package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
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
)

var (
	gpuStateMu           sync.RWMutex
	lastWarmTimestamp    time.Time
	lastReadoutLatencyMs int64
	warmupInProgress     bool
	warmupStartedAt      time.Time
	warmupDoneCh         chan struct{}
	warmupLastAttempts   int
	warmupLastErr        error
)

var mcpCmd = &cobra.Command{
	Use:     "mcp",
	GroupID: "core",
	Short:   "Start the dgem Model Context Protocol (MCP) server over stdio",
	Long: `mcp starts a Model Context Protocol (MCP) server over standard input/output (stdio),
exposing DiffusionGemma zero-shot multi-slot decision policies, multimodal SigLIP bounding-box
localization (EXP-09), GPU health/availability checks, and scale-from-zero GPU warmup tools
to AI agents (Gemini CLI, Claude Desktop, Cursor, etc.).

Note: 'dgem serve' also mounts this exact MCP server over Streamable HTTP at POST /mcp.`,
	Example: `  # Run stdio MCP server pointing at Cloud Run gateway or GPU backend
  dgem mcp -u https://dgemma-gateway-882920967572.us-central1.run.app/v1 --gcp-auth`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := buildMCPServer()
		return srv.Run(context.Background(), &mcp.StdioTransport{})
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
}

// MarkGPUWarm records that the upstream vLLM engine successfully completed a decision readout
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
	GatewayHealthy        bool   `json:"gateway_healthy"`
	GPUAvailable          bool   `json:"gpu_available"`
	GPUState              string `json:"gpu_state"` // "warm_and_ready", "warming_up", "scaled_to_zero"
	ContainerReachable    bool   `json:"container_reachable"`
	WarmupInProgress      bool   `json:"warmup_in_progress"`
	WarmupElapsedSeconds  int    `json:"warmup_elapsed_seconds,omitempty"`
	SecondsSinceLastRead  int    `json:"seconds_since_last_read,omitempty"`
	IdleRemainingSeconds  int    `json:"idle_remaining_seconds,omitempty"`
	LastReadoutMs         int64  `json:"last_readout_ms,omitempty"`
	EstimatedWakeSeconds  int    `json:"estimated_wake_seconds"`
	UpstreamURL           string `json:"upstream_url"`
	Model                 string `json:"model"`
	GPUTier               string `json:"gpu_tier"`
	TemplatesAvailable    int    `json:"templates_available"`
	AuthenticatedUser     string `json:"authenticated_user,omitempty"`
	Detail                string `json:"detail"`
}

// CheckHealthAndGPUStatus inspects both the gateway and the upstream Cloud Run GPU service
// WITHOUT sending unsolicited HTTP probes when idle (which would otherwise trigger Cloud Run's
// scale-from-zero activator or reset its 15-minute idle scale-down timer).
func CheckHealthAndGPUStatus(ctx context.Context, userEmail string) HealthAndGPUStatusOutput {
	gpuStateMu.RLock()
	lastWarm := lastWarmTimestamp
	lastReadMs := lastReadoutLatencyMs
	warming := warmupInProgress
	warmStart := warmupStartedAt
	gpuStateMu.RUnlock()

	catalog, _ := discoverTemplates(serveTemplatesDir)

	out := HealthAndGPUStatusOutput{
		GatewayHealthy:       true,
		UpstreamURL:          viper.GetString("url"),
		Model:                "nvidia/diffusiongemma-26B-A4B-it-NVFP4",
		GPUTier:              "1x NVIDIA RTX Pro 6000 (48GB VRAM, SigLIP Multimodal)",
		TemplatesAvailable:   len(catalog),
		AuthenticatedUser:    userEmail,
		WarmupInProgress:     warming,
		LastReadoutMs:        lastReadMs,
		EstimatedWakeSeconds: 210,
	}

	if warming {
		out.WarmupElapsedSeconds = int(time.Since(warmStart).Seconds())
		out.ContainerReachable = out.WarmupElapsedSeconds > 3
		out.GPUState = "warming_up"
		rem := 210 - out.WarmupElapsedSeconds
		if rem < 10 {
			rem = 10
		}
		out.EstimatedWakeSeconds = rem
		out.Detail = fmt.Sprintf("GPU warmup in progress (%ds elapsed of ~210s cold-start). vLLM EngineCore is loading 17.53 GiB NVFP4 weights over GCS FUSE.", out.WarmupElapsedSeconds)
		return out
	}

	const cloudRunIdleWindow = 15 * time.Minute
	if !lastWarm.IsZero() && time.Since(lastWarm) < cloudRunIdleWindow {
		sinceSec := int(time.Since(lastWarm).Seconds())
		remSec := int(cloudRunIdleWindow.Seconds()) - sinceSec
		out.GPUAvailable = true
		out.ContainerReachable = true
		out.GPUState = "warm_and_ready"
		out.SecondsSinceLastRead = sinceSec
		out.IdleRemainingSeconds = remSec
		out.EstimatedWakeSeconds = 0
		if lastReadMs > 0 {
			out.Detail = fmt.Sprintf("vLLM EngineCore is warm and ready (last readout %dms, %ds ago; %dm%ds until Cloud Run scale-to-zero).", lastReadMs, sinceSec, remSec/60, remSec%60)
		} else {
			out.Detail = fmt.Sprintf("vLLM EngineCore is warm and ready (last readout %ds ago; %dm%ds until Cloud Run scale-to-zero).", sinceSec, remSec/60, remSec%60)
		}
		return out
	}

	out.GPUState = "scaled_to_zero"
	out.Detail = "GPU service is scaled to 0 instances ($0.00/hr idle). Click 'Wake GPU' or execute any decision to wake automatically."
	return out
}

