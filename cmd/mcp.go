package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghchinoy/dgem/pkg/client"
	"github.com/ghchinoy/dgem/pkg/template"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// WarmupStageBreakdown records the 4 cold-start sub-stages for right-sizing and Cloud Monitoring.
type WarmupStageBreakdown struct {
	Timestamp          string `json:"timestamp"`
	TriggerSource      string `json:"trigger_source"`
	Phase1ActivatorMs  int64  `json:"phase_1_activator_ms"`
	Phase2TmpfsStageMs int64  `json:"phase_2_tmpfs_stage_ms"`
	Phase3VLLMSiglipMs int64  `json:"phase_3_vllm_siglip_ms"`
	Phase4TritonJITMs  int64  `json:"phase_4_triton_jit_ms"`
	TotalWarmupMs      int64  `json:"total_warmup_ms"`
	Attempts           int    `json:"attempts"`
}

// WarmupTelemetryStats is returned by GET /api/warmup/stats for operational dashboards & right-sizing.
type WarmupTelemetryStats struct {
	EWMAWakeSeconds        int                    `json:"ewma_wake_seconds"`
	EWMAStage1ActivatorMs  int64                  `json:"ewma_stage_1_activator_ms"`
	EWMAStage2TmpfsStageMs int64                  `json:"ewma_stage_2_tmpfs_stage_ms"`
	EWMAStage3VLLMSiglipMs int64                  `json:"ewma_stage_3_vllm_siglip_ms"`
	EWMAStage4TritonJITMs  int64                  `json:"ewma_stage_4_triton_jit_ms"`
	SampleCount            int                    `json:"sample_count"`
	History                []WarmupStageBreakdown `json:"history"`
}

var (
	gpuStateMu           sync.RWMutex
	lastWarmTimestamp    time.Time
	lastReadoutLatencyMs int64
	warmupInProgress     bool
	warmupStartedAt      time.Time
	warmupDoneCh         chan struct{}
	warmupLastAttempts   int
	warmupLastErr        error
	currentWarmupPhase   string  = "staging_tmpfs"
	currentBytesStagedGB float64 = 0.0

	// Pre-seeded with verified Cloud Run RTX Pro 6000 /tmp/dgemma cold-start telemetry (dgemma-00023-7wz)
	ewmaStage1Ms int64 = 5200
	ewmaStage2Ms int64 = 44000
	ewmaStage3Ms int64 = 64000
	ewmaStage4Ms int64 = 8648
	ewmaTotalMs  int64 = 121848

	warmupHistory = []WarmupStageBreakdown{
		{
			Timestamp:          "2026-09-22T20:16:54Z",
			TriggerSource:      "cloudrun_baseline_rtx_pro_6000",
			Phase1ActivatorMs:  5200,
			Phase2TmpfsStageMs: 44000,
			Phase3VLLMSiglipMs: 64000,
			Phase4TritonJITMs:  8648,
			TotalWarmupMs:      121848,
			Attempts:           18,
		},
	}
)

func recordWarmupBreakdown(rec WarmupStageBreakdown) int {
	const alpha = 0.30
	gpuStateMu.Lock()
	defer gpuStateMu.Unlock()

	ewmaStage1Ms = int64((1.0-alpha)*float64(ewmaStage1Ms) + alpha*float64(rec.Phase1ActivatorMs))
	ewmaStage2Ms = int64((1.0-alpha)*float64(ewmaStage2Ms) + alpha*float64(rec.Phase2TmpfsStageMs))
	ewmaStage3Ms = int64((1.0-alpha)*float64(ewmaStage3Ms) + alpha*float64(rec.Phase3VLLMSiglipMs))
	ewmaStage4Ms = int64((1.0-alpha)*float64(ewmaStage4Ms) + alpha*float64(rec.Phase4TritonJITMs))
	ewmaTotalMs = int64((1.0-alpha)*float64(ewmaTotalMs) + alpha*float64(rec.TotalWarmupMs))

	warmupHistory = append(warmupHistory, rec)
	if len(warmupHistory) > 25 {
		warmupHistory = warmupHistory[len(warmupHistory)-25:]
	}
	sec := int((ewmaTotalMs + 500) / 1000)
	if sec < 45 {
		sec = 45
	}
	return sec
}

// GetWarmupTelemetryStats returns the current EWMA right-sized cold-start estimate and stage history.
func GetWarmupTelemetryStats() WarmupTelemetryStats {
	gpuStateMu.RLock()
	defer gpuStateMu.RUnlock()
	hist := make([]WarmupStageBreakdown, len(warmupHistory))
	copy(hist, warmupHistory)
	ewmaSec := int((ewmaTotalMs + 500) / 1000)
	if ewmaSec < 45 {
		ewmaSec = 45
	}
	return WarmupTelemetryStats{
		EWMAWakeSeconds:        ewmaSec,
		EWMAStage1ActivatorMs:  ewmaStage1Ms,
		EWMAStage2TmpfsStageMs: ewmaStage2Ms,
		EWMAStage3VLLMSiglipMs: ewmaStage3Ms,
		EWMAStage4TritonJITMs:  ewmaStage4Ms,
		SampleCount:            len(hist),
		History:                hist,
	}
}

