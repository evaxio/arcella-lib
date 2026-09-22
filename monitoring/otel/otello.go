// Package otel:  https://opentelemetry.io/docs/languages/go/getting-started/
// https://github.com/open-telemetry/opentelemetry-go/tree/main/example/otel-collector
package otel

import (
	"context"
	crand "crypto/rand"
	"crypto/tls"
	"errors"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"time"
)

type Otello struct {
	//l          sync.Mutex
	//randSource *rand.Rand
	//
	config *ConfigOtello
	ctx    context.Context
	conn   *grpc.ClientConn
	//
	traceExporter  *otlptrace.Exporter
	tracerProvider *sdktrace.TracerProvider
	metricExporter *otlpmetricgrpc.Exporter
	meterProvider  *sdkmetric.MeterProvider
	logExporter    *otlploggrpc.Exporter
	loggerProvider *sdklog.LoggerProvider
	//
	inited bool
	//
	tracer trace.Tracer
	meter  metric.Meter
	logger *slog.Logger
}

func NewOtello(config *ConfigOtello) *Otello {
	//o := Otello{config: config}
	//var rngSeed int64
	//_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	//o.randSource = rand.New(rand.NewSource(rngSeed))
	//return &o
	return &Otello{config: config}
}

func (o *Otello) GetName() string {
	return "otel"
}

func (o *Otello) Start(ctx context.Context) (err error) {
	slog.Debug("Waiting for connection...")
	if o.conn != nil {
		if cerr := o.conn.Close(); cerr != nil {
			slog.Warn("Error closing otel connection", "error", cerr)
		}
		o.conn = nil
	}
	o.ctx = ctx
	conn, err := o.initConn()
	if err != nil {
		slog.Error("Failed to initialize otel", slog.String("message", err.Error()))
		return err
	}
	o.conn = conn
	slog.Debug("Connected to otel")
	var res *resource.Resource

	i := 1
	// UnqServiceName
	if o.config.UnqServiceName != "" {
		i = i + 1
	}
	attributes := make([]attribute.KeyValue, len(o.config.Labels)+i)
	attributes[0] = semconv.ServiceNameKey.String(o.config.ServiceName)
	if o.config.UnqServiceName != "" {
		attributes[0] = attribute.Key("service.unique.name").String(o.config.UnqServiceName)
	}
	for k, v := range o.config.Labels {
		// slog.Debug("label", slog.String(k, v))
		attributes[i] = attribute.String(k, v)
		i = i + 1
	}
	// slog.Debug("attributes", slog.Any("attributes", attributes))
	if res, err = resource.New(o.ctx,
		resource.WithHost(), // ???
		// resource.WithFromEnv(),
		// resource.WithTelemetrySDK(),
		resource.WithAttributes(attributes...),
	); err != nil {
		return err
	}

	// slog.Debug("ready to init tracer provider")
	if err = o.initTracerProvider(res, conn); err != nil {
		return err
	}
	// slog.Debug("ready to init logger provider")
	if err = o.initLoggerProvider(res, conn); err != nil {
		return err
	}
	if !o.config.WithoutPrometheus {
		// slog.Debug("ready to init meter provider")
		if err = o.initMeterProvider(res, conn); err != nil {
			return err
		}
	}

	o.inited = true
	o.tracer = otel.Tracer(o.config.ServiceName)
	o.logger = otelslog.NewLogger(o.config.ServiceName)
	if !o.config.WithoutPrometheus {
		o.meter = otel.Meter(o.config.ServiceName)
		slog.Debug("ready to observe")
	} else {
		slog.Debug("tracer & logger are ready")
	}
	return nil
}

func (o *Otello) Stop() error {
	var resultError error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if o.tracerProvider != nil {
		if err := o.tracerProvider.Shutdown(ctx); err != nil {
			resultError = errors.Join(resultError, err)
		}
	}
	if o.loggerProvider != nil {
		if err := o.loggerProvider.Shutdown(ctx); err != nil {
			resultError = errors.Join(resultError, err)
		}
	}
	if !o.config.WithoutPrometheus && o.meterProvider != nil {
		if err := o.meterProvider.Shutdown(ctx); err != nil {
			resultError = errors.Join(resultError, err)
		}
	}
	if o.conn != nil {
		if err := o.conn.Close(); err != nil {
			resultError = errors.Join(resultError, err)
		}
		o.conn = nil
	}
	return resultError
}

func (o *Otello) initConn() (*grpc.ClientConn, error) {
	// slog.Debug("Init connection", slog.String("url", o.config.Url))
	creds := insecure.NewCredentials()
	if o.config.TLS {
		creds = credentials.NewTLS(&tls.Config{})
	}
	return grpc.NewClient(o.config.Url, grpc.WithTransportCredentials(creds))
}

