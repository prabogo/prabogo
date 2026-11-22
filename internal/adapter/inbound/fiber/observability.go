package fiber_inbound_adapter

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"prabogo/utils/activity"
	"prabogo/utils/metrics"
)

const (
	otelCtxKey           = "otelCtx"
	tracerNameHTTPServer = "prabogo.http.server"
	tracerNameHTTPDomain = "prabogo.domain"
)

func isMiddlewareRouteLabel(r *fiber.Route, requestPath string) bool {
	if r == nil || r.Path == "" {
		return false
	}
	return r.Path != requestPath && strings.HasPrefix(requestPath, r.Path)
}

func observabilityMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		ctx := otel.GetTextMapPropagator().Extract(c.Context(), &fiberHeaderCarrier{c: c})

		tracer := otel.Tracer(tracerNameHTTPServer)
		spanCtx, span := tracer.Start(ctx, "HTTP "+c.Method(),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Method()),
				semconv.URLOriginal(c.OriginalURL()),
				semconv.URLPath(c.Path()),
				semconv.UserAgentOriginal(c.Get("User-Agent")),
				semconv.NetworkPeerAddressKey.String(c.IP()),
			),
		)
		defer span.End()

		c.Locals(otelCtxKey, spanCtx)

		err := c.Next()

		route := c.Route().Path
		if route == "" || isMiddlewareRouteLabel(c.Route(), c.Path()) {
			route = c.Path()
		}

		status := c.Response().StatusCode()

		// Batch all post-handler attributes into a single SetAttributes call
		// to minimize span lock acquisitions.
		span.SetName("HTTP " + c.Method() + " " + route)
		span.SetAttributes(
			semconv.HTTPRouteKey.String(route),
			semconv.HTTPResponseStatusCodeKey.Int(status),
		)
		if status >= 400 {
			span.SetStatus(codes.Error, "HTTP request failed")
		} else {
			span.SetStatus(codes.Ok, "")
		}

		duration := time.Since(start).Seconds()
		metrics.GetHTTPServer().RecordHTTPRequest(spanCtx, c.Method(), route, status, duration)

		return err
	}
}

func otelContextFromFiber(c *fiber.Ctx) context.Context {
	if v, ok := c.Locals(otelCtxKey).(context.Context); ok && v != nil {
		return v
	}
	return c.Context()
}

// fiberHeaderCarrier adapts fiber.Ctx to propagation.TextMapCarrier so the
// OTel propagator can extract trace context from all incoming headers
// (traceparent, tracestate, baggage) rather than a single hardcoded header.
type fiberHeaderCarrier struct {
	c *fiber.Ctx
}

func (f *fiberHeaderCarrier) Get(key string) string {
	return f.c.Get(key)
}

func (f *fiberHeaderCarrier) Set(key, value string) {
	f.c.Set(key, value)
}

func (f *fiberHeaderCarrier) Keys() []string {
	// Return nil — the propagator only uses Get for extraction.
	return nil
}

func newRequestContext(c *fiber.Ctx, action string) context.Context {
	return activity.NewContext(action, otelContextFromFiber(c))
}

func startDomainSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerNameHTTPDomain)
	return tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal))
}

func addSpanAttribute(c *fiber.Ctx, key string, value string) {
	if span := trace.SpanFromContext(otelContextFromFiber(c)); span.IsRecording() {
		span.SetAttributes(attribute.String(key, value))
	}
}
