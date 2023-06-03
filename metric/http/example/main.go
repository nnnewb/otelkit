package main

import (
	"io"
	"log"
	"net/http"

	http2 "github.com/nnnewb/otelkit/metric/http"
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

	var handler http.Handler = http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(req.Body)

		_, err := wr.Write([]byte("Hello world"))
		if err != nil {
			log.Fatal(err)
		}
	})

	handler = http2.MeasureHandler(meter)(handler)
	log.Println("server start listen at http://127.0.0.1:9998")
	http.Handle("/hello", handler)
	err = http.ListenAndServe("127.0.0.1:9998", http.DefaultServeMux)
	if err != nil {
		log.Fatal(err)
	}
}
