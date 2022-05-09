package trace

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	tracerLog "github.com/opentracing/opentracing-go/log"
)

//redisDB.AddHook(trace.TracingHook{})

type TracingHook struct{}

type RedisSpanKey string

const _RedisSpan RedisSpanKey = "_RedisSpan"

var _ redis.Hook = TracingHook{}

func (TracingHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {

	if !opentracing.IsGlobalTracerRegistered() {
		return ctx, nil
	}

	operationName := fmt.Sprintf("Redis - %s", cmd.Name())
	span, _ := opentracing.StartSpanFromContext(ctx, operationName)

	span.SetTag(string(ext.DBType), "redis")
	span.SetTag("redis.name", cmd.Name())
	span.SetTag("redis.full_name", cmd.FullName())
	span.LogKV("redis.cmd", cmd.String(), "redis.args", cmd.Args())

	withValueCtx := context.WithValue(ctx, _RedisSpan, span)

	return withValueCtx, nil

}
func (TracingHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	v := ctx.Value(_RedisSpan)
	if span, ok := v.(opentracing.Span); ok {
		defer span.Finish()

		if err := cmd.Err(); err != nil && !errors.Is(err, redis.Nil) {
			ext.Error.Set(span, true)
			span.LogFields(tracerLog.Error(cmd.Err()))
		}

	}

	return nil
}

func (TracingHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {

	if !opentracing.IsGlobalTracerRegistered() {
		return ctx, nil
	}

	operationName := fmt.Sprintf("Redis - %s", "Pipeline")
	span, _ := opentracing.StartSpanFromContext(ctx, operationName)

	cmdMap := make(map[int]string)
	for index, cmd := range cmds {
		cmdMap[index] = cmd.String()
	}

	span.SetTag(string(ext.DBType), "redis")
	span.LogKV("Pipeline", cmdMap)

	withValueCtx := context.WithValue(ctx, _RedisSpan, span)

	return withValueCtx, nil

}

func (TracingHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	v := ctx.Value(_RedisSpan)
	if span, ok := v.(opentracing.Span); ok {
		defer span.Finish()

		for _, cmd := range cmds {
			if err := cmd.Err(); err != nil {
				ext.Error.Set(span, true)
				span.LogFields(tracerLog.Error(cmd.Err()))
			}
		}

	}

	return nil

}
