package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	http2 "github.com/nnnewb/otelkit/tracing/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	url     = "http://192.168.56.4:14268/api/traces"
	service = "http-client-app"
)

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
	defer func() {
		err := tp.Shutdown(context.Background())
		if err != nil {
			log.Printf("shutdown failed, error %+v", err)
		}
	}()

	tr := tp.Tracer(service)

	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:9998/hello", nil)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
	defer cancel()
	ctx, span := tr.Start(ctx, fmt.Sprintf("client %s %s", request.Method, request.URL.String()))
	defer span.End()
	request = request.WithContext(ctx)

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

	// request.Header = http.Header{}
	http2.TraceRequest(ctx, otel.GetTextMapPropagator(), request)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}

	var attrs = make([]attribute.KeyValue, 0, len(response.Header))
	for key, values := range response.Header {
		attrs = append(attrs, attribute.String("http.response.header."+key, strings.Join(values, "\n")))
	}
	span.SetAttributes(attrs...)
	_, err = io.Copy(os.Stdout, response.Body)
	if err != nil {
		panic(err)
	}
}
