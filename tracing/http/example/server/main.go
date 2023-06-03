package main

import (
	"log"
	"net/http"

	http2 "github.com/nnnewb/otelkit/tracing/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	url     = "http://192.168.56.4:14268/api/traces"
	service = "http-app"
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

	var handler = http.Handler(http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
		_, _ = wr.Write([]byte("Hello world!"))
	}))
	handler = http2.TraceHandler(tp.Tracer("http-example"), otel.GetTextMapPropagator())(handler)
	http.DefaultServeMux.Handle("/hello", handler)
	log.Println("server start listen at http://127.0.0.1:9998")
	err = http.ListenAndServe("127.0.0.1:9998", http.DefaultServeMux)
	if err != nil {
		log.Fatal(err)
	}
}
