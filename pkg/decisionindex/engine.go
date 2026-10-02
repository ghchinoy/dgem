package decisionindex

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ghchinoy/dgem/pkg/client"
	"github.com/ghchinoy/dgem/pkg/permutation"
)

const (
	// MaxOptionsPerSlot is the hardware/tokenizer ceiling of single-letter [A-Z] slot mapping in structured_server.py.
	MaxOptionsPerSlot = 26
	// BracketSize is the chunk size used when partitioning wide choices (27..255 options) into Round-1 brackets.
	BracketSize = 20
	// MaxSlotsPerPass is the safe per-pass slot count to prevent 256-token diffusion canvas overflow ("the canvas holds").
	MaxSlotsPerPass = 8
)

// SystemOneQuestion matches apolinario/decision-index's question specification:
// {"type": "choice", "instructions": "...", "criteria": {"opt_key": "description", ...}}
type SystemOneQuestion struct {
	Type         string            `json:"type"`
	Instructions any               `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

// SystemOneRequest matches the POST /v1/systemone payload sent by decision_index.engines.http:HttpSystemOne.
type SystemOneRequest struct {
	Model     string                       `json:"model,omitempty"`
	State     any                          `json:"state"`
	Questions map[string]SystemOneQuestion `json:"questions"`
}

// SystemOneAnswer matches the per-question answer validated by decision_index.engines.base:validate().
type SystemOneAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Noul          *float64           `json:"noul,omitempty"`
	Entropy       float64            `json:"entropy,omitempty"`
	NormalizedH   float64            `json:"normalized_entropy,omitempty"`
	PassesUsed    int                `json:"passes_used,omitempty"`
}

// SystemOneResponse matches the response envelope validated by decision_index.
type SystemOneResponse struct {
	Answers         map[string]SystemOneAnswer `json:"answers"`
	EvaluationTrace *EvaluationTrace           `json:"evaluation_trace,omitempty"`
}

// EvaluationTrace records dgem's multi-slot batching and wide-option bracket telemetry.
type EvaluationTrace struct {
	TotalQuestions    int     `json:"total_questions"`
	MaxOptionsSeen    int     `json:"max_options_seen"`
	ForwardPasses     int     `json:"forward_passes"`
	WideBracketedQs   int     `json:"wide_bracketed_questions"`
	MultiSlotBatches  int     `json:"multi_slot_batches"`
	WallTimeMs        float64 `json:"wall_time_ms"`
	TemperatureScale  float64 `json:"temperature_scale,omitempty"`
}

// EngineOptions configures the Decision Index execution adapter.
type EngineOptions struct {
	MaxSlotsPerPass   int
	MaxOptionsPerSlot int
	NaiveLimits       bool    // If true, mimics naive 26-option / 10-slot capacity rejections (HTTP 422 Unsupported)
	TemperatureScale  float64 // Post-hoc slot temperature scaling T* (1.0 = unscaled, default)
	MaxConcurrency    int
	DualMirror        bool    // EXP-13C: Evaluate forward + reversed option orderings on the same diffusion canvas
	NullPriorDebias   bool    // EXP-13B: Divide out content-free positional 'A'-bias prior
	PriorAlpha        float64 // Damping exponent alpha in [0, 1] for null-prior de-biasing
	BracketSize       int     // Max options per Round-1 bracket for wide (>26) choices; <=1 means BracketSize
}

// DefaultEngineOptions returns production settings with Wide-Option Tournament + Multi-Slot Batching enabled (T*=1.0).
func DefaultEngineOptions() EngineOptions {
	return EngineOptions{
		MaxSlotsPerPass:   MaxSlotsPerPass,
		MaxOptionsPerSlot: MaxOptionsPerSlot,
		NaiveLimits:       false,
		TemperatureScale:  1.0,
		MaxConcurrency:    4,
		PriorAlpha:        0.50,
		BracketSize:       BracketSize,
	}
}

func isNoulType(t string) bool {
	lower := strings.ToLower(strings.TrimSpace(t))
	return lower == "noul" || lower == "bool" || lower == "boolean"
}

// FormatState converts an arbitrary Decision Index state (string or JSON object) into a valid JSON string for structured_server.py.
func FormatState(state any) string {
	if state == nil {
		return `{"state":"Evaluate the decision questions based on the provided option criteria."}`
	}
	switch v := state.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
			if json.Valid([]byte(trimmed)) {
				return trimmed
			}
		}
		b, _ := json.Marshal(map[string]string{"state": v})
		return string(b)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			b2, _ := json.Marshal(map[string]string{"state": fmt.Sprintf("%v", v)})
			return string(b2)
		}
		return string(b)
	}
}

// FormatInstructions converts instructions (string or structured object) into a concise string.
func FormatInstructions(instr any) string {
	if instr == nil {
		return "Select the single best option matching the criteria."
	}
	switch v := instr.(type) {
	case string:
		return v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// ExecuteSystemOne evaluates a Decision Index request against a dgem Client, automatically applying:
// 1. Wide-Option Tournament Routing when any question has >26 options (up to 255 options)
// 2. Multi-Slot Canvas Batching when a request has >MaxSlotsPerPass simultaneous questions
// 3. Post-hoc Slot Temperature Scaling T* while strictly preserving argmax and sum(p)=1.0.
func ExecuteSystemOne(ctx context.Context, cli *client.Client, req SystemOneRequest, opts EngineOptions) (*SystemOneResponse, error) {
	start := time.Now()
	if opts.MaxSlotsPerPass <= 0 {
		opts.MaxSlotsPerPass = MaxSlotsPerPass
	}
	if opts.MaxOptionsPerSlot <= 0 {
		opts.MaxOptionsPerSlot = MaxOptionsPerSlot
	}
	if opts.TemperatureScale <= 0 {
		opts.TemperatureScale = 1.0
	}

	qKeys := make([]string, 0, len(req.Questions))
	maxOpts := 0
	for k, q := range req.Questions {
		qKeys = append(qKeys, k)
		if len(q.Criteria) > maxOpts {
			maxOpts = len(q.Criteria)
		}
	}
	sort.Strings(qKeys)

	// If NaiveLimits mode is enabled, enforce raw 26-option and 10-slot canvas ceilings to demonstrate capacity failure rate.
	if opts.NaiveLimits {
		if maxOpts > 26 {
			return nil, fmt.Errorf("HTTP 422 Unsupported: options per choice (%d) exceeds 26-option [A-Z] limit", maxOpts)
		}
		if len(qKeys) > 10 {
			return nil, fmt.Errorf("HTTP 422 Unsupported: the canvas holds at most 10 questions per pass (got %d)", len(qKeys))
		}
	}

	stateText := FormatState(req.State)
	answers := make(map[string]SystemOneAnswer, len(qKeys))
	var mu sync.Mutex
	totalPasses := 0
	wideQs := 0
	multiBatches := 0

	// Partition questions into standard (K <= 26 or noul) vs wide (K > 26)
	var standardKeys []string
	var wideKeys []string
	for _, k := range qKeys {
		q := req.Questions[k]
		if !isNoulType(q.Type) && len(q.Criteria) > opts.MaxOptionsPerSlot {
			wideKeys = append(wideKeys, k)
		} else {
			standardKeys = append(standardKeys, k)
		}
	}

	// 1. Process standard questions (K <= 26) in Multi-Slot Canvas Batches of size opts.MaxSlotsPerPass
	for i := 0; i < len(standardKeys); i += opts.MaxSlotsPerPass {
		end := i + opts.MaxSlotsPerPass
		if end > len(standardKeys) {
			end = len(standardKeys)
		}
		batchKeys := standardKeys[i:end]
		multiBatches++

		batchAns, passes, err := evaluateStandardBatch(ctx, cli, stateText, batchKeys, req.Questions, opts)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		totalPasses += passes
		for k, a := range batchAns {
			answers[k] = a
		}
		mu.Unlock()
	}

	// 2. Process wide questions (27 <= K <= 255) via 2-Stage Bracket Tournament Routing
	for _, k := range wideKeys {
		wideQs++
		q := req.Questions[k]
		ans, passes, err := evaluateWideQuestionTournament(ctx, cli, stateText, k, q, opts.TemperatureScale, opts.BracketSize)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		totalPasses += passes
		answers[k] = ans
		mu.Unlock()
	}

	return &SystemOneResponse{
		Answers: answers,
		EvaluationTrace: &EvaluationTrace{
			TotalQuestions:   len(qKeys),
			MaxOptionsSeen:   maxOpts,
			ForwardPasses:    totalPasses,
			WideBracketedQs:  wideQs,
			MultiSlotBatches: multiBatches,
			WallTimeMs:       float64(time.Since(start).Milliseconds()),
			TemperatureScale: opts.TemperatureScale,
		},
	}, nil
}

// evaluateStandardBatch runs a single forward pass for up to MaxSlotsPerPass questions where each question has <= 26 options.
func evaluateStandardBatch(
	ctx context.Context,
	cli *client.Client,
	stateText string,
	batchKeys []string,
	allQuestions map[string]SystemOneQuestion,
	opts EngineOptions,
) (map[string]SystemOneAnswer, int, error) {
	questionsPayload := make([]map[string]any, 0, len(batchKeys))
	for _, qKey := range batchKeys {
		qSpec := allQuestions[qKey]
		if isNoulType(qSpec.Type) {
			questionsPayload = append(questionsPayload, map[string]any{
				"id":           qKey,
				"type":         "boolean",
				"instructions": FormatInstructions(qSpec.Instructions),
			})
		} else {
			optKeys := sortedOptionKeys(qSpec.Criteria)
			optObjs := make([]map[string]string, 0, len(optKeys))
			for _, ok := range optKeys {
				optObjs = append(optObjs, map[string]string{
					"name":        ok,
					"description": strings.TrimSpace(qSpec.Criteria[ok]),
				})
			}
			questionsPayload = append(questionsPayload, map[string]any{
				"id":           qKey,
				"type":         "choice",
				"instructions": FormatInstructions(qSpec.Instructions),
				"options":      optObjs,
			})
		}
	}

	schemaEnvelope := map[string]any{
		"instructions": "Evaluate each decision question strictly against the provided state and option descriptions.",
		"samples":      1,
		"think":        0,
		"questions":    questionsPayload,
	}
	schemaBytes, err := json.Marshal(schemaEnvelope)
	if err != nil {
		return nil, 1, err
	}
	schemaJSON := string(schemaBytes)

	var slotOpts map[string][]permutation.OptionItem
	if opts.DualMirror && len(batchKeys) <= 4 {
		schemaJSON, slotOpts, _ = permutation.InjectDualMirrorSchema(schemaJSON)
	} else if opts.NullPriorDebias {
		slotOpts = permutation.ExtractSchemaSlotOptions(schemaJSON)
	}

	if strings.TrimSpace(stateText) == "" {
		stateText = "Evaluate the decision questions based on the provided option criteria."
	}

	resp, _, err := cli.Decide(ctx, schemaJSON, stateText)
	if err != nil {
		return nil, 1, err
	}

	if (opts.DualMirror && len(batchKeys) <= 4) || opts.NullPriorDebias {
		permutation.PostProcessDecisionResponse(resp, slotOpts, opts.DualMirror && len(batchKeys) <= 4, opts.NullPriorDebias, opts.PriorAlpha)
	}

	tempScale := opts.TemperatureScale

	out := make(map[string]SystemOneAnswer, len(batchKeys))
	for _, qKey := range batchKeys {
		qSpec := allQuestions[qKey]
		rawAns, exists := resp.Answers[qKey]

		if isNoulType(qSpec.Type) {
			pYes := 0.5
			if exists {
				if rawAns.Noul > 0 {
					pYes = rawAns.Noul
				} else if rawAns.Confidence > 0 && (strings.EqualFold(rawAns.Label, "yes") || strings.EqualFold(rawAns.Label, "true")) {
					pYes = rawAns.Confidence
				} else if rawAns.Confidence > 0 && (strings.EqualFold(rawAns.Label, "no") || strings.EqualFold(rawAns.Label, "false")) {
					pYes = 1.0 - rawAns.Confidence
				} else if rawAns.Probabilities != nil {
					if py, ok := rawAns.Probabilities["yes"]; ok {
						pYes = py
					} else if pt, ok := rawAns.Probabilities["true"]; ok {
						pYes = pt
					} else if pn, ok := rawAns.Probabilities["no"]; ok {
						pYes = 1.0 - pn
					}
				}
			}
			if tempScale != 1.0 && tempScale > 0 {
				logitYes := math.Log(math.Max(1e-12, pYes)) / tempScale
				logitNo := math.Log(math.Max(1e-12, 1.0-pYes)) / tempScale
				maxLogit := math.Max(logitYes, logitNo)
				ey := math.Exp(logitYes - maxLogit)
				en := math.Exp(logitNo - maxLogit)
				pYes = ey / (ey + en)
			}
			pYes = math.Max(0.0, math.Min(1.0, pYes))
			normH := 0.0
			if rawAns.Entropy > 0 {
				normH = rawAns.Entropy / math.Ln2
			}
			out[qKey] = SystemOneAnswer{
				Type:        "noul",
				Noul:        &pYes,
				PassesUsed:  1,
				Entropy:     rawAns.Entropy,
				NormalizedH: normH,
			}
			continue
		}

		optKeys := sortedOptionKeys(qSpec.Criteria)
		probs := make(map[string]float64, len(optKeys))

		if exists && len(rawAns.Probabilities) > 0 {
			for _, ok := range optKeys {
				probs[ok] = rawAns.Probabilities[ok]
			}
		} else if exists && (rawAns.Choice != "" || rawAns.Label != "") {
			winner := rawAns.Choice
			if winner == "" {
				winner = rawAns.Label
			}
			for _, ok := range optKeys {
				if ok == winner {
					probs[ok] = 0.90
				} else {
					probs[ok] = 0.10 / float64(maxInt(1, len(optKeys)-1))
				}
			}
		} else {
			u := 1.0 / float64(len(optKeys))
			for _, ok := range optKeys {
				probs[ok] = u
			}
		}

		scaledProbs, winner, h, normH := NormalizeAndScaleProbabilities(probs, optKeys, tempScale)
		out[qKey] = SystemOneAnswer{
			Type:          "choice",
			Choice:        winner,
			Probabilities: scaledProbs,
			Entropy:       h,
			NormalizedH:   normH,
			PassesUsed:    1,
		}
	}

	return out, 1, nil
}

// evaluateWideQuestionTournament resolves a choice question with 27..255 options by:
// 1. Partitioning the K options into brackets of <= BracketSize (20) options.
// 2. Running Round-1 single-pass readouts on each bracket to extract top-2 contenders + bracket probability masses.
// 3. Running a Round-2 Finals readout over the bracket winners to calibrate the global probability distribution over all K options.
func evaluateWideQuestionTournament(
	ctx context.Context,
	cli *client.Client,
	stateText string,
	qKey string,
	qSpec SystemOneQuestion,
	tempScale float64,
	bracketSize int,
) (SystemOneAnswer, int, error) {
	optKeys := sortedOptionKeys(qSpec.Criteria)
	numOpts := len(optKeys)
	if numOpts <= MaxOptionsPerSlot {
		batchMap, passes, err := evaluateStandardBatch(ctx, cli, stateText, []string{qKey}, map[string]SystemOneQuestion{qKey: qSpec}, EngineOptions{TemperatureScale: tempScale})
		if err != nil {
			return SystemOneAnswer{}, passes, err
		}
		return batchMap[qKey], passes, nil
	}

	if bracketSize <= 1 || bracketSize > MaxOptionsPerSlot {
		bracketSize = BracketSize
	}
	brackets := balancedBrackets(optKeys, bracketSize)

	// Pack up to MaxSlotsPerPass bracket sub-questions into a single Round-1 canvas pass!
	round1Questions := make(map[string]SystemOneQuestion, len(brackets))
	round1Keys := make([]string, 0, len(brackets))
	for bIdx, bOpts := range brackets {
		bKey := fmt.Sprintf("%s_b%02d", qKey, bIdx)
		round1Keys = append(round1Keys, bKey)
		subCriteria := make(map[string]string, len(bOpts))
		for _, ok := range bOpts {
			subCriteria[ok] = qSpec.Criteria[ok]
		}
		round1Questions[bKey] = SystemOneQuestion{
			Type:         "choice",
			Instructions: FormatInstructions(qSpec.Instructions),
			Criteria:     subCriteria,
		}
	}

	round1Results := make(map[string]SystemOneAnswer, len(brackets))
	passesUsed := 0
	for i := 0; i < len(round1Keys); i += MaxSlotsPerPass {
		end := i + MaxSlotsPerPass
		if end > len(round1Keys) {
			end = len(round1Keys)
		}
		subKeys := round1Keys[i:end]
		subAns, p, err := evaluateStandardBatch(ctx, cli, stateText, subKeys, round1Questions, EngineOptions{TemperatureScale: 1.0})
		if err != nil {
			return SystemOneAnswer{}, passesUsed + p, err
		}
		passesUsed += p
		for k, v := range subAns {
			round1Results[k] = v
		}
	}

	// Collect top-2 contenders from each bracket for Round-2 Finals (capped at 24 finalists)
	type contender struct {
		key        string
		bracketKey string
		localProb  float64
	}
	var finalists []contender
	for bIdx, bKey := range round1Keys {
		bAns := round1Results[bKey]
		bOpts := brackets[bIdx]
		sortedLocal := make([]contender, 0, len(bOpts))
		for _, ok := range bOpts {
			sortedLocal = append(sortedLocal, contender{
				key:        ok,
				bracketKey: bKey,
				localProb:  bAns.Probabilities[ok],
			})
		}
		sort.Slice(sortedLocal, func(i, j int) bool {
			return sortedLocal[i].localProb > sortedLocal[j].localProb
		})
		take := 2
		if len(sortedLocal) < take {
			take = len(sortedLocal)
		}
		finalists = append(finalists, sortedLocal[:take]...)
	}

	if len(finalists) > 24 {
		sort.Slice(finalists, func(i, j int) bool {
			return finalists[i].localProb > finalists[j].localProb
		})
		finalists = finalists[:24]
	}

	finalCriteria := make(map[string]string, len(finalists))
	for _, f := range finalists {
		finalCriteria[f.key] = qSpec.Criteria[f.key]
	}
	finalQ := map[string]SystemOneQuestion{
		qKey: {
			Type:         "choice",
			Instructions: FormatInstructions(qSpec.Instructions),
			Criteria:     finalCriteria,
		},
	}

	finalBatch, p, err := evaluateStandardBatch(ctx, cli, stateText, []string{qKey}, finalQ, EngineOptions{TemperatureScale: 1.0})
	if err != nil {
		return SystemOneAnswer{}, passesUsed + p, err
	}
	passesUsed += p
	finalAns := finalBatch[qKey]

	finalistSet := make(map[string]bool, len(finalists))
	for _, f := range finalists {
		finalistSet[f.key] = true
	}
	round1 := make([]map[string]float64, len(brackets))
	for bIdx, bKey := range round1Keys {
		round1[bIdx] = round1Results[bKey].Probabilities
	}
	combined := fuseBracketProbabilities(brackets, round1, finalistSet, finalAns.Probabilities)

	scaledProbs, winner, h, normH := NormalizeAndScaleProbabilities(combined, optKeys, tempScale)
	return SystemOneAnswer{
		Type:          "choice",
		Choice:        winner,
		Probabilities: scaledProbs,
		Entropy:       h,
		NormalizedH:   normH,
		PassesUsed:    passesUsed,
	}, passesUsed, nil
}

// NormalizeAndScaleProbabilities applies temperature scaling T > 0, guarantees strict positivity and
// exact normalization (abs(sum(p)-1) < 1e-9) required by decision_index/engines/base.py:validate(),
// and returns (probabilities, argmaxWinner, shannonEntropyNats, normalizedEntropy).
func NormalizeAndScaleProbabilities(raw map[string]float64, optKeys []string, tempScale float64) (map[string]float64, string, float64, float64) {
	if tempScale <= 0 {
		tempScale = 1.0
	}
	kCount := len(optKeys)
	if kCount == 0 {
		return map[string]float64{}, "", 0, 0
	}

	scaled := make(map[string]float64, kCount)
	sum := 0.0
	for _, k := range optKeys {
		p := raw[k]
		if math.IsNaN(p) || math.IsInf(p, 0) || p <= 1e-9 {
			p = 1e-6
		}
		sp := math.Pow(p, 1.0/tempScale)
		scaled[k] = sp
		sum += sp
	}

	winner := optKeys[0]
	bestP := -1.0
	entropy := 0.0
	for _, k := range optKeys {
		p := scaled[k] / sum
		scaled[k] = p
		if p > bestP {
			bestP = p
			winner = k
		}
		if p > 0 {
			entropy -= p * math.Log(p)
		}
	}

	normH := 0.0
	if kCount > 1 {
		normH = entropy / math.Log(float64(kCount))
	}
	return scaled, winner, entropy, normH
}

func sortedOptionKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// NewSystemOneHTTPHandler returns an http.HandlerFunc serving POST /v1/systemone compatible with
// `python -m decision_index run --engine http`.
func NewSystemOneHTTPHandler(cli *client.Client, opts EngineOptions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var req SystemOneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid JSON body: %v", err), http.StatusBadRequest)
			return
		}
		resp, err := ExecuteSystemOne(r.Context(), cli, req, opts)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "HTTP 422") ||
				strings.Contains(errStr, "options per choice") ||
				strings.Contains(errStr, "the canvas holds at most") ||
				strings.Contains(errStr, "too many tokens") ||
				strings.Contains(errStr, "maximum context length") ||
				strings.Contains(errStr, "context window") {
				http.Error(w, errStr, http.StatusUnprocessableEntity)
				return
			}
			http.Error(w, errStr, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// balancedBrackets splits keys into ceil(K/size) brackets of near-equal size (differing by at most one).
// Fixed-size chunking left a 1-option tail whenever K % size == 1 (e.g. 41, 61, 101 options), which the
// server rejects ("needs at least two alternatives").
func balancedBrackets(keys []string, size int) [][]string {
	if size < 2 {
		size = 2
	}
	n := len(keys)
	if n == 0 {
		return nil
	}
	nb := (n + size - 1) / size
	base, extra := n/nb, n%nb
	out := make([][]string, 0, nb)
	i := 0
	for b := 0; b < nb; b++ {
		sz := base
		if b < extra {
			sz++
		}
		out = append(out, keys[i:i+sz])
		i += sz
	}
	return out
}

// fuseBracketProbabilities combines Round-1 bracket readouts and the Round-2 final into one distribution over all K
// options, using the model's own outputs instead of a fixed finalist/non-finalist split (previously 92% / 8%, which
// capped every wide-option confidence at ~0.92):
//
//	P(b)          = Σ_{f finalist in b} p_final(f)          (the final round is the evidence for which bracket wins)
//	non-final o∈b = P(b) · p_b(o)                           (Round-1 share inside that bracket)
//	finalist  f   = p_final(f) · (1 − Σ non-finalist mass)
//
// A uniform bracket prior (w_b = |b|/K) was rejected: Round-1 distributions sum to 1 even in brackets with no good
// option, so it hands most of the mass to brackets the final round already rejected.
// Every option gets a strictly positive probability, the result sums to 1, and the final-round winner stays the
// argmax (a non-finalist is capped just below it before renormalising, which preserves order).
func fuseBracketProbabilities(brackets [][]string, round1 []map[string]float64, finalist map[string]bool, final map[string]float64) map[string]float64 {
	const eps = 1e-6
	out := make(map[string]float64)
	finTot := 0.0
	for o := range finalist {
		finTot += math.Max(eps, final[o])
	}
	if finTot == 0 {
		finTot = 1
	}
	tail := 0.0
	nonFin := make(map[string]float64)
	for bIdx, b := range brackets {
		mass, pb := 0.0, 0.0
		for _, o := range b {
			mass += math.Max(eps, round1[bIdx][o])
			if finalist[o] {
				pb += math.Max(eps, final[o]) / finTot
			}
		}
		pb = math.Max(eps, pb)
		for _, o := range b {
			if !finalist[o] {
				v := pb * math.Max(eps, round1[bIdx][o]) / mass
				nonFin[o] = v
				tail += v
			}
		}
	}
	tail = math.Min(1-eps, tail)
	winner, best := "", -1.0
	for o := range finalist {
		v := (1 - tail) * math.Max(eps, final[o]) / finTot
		out[o] = v
		if v > best || (v == best && o < winner) {
			winner, best = o, v
		}
	}
	for o, v := range nonFin {
		if winner != "" && v >= best {
			v = best * 0.999
		}
		out[o] = math.Max(eps*eps, v)
	}
	sum := 0.0
	for _, v := range out {
		sum += v
	}
	for o := range out {
		out[o] /= sum
	}
	return out
}
