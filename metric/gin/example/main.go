package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	gin2 "github.com/nnnewb/otelkit/metric/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

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

	app := gin.New()
	app.Use(gin2.MeasureHandleFunc(meter))
	app.Handle(http.MethodGet, "/hello", func(c *gin.Context) {
		c.JSON(200, "Hello world")
	})

	log.Println("server start listen at http://127.0.0.1:9998")
	err = http.ListenAndServe("127.0.0.1:9998", app)
	if err != nil {
		log.Fatal(err)
	}
}
