package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TraceHandler(tracer trace.Tracer, propagator propagation.TextMapPropagator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))
			ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", req.Method, req.URL.String()))
			defer span.End()
			defer func() {
				var attrs = make([]attribute.KeyValue, 0, len(wr.Header()))
				for key, values := range wr.Header() {
					attrs = append(attrs, attribute.String("http.response.header."+key, strings.Join(values, "\n")))
				}
				span.SetAttributes(attrs...)
			}()

			var attrs []attribute.KeyValue
			for key, values := range req.Header {
				attrs = append(attrs, attribute.String("http.request.header."+key, strings.Join(values, "\n")))
			}
			span.SetAttributes(
				attribute.Int64("http.request_content_length", req.ContentLength),
				attribute.String("http.method", req.Method),
				attribute.String("net.protocol.name", "http"),
				attribute.String("net.protocol.version", req.Proto),
				attribute.String("net.sock.peer.addr", req.RemoteAddr),
				attribute.String("user_agent.original", req.Header.Get("User-Agent")))
			span.SetAttributes(attrs...)
			req = req.WithContext(ctx)
			next.ServeHTTP(wr, req)
		})
	}
}

func TraceRequest(ctx context.Context, propagator propagation.TextMapPropagator, req *http.Request) {
	injectHttpHeader(ctx, propagator, req.Header)
	span := trace.SpanFromContext(ctx)
	var attrs []attribute.KeyValue
	for key, values := range req.Header {
		attrs = append(attrs, attribute.String("http.request.header."+key, strings.Join(values, "\n")))
	}
	span.SetAttributes(
		attribute.Int64("http.request_content_length", req.ContentLength),
		attribute.String("http.method", req.Method),
		attribute.String("net.protocol.name", "http"),
		attribute.String("net.protocol.version", req.Proto),
		attribute.String("net.sock.peer.addr", req.RemoteAddr),
		attribute.String("user_agent.original", req.Header.Get("User-Agent")))
	span.SetAttributes(attrs...)
}

func injectHttpHeader(ctx context.Context, propagator propagation.TextMapPropagator, header http.Header) {
	propagator.Inject(ctx, propagation.HeaderCarrier(header))
}

func injectMap(ctx context.Context, propagator propagation.TextMapPropagator, m map[string]string) {
	propagator.Inject(ctx, propagation.MapCarrier(m))
}