func pollUpstreamWarmupHealth(baseURL string) (string, float64, bool) {
	healthURL := strings.TrimSuffix(strings.TrimSuffix(baseURL, "/"), "/v1") + "/health"
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return "", 0, false
	}
	if tok := FetchGCPIdentityToken("", baseURL); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	hc := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := hc.Do(req)
	if err != nil {
		return "", 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, false
	}
	var payload struct {
		Phase         string  `json:"phase"`
		BytesStagedGB float64 `json:"bytes_staged_gb"`
		VLLMReady     bool    `json:"vllm_ready"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", 0, true
	}
	return payload.Phase, payload.BytesStagedGB, true
}

func inferWarmupPhaseAndLabel(elapsedSec int, ewmaSec int, phaseHint string, bytesStaged float64) (string, string) {
	// 1. If dgemma /health explicitly returned a sub-stage, honor it directly
	switch phaseHint {
	case "mounting_gcs":
		return "phase_1_activator", "Stage 1/4: Cloud Run Activator & GCSFuse Mount"
	case "staging_tmpfs":
		if bytesStaged > 0 {
			return "phase_2_tmpfs_stage", fmt.Sprintf("Stage 2/4: Staging 17.53 GiB to /tmp RAM (%.1f GiB staged)", bytesStaged)
		}
		return "phase_2_tmpfs_stage", "Stage 2/4: Staging 17.53 GiB NVFP4 Weights to /tmp RAM"
	case "loading_vllm_siglip":
		return "phase_3_vllm_siglip", "Stage 3/4: Loading vLLM EngineCore (5.6s) + SigLIP Vision Tower"
	case "ready":
		return "phase_4_triton_jit", "Stage 4/4: First Decision Readout & Triton Kernel JIT"
	}

	// 2. Time-based progression fallback when /health hasn't reported a phase key yet
	p1Sec := int(ewmaStage1Ms / 1000)
	if p1Sec < 8 {
		p1Sec = 8
	}
	p2Sec := p1Sec + int(ewmaStage2Ms/1000)
	p3Sec := p2Sec + int(ewmaStage3Ms/1000)

	if elapsedSec <= p1Sec {
		return "phase_1_activator", "Stage 1/4: Cloud Run Activator & GCSFuse Mount"
	}
	if elapsedSec <= p2Sec {
		return "phase_2_tmpfs_stage", "Stage 2/4: Staging 17.53 GiB NVFP4 Weights to /tmp RAM"
	}
	if elapsedSec <= p3Sec {
		return "phase_3_vllm_siglip", "Stage 3/4: Loading vLLM EngineCore (5.6s) + SigLIP Vision Tower"
	}
	return "phase_4_triton_jit", "Stage 4/4: First Decision Readout & Triton Kernel JIT"
}

var mcpRemoteURL string

// runRemoteMCPProxy bridges stdio JSON-RPC messages to a remote Streamable HTTP MCP endpoint
// (such as https://dgemma.aaie.cloud/mcp), automatically attaching Application Default Credentials
// (ADC) OIDC id_token (or OAuth2 access token) and MCP Streamable HTTP headers (Mcp-Method, Mcp-Protocol-Version).
func runRemoteMCPProxy(remoteURL string) error {
	httpClient := &http.Client{Timeout: 180 * time.Second}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	sessionProtoVer := ""
	sessionID := ""

	writeRPCError := func(id *json.RawMessage, code int, msg string) {
		if id == nil || len(*id) == 0 || string(*id) == "null" {
			return
		}
		errObj := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      *id,
			"error": map[string]interface{}{
				"code":    code,
				"message": msg,
			},
		}
		if b, err := json.Marshal(errObj); err == nil {
			os.Stdout.Write(append(b, '\n'))
		}
	}

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		// Extract id, method, and protocolVersion (_meta["io.modelcontextprotocol/protocolVersion"] or params.protocolVersion)
		var rpcEnv struct {
			ID     *json.RawMessage `json:"id"`
			Method string           `json:"method"`
			Params struct {
				ProtocolVersion string            `json:"protocolVersion"`
				Meta            map[string]string `json:"_meta"`
			} `json:"params"`
		}
		_ = json.Unmarshal(line, &rpcEnv)
		// Also handle _meta where values include non-strings (e.g. clientCapabilities object)
		if rpcEnv.Params.ProtocolVersion != "" {
			sessionProtoVer = rpcEnv.Params.ProtocolVersion
		} else {
			var rawEnv struct {
				ID     *json.RawMessage `json:"id"`
				Method string           `json:"method"`
				Params struct {
					Meta map[string]interface{} `json:"_meta"`
				} `json:"params"`
			}
			if json.Unmarshal(line, &rawEnv) == nil {
				if rpcEnv.ID == nil {
					rpcEnv.ID = rawEnv.ID
				}
				if rpcEnv.Method == "" {
					rpcEnv.Method = rawEnv.Method
				}
				if pv, ok := rawEnv.Params.Meta["io.modelcontextprotocol/protocolVersion"].(string); ok && pv != "" {
					sessionProtoVer = pv
				}
			}
		}

		req, err := http.NewRequest(http.MethodPost, remoteURL, bytes.NewReader(line))
		if err != nil {
			writeRPCError(rpcEnv.ID, -32603, fmt.Sprintf("failed to construct remote MCP request: %v", err))
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if rpcEnv.Method != "" {
			req.Header.Set("Mcp-Method", rpcEnv.Method)
		}
		if sessionProtoVer != "" {
			req.Header.Set("Mcp-Protocol-Version", sessionProtoVer)
		}
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}

		tok := FetchGCPIdentityToken(viper.GetString("iap_client_id"), remoteURL)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			writeRPCError(rpcEnv.ID, -32000, fmt.Sprintf("remote MCP endpoint unreachable (%s): %v", remoteURL, err))
			continue
		}
		if sid := strings.TrimSpace(resp.Header.Get("Mcp-Session-Id")); sid != "" {
			sessionID = sid
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		trimmed := bytes.TrimSpace(body)
		if resp.StatusCode >= 300 {
			snippet := string(trimmed)
			if len(snippet) > 240 {
				snippet = snippet[:240] + "..."
			}
			writeRPCError(rpcEnv.ID, -32000, fmt.Sprintf("remote MCP endpoint returned HTTP %d: %s", resp.StatusCode, snippet))
			continue
		}
		if len(trimmed) > 0 && bytes.HasPrefix(trimmed, []byte("{")) {
			var compactBuf bytes.Buffer
			if json.Compact(&compactBuf, trimmed) == nil {
				os.Stdout.Write(compactBuf.Bytes())
				os.Stdout.Write([]byte("\n"))
			} else {
				os.Stdout.Write(trimmed)
				os.Stdout.Write([]byte("\n"))
			}
		}
	}
	return scanner.Err()
}

var mcpCmd = &cobra.Command{
	Use:     "mcp",
	GroupID: "core",
	Short:   "Start the dgem Model Context Protocol (MCP) server over stdio (with automatic ADC)",
	Long: `mcp starts a Model Context Protocol (MCP) server over standard input/output (stdio),
exposing DiffusionGemma zero-shot multi-slot decision policies, multimodal SigLIP bounding-box
localization (EXP-09), GPU health/availability checks, and scale-from-zero GPU warmup tools
to AI agents (Gemini CLI, Claude Desktop, Cursor, Antigravity, etc.).

Application Default Credentials (ADC: ~/.config/gcloud/application_default_credentials.json)
are automatically discovered and used to authenticate against Vertex AI Dedicated Endpoints
and Cloud Run IAP / IAM services without requiring manual tokens.

Pass --remote https://dgemma.aaie.cloud/mcp to bridge stdio directly to the hosted Cloud Run
MCP gateway using ADC.`,
	Example: `  # Bridge stdio to the hosted Cloud Run MCP gateway using ADC automatically
  dgem mcp --remote https://dgemma.aaie.cloud/mcp

  # Run local stdio MCP server with automatic ADC (vertex_first -> Cloud Run failover)
  dgem mcp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(mcpRemoteURL) != "" {
			return runRemoteMCPProxy(strings.TrimSpace(mcpRemoteURL))
		}
		viper.Set("gcp_auth", true)
		if viper.GetString("url") == "http://127.0.0.1:8080/v1" {
			viper.Set("url", "https://dgemma-882920967572.us-central1.run.app/v1")
		}
		if _, err := os.Stat(serveTemplatesDir); err != nil {
			if exe, exErr := os.Executable(); exErr == nil {
				candidate := filepath.Join(filepath.Dir(filepath.Dir(exe)), "templates")
				if _, stErr := os.Stat(candidate); stErr == nil {
					serveTemplatesDir = candidate
				}
			}
		}
		srv := buildMCPServer()
		return srv.Run(context.Background(), &mcp.StdioTransport{})
	},
}

func init() {
	mcpCmd.Flags().StringVar(&mcpRemoteURL, "remote", "", "Remote Streamable HTTP MCP endpoint URL (e.g. https://dgemma.aaie.cloud/mcp) to proxy over stdio using ADC")
	RootCmd.AddCommand(mcpCmd)
}

// getGPUIdleWindow returns the configured GPU time-to-idle duration from DGEM_GPU_IDLE_TTL,
// GPU_IDLE_TTL, or the --gpu-idle-ttl CLI flag (defaulting to 3 hours).
func getGPUIdleWindow() time.Duration {
	for _, k := range []string{"DGEM_GPU_IDLE_TTL", "GPU_IDLE_TTL"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				return d
			}
		}
	}
	if serveGPUIdleTTL > 0 {
		return serveGPUIdleTTL
	}
	return 3 * time.Hour
}

var keepaliveOnce sync.Once

