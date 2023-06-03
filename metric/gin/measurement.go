package gin

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func MeasureHandleFunc(meter metric.Meter) gin.HandlerFunc {
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

	return func(c *gin.Context) {
		req := c.Request

		requestCounter.Add(
			req.Context(), 1,
			metric.WithAttributes(
				attribute.String("url", req.URL.String()),
				attribute.String("method", req.Method),
				attribute.String("peer", req.RemoteAddr),
			),
		)
		start := time.Now()
		defer func() {
			durationHistogram.Record(
				req.Context(),
				time.Since(start).Milliseconds(),
				metric.WithAttributes(
					attribute.String("url", req.URL.String()),
					attribute.String("method", req.Method),
					attribute.String("peer", req.RemoteAddr),
				),
			)
		}()

		c.Next()
	}
}
