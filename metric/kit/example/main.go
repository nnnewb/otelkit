package main

import (
	"context"
	"log"
	"net/http"

	khttp "github.com/go-kit/kit/transport/http"
	"github.com/nnnewb/otelkit/metric/kit"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

type ExampleEndpointRequest struct{}
type ExampleEndpointResponse struct {
	Msg string `json:"msg"`
}

func example(_ context.Context, _ *ExampleEndpointRequest) (*ExampleEndpointResponse, error) {
	return &ExampleEndpointResponse{
		Msg: "hello world",
	}, nil
}

func decodeExampleRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return &ExampleEndpointRequest{}, nil
}

func endpoint(ctx context.Context, req interface{}) (interface{}, error) {
	return example(ctx, req.(*ExampleEndpointRequest))
}

func main() {
	promExporter, err := prometheus.New()
	if err != nil {
		log.Fatal(err)
	}

	provider := metric.NewMeterProvider(metric.WithReader(promExporter))
	meter := provider.Meter("http-example")

	// serving /metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("prometheus scrap endpoint start serving at http://192.168.56.1:23333/metrics")
		err := http.ListenAndServe("192.168.56.1:23333", http.DefaultServeMux)
		if err != nil {
			log.Fatal(err)
		}
	}()

	svr := khttp.NewServer(
		endpoint,
		decodeExampleRequest,
		khttp.EncodeJSONResponse,
		kit.MeasureServerBefore(meter),
		kit.MeasureServerFinalizer(meter),
	)

	http.DefaultServeMux.Handle("/hello", svr)
	log.Println("server start listen at http://127.0.0.1:9998")
	err = http.ListenAndServe("127.0.0.1:9998", http.DefaultServeMux)
	if err != nil {
		log.Fatal(err)
	}
}