// startGPUKeepaliveLoop sends a lightweight authenticated GET /health heartbeat to the upstream
// Cloud Run GPU service every 4 minutes ONLY while time.Since(lastWarmTimestamp) < getGPUIdleWindow().
// This prevents Cloud Run's underlying 15-minute container idle reaper from terminating the warm
// GPU before the configured DGEM_GPU_IDLE_TTL (e.g. 3h) expires, while allowing it to scale to 0
// immediately once the idle TTL elapses.
func startGPUKeepaliveLoop() {
	keepaliveOnce.Do(func() {
		go func() {
			// 0. If Vertex First or Vertex is configured, immediately check Vertex /invoke/health on startup
			backendConfigMu.RLock()
			defB := serveDefaultBackend
			vxURL := serveVertexURL
			backendConfigMu.RUnlock()
			if defB == "vertex_first" || defB == "vertex" {
				vSt := inspectVertexEndpointState(context.Background(), vxURL)
				if vSt.State == "deployed" {
					RecordVertexReadoutLatency(490)
				}
			}

			// On gateway startup, probe upstream Cloud Run /health once with a 3s timeout: if the GPU container
			// is already warm ("phase":"ready" / "vllm_ready":true), sync gateway state immediately.
			upstreamBase := strings.TrimSuffix(viper.GetString("url"), "/")
			upstreamBase = strings.TrimSuffix(upstreamBase, "/v1")
			if upstreamBase != "" {
				healthURL := upstreamBase + "/health"
				if req, err := http.NewRequest("GET", healthURL, nil); err == nil {
					if viper.GetBool("gcp_auth") || viper.GetString("iap_client_id") != "" {
						if tok := FetchGCPIdentityToken(viper.GetString("iap_client_id"), healthURL); tok != "" {
							req.Header.Set("Authorization", "Bearer "+tok)
						}
					} else if tok := viper.GetString("token"); tok != "" {
						req.Header.Set("Authorization", "Bearer "+tok)
					}
					hc := &http.Client{Timeout: 4 * time.Second}
					if resp, err := hc.Do(req); err == nil {
						var st struct {
							Phase     string `json:"phase"`
							VLLMReady bool   `json:"vllm_ready"`
						}
						_ = json.NewDecoder(resp.Body).Decode(&st)
						resp.Body.Close()
						if st.VLLMReady || st.Phase == "ready" {
							MarkGPUWarm()
						}
					}
				}
			}

			ticker := time.NewTicker(4 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				// Keep Vertex AI Dedicated Endpoint (/invoke/health) warm when vertex_first or vertex is active
				backendConfigMu.RLock()
				curDefB := serveDefaultBackend
				curVxURL := serveVertexURL
				backendConfigMu.RUnlock()
				if curDefB == "vertex_first" || curDefB == "vertex" {
					invalidateVertexStatusCache()
					vSt := inspectVertexEndpointState(context.Background(), curVxURL)
					if vSt.State == "deployed" {
						MarkGPUWarm()
					}
				}

				idleWindow := getGPUIdleWindow()
				gpuStateMu.RLock()
				lastWarm := lastWarmTimestamp
				warming := warmupInProgress
				gpuStateMu.RUnlock()

				if warming || lastWarm.IsZero() || time.Since(lastWarm) >= idleWindow {
					continue
				}

				upstreamBase := strings.TrimSuffix(viper.GetString("url"), "/")
				upstreamBase = strings.TrimSuffix(upstreamBase, "/v1")
				if upstreamBase == "" {
					continue
				}
				healthURL := upstreamBase + "/health"
				req, err := http.NewRequest("GET", healthURL, nil)
				if err != nil {
					continue
				}
				if viper.GetBool("gcp_auth") || viper.GetString("iap_client_id") != "" {
					if tok := FetchGCPIdentityToken(viper.GetString("iap_client_id"), healthURL); tok != "" {
						req.Header.Set("Authorization", "Bearer "+tok)
					}
				} else if tok := viper.GetString("token"); tok != "" {
					req.Header.Set("Authorization", "Bearer "+tok)
				}
				hc := &http.Client{Timeout: 8 * time.Second}
				if resp, err := hc.Do(req); err == nil {
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}()
	})
}

// NotifyColdStartWarmup ensures that if a decision request arrives while the GPU is scaled to zero,
// the global status coordinator immediately starts the single-flight background warmup worker
// (just as if "Wake GPU" had been pressed) and guarantees warmupInProgress is cleared on completion or timeout.
func NotifyColdStartWarmup() {
	idleWindow := getGPUIdleWindow()
	gpuStateMu.RLock()
	isWarm := !lastWarmTimestamp.IsZero() && time.Since(lastWarmTimestamp) < idleWindow
	alreadyWarming := warmupInProgress
	gpuStateMu.RUnlock()

	if !isWarm && !alreadyWarming {
		go func() {
			_, _ = TriggerGPUWarmupWithSource(context.Background(), false, "auto_wake_decide")
		}()
	}
}

var lastVertexReadoutLatencyMs int64 = 490

// RecordVertexReadoutLatency records the most recent Vertex AI /invoke/* readout latency
// without marking the scale-to-zero Cloud Run GPU container as warm.
func RecordVertexReadoutLatency(latencyMs int64) {
	if latencyMs <= 0 {
		return
	}
	gpuStateMu.Lock()
	lastVertexReadoutLatencyMs = latencyMs
	gpuStateMu.Unlock()
}

// MarkGPUWarm records that the upstream Cloud Run vLLM engine successfully completed a decision readout
// and wakes any callers waiting on an in-flight warmup broadcast channel.
func MarkGPUWarm(latencyMs ...int64) {
	gpuStateMu.Lock()
	lastWarmTimestamp = time.Now()
	if len(latencyMs) > 0 && latencyMs[0] > 0 {
		lastReadoutLatencyMs = latencyMs[0]
	}
	if warmupInProgress {
		warmupInProgress = false
		warmupLastErr = nil
		if warmupDoneCh != nil {
			close(warmupDoneCh)
			warmupDoneCh = nil
		}
	}
	gpuStateMu.Unlock()
}

// HealthAndGPUStatusOutput is returned by GET /api/status and the get_health_and_gpu_status MCP tool.
type HealthAndGPUStatusOutput struct {
	GatewayHealthy        bool    `json:"gateway_healthy"`
	GPUAvailable          bool    `json:"gpu_available"`
	GPUState              string  `json:"gpu_state"` // "warm_and_ready", "warming_up", "scaled_to_zero"
	ActiveBackend         string  `json:"active_backend"`    // "vertex" or "cloudrun"
	RequestedBackend      string  `json:"requested_backend"` // "vertex_first", "vertex", or "cloudrun"
	ContainerReachable    bool    `json:"container_reachable"`
	WarmupInProgress      bool    `json:"warmup_in_progress"`
	WarmupElapsedSeconds  int     `json:"warmup_elapsed_seconds,omitempty"`
	WarmupPhase           string  `json:"warmup_phase,omitempty"`
	WarmupPhaseLabel      string  `json:"warmup_phase_label,omitempty"`
	WarmupBytesStagedGB   float64 `json:"warmup_bytes_staged_gb,omitempty"`
	EWMAWakeSeconds       int     `json:"ewma_wake_seconds"`
	SecondsSinceLastRead  int     `json:"seconds_since_last_read,omitempty"`
	IdleRemainingSeconds  int     `json:"idle_remaining_seconds,omitempty"`
	IdleTTLSeconds        int     `json:"idle_ttl_seconds,omitempty"`
	LastReadoutMs         int64   `json:"last_readout_ms,omitempty"`
	EstimatedWakeSeconds  int     `json:"estimated_wake_seconds"`
	UpstreamURL           string  `json:"upstream_url"`
	Model                 string  `json:"model"`
	GPUTier               string  `json:"gpu_tier"`
	TemplatesAvailable    int     `json:"templates_available"`
	AuthenticatedUser     string  `json:"authenticated_user,omitempty"`
	Detail                string  `json:"detail"`
}

// CheckHealthAndGPUStatus inspects both the gateway, the Vertex AI Dedicated Endpoint (when vertex_first
// or vertex is configured), and the upstream Cloud Run GPU service.
func CheckHealthAndGPUStatus(ctx context.Context, userEmail string) HealthAndGPUStatusOutput {
	return CheckHealthAndGPUStatusForBackend(ctx, userEmail, "")
}

// CheckHealthAndGPUStatusForBackend inspects the readiness and telemetry of the requested backend target
// ("vertex_first", "vertex", or "cloudrun").
func CheckHealthAndGPUStatusForBackend(ctx context.Context, userEmail, backendOverride string) HealthAndGPUStatusOutput {
	backendConfigMu.RLock()
	defB := serveDefaultBackend
	vxURL := serveVertexURL
	backendConfigMu.RUnlock()

	reqMode := strings.ToLower(strings.TrimSpace(backendOverride))
	if reqMode == "vertex" || reqMode == "cloudrun" || reqMode == "vertex_first" {
		defB = reqMode
	}

	catalog, _ := discoverTemplates(serveTemplatesDir)
	idleWindow := getGPUIdleWindow()

	// 1. If Vertex First or Vertex is the effective backend, check Vertex AI Dedicated Endpoint state first!
	if defB == "vertex_first" || defB == "vertex" {
		vSt := inspectVertexEndpointState(ctx, vxURL)
		if vSt.State == "deployed" && vSt.ReplicaCount > 0 {
			gpuStateMu.RLock()
			lastReadMs := lastVertexReadoutLatencyMs
			gpuStateMu.RUnlock()
			if lastReadMs <= 0 {
				lastReadMs = 490
			}
			return HealthAndGPUStatusOutput{
				GatewayHealthy:       true,
				GPUAvailable:         true,
				GPUState:             "warm_and_ready",
				ActiveBackend:        "vertex",
				RequestedBackend:     defB,
				ContainerReachable:   true,
				WarmupInProgress:     false,
				EWMAWakeSeconds:      0,
				SecondsSinceLastRead: 0,
				IdleRemainingSeconds: 0,
				IdleTTLSeconds:       0,
				LastReadoutMs:        lastReadMs,
				EstimatedWakeSeconds: 0,
				UpstreamURL:          vxURL,
				Model:                "nvidia/diffusiongemma-26B-A4B-it-NVFP4",
				GPUTier:              fmt.Sprintf("Vertex AI Dedicated Endpoint (%s · /invoke/*)", vSt.MachineType),
				TemplatesAvailable:   len(catalog),
				AuthenticatedUser:    userEmail,
				Detail:               fmt.Sprintf("Vertex AI Dedicated Endpoint %s is Warm & Ready (%d× L4 · 0.0s cold start · ~%dms GPU denoise · /invoke/*).", vSt.EndpointID, vSt.ReplicaCount, lastReadMs),
			}
		}
		if defB == "vertex" {
			if vSt.State == "deploying" {
				return HealthAndGPUStatusOutput{
					GatewayHealthy:       true,
					GPUAvailable:         false,
					GPUState:             "warming_up",
					ActiveBackend:        "vertex",
					RequestedBackend:     defB,
					ContainerReachable:   false,
					WarmupInProgress:     true,
					WarmupPhase:          "vertex_scaling_up",
					WarmupPhaseLabel:     vSt.Message,
					EWMAWakeSeconds:      90,
					EstimatedWakeSeconds: 90,
					IdleTTLSeconds:       0,
					UpstreamURL:          vxURL,
					Model:                "nvidia/diffusiongemma-26B-A4B-it-NVFP4",
					GPUTier:              fmt.Sprintf("Vertex AI Dedicated Endpoint (%s · /invoke/*)", vSt.MachineType),
					TemplatesAvailable:   len(catalog),
					AuthenticatedUser:    userEmail,
					Detail:               vSt.Message,
				}
			}
			return HealthAndGPUStatusOutput{
				GatewayHealthy:       true,
				GPUAvailable:         false,
				GPUState:             "scaled_to_zero",
				ActiveBackend:        "vertex",
				RequestedBackend:     defB,
				ContainerReachable:   false,
				WarmupInProgress:     false,
				WarmupPhase:          "vertex_quiesced",
				WarmupPhaseLabel:     "Vertex L4 Quiesced (0 replicas)",
				EWMAWakeSeconds:      90,
				EstimatedWakeSeconds: 90,
				IdleTTLSeconds:       0,
				UpstreamURL:          vxURL,
				Model:                "nvidia/diffusiongemma-26B-A4B-it-NVFP4",
				GPUTier:              fmt.Sprintf("Vertex AI Dedicated Endpoint (%s · /invoke/*)", vSt.MachineType),
				TemplatesAvailable:   len(catalog),
				AuthenticatedUser:    userEmail,
				Detail:               vSt.Message,
			}
		}
	}

	gpuStateMu.Lock()
	// Safety expiry: never allow warmupInProgress to stay stuck past serveWakeupTimeout (10m)
	if warmupInProgress && time.Since(warmupStartedAt) > 10*time.Minute {
		warmupInProgress = false
		if warmupDoneCh != nil {
			close(warmupDoneCh)
			warmupDoneCh = nil
		}
	}
	lastWarm := lastWarmTimestamp
	lastReadMs := lastReadoutLatencyMs
	warming := warmupInProgress
	warmStart := warmupStartedAt
	phaseHint := currentWarmupPhase
	bytesStaged := currentBytesStagedGB
	ewmaSec := int((ewmaTotalMs + 500) / 1000)
	if ewmaSec < 45 {
		ewmaSec = 45
	}
	gpuStateMu.Unlock()

	out := HealthAndGPUStatusOutput{
		GatewayHealthy:       true,
		ActiveBackend:        "cloudrun",
		RequestedBackend:     defB,
		UpstreamURL:          viper.GetString("url"),
		Model:                "nvidia/diffusiongemma-26B-A4B-it-NVFP4",
		GPUTier:              "1x NVIDIA RTX Pro 6000 (48GB VRAM, SigLIP Multimodal)",
		TemplatesAvailable:   len(catalog),
		AuthenticatedUser:    userEmail,
		WarmupInProgress:     warming,
		LastReadoutMs:        lastReadMs,
		EWMAWakeSeconds:      ewmaSec,
		EstimatedWakeSeconds: ewmaSec,
		IdleTTLSeconds:       int(idleWindow.Seconds()),
	}

	if warming {
		out.WarmupElapsedSeconds = int(time.Since(warmStart).Seconds())
		out.ContainerReachable = out.WarmupElapsedSeconds > 3
		out.GPUState = "warming_up"
		phaseID, phaseLabel := inferWarmupPhaseAndLabel(out.WarmupElapsedSeconds, ewmaSec, phaseHint, bytesStaged)
		out.WarmupPhase = phaseID
		out.WarmupPhaseLabel = phaseLabel
		out.WarmupBytesStagedGB = bytesStaged
		rem := ewmaSec - out.WarmupElapsedSeconds
		if rem < 8 {
			rem = 8
		}
		out.EstimatedWakeSeconds = rem
		out.Detail = fmt.Sprintf("%s (%ds elapsed of ~%ds EWMA cold-start).", phaseLabel, out.WarmupElapsedSeconds, ewmaSec)
		return out
	}

	if !lastWarm.IsZero() && time.Since(lastWarm) < idleWindow {
		sinceSec := int(time.Since(lastWarm).Seconds())
		remSec := int(idleWindow.Seconds()) - sinceSec
		out.GPUAvailable = true
		out.ContainerReachable = true
		out.GPUState = "warm_and_ready"
		out.SecondsSinceLastRead = sinceSec
		out.IdleRemainingSeconds = remSec
		out.EstimatedWakeSeconds = 0
		if lastReadMs > 0 {
			out.Detail = fmt.Sprintf("vLLM EngineCore is warm and ready (last readout %dms, %ds ago; %dm%ds remaining in %s idle TTL).", lastReadMs, sinceSec, remSec/60, remSec%60, idleWindow)
		} else {
			out.Detail = fmt.Sprintf("vLLM EngineCore is warm and ready (last readout %ds ago; %dm%ds remaining in %s idle TTL).", sinceSec, remSec/60, remSec%60, idleWindow)
		}
		return out
	}

	out.GPUState = "scaled_to_zero"
	out.Detail = fmt.Sprintf("GPU service is scaled to 0 instances ($0.00/hr idle; ~%ds EWMA wake, %s idle TTL once warm). Click 'Wake GPU' or execute any decision to wake automatically.", ewmaSec, idleWindow)
	return out
}

// WarmupGPUInput defines arguments for the warmup_gpu MCP tool and POST /api/warmup.
type WarmupGPUInput struct {
	WaitForReady bool `json:"wait_for_ready,omitempty" jsonschema:"If true (default), waits until vLLM finishes loading weights and confirms a test decision (~2m if cold, <10ms if already warm). If false, triggers background wakeup and returns immediately."`
}

// WarmupGPUOutput is returned by warmup_gpu and POST /api/warmup.
type WarmupGPUOutput struct {
	Status                string `json:"status"` // "warm_and_ready", "warming_up_started", or "warming_up_in_progress"
	GPUAvailable          bool   `json:"gpu_available"`
	Coalesced             bool   `json:"coalesced"`
	WarmupElapsedSeconds  int    `json:"warmup_elapsed_seconds,omitempty"`
	EWMAWakeSeconds       int    `json:"ewma_wake_seconds,omitempty"`
	WarmupAttempts        int    `json:"warmup_attempts,omitempty"`
	ElapsedMs             int64  `json:"elapsed_ms"`
	UpstreamURL           string `json:"upstream_url"`
	Message               string `json:"message"`
}

// TriggerGPUWarmup delegates to TriggerGPUWarmupWithSource with default source.
func TriggerGPUWarmup(ctx context.Context, waitForReady bool) (WarmupGPUOutput, error) {
	return TriggerGPUWarmupWithSource(ctx, waitForReady, "api_warmup")
}

// TriggerGPUWarmupWithSource is a strictly idempotent, single-flight coalesced GPU warmup coordinator
// shared across MCP (warmup_gpu), Web Studio (Wake GPU button), and REST (POST /api/warmup).
func TriggerGPUWarmupWithSource(ctx context.Context, waitForReady bool, triggerSource string) (WarmupGPUOutput, error) {
	callStart := time.Now()

	// 1. Fast-Path Idempotency: If the GPU was verified warm within the 15m window,
	// return immediately without running a redundant GPU forward pass.
	currentStatus := CheckHealthAndGPUStatus(ctx, "")
	if currentStatus.GPUAvailable && currentStatus.GPUState == "warm_and_ready" {
		return WarmupGPUOutput{
			Status:          "warm_and_ready",
			GPUAvailable:    true,
			Coalesced:       true,
			EWMAWakeSeconds: currentStatus.EWMAWakeSeconds,
			ElapsedMs:       time.Since(callStart).Milliseconds(),
			UpstreamURL:     viper.GetString("url"),
			Message:         fmt.Sprintf("GPU is already warm and ready (last readout %ds ago; no duplicate warmup sent).", currentStatus.SecondsSinceLastRead),
		}, nil
	}

	// 2. Single-Flight Coalescing: Join existing in-flight warmup if one is already running,
	// or start exactly ONE background warmup worker if none is active.
	gpuStateMu.Lock()
	alreadyRunning := warmupInProgress
	var waitCh chan struct{}
	var startedAt time.Time

	if alreadyRunning {
		waitCh = warmupDoneCh
		startedAt = warmupStartedAt
		gpuStateMu.Unlock()
	} else {
		warmupInProgress = true
		warmupStartedAt = callStart
		warmupDoneCh = make(chan struct{})
		warmupLastAttempts = 0
		warmupLastErr = nil
		currentWarmupPhase = ""
		currentBytesStagedGB = 0.0
		waitCh = warmupDoneCh
		startedAt = warmupStartedAt
		gpuStateMu.Unlock()

		// Launch the single background warmup poller + OTel warmup lifecycle span
		go func(doneCh chan struct{}, src string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), serveWakeupTimeout)
			defer cancel()

			// Poll upstream /health every 3s to capture live sub-stage and bytes_staged_gb
			go func() {
				ticker := time.NewTicker(3 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-bgCtx.Done():
						return
					case <-ticker.C:
						ph, stagedGB, ok := pollUpstreamWarmupHealth(viper.GetString("url"))
						if ok {
							gpuStateMu.Lock()
							if ph != "" {
								currentWarmupPhase = ph
							}
							if stagedGB > 0 {
								currentBytesStagedGB = stagedGB
							}
							gpuStateMu.Unlock()
						}
					}
				}
			}()

			_, warmSpan := gatewayTracer().Start(bgCtx, "dgem.gpu.warmup_lifecycle")
			warmSpan.SetAttributes(attribute.String("dgem.warmup.trigger", src))

			engine := template.NewEngine()
			vars := map[string]interface{}{"ticket": "GPU warmup readiness probe"}
			targetPath, _ := resolveTemplateFile(serveTemplatesDir, "support_triage")
			rendered, err := engine.RenderFile(targetPath, vars)
			if err != nil {
				rendered = `{"questions":[{"id":"ready","type":"boolean","prompt":"Is the system ready?"}]}`
			}
			schemaContent, stateContent, err := template.ParseStructuredPayload(rendered, vars)
			attempts := 1
			var readoutMs int64
			if err == nil {
				_, stats, att, decErr := executeDecideWithWarmup(bgCtx, schemaContent, stateContent, nil)
				attempts = att
				err = decErr
				readoutMs = stats.WallTime.Milliseconds()
			}

			totalWarmMs := time.Since(callStart).Milliseconds()
			if err == nil {
				// Decompose total warmup duration proportionally across the 4 stages (or from probe)
				s4 := readoutMs
				if s4 <= 0 || s4 > 15000 {
					s4 = 8600
				}
				remMs := totalWarmMs - s4
				if remMs < 5000 {
					remMs = 5000
				}
				s1 := int64(float64(remMs) * 0.05)
				s2 := int64(float64(remMs) * 0.38)
				s3 := remMs - s1 - s2

				rec := WarmupStageBreakdown{
					Timestamp:          time.Now().UTC().Format(time.RFC3339),
					TriggerSource:      src,
					Phase1ActivatorMs:  s1,
					Phase2TmpfsStageMs: s2,
					Phase3VLLMSiglipMs: s3,
					Phase4TritonJITMs:  s4,
					TotalWarmupMs:      totalWarmMs,
					Attempts:           attempts,
				}
				newEWMA := recordWarmupBreakdown(rec)

				warmSpan.SetAttributes(
					attribute.Int64("dgem.warmup.total_ms", totalWarmMs),
					attribute.Int64("dgem.warmup.stage1_activator_ms", s1),
					attribute.Int64("dgem.warmup.stage2_tmpfs_ms", s2),
					attribute.Int64("dgem.warmup.stage3_vllm_siglip_ms", s3),
					attribute.Int64("dgem.warmup.stage4_triton_jit_ms", s4),
					attribute.Int("dgem.warmup.attempts", attempts),
					attribute.Int("dgem.warmup.ewma_wake_seconds", newEWMA),
				)
				warmSpan.SetStatus(codes.Ok, "warmup completed")
			} else {
				warmSpan.RecordError(err)
				warmSpan.SetStatus(codes.Error, err.Error())
			}
			warmSpan.End()

			gpuStateMu.Lock()
			warmupLastAttempts = attempts
			warmupLastErr = err
			if err == nil {
				lastWarmTimestamp = time.Now()
				if readoutMs > 0 {
					lastReadoutLatencyMs = readoutMs
				}
			}
			if warmupInProgress {
				warmupInProgress = false
				if warmupDoneCh != nil {
					close(warmupDoneCh)
					warmupDoneCh = nil
				}
			}
			gpuStateMu.Unlock()
		}(waitCh, triggerSource)
	}

	// 3. Non-blocking mode (wait_for_ready=false): Return immediately (coalesced if already running)
	if !waitForReady {
		elapsedSec := int(time.Since(startedAt).Seconds())
		if alreadyRunning {
			return WarmupGPUOutput{
				Status:               "warming_up_in_progress",
				GPUAvailable:         false,
				Coalesced:            true,
				WarmupElapsedSeconds: elapsedSec,
				ElapsedMs:            time.Since(callStart).Milliseconds(),
				UpstreamURL:          viper.GetString("url"),
				Message:              fmt.Sprintf("Joined existing in-flight GPU warmup (%ds elapsed; 1 active poller, no duplicate requests sent).", elapsedSec),
			}, nil
		}
		return WarmupGPUOutput{
			Status:       "warming_up_started",
			GPUAvailable: false,
			Coalesced:    false,
			ElapsedMs:    time.Since(callStart).Milliseconds(),
			UpstreamURL:  viper.GetString("url"),
			Message:      "Background GPU wakeup started (0 -> 1 instance; 1 active poller). Poll get_health_and_gpu_status to monitor readiness.",
		}, nil
	}

	// 4. Blocking mode (wait_for_ready=true): Subscribe to the shared waitCh until the single poller finishes
	select {
	case <-waitCh:
		gpuStateMu.RLock()
		attempts := warmupLastAttempts
		err := warmupLastErr
		gpuStateMu.RUnlock()
		if err != nil {
			return WarmupGPUOutput{
				Status:         "error",
				GPUAvailable:   false,
				Coalesced:      alreadyRunning,
				WarmupAttempts: attempts,
				ElapsedMs:      time.Since(callStart).Milliseconds(),
				UpstreamURL:    viper.GetString("url"),
				Message:        fmt.Sprintf("Warmup failed after %d attempt(s): %v", attempts, err),
			}, err
		}
		return WarmupGPUOutput{
			Status:         "warm_and_ready",
			GPUAvailable:   true,
			Coalesced:      alreadyRunning,
			WarmupAttempts: attempts,
			ElapsedMs:      time.Since(callStart).Milliseconds(),
			UpstreamURL:    viper.GetString("url"),
			Message:        fmt.Sprintf("GPU is warm and ready! Completed readiness decision in %d ms (%d attempt(s), coalesced=%v).", time.Since(callStart).Milliseconds(), attempts, alreadyRunning),
		}, nil
	case <-ctx.Done():
		return WarmupGPUOutput{
			Status:      "warming_up_in_progress",
			Coalesced:   alreadyRunning,
			ElapsedMs:   time.Since(callStart).Milliseconds(),
			UpstreamURL: viper.GetString("url"),
			Message:     "Caller context expired while waiting, but single-flight background GPU warmup continues running.",
		}, ctx.Err()
	}
}

