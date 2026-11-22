package otel

import (
	"context"
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

const (
	DefaultEndpoint = "127.0.0.1:4317"
)

type Options struct {
	Enabled  bool
	Endpoint string
}

func enabled(signal string) bool {
	return os.Getenv("OTEL_"+signal+"_ENABLED") == "true"
}

func endpoint(signal string) string {
	if v := os.Getenv("OTEL_" + signal + "_ENDPOINT"); v != "" {
		return v
	}
	return DefaultEndpoint
}

func newResource() (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName()),
			semconv.ServiceVersionKey.String(serviceVersion()),
			semconv.DeploymentEnvironmentKey.String(serviceEnvironment()),
		),
	)
}

func serviceName() string {
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		return v
	}
	return "prabogo"
}

func serviceVersion() string {
	if v := os.Getenv("OTEL_SERVICE_VERSION"); v != "" {
		return v
	}
	return "1.0.0"
}

func serviceEnvironment() string {
	if v := os.Getenv("APP_MODE"); v != "" {
		return v
	}
	return "development"
}

func Setup(ctx context.Context) (func(context.Context) error, error) {
	res, err := newResource()
	if err != nil {
		return nil, err
	}

	var (
		shutdownFuncs []func(context.Context) error
		errs          []error
		flushTimeout  = 10 * time.Second
	)

	if enabled("TRACE") {
		traceExporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint("TRACE")),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			errs = append(errs, err)
		} else {
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(traceExporter),
				sdktrace.WithResource(res),
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
			)
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
		}
	}

	if enabled("METRIC") {
		metricExporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(endpoint("METRIC")),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			errs = append(errs, err)
		} else {
			mp := metric.NewMeterProvider(
				metric.WithReader(metric.NewPeriodicReader(metricExporter,
					metric.WithInterval(15*time.Second)),
				),
				metric.WithResource(res),
			)
			otel.SetMeterProvider(mp)
			shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
		}
	}

	if enabled("LOG") {
		logExporter, err := otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(endpoint("LOG")),
			otlploggrpc.WithInsecure(),
		)
		if err != nil {
			errs = append(errs, err)
		} else {
			lp := log.NewLoggerProvider(
				log.WithProcessor(log.NewBatchProcessor(logExporter)),
				log.WithResource(res),
			)
			global.SetLoggerProvider(lp)
			shutdownFuncs = append(shutdownFuncs, lp.Shutdown)
		}
	}

	return func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, flushTimeout)
		defer cancel()

		var shutdownErrs []error
		for _, fn := range shutdownFuncs {
			if err := fn(shutdownCtx); err != nil {
				shutdownErrs = append(shutdownErrs, err)
			}
		}
		return errors.Join(shutdownErrs...)
	}, errors.Join(errs...)
}
