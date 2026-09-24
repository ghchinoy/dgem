package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghchinoy/dgem/pkg/client"
	"google.golang.org/genai"
)

const (
	DefaultCascadeGeminiModel  = "gemini-3.8-flash"
	DefaultCascadeGeminiModels = "gemini-3.8-flash,gemini-3.7-flash,gemini-3.5-flash-lite"
)

// CascadeSlotTelemetry records Stage 1 (dgemma) vs Stage 2 (Gemini 3.x) per-slot resolution.
type CascadeSlotTelemetry struct {
	QuestionID       string  `json:"question_id"`
	Escalated        bool    `json:"escalated"`
	Reason           string  `json:"reason,omitempty"`
	Stage1Value      string  `json:"stage1_value"`
	Stage1Confidence float64 `json:"stage1_confidence"`
	Stage1Entropy    float64 `json:"stage1_entropy"`
	Stage2Value      string  `json:"stage2_value,omitempty"`
	Stage2Confidence float64 `json:"stage2_confidence,omitempty"`
	Explanation      string  `json:"explanation,omitempty"`
}

// CascadeExecutionSummary summarizes Stage 2 Gemini escalation for a decision call.
type CascadeExecutionSummary struct {
	Mode           string                          `json:"mode"`
	Model          string                          `json:"model"`
	Threshold      float64                         `json:"threshold"`
	Triggered      bool                            `json:"triggered"`
	EscalatedCount int                             `json:"escalated_count"`
	TotalSlots     int                             `json:"total_slots"`
	LatencyMs      int64                           `json:"latency_ms"`
	Error          string                          `json:"error,omitempty"`
	Slots          map[string]CascadeSlotTelemetry `json:"slots,omitempty"`
}

var (
	cascadeGenaiMu     sync.Mutex
	cascadeGenaiClient *genai.Client
	cascadeGenaiProj   string
)

// SanitizeCascadeModel enforces the Gemini 3.x series mandate (defaulting to gemini-3.8-flash,
// remapping gemini-3.5-flash -> gemini-3.7-flash and gemini-3.1-flash-lite -> gemini-3.5-flash-lite,
// and upgrading any legacy gemini-1.x / gemini-2.x reference to gemini-3.8-flash).
func SanitizeCascadeModel(raw string) string {
	m := strings.TrimSpace(raw)
	if m == "" {
		if envM := strings.TrimSpace(os.Getenv("DGEM_CASCADE_MODEL")); envM != "" {
			m = envM
		} else if serveCascadeModel != "" {
			m = serveCascadeModel
		} else {
			m = DefaultCascadeGeminiModel
		}
	}
	lower := strings.ToLower(m)
	switch lower {
	case "gemini-3.5-flash":
		return "gemini-3.7-flash"
	case "gemini-3.1-flash-lite":
		return "gemini-3.5-flash-lite"
	}
	if strings.HasPrefix(lower, "gemini-2") || strings.HasPrefix(lower, "gemini-1") {
		return DefaultCascadeGeminiModel
	}
	return m
}

// GetConfiguredCascadeModels returns the ordered list of selectable Stage-2 Gemini 3.x models
// from DGEM_CASCADE_MODELS or --cascade-models (defaulting to gemini-3.8-flash, gemini-3.7-flash, gemini-3.5-flash-lite).
func GetConfiguredCascadeModels() []string {
	raw := strings.TrimSpace(os.Getenv("DGEM_CASCADE_MODELS"))
	if raw == "" {
		raw = strings.TrimSpace(serveCascadeModels)
	}
	if raw == "" {
		raw = DefaultCascadeGeminiModels
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]bool)
	var out []string
	for _, p := range parts {
		m := SanitizeCascadeModel(p)
		if m != "" && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return []string{"gemini-3.8-flash", "gemini-3.7-flash", "gemini-3.5-flash-lite"}
	}
	return out
}

func getSharedGenaiClient(ctx context.Context) (*genai.Client, string, error) {
	cascadeGenaiMu.Lock()
	defer cascadeGenaiMu.Unlock()

	proj := resolveGCPProject(serveVertexProject)
	if proj == "" {
		proj = "genai-blackbelt-fishfooding"
	}
	if cascadeGenaiClient != nil && cascadeGenaiProj == proj {
		return cascadeGenaiClient, proj, nil
	}
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  proj,
		Location: "global",
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, proj, err
	}
	cascadeGenaiClient = c
	cascadeGenaiProj = proj
	return c, proj, nil
}