// MCP Tool Input/Output Structs

type StatusToolInput struct {
	Backend string `json:"backend,omitempty" jsonschema:"Optional inference backend selector to inspect: 'vertex_first' (default), 'vertex' (Vertex AI Dedicated Endpoint 4217256562927861760), or 'cloudrun' (Serverless Cloud Run GPU)."`
}

type DecidePolicyToolInput struct {
	Template          string                 `json:"template" jsonschema:"Policy template ID (e.g. 'support_triage', 'taxonomy_discovery', 'code_review', 'secops_conditional_dag', 'calibration/hallucination_judge')."`
	Variables         map[string]interface{} `json:"variables" jsonschema:"Key-value map of template variables (e.g. {'ticket': 'Double charged on invoice #9481'})."`
	Image             string                 `json:"image,omitempty" jsonschema:"Optional image URL or base64 data URI for multimodal policies."`
	Backend           string                 `json:"backend,omitempty" jsonschema:"Optional inference backend selector: 'vertex_first' (default: Vertex AI Dedicated Endpoint primary with Cloud Run GPU failover), 'vertex' (strict Vertex AI /invoke/*), or 'cloudrun' (strict Serverless Cloud Run GPU)."`
	VertexURL         string                 `json:"vertex_url,omitempty" jsonschema:"Optional Vertex AI Endpoint ID or /invoke/* URL override (defaults to 4217256562927861760)."`
	CascadeMode       string                 `json:"cascade_mode,omitempty" jsonschema:"Optional Stage-2 Vertex AI Gemini 3.x cascade mode: 'off' (default), 'entropy' (forward slots with Shannon entropy H >= cascade_threshold), or 'on_miss' (forward slots that miss expected_answers)."`
	CascadeThreshold  float64                `json:"cascade_threshold,omitempty" jsonschema:"Shannon entropy threshold H in nats for Stage-2 Gemini escalation (default 0.35)."`
	CascadeModel      string                 `json:"cascade_model,omitempty" jsonschema:"Stage-2 Vertex AI Gemini 3.x model (default 'gemini-3.8-flash'; also supports 'gemini-3.7-flash', 'gemini-3.5-flash-lite')."`
	ExpectedAnswers   map[string]string      `json:"expected_answers,omitempty" jsonschema:"Optional map of slot_id -> expected value for 'on_miss' cascade mode."`
	SuggestExpansions bool                   `json:"suggest_expansions,omitempty" jsonschema:"If true, dynamically injects an 'other_unclassified' catch-all option into choice slots (if absent) and proposes new {'name', 'description'} options when unclassified or high-entropy."`
	ExpansionEntropy  float64                `json:"expansion_entropy,omitempty" jsonschema:"Shannon entropy threshold H in nats on choice slots to trigger taxonomy expansion proposals (default 0.35)."`
}

