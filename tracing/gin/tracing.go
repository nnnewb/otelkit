package gin

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TraceMiddleware(tracer trace.Tracer, propagator propagation.TextMapPropagator) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request
		ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))
		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", req.Method, req.URL.String()))
		defer span.End()
		defer func() {
			wr := c.Writer
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
		c.Set("span", span)
		c.Next()
	}
}