func extractAnswerString(qa client.QuestionAnswer) string {
	if dv := strings.TrimSpace(qa.DisplayValue()); dv != "" {
		return normalizeDecisionToken(dv)
	}
	if strings.TrimSpace(qa.Choice) != "" {
		return normalizeDecisionToken(qa.Choice)
	}
	if strings.TrimSpace(qa.Level) != "" {
		return normalizeDecisionToken(qa.Level)
	}
	if strings.TrimSpace(qa.Label) != "" {
		return normalizeDecisionToken(qa.Label)
	}
	if qa.Type == "noul" || qa.Type == "boolean" || qa.Type == "bool" {
		if qa.Noul >= 0.5 {
			return "yes"
		}
		return "no"
	}
	return ""
}

func normalizeDecisionToken(val string) string {
	s := strings.TrimSpace(val)
	low := strings.ToLower(s)
	if low == "true" {
		return "yes"
	}
	if low == "false" {
		return "no"
	}
	return s
}

func decisionValuesMatch(predicted, expected string) bool {
	p := strings.ToLower(normalizeDecisionToken(predicted))
	e := strings.ToLower(normalizeDecisionToken(expected))
	if e == "" || e == "—" || e == "-" {
		return true
	}
	return p != "" && p == e
}

func computeSlotEntropyNats(qID string, qa client.QuestionAnswer, resp *client.StructuredDecisionResponse) float64 {
	if qa.Entropy > 0 {
		return qa.Entropy
	}
	if resp != nil {
		if qd, ok := resp.Diagnostics.Questions[qID]; ok && qd.Entropy > 0 {
			return qd.Entropy
		}
	}
	if len(qa.Probabilities) > 0 {
		var h float64
		for _, p := range qa.Probabilities {
			if p > 1e-12 {
				h -= p * math.Log(p)
			}
		}
		return h
	}
	return 0
}