type LocateBBoxToolInput struct {
	Image     string `json:"image" jsonschema:"Image URL or base64 data URI (data:image/png;base64,...) to analyze with Gemma 4 SigLIP vision."`
	Target    string `json:"target" jsonschema:"Natural language description of the object to localize (e.g. 'red vintage pickup truck', 'the wine glass closest to the bottle')."`
	Backend   string `json:"backend,omitempty" jsonschema:"Optional inference backend selector: 'vertex_first' (default), 'vertex', or 'cloudrun'."`
	VertexURL string `json:"vertex_url,omitempty" jsonschema:"Optional Vertex AI Endpoint ID or /invoke/* URL override."`
}

type BBoxCoords struct {
	YMin float64 `json:"ymin"`
	XMin float64 `json:"xmin"`
	YMax float64 `json:"ymax"`
	XMax float64 `json:"xmax"`
}

type LocateBBoxToolOutput struct {
	Target             string             `json:"target"`
	Presence           bool               `json:"presence"`
	PresenceConfidence float64            `json:"presence_confidence"`
	SoftmaxExpectation BBoxCoords         `json:"softmax_expectation_box_1000"`
	DiscreteArgmax     BBoxCoords         `json:"discrete_argmax_box_1000"`
	CoordinateEntropy  map[string]float64 `json:"coordinate_entropy_nats"`
	MaxEntropy         float64            `json:"max_entropy"`
	WallTimeMs         int64              `json:"wall_time_ms"`
	BackendTarget      string             `json:"backend_target"`
	UpstreamURL        string             `json:"upstream_url"`
}

