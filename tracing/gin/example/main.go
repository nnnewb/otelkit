package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	gin2 "github.com/nnnewb/otelkit/tracing/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	url     = "http://192.168.56.4:14268/api/traces"
	service = "gin-app"
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		err := tp.Shutdown(ctx)
		if err != nil {
			log.Printf("shutdown returned error %+v", err)
		}
	}()

	app := gin.New()
	app.Use(gin2.TraceMiddleware(tp.Tracer("gin-example"), otel.GetTextMapPropagator()))
	app.Handle(http.MethodGet, "/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "Hello world"})
	})
	log.Println("server start listen at http://127.0.0.1:9998")
	err = http.ListenAndServe("127.0.0.1:9998", app)
	if err != nil {
		log.Fatal(err)
	}
}