// ExecuteStage2GeminiCascade evaluates whether any slot in resp.Answers requires Stage 2 escalation
// under cascadeMode ("entropy" or "on_miss") and forwards escalated slots to Vertex AI Gemini 3.x
// (defaulting to gemini-3.8-flash) with Stage 1 DiffusionGemma restricted-softmax priors.
func ExecuteStage2GeminiCascade(
	ctx context.Context,
	cascadeMode string,
	cascadeThreshold float64,
	requestedModel string,
	expectedAnswers map[string]string,
	schemaContent string,
	stateContent string,
	resp *client.StructuredDecisionResponse,
) *CascadeExecutionSummary {
	mode := strings.ToLower(strings.TrimSpace(cascadeMode))
	if mode == "" || mode == "off" || mode == "none" || mode == "false" || resp == nil {
		return nil
	}
	if cascadeThreshold <= 0 {
		cascadeThreshold = 0.35
	}
	model := SanitizeCascadeModel(requestedModel)

	// Parse schemaContent to discover question definitions (type, options, levels, instructions)
	var schemaMap map[string]interface{}
	_ = json.Unmarshal([]byte(schemaContent), &schemaMap)
	globalInstr, _ := schemaMap["instructions"].(string)

	qDefs := make(map[string]map[string]interface{})
	if qs, ok := schemaMap["questions"].([]interface{}); ok {
		for _, qItem := range qs {
			if qm, ok := qItem.(map[string]interface{}); ok {
				if id, ok := qm["id"].(string); ok && id != "" {
					qDefs[id] = qm
				}
			}
		}
	}

	summary := &CascadeExecutionSummary{
		Mode:       mode,
		Model:      model,
		Threshold:  cascadeThreshold,
		TotalSlots: len(resp.Answers),
		Slots:      make(map[string]CascadeSlotTelemetry, len(resp.Answers)),
	}

	var escalatedIDs []string
	for qID, qa := range resp.Answers {
		stage1Val := extractAnswerString(qa)
		ent := computeSlotEntropyNats(qID, qa, resp)
		conf := qa.Confidence
		if conf <= 0 && (qa.Type == "noul" || qa.Type == "boolean") {
			if qa.Noul >= 0.5 {
				conf = qa.Noul
			} else {
				conf = 1.0 - qa.Noul
			}
		}

		shouldEscalate := false
		reason := ""

		switch mode {
		case "on_miss", "miss":
			expVal := ""
			if expectedAnswers != nil {
				expVal = strings.TrimSpace(expectedAnswers[qID])
			}
			if expVal != "" && expVal != "—" {
				if !decisionValuesMatch(stage1Val, expVal) {
					shouldEscalate = true
					reason = fmt.Sprintf("stage1_miss (dgemma=%q != expected=%q)", stage1Val, expVal)
				}
			} else if ent >= cascadeThreshold {
				shouldEscalate = true
				reason = fmt.Sprintf("entropy (H=%.4f >= %.2f)", ent, cascadeThreshold)
			}
		case "entropy", "auto", "true", "prior_guided":
			if ent >= cascadeThreshold {
				shouldEscalate = true
				reason = fmt.Sprintf("entropy (H=%.4f >= %.2f)", ent, cascadeThreshold)
			}
		}

		slotTel := CascadeSlotTelemetry{
			QuestionID:       qID,
			Escalated:        shouldEscalate,
			Reason:           reason,
			Stage1Value:      stage1Val,
			Stage1Confidence: conf,
			Stage1Entropy:    ent,
		}
		summary.Slots[qID] = slotTel
		if shouldEscalate {
			escalatedIDs = append(escalatedIDs, qID)
		}
	}

	sort.Strings(escalatedIDs)
	if len(escalatedIDs) == 0 {
		return summary
	}

	summary.Triggered = true
	summary.EscalatedCount = len(escalatedIDs)

	genaiClient, _, err := getSharedGenaiClient(ctx)
	if err != nil {
		summary.Error = fmt.Sprintf("Vertex AI client init failed: %v", err)
		return summary
	}

	// Build a structured ResponseSchema and Prior-Guided Prompt for all escalated question slots
	slotProps := make(map[string]*genai.Schema, len(escalatedIDs))
	var priorLines []string

	for _, qID := range escalatedIDs {
		tel := summary.Slots[qID]
		qa := resp.Answers[qID]
		qDef := qDefs[qID]
		qType, _ := qDef["type"].(string)
		if qType == "" {
			qType = qa.Type
		}
		qInstr, _ := qDef["instructions"].(string)
		if qInstr == "" {
			qInstr, _ = qDef["question"].(string)
		}

		// Format restricted-softmax prior distribution from Stage 1 dgemma
		var distParts []string
		for k, p := range qa.Probabilities {
			distParts = append(distParts, fmt.Sprintf("%s: %.1f%%", k, p*100.0))
		}
		sort.Strings(distParts)
		priorLines = append(priorLines, fmt.Sprintf(
			"- Slot %q (%s): Stage-1 dgemma candidate=%q (conf=%.1f%%, H=%.4f nats, reason=%s). Allowed distribution: {%s}. Question: %s",
			qID, qType, tel.Stage1Value, tel.Stage1Confidence*100.0, tel.Stage1Entropy, tel.Reason, strings.Join(distParts, ", "), qInstr,
		))

		var valSchema *genai.Schema
		switch strings.ToLower(qType) {
		case "boolean", "bool", "noul":
			valSchema = &genai.Schema{
				Type:        genai.TypeString,
				Enum:        []string{"yes", "no"},
				Description: fmt.Sprintf("Resolved boolean decision ('yes' or 'no') for slot %s: %s", qID, qInstr),
			}
		case "choice":
			var enumVals []string
			if opts, ok := qDef["options"].([]interface{}); ok {
				for _, o := range opts {
					switch ov := o.(type) {
					case string:
						enumVals = append(enumVals, ov)
					case map[string]interface{}:
						if n, ok := ov["name"].(string); ok && n != "" {
							enumVals = append(enumVals, n)
						}
					}
				}
			}
			if len(enumVals) == 0 && len(qa.Probabilities) > 0 {
				for k := range qa.Probabilities {
					enumVals = append(enumVals, k)
				}
				sort.Strings(enumVals)
			}
			valSchema = &genai.Schema{
				Type:        genai.TypeString,
				Enum:        enumVals,
				Description: fmt.Sprintf("Resolved option name for slot %s: %s", qID, qInstr),
			}
		default:
			var levelVals []string
			if lvls, ok := qDef["levels"].([]interface{}); ok {
				for _, l := range lvls {
					levelVals = append(levelVals, fmt.Sprintf("%v", l))
				}
			}
			valSchema = &genai.Schema{
				Type:        genai.TypeString,
				Enum:        levelVals,
				Description: fmt.Sprintf("Resolved score/level for slot %s: %s", qID, qInstr),
			}
		}

		slotProps[qID] = &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"decision":    valSchema,
				"confidence":  {Type: genai.TypeNumber, Description: "Calibrated confidence in [0.0, 1.0]."},
				"explanation": {Type: genai.TypeString, Description: "Concise 1-sentence justification."},
			},
			Required: []string{"decision", "confidence", "explanation"},
		}
	}

	respSchema := &genai.Schema{
		Type:       genai.TypeObject,
		Properties: slotProps,
		Required:   escalatedIDs,
	}

	prompt := fmt.Sprintf(
		"You are Stage-2 (%s) in a 2-Stage DiffusionGemma -> Gemini Decision Cascade.\n"+
			"Global Policy Instructions:\n%s\n\n"+
			"Input State / Context:\n%s\n\n"+
			"[TIER-1 DISCRETE DIFFUSION PRIOR TELEMETRY]\n%s\n\n"+
			"Carefully verify the input state against the policy instructions and resolve each escalated slot.",
		model, globalInstr, stateContent, strings.Join(priorLines, "\n"),
	)

	temp := float32(0.0)
	cfg := &genai.GenerateContentConfig{
		Temperature:      &temp,
		ResponseMIMEType: "application/json",
		ResponseSchema:   respSchema,
	}

	t0 := time.Now()
	candidateModels := []string{model}
	// Transparent Gemini 3.x publisher fallback if the alias is not directly bound on the project's Vertex endpoint
	for _, fallback := range []string{"gemini-3.8-flash", "gemini-3.7-flash", "gemini-3.5-flash-lite", "gemini-3-flash-preview"} {
		if fallback != model {
			candidateModels = append(candidateModels, fallback)
		}
	}

	var genResp *genai.GenerateContentResponse
	var genErr error
	for _, cand := range candidateModels {
		callCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		genResp, genErr = genaiClient.Models.GenerateContent(callCtx, cand, genai.Text(prompt), cfg)
		cancel()
		if genErr == nil && genResp != nil {
			break
		}
	}
	summary.LatencyMs = time.Since(t0).Milliseconds()
	if genErr != nil {
		summary.Error = fmt.Sprintf("Stage-2 Gemini (%s) call failed: %v", model, genErr)
		return summary
	}

	rawJSON := strings.TrimSpace(genResp.Text())
	var parsed map[string]struct {
		Decision    string  `json:"decision"`
		Confidence  float64 `json:"confidence"`
		Explanation string  `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		summary.Error = fmt.Sprintf("Stage-2 Gemini invalid JSON: %v", err)
		return summary
	}

	for _, qID := range escalatedIDs {
		resSlot, ok := parsed[qID]
		if !ok || strings.TrimSpace(resSlot.Decision) == "" {
			continue
		}
		decisionVal := normalizeDecisionToken(resSlot.Decision)
		conf := resSlot.Confidence
		if conf <= 0 || conf > 1.0 {
			conf = 0.95
		}

		// Update the CascadeSlotTelemetry record
		tel := summary.Slots[qID]
		tel.Stage2Value = decisionVal
		tel.Stage2Confidence = conf
		tel.Explanation = resSlot.Explanation
		summary.Slots[qID] = tel

		// Update resp.Answers[qID] so the caller receives the Stage-2 resolved decision
		qa := resp.Answers[qID]
		qa.Confidence = conf
		switch strings.ToLower(qa.Type) {
		case "boolean", "bool", "noul":
			isYes := decisionVal == "yes" || decisionVal == "true"
			qa.Label = decisionVal
			qa.Choice = decisionVal
			if isYes {
				qa.Noul = conf
			} else {
				qa.Noul = 1.0 - conf
			}
		case "choice":
			qa.Choice = decisionVal
			qa.Label = decisionVal
		default:
			qa.Level = decisionVal
			qa.Label = decisionVal
			if f, err := strconv.ParseFloat(decisionVal, 64); err == nil {
				qa.Score = f
			}
		}
		resp.Answers[qID] = qa
	}

	return summary
}