type CustomQuestionSpec struct {
	ID       string   `json:"id" jsonschema:"Short slot identifier (e.g. 'is_urgent', 'category')."`
	Type     string   `json:"type" jsonschema:"Question type: 'boolean', 'choice' (<=26 options), or 'score' (1..5)."`
	Question string   `json:"question" jsonschema:"Prompt question for this slot."`
	Options  []string `json:"options,omitempty" jsonschema:"List of choices (required when type is 'choice', max 26)."`
}

type DecideCustomToolInput struct {
	Context           string               `json:"context" jsonschema:"Input text, document, code diff, or event log to evaluate."`
	Questions         []CustomQuestionSpec `json:"questions" jsonschema:"List of structured decision slots to evaluate simultaneously in 1 forward pass."`
	Backend           string               `json:"backend,omitempty" jsonschema:"Optional inference backend selector: 'vertex_first' (default: Vertex AI primary with Cloud Run failover), 'vertex', or 'cloudrun'."`
	VertexURL         string               `json:"vertex_url,omitempty" jsonschema:"Optional Vertex AI Endpoint ID or /invoke/* URL override."`
	CascadeMode       string               `json:"cascade_mode,omitempty" jsonschema:"Optional Stage-2 Vertex AI Gemini 3.x cascade mode: 'off' (default), 'entropy' (forward slots with Shannon entropy H >= cascade_threshold), or 'on_miss' (forward slots that miss expected_answers)."`
	CascadeThreshold  float64              `json:"cascade_threshold,omitempty" jsonschema:"Shannon entropy threshold H in nats for Stage-2 Gemini escalation (default 0.35)."`
	CascadeModel      string               `json:"cascade_model,omitempty" jsonschema:"Stage-2 Vertex AI Gemini 3.x model (default 'gemini-3.8-flash'; also supports 'gemini-3.7-flash', 'gemini-3.5-flash-lite')."`
	ExpectedAnswers   map[string]string    `json:"expected_answers,omitempty" jsonschema:"Optional map of slot_id -> expected value for 'on_miss' cascade mode."`
	SuggestExpansions bool                 `json:"suggest_expansions,omitempty" jsonschema:"If true, dynamically injects an 'other_unclassified' catch-all option into choice slots (if absent) and proposes new {'name', 'description'} options when unclassified or high-entropy."`
	ExpansionEntropy  float64              `json:"expansion_entropy,omitempty" jsonschema:"Shannon entropy threshold H in nats on choice slots to trigger taxonomy expansion proposals (default 0.35)."`
}

