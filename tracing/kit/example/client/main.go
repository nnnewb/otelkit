package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	khttp "github.com/go-kit/kit/transport/http"
	"github.com/nnnewb/otelkit/tracing/kit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	jaegerUrl = "http://192.168.56.4:14268/api/traces"
	service   = "kit-client-app"
)

type ExampleEndpointRequest struct{}
type ExampleEndpointResponse struct {
	Msg string `json:"msg"`
}

func encodeExampleRequest(_ context.Context, request *http.Request, _ interface{}) error {
	request.Body = nil
	return nil
}

func decodeExampleResponse(_ context.Context, response *http.Response) (interface{}, error) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("close body failed, error %+v", err)
		}
	}(response.Body)
	var ret ExampleEndpointResponse
	err := json.NewDecoder(response.Body).Decode(&ret)
	return ret, err
}

func main() {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerUrl)))
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
	defer func() {
		err := tp.Shutdown(context.Background())
		if err != nil {
			log.Printf("shutdown failed, error %+v", err)
		}
	}()

	tr := tp.Tracer(service)
	endpointUrl, _ := url.Parse("http://127.0.0.1:9998/hello")
	client := khttp.NewClient(
		http.MethodGet,
		endpointUrl,
		encodeExampleRequest,
		decodeExampleResponse,
		kit.TraceClientBefore(tr, otel.GetTextMapPropagator()),
		kit.TraceClientAfter(),
		kit.TraceClientFinalizer(),
	)

	client.Endpoint()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()

	ctx, span := tr.Start(ctx, fmt.Sprintf("call endpoint %s", endpointUrl))
	defer span.End()

	b, err := baggage.New()
	if err != nil {
		panic(err)
	}

	member, err := baggage.NewMember("Hello", "world")
	if err != nil {
		panic(err)
	}

	b, err = b.SetMember(member)
	if err != nil {
		panic(err)
	}

	ctx = baggage.ContextWithBaggage(ctx, b)

	resp, err := client.Endpoint()(ctx, &ExampleEndpointRequest{})
	if err != nil {
		panic(err)
	}
	log.Printf("%#v", resp)
}
