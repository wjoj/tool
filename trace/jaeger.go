package trace

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
)

func NewJaeger(c *TracerCfg, srvName string, isGlobal bool) (opentracing.Tracer, error) {
	if !c.IsOpen {
		return nil, nil
	}
	if len(c.EndpointURL) == 0 {
		return nil, fmt.Errorf("EndpointURL is empty")
	}
	cfg := &config.Configuration{
		ServiceName: srvName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LocalAgentHostPort: c.EndpointURL,
			LogSpans:           true,
		},
	}
	tracer, _, err := cfg.NewTracer(config.Logger(jaeger.StdLogger), config.Metrics(metrics.NullFactory))
	if err != nil {
		return nil, err
	}
	if isGlobal {
		opentracing.SetGlobalTracer(tracer)
	}
	return tracer, nil
}