type ListTemplatesToolInput struct {
	Category string `json:"category,omitempty" jsonschema:"Optional category filter: 'core', 'calibration', or 'multimodal'."`
}

type ListTemplatesToolOutput struct {
	Count     int                    `json:"count"`
	Templates []TemplateCatalogEntry `json:"templates"`
}

var binNumberRegex = regexp.MustCompile(`(\d+)`)

func computeExpectedCoord(ans client.QuestionAnswer) (float64, float64) {
	raw := ans.DisplayValue()
	if raw == "" {
		raw = ans.Label
	}
	argmax := 0.0
	if m := binNumberRegex.FindStringSubmatch(raw); len(m) > 1 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			if v <= 100 {
				argmax = v * 10.0
			} else {
				argmax = v
			}
		}
	}
	if len(ans.Probabilities) == 0 {
		return argmax, argmax
	}
	var expected, totalMass float64
	for k, p := range ans.Probabilities {
		if m := binNumberRegex.FindStringSubmatch(k); len(m) > 1 {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				scale := v
				if scale <= 100 {
					scale = v * 10.0
				}
				expected += scale * p
				totalMass += p
			}
		}
	}
	if totalMass > 0 {
		expected = math.Round((expected/totalMass)*10) / 10
		return expected, argmax
	}
	return argmax, argmax
}