// Initializes an OTLP exporter, and configures the corresponding trace provider.
func (o *Otello) initTracerProvider(res *resource.Resource, conn *grpc.ClientConn) (err error) {
	// slog.Debug("initTracerProvider", slog.String("state", conn.GetState().String()))
	ratio := o.config.SampleRatio
	if ratio <= 0 {
		ratio = 1.0
	}
	if o.traceExporter, err = otlptracegrpc.New(o.ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithGRPCConn(conn)); err == nil {
		bsp := sdktrace.NewBatchSpanProcessor(o.traceExporter)
		o.tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(bsp),
		) // Set global propagator to tracecontext (the default is no-op).
		otel.SetTracerProvider(o.tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
		return nil
	} else {
		return err
	}
}

// Initializes an OTLP exporter, and configures the corresponding meter provider.
func (o *Otello) initMeterProvider(res *resource.Resource, conn *grpc.ClientConn) (err error) {
	// slog.Debug("initMeterProvider", slog.String("state", conn.GetState().String()))
	if o.metricExporter, err = otlpmetricgrpc.New(o.ctx, otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithGRPCConn(conn),
	); err == nil {
		o.meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(o.metricExporter)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(o.meterProvider)
		return nil
	} else {
		return err
	}
}

func (o *Otello) initLoggerProvider(res *resource.Resource, conn *grpc.ClientConn) (err error) {
	// slog.Debug("initLoggerProvider", slog.String("state", conn.GetState().String()))
	if o.logExporter, err = otlploggrpc.New(o.ctx, otlploggrpc.WithInsecure(), otlploggrpc.WithGRPCConn(conn)); err == nil {
		bsp := sdklog.NewBatchProcessor(o.logExporter,
			sdklog.WithMaxQueueSize(2048),
			sdklog.WithExportMaxBatchSize(512),
			sdklog.WithExportInterval(1*time.Second),
			sdklog.WithExportTimeout(1*time.Second),
		)
		o.loggerProvider = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(bsp),
		)
		global.SetLoggerProvider(o.loggerProvider)
		return nil
	} else {
		return err
	}
}

func (o *Otello) GetTracer() trace.Tracer {
	return o.tracer
}

func (o *Otello) GetLogger() *slog.Logger {
	return o.logger
}

func (o *Otello) GetMeter() metric.Meter {
	return o.meter
}

func (o *Otello) Inited() bool {
	return o.inited
}

func (o *Otello) ChildCtx(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return o.tracer.Start(ctx, spanName)
}

func (o *Otello) GetFromRemoteWithTraceID(traceIDString string, remote bool) (context.Context, error) {
	if tid, err := trace.TraceIDFromHex(traceIDString); err == nil {
		ctx := trace.ContextWithRemoteSpanContext(context.Background(),
			trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    tid,
				SpanID:     o.NewSpanID(),
				TraceFlags: trace.FlagsSampled,
				Remote:     remote,
			}))
		return ctx, err
	} else {
		return nil, err
	}
}

func (o *Otello) NewSpanID() trace.SpanID {
	var sid trace.SpanID
	for {
		if _, err := crand.Read(sid[:]); err == nil && sid.IsValid() {
			return sid
		}
	}
}

/*
	func (o *Otello) GetTraceIdFromSpanCtx(ctx context.Context) string {
		carrier := propagation.MapCarrier{}
		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
		propagator.Inject(ctx, carrier)
		return carrier.Get(TraceParentName)
	}

	func (o *Otello) GetChildSpanFromTraceParent(traceId, serviceName string) (context.Context, trace.Span) {
		carrier := propagation.MapCarrier{TraceParentName: traceId}
		p := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
		parentCtx := p.Extract(context.Background(), carrier)
		return o.ChildCtx(parentCtx, serviceName)
	}

	func (o *Otello) GetFromRemoteWithTraceIDAndSpanId(traceIDString, spanIdString string) (context.Context, error) {
		if tid, err := trace.TraceIDFromHex(traceIDString); err == nil {
			if sid, err := trace.SpanIDFromHex(spanIdString); err != nil {
				return nil, err
			} else {
				ctx := trace.ContextWithRemoteSpanContext(context.Background(),
					trace.NewSpanContext(trace.SpanContextConfig{
						TraceID:    tid,
						SpanID:     sid,
						TraceFlags: trace.FlagsSampled,
						Remote:     true,
					}))
				return ctx, err
			}
		} else {
			return nil, err
		}
	}

	func (o *Otello) NewIDs() (trace.TraceID, trace.SpanID) {
		o.l.Lock()
		var randSource *rand.Rand
		defer o.l.Unlock()
		tid := trace.TraceID{}
		sid := trace.SpanID{}
		for {
			_, _ = randSource.Read(tid[:])
			if tid.IsValid() {
				break
			}
		}
		for {
			_, _ = randSource.Read(sid[:])
			if sid.IsValid() {
				break
			}
		}
		return tid, sid
	}

// NewSpanIDRand returns a non-zero span ID from a randomly-chosen sequence.

	func (o *Otello) NewSpanIDRand() trace.SpanID {
		o.l.Lock()
		defer o.l.Unlock()
		sid := trace.SpanID{}
		for {
			_, _ = o.randSource.Read(sid[:])
			if sid.IsValid() {
				break
			}
		}
		return sid
	}
*/
