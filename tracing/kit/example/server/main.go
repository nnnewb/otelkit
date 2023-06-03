package main

import (
	"context"
	"log"
	"net/http"

	khttp "github.com/go-kit/kit/transport/http"
	"github.com/nnnewb/otelkit/tracing/kit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	url     = "http://192.168.56.4:14268/api/traces"
	service = "kit-http-app"
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
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		log.Fatal(err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(service),
		)),
	)

	svr := khttp.NewServer(
		endpoint,
		decodeExampleRequest,
		khttp.EncodeJSONResponse,
		kit.TraceServerBefore(tp.Tracer("http-example"), otel.GetTextMapPropagator()),
		kit.TraceServerAfter(),
		kit.TraceServerFinalizer())

	http.DefaultServeMux.Handle("/hello", svr)
	log.Println("server start listen at http://127.0.0.1:9998")
	err = http.ListenAndServe("127.0.0.1:9998", http.DefaultServeMux)
	if err != nil {
		log.Fatal(err)
	}
}
