package trace

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	zipkinot "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/reporter"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	zipkinkafka "github.com/openzipkin/zipkin-go/reporter/kafka"
)

func NewZikpin(cfg *TracerCfg, srvName string, isGlobal bool) (opentracing.Tracer, error) {
	if !cfg.IsOpen {
		return nil, nil
	}
	if len(cfg.EndpointURL) == 0 && len(cfg.Kafkas) == 0 {
		return nil, fmt.Errorf("EndpointURL or Kafkas is empty")
	}
	endpointURL := cfg.EndpointURL
	if len(srvName) == 0 {
		return nil, fmt.Errorf("create tracer server name not is null")
	}
	kfs := cfg.Kafkas
	hostURL := cfg.HostURL
	var reporter reporter.Reporter
	var err error
	if len(kfs) != 0 {
		reporter, err = zipkinkafka.NewReporter(kfs) // zipkinkafka.Producer(p),
		if err != nil {
			return nil, fmt.Errorf("unable to lint kafka endpoint: %+v", err)
		}
	} else if len(endpointURL) != 0 {
		reporter = zipkinhttp.NewReporter(endpointURL)
	}

	endpoint, err := zipkin.NewEndpoint(srvName, hostURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create local endpoint: %+v", err)
	}
	nativeTracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("unable to create tracer: %+v", err)
	}
	tracer := zipkinot.Wrap(nativeTracer)
	if isGlobal {
		opentracing.SetGlobalTracer(tracer)
	}

	return tracer, nil
}
