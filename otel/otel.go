package otel

import (
	"context"
	"fmt"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
	"os"
)

type Shutdown func(ctx context.Context) error

func Setup(ctx context.Context, name, version string) (Shutdown, error) {
	rs, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			otelsemconv.ServiceNameKey.String(name),
			otelsemconv.ServiceVersionKey.String(version),
		),
	)
	if err != nil {
		return nil, err
	}

	tp, err := traces(ctx, rs)
	if err != nil {
		return nil, err
	}

	mp, err := metrics(ctx, rs)
	if err != nil {
		tp.Shutdown(ctx)
		return nil, err
	}

	lp, err := logs(ctx, rs)
	if err != nil {
		tp.Shutdown(ctx)
		mp.Shutdown(ctx)
		return nil, err
	}

	shutdown := func(ctx context.Context) error {
		wg, ctx := errgroup.WithContext(ctx)
		wg.Go(func() error {
			return tp.Shutdown(ctx)
		})
		wg.Go(func() error {
			return mp.Shutdown(ctx)
		})
		wg.Go(func() error {
			return lp.Shutdown(ctx)
		})
		return wg.Wait()
	}

	err = runtime.Start()
	if err != nil {
		slog.ErrorContext(ctx, "failed to start runtime metrics", "error", err)
		shutdown(ctx)
		return nil, err
	}

	return shutdown, nil
}

func traces(ctx context.Context, rs *resource.Resource) (*trace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(rs),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

func metrics(ctx context.Context, rs *resource.Resource) (*metric.MeterProvider, error) {
	exp, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	reader := metric.NewPeriodicReader(exp, metric.WithProducer(runtime.NewProducer()))

	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(rs),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}

func logs(ctx context.Context, rs *resource.Resource) (*log.LoggerProvider, error) {
	exp, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exp)),
		log.WithResource(rs),
	)

	// Fan out to stdout as well as OTel: slog.SetDefault also redirects the standard
	// log package, so without a local handler all log output would only reach the collector.
	slog.SetDefault(slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(os.Stdout, nil),
		otelslog.NewHandler("name", otelslog.WithLoggerProvider(lp)),
	)))

	return lp, nil
}

func Trace(h http.Handler) (http.Handler, error) {
	return otelhttp.NewHandler(h, "action"), nil
}