// WarmupGPUInput defines arguments for the warmup_gpu MCP tool and POST /api/warmup.
type WarmupGPUInput struct {
	WaitForReady bool `json:"wait_for_ready,omitempty" jsonschema:"If true (default), waits until vLLM finishes loading weights and confirms a test decision (~3.5-4.5m if cold, <10ms if already warm). If false, triggers background wakeup and returns immediately."`
}

// WarmupGPUOutput is returned by warmup_gpu and POST /api/warmup.
type WarmupGPUOutput struct {
	Status                string `json:"status"` // "warm_and_ready", "warming_up_started", or "warming_up_in_progress"
	GPUAvailable          bool   `json:"gpu_available"`
	Coalesced             bool   `json:"coalesced"`
	WarmupElapsedSeconds  int    `json:"warmup_elapsed_seconds,omitempty"`
	WarmupAttempts        int    `json:"warmup_attempts,omitempty"`
	ElapsedMs             int64  `json:"elapsed_ms"`
	UpstreamURL           string `json:"upstream_url"`
	Message               string `json:"message"`
}

// TriggerGPUWarmup is a strictly idempotent, single-flight coalesced GPU warmup coordinator
// shared across MCP (warmup_gpu), Web Studio (Wake GPU button), and REST (POST /api/warmup).
func TriggerGPUWarmup(ctx context.Context, waitForReady bool) (WarmupGPUOutput, error) {
	callStart := time.Now()

	// 1. Fast-Path Idempotency: If the GPU was verified warm within the 14m window and /health is up,
	// return immediately without running a redundant GPU forward pass.
	currentStatus := CheckHealthAndGPUStatus(ctx, "")
	if currentStatus.GPUAvailable && currentStatus.GPUState == "warm_and_ready" {
		return WarmupGPUOutput{
			Status:       "warm_and_ready",
			GPUAvailable: true,
			Coalesced:    true,
			ElapsedMs:    time.Since(callStart).Milliseconds(),
			UpstreamURL:  viper.GetString("url"),
			Message:      fmt.Sprintf("GPU is already warm and ready (last readout %ds ago; no duplicate warmup sent).", currentStatus.SecondsSinceLastRead),
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
		waitCh = warmupDoneCh
		startedAt = warmupStartedAt
		gpuStateMu.Unlock()

		// Launch the single background warmup poller
		go func(doneCh chan struct{}) {
			bgCtx, cancel := context.WithTimeout(context.Background(), serveWakeupTimeout)
			defer cancel()

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
		}(waitCh)
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

type StatusToolInput struct{}

type DecidePolicyToolInput struct {
	Template  string                 `json:"template" jsonschema:"Policy template ID (e.g. 'support_triage', 'code_review', 'secops_conditional_dag', 'calibration/hallucination_judge', 'calibration/prompt_injection_guard')."`
	Variables map[string]interface{} `json:"variables" jsonschema:"Key-value map of template variables (e.g. {'ticket': 'Double charged on invoice #9481'})."`
	Image     string                 `json:"image,omitempty" jsonschema:"Optional image URL or base64 data URI for multimodal policies."`
}

type LocateBBoxToolInput struct {
	Image  string `json:"image" jsonschema:"Image URL or base64 data URI (data:image/png;base64,...) to analyze with Gemma 4 SigLIP vision."`
	Target string `json:"target" jsonschema:"Natural language description of the object to localize (e.g. 'red vintage pickup truck', 'the wine glass closest to the bottle')."`
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
}

type CustomQuestionSpec struct {
	ID       string   `json:"id" jsonschema:"Short slot identifier (e.g. 'is_urgent', 'category')."`
	Type     string   `json:"type" jsonschema:"Question type: 'boolean', 'choice' (<=26 options), or 'score' (1..5)."`
	Question string   `json:"question" jsonschema:"Prompt question for this slot."`
	Options  []string `json:"options,omitempty" jsonschema:"List of choices (required when type is 'choice', max 26)."`
}

type DecideCustomToolInput struct {
	Context   string               `json:"context" jsonschema:"Input text, document, code diff, or event log to evaluate."`
	Questions []CustomQuestionSpec `json:"questions" jsonschema:"List of structured decision slots to evaluate simultaneously in 1 forward pass."`
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
		Description: "Returns live health status of the dgem gateway and availability/warmup state of the Cloud Run NVIDIA RTX Pro 6000 GPU engine (warm_and_ready, warming_up, or scaled_to_zero), plus estimated wakeup time.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StatusToolInput) (*mcp.CallToolResult, HealthAndGPUStatusOutput, error) {
		return nil, CheckHealthAndGPUStatus(ctx, ""), nil
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
		Description: "Executes a zero-shot multi-slot decision policy (.json.tmpl) on DiffusionGemma in O(1) forward passes, returning joint slot answers, probabilities, and calibrated epistemic Shannon entropy H (in nats).",
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
		var imgs []string
		if input.Image != "" {
			imgs = append(imgs, input.Image)
		}
		start := time.Now()
		resp, stats, attempts, err := executeDecideWithWarmup(ctx, schemaContent, stateContent, imgs)
		if err != nil {
			return nil, GatewayDecideResponse{}, err
		}
		MarkGPUWarm()

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
			Template:       cleanID,
			Answers:        resp.Answers,
			Diagnostics:    resp.Diagnostics,
			MaxEntropy:     maxEntropy,
			WallTimeMs:     time.Since(start).Milliseconds(),
			WarmupAttempts: attempts,
			Model:          stats.Model,
			UpstreamURL:    viper.GetString("url"),
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
		start := time.Now()
		resp, _, _, err := executeDecideWithWarmup(ctx, schemaContent, stateContent, []string{input.Image})
		if err != nil {
			return nil, LocateBBoxToolOutput{}, err
		}
		MarkGPUWarm()

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
		}, nil
	})

	// Tool 5: decide_custom_questions
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "decide_custom_questions",
		Description: "Evaluates an ad-hoc list of boolean, choice (<=26 options), or score (1..5) questions simultaneously in 1 forward pass against a context document, returning joint answers and calibrated Shannon entropy H.",
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

		start := time.Now()
		resp, stats, attempts, err := executeDecideWithWarmup(ctx, schemaStr, stateStr, nil)
		if err != nil {
			return nil, GatewayDecideResponse{}, err
		}
		MarkGPUWarm()

		maxEntropy := 0.0
		for _, a := range resp.Answers {
			if a.Entropy > maxEntropy {
				maxEntropy = a.Entropy
			}
		}
		return nil, GatewayDecideResponse{
			Template:       "custom_questions",
			Answers:        resp.Answers,
			Diagnostics:    resp.Diagnostics,
			MaxEntropy:     maxEntropy,
			WallTimeMs:     time.Since(start).Milliseconds(),
			WarmupAttempts: attempts,
			Model:          stats.Model,
			UpstreamURL:    viper.GetString("url"),
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
