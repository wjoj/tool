package monitoring

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

// 添加 Prometheus 指标定义
var (
	redisConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "redis_connections_active",
		Help: "Current number of active connections in the pool",
	})

	redisConnectionsIdle = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "redis_connections_idle",
		Help: "Current number of idle connections in the pool",
	})

	redisCommandsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_commands_total",
		Help: "Total number of Redis commands executed",
	}, []string{"command"})

	redisCommandDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "redis_command_duration_seconds",
		Help:    "Duration of Redis commands execution",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"command"})
	redisHealthMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "redis_health_status",
		Help: "Redis health status (1 = healthy, 0 = unhealthy)",
	})
	redisLatencyMetric = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "redis_latency_seconds",
		Help:    "Redis command latency in seconds",
		Buckets: prometheus.DefBuckets,
	})
)

// 添加指标收集方法
func collectMetrics(stats *redis.PoolStats) {
	if stats != nil {
		redisConnectionsActive.Set(float64(stats.TotalConns - stats.IdleConns))
		redisConnectionsIdle.Set(float64(stats.IdleConns))
	}
}

// 包装命令执行以收集指标
func instrumentedCommand(ctx context.Context, cmd string, fn func() error) error {
	start := time.Now()
	defer func() {
		redisCommandDuration.WithLabelValues(cmd).Observe(time.Since(start).Seconds())
		redisCommandsTotal.WithLabelValues(cmd).Inc()
	}()
	return fn()
}

// 添加健康检查方法
func CheckHealth(rd redis.Cmdable, ctx context.Context) bool {
	start := time.Now()
	err := rd.Ping(ctx).Err()
	duration := time.Since(start).Seconds()

	redisLatencyMetric.Observe(duration)

	if err != nil {
		redisHealthMetric.Set(0)
		return false
	}

	redisHealthMetric.Set(1)
	return true
}

// 添加 Prometheus 指标收集器
func RegisterMetrics(registry prometheus.Registerer) {
	registry.MustRegister(redisHealthMetric)
	registry.MustRegister(redisLatencyMetric)
}