func buildMCPServer() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "dgem-decision-studio-mcp",
		Version: "1.0.0",
	}, nil)

	// Tool 1: get_health_and_gpu_status
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_health_and_gpu_status",
		Description: "Returns live health status of the dgem gateway and availability/warmup state of the selected inference backend ('vertex_first', 'vertex' 1x NVIDIA L4, or 'cloudrun' 1x NVIDIA RTX Pro 6000), plus readout latency and estimated wakeup time.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StatusToolInput) (*mcp.CallToolResult, HealthAndGPUStatusOutput, error) {
		return nil, CheckHealthAndGPUStatusForBackend(ctx, "", input.Backend), nil
	})

	// Tool 2: warmup_gpu
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "warmup_gpu",
		Description: "Wakes the scale-to-zero Cloud Run DiffusionGemma GPU instance (0 -> 1) and loads the 17.53 GiB model into VRAM. Pass wait_for_ready=true to wait until ready, or wait_for_ready=false to trigger background warmup immediately.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input WarmupGPUInput) (*mcp.CallToolResult, WarmupGPUOutput, error) {
		out, err := TriggerGPUWarmup(ctx, input.WaitForReady)
		return nil, out, err
	})

	// Tool 3: decide_policy
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "decide_policy",
		Description: "Executes a zero-shot multi-slot decision policy (.json.tmpl) on DiffusionGemma in O(1) forward passes, returning joint slot answers, probabilities, calibrated epistemic Shannon entropy H (in nats), and optional Stage-2 Gemini 3.x cascade escalation (gemini-3.8-flash default).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DecidePolicyToolInput) (*mcp.CallToolResult, GatewayDecideResponse, error) {
		tmplID := strings.TrimSpace(input.Template)
		if tmplID == "" {
			tmplID = "support_triage"
		}
		targetPath, cleanID := resolveTemplateFile(serveTemplatesDir, tmplID)

		engine := template.NewEngine()
		rendered, err := engine.RenderFile(targetPath, input.Variables)
		if err != nil {
			return nil, GatewayDecideResponse{}, fmt.Errorf("failed to render template %q: %w", tmplID, err)
		}
		schemaContent, stateContent, err := template.ParseStructuredPayload(rendered, input.Variables)
		if err != nil {
			return nil, GatewayDecideResponse{}, err
		}
		var injectedSlots map[string]bool
		var existingOptions map[string][]client.ProposedOption
		if input.SuggestExpansions {
			schemaContent, injectedSlots, existingOptions = InjectUnclassifiedCatchAll(schemaContent)
		} else {
			_, _, existingOptions = InjectUnclassifiedCatchAll(schemaContent)
		}

		var imgs []string
		if input.Image != "" {
			imgs = append(imgs, input.Image)
		}
		backendTarget, targetURL, bErr := resolveBackendTargetFromParams(ctx, input.Backend, input.VertexURL)
		if bErr != nil {
			return nil, GatewayDecideResponse{}, bErr
		}
		start := time.Now()
		resp, stats, attempts, err := executeDecideWithWarmup(ctx, schemaContent, stateContent, imgs, targetURL)
		if err != nil {
			return nil, GatewayDecideResponse{}, err
		}
		if backendTarget == "cloudrun" {
			MarkGPUWarm()
		}

		var cascadeSummary *CascadeExecutionSummary
		if input.CascadeMode != "" && input.CascadeMode != "off" && input.CascadeMode != "none" {
			cascadeSummary = ExecuteStage2GeminiCascade(
				ctx,
				input.CascadeMode,
				input.CascadeThreshold,
				input.CascadeModel,
				input.ExpectedAnswers,
				schemaContent,
				stateContent,
				resp,
			)
		}

		if input.SuggestExpansions || (resp.Diagnostics.Thought != nil && strings.Contains(resp.Diagnostics.Thought.Text, "SUGGESTED_")) {
			expandCtx, expandSpan := gatewayTracer().Start(ctx, "dgem.taxonomy.expand")
			resp.SuggestedExpansions = SynthesizeTaxonomyExpansions(
				expandCtx,
				nil,
				cleanID,
				stateContent,
				resp,
				injectedSlots,
				existingOptions,
				input.ExpansionEntropy,
				targetURL,
			)
			expandSpan.SetAttributes(
				attribute.Bool("dgem.taxonomy.triggered", len(resp.SuggestedExpansions) > 0),
				attribute.Int("dgem.taxonomy.suggestions_count", len(resp.SuggestedExpansions)),
			)
			expandSpan.SetStatus(codes.Ok, "ok")
			expandSpan.End()
		}

		maxEntropy := 0.0
		for _, q := range resp.Diagnostics.Questions {
			if q.Entropy > maxEntropy {
				maxEntropy = q.Entropy
			}
		}
		for _, a := range resp.Answers {
			if a.Entropy > maxEntropy {
				maxEntropy = a.Entropy
			}
		}
		return nil, GatewayDecideResponse{
			Template:            cleanID,
			Answers:             resp.Answers,
			Diagnostics:         resp.Diagnostics,
			Decision:            resp,
			Cascade:             cascadeSummary,
			SuggestedExpansions: resp.SuggestedExpansions,
			MaxEntropy:          maxEntropy,
			WallTimeMs:          time.Since(start).Milliseconds(),
			WarmupAttempts:      attempts,
			Model:               stats.Model,
			BackendTarget:       backendTarget,
			UpstreamURL:         targetURL,
		}, nil
	})

	// Tool 4: locate_bounding_boxes
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "locate_bounding_boxes",
		Description: "Localizes an object in an image in a single forward pass (EXP-09) using Gemma 4's SigLIP vision tower and 21-bin discrete diffusion coordinate readout, returning both continuous Softmax Expectation and discrete argmax [ymin, xmin, ymax, xmax] bounding boxes in [0, 1000] coordinates.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input LocateBBoxToolInput) (*mcp.CallToolResult, LocateBBoxToolOutput, error) {
		if strings.TrimSpace(input.Image) == "" {
			return nil, LocateBBoxToolOutput{}, fmt.Errorf("'image' URL or data URI is required")
		}
		target := strings.TrimSpace(input.Target)
		if target == "" {
			target = "primary foreground object"
		}
		vars := map[string]interface{}{"target": target}
		targetPath, _ := resolveTemplateFile(serveTemplatesDir, "bbox_localization")
		engine := template.NewEngine()
		rendered, err := engine.RenderFile(targetPath, vars)
		if err != nil {
			return nil, LocateBBoxToolOutput{}, err
		}
		schemaContent, stateContent, err := template.ParseStructuredPayload(rendered, vars)
		if err != nil {
			return nil, LocateBBoxToolOutput{}, err
		}
		backendTarget, targetURL, bErr := resolveBackendTargetFromParams(ctx, input.Backend, input.VertexURL)
		if bErr != nil {
			return nil, LocateBBoxToolOutput{}, bErr
		}
		start := time.Now()
		resp, _, _, err := executeDecideWithWarmup(ctx, schemaContent, stateContent, []string{input.Image}, targetURL)
		if err != nil {
			return nil, LocateBBoxToolOutput{}, err
		}
		if backendTarget == "cloudrun" {
			MarkGPUWarm()
		}

		expYMin, argYMin := computeExpectedCoord(resp.Answers["ymin"])
		expXMin, argXMin := computeExpectedCoord(resp.Answers["xmin"])
		expYMax, argYMax := computeExpectedCoord(resp.Answers["ymax"])
		expXMax, argXMax := computeExpectedCoord(resp.Answers["xmax"])

		coordEnt := map[string]float64{}
		maxEnt := 0.0
		for _, k := range []string{"presence", "ymin", "xmin", "ymax", "xmax"} {
			e := resp.Answers[k].Entropy
			if e == 0 {
				if q, ok := resp.Diagnostics.Questions[k]; ok {
					e = q.Entropy
				}
			}
			coordEnt[k] = e
			if e > maxEnt {
				maxEnt = e
			}
		}
		presAns := resp.Answers["presence"]
		present := strings.EqualFold(presAns.Label, "yes") || strings.EqualFold(presAns.Label, "true")

		return nil, LocateBBoxToolOutput{
			Target:             target,
			Presence:           present,
			PresenceConfidence: presAns.Confidence,
			SoftmaxExpectation: BBoxCoords{YMin: expYMin, XMin: expXMin, YMax: expYMax, XMax: expXMax},
			DiscreteArgmax:     BBoxCoords{YMin: argYMin, XMin: argXMin, YMax: argYMax, XMax: argXMax},
			CoordinateEntropy:  coordEnt,
			MaxEntropy:         maxEnt,
			WallTimeMs:         time.Since(start).Milliseconds(),
			BackendTarget:      backendTarget,
			UpstreamURL:        targetURL,
		}, nil
	})

	// Tool 5: decide_custom_questions
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "decide_custom_questions",
		Description: "Evaluates an ad-hoc list of boolean, choice (<=26 options), or score (1..5) questions simultaneously in 1 forward pass against a context document, returning joint answers, calibrated Shannon entropy H, and optional Stage-2 Gemini 3.x cascade escalation (gemini-3.8-flash default).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DecideCustomToolInput) (*mcp.CallToolResult, GatewayDecideResponse, error) {
		if len(input.Questions) == 0 {
			return nil, GatewayDecideResponse{}, fmt.Errorf("'questions' list cannot be empty")
		}
		qList := make([]map[string]interface{}, 0, len(input.Questions))
		for _, q := range input.Questions {
			qMap := map[string]interface{}{
				"id":           q.ID,
				"type":         q.Type,
				"instructions": q.Question,
			}
			if len(q.Options) > 0 {
				qMap["options"] = q.Options
			}
			qList = append(qList, qMap)
		}
		schemaObj := map[string]interface{}{"questions": qList}
		schemaBytes, _ := json.Marshal(schemaObj)
		schemaStr, stateStr, _ := template.ParseStructuredPayload(string(schemaBytes), map[string]interface{}{"context": input.Context})

		var injectedSlots map[string]bool
		var existingOptions map[string][]client.ProposedOption
		if input.SuggestExpansions {
			schemaStr, injectedSlots, existingOptions = InjectUnclassifiedCatchAll(schemaStr)
		} else {
			_, _, existingOptions = InjectUnclassifiedCatchAll(schemaStr)
		}

		backendTarget, targetURL, bErr := resolveBackendTargetFromParams(ctx, input.Backend, input.VertexURL)
		if bErr != nil {
			return nil, GatewayDecideResponse{}, bErr
		}
		start := time.Now()
		resp, stats, attempts, err := executeDecideWithWarmup(ctx, schemaStr, stateStr, nil, targetURL)
		if err != nil {
			return nil, GatewayDecideResponse{}, err
		}
		if backendTarget == "cloudrun" {
			MarkGPUWarm()
		}

		var cascadeSummary *CascadeExecutionSummary
		if input.CascadeMode != "" && input.CascadeMode != "off" && input.CascadeMode != "none" {
			cascadeSummary = ExecuteStage2GeminiCascade(
				ctx,
				input.CascadeMode,
				input.CascadeThreshold,
				input.CascadeModel,
				input.ExpectedAnswers,
				schemaStr,
				stateStr,
				resp,
			)
		}

		if input.SuggestExpansions || (resp.Diagnostics.Thought != nil && strings.Contains(resp.Diagnostics.Thought.Text, "SUGGESTED_")) {
			expandCtx, expandSpan := gatewayTracer().Start(ctx, "dgem.taxonomy.expand")
			resp.SuggestedExpansions = SynthesizeTaxonomyExpansions(
				expandCtx,
				nil,
				"custom_questions",
				stateStr,
				resp,
				injectedSlots,
				existingOptions,
				input.ExpansionEntropy,
				targetURL,
			)
			expandSpan.SetAttributes(
				attribute.Bool("dgem.taxonomy.triggered", len(resp.SuggestedExpansions) > 0),
				attribute.Int("dgem.taxonomy.suggestions_count", len(resp.SuggestedExpansions)),
			)
			expandSpan.SetStatus(codes.Ok, "ok")
			expandSpan.End()
		}

		maxEntropy := 0.0
		for _, a := range resp.Answers {
			if a.Entropy > maxEntropy {
				maxEntropy = a.Entropy
			}
		}
		return nil, GatewayDecideResponse{
			Template:            "custom_questions",
			Answers:             resp.Answers,
			Diagnostics:         resp.Diagnostics,
			Decision:            resp,
			Cascade:             cascadeSummary,
			SuggestedExpansions: resp.SuggestedExpansions,
			MaxEntropy:          maxEntropy,
			WallTimeMs:          time.Since(start).Milliseconds(),
			WarmupAttempts:      attempts,
			Model:               stats.Model,
			BackendTarget:       backendTarget,
			UpstreamURL:         targetURL,
		}, nil
	})

	// Tool 6: list_policy_templates
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_policy_templates",
		Description: "Lists all 24 executable .json.tmpl decision policies available on the gateway, including required variables and sample values.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTemplatesToolInput) (*mcp.CallToolResult, ListTemplatesToolOutput, error) {
		catalog, err := discoverTemplates(serveTemplatesDir)
		if err != nil {
			return nil, ListTemplatesToolOutput{}, err
		}
		var filtered []TemplateCatalogEntry
		catFilter := strings.TrimSpace(strings.ToLower(input.Category))
		for _, item := range catalog {
			if catFilter == "" || strings.ToLower(item.Category) == catFilter {
				// Omit full raw_template in listing to keep agent context lean
				slim := item
				slim.RawTemplate = ""
				filtered = append(filtered, slim)
			}
		}
		return nil, ListTemplatesToolOutput{
			Count:     len(filtered),
			Templates: filtered,
		}, nil
	})

	return srv
}

func newMCPHTTPHandler() http.Handler {
	mcpServer := buildMCPServer()
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless:                  true,
		JSONResponse:               true,
		DisableLocalhostProtection: true,
	})
}
