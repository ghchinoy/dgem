package cmd

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const gatewayTracerName = "github.com/ghchinoy/dgem/gateway"

// TraceSpanRecord represents a serialized OpenTelemetry span for API responses & /api/traces.
type TraceSpanRecord struct {
	TraceID      string                 `json:"trace_id"`
	SpanID       string                 `json:"span_id"`
	ParentSpanID string                 `json:"parent_span_id,omitempty"`
	Name         string                 `json:"name"`
	StartTime    string                 `json:"start_time"`
	EndTime      string                 `json:"end_time"`
	DurationMs   float64                `json:"duration_ms"`
	Status       string                 `json:"status"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

var (
	traceRingMu   sync.Mutex
	traceRing     []TraceSpanRecord
	maxTraceItems = 250
	gcpProjectID  string
)

func detectGCPProjectID() string {
	for _, env := range []string{"GOOGLE_CLOUD_PROJECT", "GCP_PROJECT", "GCLOUD_PROJECT"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
	}
	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
	if err == nil {
		req.Header.Set("Metadata-Flavor", "Google")
		hc := &http.Client{Timeout: 800 * time.Millisecond}
		if resp, err := hc.Do(req); err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if b, err := io.ReadAll(resp.Body); err == nil {
					if p := strings.TrimSpace(string(b)); p != "" {
						return p
					}
				}
			}
		}
	}
	return ""
}

type gatewaySpanProcessor struct {
	projectID string
}

func (p *gatewaySpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {}

func (p *gatewaySpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	sc := s.SpanContext()
	traceID := sc.TraceID().String()
	spanID := sc.SpanID().String()
	parentID := ""
	if s.Parent().IsValid() {
		parentID = s.Parent().SpanID().String()
	}

	durMs := float64(s.EndTime().Sub(s.StartTime()).Microseconds()) / 1000.0
	attrs := make(map[string]interface{}, len(s.Attributes()))
	for _, kv := range s.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsInterface()
	}

	rec := TraceSpanRecord{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentID,
		Name:         s.Name(),
		StartTime:    s.StartTime().UTC().Format(time.RFC3339Nano),
		EndTime:      s.EndTime().UTC().Format(time.RFC3339Nano),
		DurationMs:   durMs,
		Status:       s.Status().Code.String(),
		Attributes:   attrs,
	}

	traceRingMu.Lock()
	traceRing = append(traceRing, rec)
	if len(traceRing) > maxTraceItems {
		traceRing = traceRing[len(traceRing)-maxTraceItems:]
	}
	traceRingMu.Unlock()

	// Emit Cloud Run structured JSON log entry correlated with Cloud Trace
	logEntry := map[string]interface{}{
		"severity":    "INFO",
		"message":     fmt.Sprintf("[OTel Span] %s completed in %.2f ms", s.Name(), durMs),
		"span_name":   s.Name(),
		"duration_ms": durMs,
		"attributes":  attrs,
	}
	if p.projectID != "" && sc.TraceID().IsValid() {
		logEntry["logging.googleapis.com/trace"] = fmt.Sprintf("projects/%s/traces/%s", p.projectID, traceID)
		logEntry["logging.googleapis.com/spanId"] = spanID
		logEntry["logging.googleapis.com/trace_sampled"] = sc.IsSampled()
	}
	if b, err := json.Marshal(logEntry); err == nil {
		fmt.Fprintln(os.Stdout, string(b))
	}
}

func (p *gatewaySpanProcessor) Shutdown(ctx context.Context) error   { return nil }
func (p *gatewaySpanProcessor) ForceFlush(ctx context.Context) error { return nil }

// initGatewayTracer initializes the OpenTelemetry TracerProvider with Google Cloud Trace export
// (when on GCP) and the in-memory + Cloud Logging span processor.
func initGatewayTracer(ctx context.Context) func(context.Context) error {
	gcpProjectID = detectGCPProjectID()

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("dgemma-gateway"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("gcp.project_id", gcpProjectID),
		),
	)

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(&gatewaySpanProcessor{projectID: gcpProjectID}),
	}

	if gcpProjectID != "" {
		if exporter, err := cloudtrace.New(cloudtrace.WithProjectID(gcpProjectID)); err == nil {
			opts = append(opts, sdktrace.WithBatcher(exporter))
		}
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp.Shutdown
}

func gatewayTracer() trace.Tracer {
	return otel.Tracer(gatewayTracerName)
}

// extractTraceContextFromRequest extracts W3C traceparent or Google Cloud Run X-Cloud-Trace-Context
// so all gateway and dgemma model spans attach to the incoming Cloud Run request trace.
func extractTraceContextFromRequest(r *http.Request) context.Context {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	if trace.SpanContextFromContext(ctx).IsValid() {
		return ctx
	}
	// Fallback: parse Google Cloud Run X-Cloud-Trace-Context: TRACE_ID/SPAN_ID_DEC;o=1
	xctc := strings.TrimSpace(r.Header.Get("X-Cloud-Trace-Context"))
	if xctc == "" {
		return ctx
	}
	slashIdx := strings.Index(xctc, "/")
	if slashIdx != 32 {
		return ctx
	}
	traceHex := xctc[:32]
	rest := xctc[33:]
	semiIdx := strings.Index(rest, ";")
	spanDecStr := rest
	if semiIdx >= 0 {
		spanDecStr = rest[:semiIdx]
	}
	tID, err := trace.TraceIDFromHex(traceHex)
	if err != nil {
		return ctx
	}
	var sID trace.SpanID
	if spanUint, err := strconv.ParseUint(spanDecStr, 10, 64); err == nil && spanUint > 0 {
		binary.BigEndian.PutUint64(sID[:], spanUint)
	} else {
		_, _ = hex.Decode(sID[:], []byte("0000000000000001"))
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tID,
		SpanID:     sID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

// injectTraceContextToRequest propagates W3C traceparent and X-Cloud-Trace-Context to upstream dgemma.
func injectTraceContextToRequest(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		spanIDBytes := sc.SpanID()
		spanDec := binary.BigEndian.Uint64(spanIDBytes[:])
		req.Header.Set("X-Cloud-Trace-Context", fmt.Sprintf("%s/%d;o=1", sc.TraceID().String(), spanDec))
	}
}

// getTraceSpansByTraceID returns all recorded spans for a specific TraceID in chronological order.
func getTraceSpansByTraceID(traceID string) []TraceSpanRecord {
	if traceID == "" {
		return nil
	}
	traceRingMu.Lock()
	defer traceRingMu.Unlock()
	var out []TraceSpanRecord
	for _, s := range traceRing {
		if s.TraceID == traceID {
			out = append(out, s)
		}
	}
	return out
}

// getRecentTraces returns the most recent N spans from the ring buffer.
func getRecentTraces(limit int) []TraceSpanRecord {
	if limit <= 0 || limit > maxTraceItems {
		limit = 50
	}
	traceRingMu.Lock()
	defer traceRingMu.Unlock()
	if len(traceRing) <= limit {
		cp := make([]TraceSpanRecord, len(traceRing))
		copy(cp, traceRing)
		return cp
	}
	cp := make([]TraceSpanRecord, limit)
	copy(cp, traceRing[len(traceRing)-limit:])
	return cp
}
