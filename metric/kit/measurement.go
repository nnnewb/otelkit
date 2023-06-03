package kit

import (
	"context"
	"net/http"
	"time"

	khttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type startTimeKeyT struct{}
type requestKeyT struct{}

var startTimeKey startTimeKeyT
var requestKey requestKeyT

func MeasureServerBefore(meter metric.Meter) khttp.ServerOption {
	// throughput
	requestCounter, err := meter.Int64Counter("request-count")
	if err != nil {
		panic(err)
	}

	return khttp.ServerBefore(func(ctx context.Context, request *http.Request) context.Context {
		requestCounter.Add(
			request.Context(), 1,
			metric.WithAttributes(
				attribute.String("url", request.URL.String()),
				attribute.String("method", request.Method),
				attribute.String("peer", request.RemoteAddr),
			),
		)

		ctx = context.WithValue(ctx, startTimeKey, time.Now())
		ctx = context.WithValue(ctx, requestKey, request)
		return ctx
	})
}

func MeasureServerFinalizer(meter metric.Meter) khttp.ServerOption {
	// request duration
	durationHistogram, err := meter.Int64Histogram("request-duration-milli")
	if err != nil {
		panic(err)
	}

	return khttp.ServerFinalizer(func(ctx context.Context, code int, r *http.Request) {
		if req, ok := ctx.Value(requestKey).(*http.Request); ok {
			if start, ok := ctx.Value(startTimeKey).(time.Time); ok {
				durationHistogram.Record(
					req.Context(),
					time.Since(start).Milliseconds(),
					metric.WithAttributes(
						attribute.String("url", req.URL.String()),
						attribute.String("method", req.Method),
						attribute.String("peer", req.RemoteAddr),
					),
				)
			}
		}
	})
}
