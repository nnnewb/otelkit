package http

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func MeasureHandler(meter metric.Meter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// throughput
		requestCounter, err := meter.Int64Counter("request-count")
		if err != nil {
			panic(err)
		}

		// request duration
		durationHistogram, err := meter.Int64Histogram("request-duration-milli")
		if err != nil {
			panic(err)
		}

		return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
			requestCounter.Add(
				req.Context(), 1,
				metric.WithAttributes(
					attribute.String("url", req.URL.String()),
					attribute.String("method", req.Method),
					attribute.String("peer", req.RemoteAddr),
				),
			)

			start := time.Now()
			next.ServeHTTP(wr, req)
			durationHistogram.Record(
				req.Context(),
				time.Since(start).Milliseconds(),
				metric.WithAttributes(
					attribute.String("url", req.URL.String()),
					attribute.String("method", req.Method),
					attribute.String("peer", req.RemoteAddr),
				),
			)
		})
	}
}
