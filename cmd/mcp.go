package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
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
)

var (
	gpuStateMu        sync.RWMutex
	lastWarmTimestamp time.Time
	warmupInProgress  bool
	warmupStartedAt   time.Time
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

// MarkGPUWarm records that the upstream vLLM engine successfully completed a decision readout.
func MarkGPUWarm() {
	gpuStateMu.Lock()
	lastWarmTimestamp = time.Now()
	warmupInProgress = false
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
	EstimatedWakeSeconds  int    `json:"estimated_wake_seconds"`
	UpstreamURL           string `json:"upstream_url"`
	Model                 string `json:"model"`
	GPUTier               string `json:"gpu_tier"`
	TemplatesAvailable    int    `json:"templates_available"`
	AuthenticatedUser     string `json:"authenticated_user,omitempty"`
	Detail                string `json:"detail"`
}

// CheckHealthAndGPUStatus inspects both the gateway and the upstream Cloud Run GPU service.
func CheckHealthAndGPUStatus(ctx context.Context, userEmail string) HealthAndGPUStatusOutput {
	upstream := strings.TrimSuffix(viper.GetString("url"), "/v1")
	healthURL := strings.TrimSuffix(upstream, "/") + "/health"

	gpuStateMu.RLock()
	lastWarm := lastWarmTimestamp
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
		EstimatedWakeSeconds: 210,
	}

	if warming {
		out.WarmupElapsedSeconds = int(time.Since(warmStart).Seconds())
	}

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err == nil {
		if viper.GetBool("gcp_auth") || viper.GetString("iap_client_id") != "" {
			if tok := FetchGCPIdentityToken(viper.GetString("iap_client_id"), upstream); tok != "" {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
		}
		hc := &http.Client{Timeout: 3500 * time.Millisecond}
		if resp, doErr := hc.Do(req); doErr == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				out.ContainerReachable = true
			}
		}
	}

	if !lastWarm.IsZero() && time.Since(lastWarm) < 14*time.Minute && out.ContainerReachable {
		out.GPUAvailable = true
		out.GPUState = "warm_and_ready"
		out.SecondsSinceLastRead = int(time.Since(lastWarm).Seconds())
		out.EstimatedWakeSeconds = 0
		out.Detail = fmt.Sprintf("vLLM EngineCore is warm and ready (last readout %ds ago; ~450-700ms latency).", out.SecondsSinceLastRead)
		return out
	}

	if out.ContainerReachable {
		out.GPUState = "warming_up"
		out.EstimatedWakeSeconds = 90
		out.Detail = "GPU container is allocated and reachable; vLLM EngineCore is loading 17.53 GiB bfloat16/NVFP4 weights over GCS FUSE."
		return out
	}

	if warming {
		out.GPUState = "warming_up"
		out.Detail = fmt.Sprintf("GPU warmup in progress (%ds elapsed of ~210s cold-start). Container scaling 0 -> 1.", out.WarmupElapsedSeconds)
		return out
	}

	out.GPUState = "scaled_to_zero"
	out.Detail = "GPU service is scaled to 0 instances ($0.00/hr idle). Call warmup_gpu or execute any decision to wake automatically."
	return out
}

// WarmupGPUInput defines arguments for the warmup_gpu MCP tool and POST /api/warmup.
type WarmupGPUInput struct {
	WaitForReady bool `json:"wait_for_ready,omitempty" jsonschema:"If true (default), waits until vLLM finishes loading weights and confirms a test decision (~3.5-4.5m if cold, ~500ms if already warm). If false, triggers background wakeup and returns immediately."`
}

// WarmupGPUOutput is returned by warmup_gpu and POST /api/warmup.
type WarmupGPUOutput struct {
	Status         string `json:"status"` // "warm_and_ready" or "warming_up_started"
	GPUAvailable   bool   `json:"gpu_available"`
	WarmupAttempts int    `json:"warmup_attempts,omitempty"`
	ElapsedMs      int64  `json:"elapsed_ms"`
	UpstreamURL    string `json:"upstream_url"`
	Message        string `json:"message"`
}

// TriggerGPUWarmup starts or awaits a cold-start wakeup probe against the upstream GPU service.
func TriggerGPUWarmup(ctx context.Context, waitForReady bool) (WarmupGPUOutput, error) {
	start := time.Now()
	gpuStateMu.Lock()
	if !warmupInProgress {
		warmupInProgress = true
		warmupStartedAt = start
	}
	gpuStateMu.Unlock()

	probeTask := func(runCtx context.Context) (int, error) {
		engine := template.NewEngine()
		vars := map[string]interface{}{"ticket": "GPU warmup readiness probe"}
		targetPath := filepath.Join(serveTemplatesDir, "support_triage.json.tmpl")
		rendered, err := engine.RenderFile(targetPath, vars)
		if err != nil {
			// Fallback minimal schema if templates dir is elsewhere
			rendered = `{"questions":[{"id":"ready","type":"boolean","prompt":"Is the system ready?"}]}`
		}
		schemaContent, stateContent, err := template.ParseStructuredPayload(rendered, vars)
		if err != nil {
			return 1, err
		}
		_, _, attempts, err := executeDecideWithWarmup(runCtx, schemaContent, stateContent, nil)
		if err == nil {
			MarkGPUWarm()
		} else {
			gpuStateMu.Lock()
			warmupInProgress = false
			gpuStateMu.Unlock()
		}
		return attempts, err
	}

	if !waitForReady {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), serveWakeupTimeout)
			defer cancel()
			_, _ = probeTask(bgCtx)
		}()
		return WarmupGPUOutput{
			Status:       "warming_up_started",
			GPUAvailable: false,
			ElapsedMs:    time.Since(start).Milliseconds(),
			UpstreamURL:  viper.GetString("url"),
			Message:      "Background GPU wakeup triggered (0 -> 1 instance). Poll get_health_and_gpu_status to monitor readiness.",
		}, nil
	}

	attempts, err := probeTask(ctx)
	if err != nil {
		return WarmupGPUOutput{
			Status:         "error",
			GPUAvailable:   false,
			WarmupAttempts: attempts,
			ElapsedMs:      time.Since(start).Milliseconds(),
			UpstreamURL:    viper.GetString("url"),
			Message:        fmt.Sprintf("Warmup failed after %d attempt(s): %v", attempts, err),
		}, err
	}

	return WarmupGPUOutput{
		Status:         "warm_and_ready",
		GPUAvailable:   true,
		WarmupAttempts: attempts,
		ElapsedMs:      time.Since(start).Milliseconds(),
		UpstreamURL:    viper.GetString("url"),
		Message:        fmt.Sprintf("GPU is warm and ready! Completed readiness decision in %d ms (%d attempt(s)).", time.Since(start).Milliseconds(), attempts),
	}, nil
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
		cleanID := strings.TrimSuffix(tmplID, ".json.tmpl")
		targetPath := filepath.Join(serveTemplatesDir, cleanID+".json.tmpl")

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
		targetPath := filepath.Join(serveTemplatesDir, "multimodal", "bbox_single.json.tmpl")
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
				"id":     q.ID,
				"type":   q.Type,
				"prompt": q.Question,
			}
			if len(q.Options) > 0 {
				qMap["options"] = q.Options
			}
			qList = append(qList, qMap)
		}
		schemaObj := map[string]interface{}{"questions": qList}
		schemaBytes, _ := json.Marshal(schemaObj)
		stateBytes, _ := json.Marshal(map[string]interface{}{"context": input.Context})

		start := time.Now()
		resp, stats, attempts, err := executeDecideWithWarmup(ctx, string(schemaBytes), string(stateBytes), nil)
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
