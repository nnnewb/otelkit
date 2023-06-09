package kit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	khttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TraceServerBefore(tr trace.Tracer, propagator propagation.TextMapPropagator) khttp.ServerOption {
	return khttp.ServerBefore(func(ctx context.Context, request *http.Request) context.Context {
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(request.Header))
		ctx, span := tr.Start(ctx, request.URL.Path)
		var attrs []attribute.KeyValue
		for key, values := range request.Header {
			attrs = append(attrs, attribute.String("http.request.header."+key, strings.Join(values, "\n")))
		}
		span.SetAttributes(
			attribute.Int64("http.request_content_length", request.ContentLength),
			attribute.String("http.method", request.Method),
			attribute.String("net.protocol.name", "http"),
			attribute.String("net.protocol.version", request.Proto),
			attribute.String("net.sock.peer.addr", request.RemoteAddr),
			attribute.String("user_agent.original", request.Header.Get("User-Agent")))
		span.SetAttributes(attrs...)
		return ctx
	})
}

func TraceServerAfter() khttp.ServerOption {
	return khttp.ServerAfter(func(ctx context.Context, wr http.ResponseWriter) context.Context {
		span := trace.SpanFromContext(ctx)
		var attrs []attribute.KeyValue
		for key, values := range wr.Header() {
			attrs = append(attrs, attribute.String("http.response.header."+key, strings.Join(values, "\n")))
		}
		span.SetAttributes(attrs...)
		return ctx
	})
}

func TraceServerFinalizer() khttp.ServerOption {
	return khttp.ServerFinalizer(func(ctx context.Context, code int, req *http.Request) {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.Int("http.status_code", code))
		if span != nil {
			span.End()
		}
	})
}

func TraceClientBefore(tr trace.Tracer, propagator propagation.TextMapPropagator) khttp.ClientOption {
	return khttp.ClientBefore(func(ctx context.Context, request *http.Request) context.Context {
		ctx, span := tr.Start(ctx, request.Method+" "+request.URL.String())
		var attrs []attribute.KeyValue
		for key, values := range request.Header {
			attrs = append(attrs, attribute.String("http.request.header."+key, strings.Join(values, "\n")))
		}
		span.SetAttributes(attrs...)
		var port int
		portStr := request.URL.Port()
		if portStr != "" {
			port, _ = strconv.Atoi(portStr)
		}
		span.SetAttributes(
			attribute.String("http.method", request.Method),
			attribute.String("http.flavor", fmt.Sprintf("%d.%d", request.ProtoMajor, request.ProtoMinor)),
			attribute.String("http.url", request.URL.String()),
			attribute.String("net.sock.peer.name", request.URL.Hostname()),
			attribute.Int("net.sock.peer.port", port))

		propagator.Inject(ctx, propagation.HeaderCarrier(request.Header))

		return ctx
	})
}

func TraceClientAfter() khttp.ClientOption {
	return khttp.ClientAfter(func(ctx context.Context, response *http.Response) context.Context {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.Int("http.status_code", response.StatusCode))
		var attrs = make([]attribute.KeyValue, 0, len(response.Header))
		for key, values := range response.Header {
			attrs = append(attrs, attribute.String("http.response.header."+key, strings.Join(values, "\n")))
		}
		span.SetAttributes(attrs...)
		return ctx
	})
}

func TraceClientFinalizer() khttp.ClientOption {
	return khttp.ClientFinalizer(func(ctx context.Context, err error) {
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
		span.End()
	})
}
