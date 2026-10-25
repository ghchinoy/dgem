// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package decisionindex

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// ConformalMethod identifies the nonconformity score family used for split conformal prediction.
type ConformalMethod string

const (
	// ConformalLAC uses Least Ambiguous Set Classifiers nonconformity s(x, y) = 1 - p(y|x).
	ConformalLAC ConformalMethod = "lac"
	// ConformalAPS uses Adaptive Prediction Sets cumulative mass up to and including y.
	ConformalAPS ConformalMethod = "aps"
	// ConformalRAPS adds rank-thresholded size regularization above KReg to APS.
	ConformalRAPS ConformalMethod = "raps"
)

const (
	// ContextActionFull retains the complete trajectory block verbatim.
	ContextActionFull = "full"
	// ContextActionStub replaces the block body with its compact summary or signature stub.
	ContextActionStub = "stub"
	// ContextActionDrop omits the block from the active context window.
	ContextActionDrop = "drop"
)

const conformalScoreTolerance = 1e-12

var defaultAbstainLabels = []string{
	"abstain",
	"none_of_the_above",
	"insufficient_evidence",
	"unknown",
}

var (
	// ErrEmptyCalibration is returned when an empty calibration slice is supplied.
	ErrEmptyCalibration = errors.New("decisionindex: calibration set must be non-empty")
	// ErrCalibrationLenMismatch is returned when calibration probability maps and gold labels differ in length.
	ErrCalibrationLenMismatch = errors.New("decisionindex: calibration probabilities and labels length mismatch")
	// ErrInvalidAlpha is returned when alpha lies outside the open interval (0, 1).
	ErrInvalidAlpha = errors.New("decisionindex: alpha must be strictly inside (0, 1)")
	// ErrUnknownCalibrationKey is returned when a gold calibration label is missing from its probability map.
	ErrUnknownCalibrationKey = errors.New("decisionindex: gold calibration label missing from probability map")
	// ErrInvalidConformalMethod is returned when ConformalConfig.Method is not recognized.
	ErrInvalidConformalMethod = errors.New("decisionindex: unknown conformal method")
	// ErrTooManyContextBlocks is returned when more than MaxSlotsPerPass (8) blocks are passed for single-pass pruning.
	ErrTooManyContextBlocks = errors.New("decisionindex: context block count exceeds MaxSlotsPerPass (8)")
	// ErrNegativeTokenBudget is returned when ContextPruneConfig.TokenBudget is negative.
	ErrNegativeTokenBudget = errors.New("decisionindex: token budget must be non-negative")
)

// ConformalConfig configures split conformal score computation and prediction-set construction.
type ConformalConfig struct {
	// Method selects "lac" (default when empty), "aps", or "raps".
	Method ConformalMethod `json:"method,omitempty"`
	// KReg is the unpenalized top-rank prefix size for RAPS (defaults to 1 when Method == ConformalRAPS and KReg <= 0).
	KReg int `json:"k_reg,omitempty"`
	// LambdaReg is the per-rank penalty weight above KReg for RAPS (clamped to >= 0).
	LambdaReg float64 `json:"lambda_reg,omitempty"`
	// Randomized enables deterministic seeded U[0,1) boundary smoothing for APS and RAPS.
	Randomized bool `json:"randomized,omitempty"`
	// Seed makes Randomized boundary smoothing deterministic across calibration and inference.
	Seed int64 `json:"seed,omitempty"`
	// AbstainLabels lists option names treated as abstention triggers when included in the prediction set.
	// Defaults to {"abstain", "none_of_the_above", "insufficient_evidence", "unknown"} when nil.
	AbstainLabels []string `json:"abstain_labels,omitempty"`
}

// ConformalCalibrationResult records the calibrated split-conformal quantile and calibration summary.
type ConformalCalibrationResult struct {
	Method         ConformalMethod `json:"method"`
	Alpha          float64         `json:"alpha"`
	NumCalibration int             `json:"num_calibration"`
	OrderIndex     int             `json:"order_index"`
	QHat           float64         `json:"q_hat"`
	EmpiricalCover float64         `json:"empirical_coverage"`
	SortedScores   []float64       `json:"sorted_scores,omitempty"`
}

// ConformalSetResult holds the calibrated prediction set and cascade escalation decision for one slot.
type ConformalSetResult struct {
	PredictionSet   []string           `json:"prediction_set"`
	Scores          map[string]float64 `json:"scores"`
	QHat            float64            `json:"q_hat"`
	TopOption       string             `json:"top_option"`
	TopProbability  float64            `json:"top_probability"`
	SetSize         int                `json:"set_size"`
	Singleton       bool               `json:"singleton"`
	ContainsAbstain bool               `json:"contains_abstain"`
	// ShouldEscalate is true iff SetSize != 1 || ContainsAbstain.
	ShouldEscalate bool `json:"should_escalate"`
}

// QuestionResult carries post-fusion and optional pre-prior-fusion option probabilities
// so conformal escalation gates can inspect raw likelihoods before caller prior fusion.
type QuestionResult struct {
	Choice           string             `json:"choice,omitempty"`
	Probabilities    map[string]float64 `json:"probabilities,omitempty"`
	RawProbabilities map[string]float64 `json:"raw_probabilities,omitempty"`
	PriorFused       bool               `json:"prior_fused,omitempty"`
}

type optionProb struct {
	name string
	prob float64
}

func parseConformalMethod(m ConformalMethod) (ConformalMethod, error) {
	norm := strings.ToLower(strings.TrimSpace(string(m)))
	switch norm {
	case "", string(ConformalLAC):
		return ConformalLAC, nil
	case string(ConformalAPS):
		return ConformalAPS, nil
	case string(ConformalRAPS):
		return ConformalRAPS, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidConformalMethod, m)
	}
}

func normalizeOrderedProbs(probs map[string]float64) []optionProb {
	if len(probs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(probs))
	for k := range probs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	cleaned := make([]optionProb, len(keys))
	sum := 0.0
	for i, k := range keys {
		p := probs[k]
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 {
			p = 0
		}
		cleaned[i] = optionProb{name: k, prob: p}
		sum += p
	}

	if sum > 0 {
		for i := range cleaned {
			cleaned[i].prob /= sum
		}
	} else {
		u := 1.0 / float64(len(cleaned))
		for i := range cleaned {
			cleaned[i].prob = u
		}
	}

	sort.Slice(cleaned, func(i, j int) bool {
		if cleaned[i].prob != cleaned[j].prob {
			return cleaned[i].prob > cleaned[j].prob
		}
		return cleaned[i].name < cleaned[j].name
	})
	return cleaned
}

func fnv1a64(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

func splitMix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

func uniformForConformalOption(seed int64, ordered []optionProb, label string) float64 {
	h := splitMix64(uint64(seed))
	for _, opt := range ordered {
		qProb := uint64(math.Round(opt.prob * 1e9))
		h = splitMix64(h ^ fnv1a64(opt.name))
		h = splitMix64(h ^ qProb)
	}
	h = splitMix64(h ^ fnv1a64(label))
	const inv53 = 1.0 / float64(uint64(1)<<53)
	return float64(h>>11) * inv53
}

func computeOrderedScores(ordered []optionProb, method ConformalMethod, cfg ConformalConfig) map[string]float64 {
	scores := make(map[string]float64, len(ordered))
	if len(ordered) == 0 {
		return scores
	}

	kReg := cfg.KReg
	if kReg <= 0 {
		kReg = 1
	}
	lamReg := cfg.LambdaReg
	if math.IsNaN(lamReg) || math.IsInf(lamReg, 0) || lamReg < 0 {
		lamReg = 0
	}

	cumBefore := 0.0
	for idx, opt := range ordered {
		rank := idx + 1
		var s float64
		switch method {
		case ConformalLAC:
			s = 1.0 - opt.prob
		case ConformalAPS, ConformalRAPS:
			u := 1.0
			if cfg.Randomized {
				u = uniformForConformalOption(cfg.Seed, ordered, opt.name)
			}
			s = cumBefore + u*opt.prob
			if method == ConformalRAPS && rank > kReg {
				s += lamReg * float64(rank-kReg)
			}
		default:
			s = 1.0 - opt.prob
		}
		scores[opt.name] = s
		cumBefore += opt.prob
	}
	return scores
}

// ConformalScore computes the nonconformity score s(x, label) for one probability map and candidate label.
func ConformalScore(probs map[string]float64, label string, cfg ConformalConfig) (float64, error) {
	method, err := parseConformalMethod(cfg.Method)
	if err != nil {
		return 0, err
	}
	if _, ok := probs[label]; !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnknownCalibrationKey, label)
	}
	ordered := normalizeOrderedProbs(probs)
	scores := computeOrderedScores(ordered, method, cfg)
	return scores[label], nil
}

// CalibrateConformalQuantile computes the finite-sample split conformal quantile
// at order index k = ceil((n + 1) * (1 - alpha)) over held-out calibration examples.
func CalibrateConformalQuantile(
	calibProbs []map[string]float64,
	calibLabels []string,
	alpha float64,
	cfg ConformalConfig,
) (ConformalCalibrationResult, error) {
	if len(calibProbs) == 0 {
		return ConformalCalibrationResult{}, ErrEmptyCalibration
	}
	if len(calibProbs) != len(calibLabels) {
		return ConformalCalibrationResult{}, fmt.Errorf(
			"%w: got %d probability maps and %d labels",
			ErrCalibrationLenMismatch, len(calibProbs), len(calibLabels),
		)
	}
	if math.IsNaN(alpha) || math.IsInf(alpha, 0) || alpha <= 0 || alpha >= 1 {
		return ConformalCalibrationResult{}, fmt.Errorf("%w: got %v", ErrInvalidAlpha, alpha)
	}
	method, err := parseConformalMethod(cfg.Method)
	if err != nil {
		return ConformalCalibrationResult{}, err
	}

	n := len(calibProbs)
	rawScores := make([]float64, n)
	for i := 0; i < n; i++ {
		lbl := calibLabels[i]
		if _, ok := calibProbs[i][lbl]; !ok {
			return ConformalCalibrationResult{}, fmt.Errorf("%w at index %d: %q", ErrUnknownCalibrationKey, i, lbl)
		}
		ordered := normalizeOrderedProbs(calibProbs[i])
		sMap := computeOrderedScores(ordered, method, cfg)
		rawScores[i] = sMap[lbl]
	}

	sorted := append([]float64(nil), rawScores...)
	sort.Float64s(sorted)

	k := int(math.Ceil(float64(n+1) * (1.0 - alpha)))
	if k < 1 {
		k = 1
	}

	var qHat float64
	orderIdx := k
	if k > n {
		orderIdx = n + 1
		qHat = math.Inf(1)
	} else {
		qHat = sorted[k-1]
	}

	covered := 0
	for _, s := range rawScores {
		if s <= qHat+conformalScoreTolerance {
			covered++
		}
	}

	return ConformalCalibrationResult{
		Method:         method,
		Alpha:          alpha,
		NumCalibration: n,
		OrderIndex:     orderIdx,
		QHat:           qHat,
		EmpiricalCover: float64(covered) / float64(n),
		SortedScores:   sorted,
	}, nil
}

// ConformalPredictionSet constructs the calibrated prediction set C_alpha(x) = {y : s(x, y) <= qHat}
// ordered by descending probability, and sets ShouldEscalate when the set is not a non-abstain singleton.
func ConformalPredictionSet(probs map[string]float64, qHat float64, cfg ConformalConfig) ConformalSetResult {
	method, err := parseConformalMethod(cfg.Method)
	if err != nil {
		method = ConformalLAC
	}

	ordered := normalizeOrderedProbs(probs)
	scores := computeOrderedScores(ordered, method, cfg)

	abstainList := cfg.AbstainLabels
	if abstainList == nil {
		abstainList = defaultAbstainLabels
	}
	abstainSet := make(map[string]bool, len(abstainList))
	for _, a := range abstainList {
		trimmed := strings.ToLower(strings.TrimSpace(a))
		if trimmed != "" {
			abstainSet[trimmed] = true
		}
	}

	predSet := make([]string, 0, len(ordered))
	containsAbstain := false
	for _, opt := range ordered {
		if scores[opt.name] <= qHat+conformalScoreTolerance {
			predSet = append(predSet, opt.name)
			if abstainSet[strings.ToLower(strings.TrimSpace(opt.name))] {
				containsAbstain = true
			}
		}
	}

	topOpt := ""
	topProb := 0.0
	if len(ordered) > 0 {
		topOpt = ordered[0].name
		topProb = ordered[0].prob
	}

	singleton := len(predSet) == 1
	shouldEscalate := !singleton || containsAbstain

	return ConformalSetResult{
		PredictionSet:   predSet,
		Scores:          scores,
		QHat:            qHat,
		TopOption:       topOpt,
		TopProbability:  topProb,
		SetSize:         len(predSet),
		Singleton:       singleton,
		ContainsAbstain: containsAbstain,
		ShouldEscalate:  shouldEscalate,
	}
}

// EvaluatePreFusionConformalGate evaluates ConformalPredictionSet on qr.RawProbabilities when populated,
// falling back to qr.Probabilities when no prior fusion occurred, so caller priors cannot mask ambiguity.
func EvaluatePreFusionConformalGate(qr QuestionResult, qHat float64, cfg ConformalConfig) ConformalSetResult {
	target := qr.Probabilities
	if len(qr.RawProbabilities) > 0 {
		target = qr.RawProbabilities
	}
	return ConformalPredictionSet(target, qHat, cfg)
}

// RecoverPreFusionProbabilities inverts log-linear prior fusion when ans.PriorFused is true,
// returning the pre-fusion option probability distribution for downstream conformal gating.
func RecoverPreFusionProbabilities(ans SystemOneAnswer, priorLogits map[string]float64, priorWeight float64) map[string]float64 {
	if !ans.PriorFused || len(priorLogits) == 0 || priorWeight <= 0 || len(ans.Probabilities) == 0 {
		out := make(map[string]float64, len(ans.Probabilities))
		for k, v := range ans.Probabilities {
			out[k] = v
		}
		return out
	}

	keys := make([]string, 0, len(ans.Probabilities))
	for k := range ans.Probabilities {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	logits := make(map[string]float64, len(keys))
	maxLogit := math.Inf(-1)
	for _, k := range keys {
		p := math.Max(1e-12, ans.Probabilities[k])
		l := math.Log(p) - priorWeight*priorLogits[k]
		logits[k] = l
		if l > maxLogit {
			maxLogit = l
		}
	}

	raw := make(map[string]float64, len(keys))
	sum := 0.0
	for _, k := range keys {
		v := math.Exp(logits[k] - maxLogit)
		raw[k] = v
		sum += v
	}
	if sum > 0 {
		for _, k := range keys {
			raw[k] /= sum
		}
	}
	return raw
}

// EvaluateSystemOnePreFusionGate evaluates ConformalPredictionSet on a SystemOneAnswer after
// recovering pre-prior-fusion probabilities when PriorFused is true.
func EvaluateSystemOnePreFusionGate(
	ans SystemOneAnswer,
	priorLogits map[string]float64,
	priorWeight float64,
	qHat float64,
	cfg ConformalConfig,
) ConformalSetResult {
	raw := RecoverPreFusionProbabilities(ans, priorLogits, priorWeight)
	return ConformalPredictionSet(raw, qHat, cfg)
}

// NormalizedShannonEntropy computes H(p) / ln(K) in [0, 1] for a non-negative slice of K probabilities.
func NormalizedShannonEntropy(probs []float64) float64 {
	k := len(probs)
	if k <= 1 {
		return 0
	}
	sum := 0.0
	for _, p := range probs {
		if !math.IsNaN(p) && !math.IsInf(p, 0) && p > 0 {
			sum += p
		}
	}
	if sum <= 0 {
		return 0
	}
	h := 0.0
	for _, p := range probs {
		if !math.IsNaN(p) && !math.IsInf(p, 0) && p > 0 {
			pn := p / sum
			h -= pn * math.Log(pn)
		}
	}
	normH := h / math.Log(float64(k))
	if normH < 0 {
		return 0
	}
	if normH > 1 {
		return 1
	}
	return normH
}

// ContextBlockSpec describes one candidate trajectory block (up to MaxSlotsPerPass = 8 blocks per pass).
type ContextBlockSpec struct {
	// ID is the caller's block identifier (mapped in source order to neutral slot IDs b0..b7).
	ID string `json:"id"`
	// Kind categorizes the block (for example "tool_output", "file_excerpt", "test_trace", "turn_summary").
	Kind string `json:"kind,omitempty"`
	// Summary is a concise description rendered into the slot instructions prompt.
	Summary string `json:"summary,omitempty"`
	// FullText is the unpruned block content.
	FullText string `json:"full_text,omitempty"`
	// StubText is the compact header, signature, or digest retained when Action == "stub".
	StubText string `json:"stub_text,omitempty"`
	// FullTokens is the token cost of keeping FullText (clamped to >= StubTokens >= 0).
	FullTokens int `json:"full_tokens"`
	// StubTokens is the token cost of keeping StubText.
	StubTokens int `json:"stub_tokens"`
	// Pinned forces Action == "full" regardless of model probabilities.
	Pinned bool `json:"pinned,omitempty"`
	// ImportanceWeight scales this block's expected utility in knapsack selection (defaults to 1.0 when <= 0).
	ImportanceWeight float64 `json:"importance_weight,omitempty"`
}

// ContextPruneConfig controls expected-utility Multiple-Choice Knapsack (MCKP) context pruning.
type ContextPruneConfig struct {
	// TokenBudget is the maximum total tokens allowed across retained full and stub blocks.
	TokenBudget int `json:"token_budget"`
	// StubUtilityRatio is the utility credit for retaining a stub when the true best state is "full"
	// (defaults to 0.65 when <= 0 or >= 1).
	StubUtilityRatio float64 `json:"stub_utility_ratio,omitempty"`
	// UncertaintyStubBonus adds normalized-entropy weight toward "stub" so high-hesitation blocks
	// compress to stubs instead of dropping irreversibly (defaults to 0.25 when == 0; set < 0 to disable).
	UncertaintyStubBonus float64 `json:"uncertainty_stub_bonus,omitempty"`
}

// ContextBlockDecision records the selected action, token cost, and posterior distribution for one block.
type ContextBlockDecision struct {
	BlockID           string             `json:"block_id"`
	SlotID            string             `json:"slot_id"`
	Action            string             `json:"action"`
	RetainedText      string             `json:"retained_text,omitempty"`
	RetainedTokens    int                `json:"retained_tokens"`
	SavedTokens       int                `json:"saved_tokens"`
	NormalizedEntropy float64            `json:"normalized_entropy"`
	Probabilities     map[string]float64 `json:"probabilities"`
}

// ContextPruneResult aggregates per-block pruning decisions and token accounting.
type ContextPruneResult struct {
	Decisions       []ContextBlockDecision `json:"decisions"`
	OriginalTokens  int                    `json:"original_tokens"`
	RetainedTokens  int                    `json:"retained_tokens"`
	SavedTokens     int                    `json:"saved_tokens"`
	TokenBudget     int                    `json:"token_budget"`
	BudgetSatisfied bool                   `json:"budget_satisfied"`
	ExpectedUtility float64                `json:"expected_utility"`
}

// BuildContextPruningQuestions constructs up to MaxSlotsPerPass (8) independent choice questions
// keyed by neutral slot IDs b0..b7 with options {"full", "stub", "drop"}.
func BuildContextPruningQuestions(blocks []ContextBlockSpec) (map[string]SystemOneQuestion, []string, error) {
	if len(blocks) > MaxSlotsPerPass {
		return nil, nil, fmt.Errorf("%w: got %d blocks", ErrTooManyContextBlocks, len(blocks))
	}
	questions := make(map[string]SystemOneQuestion, len(blocks))
	slotIDs := make([]string, 0, len(blocks))
	for i, b := range blocks {
		slotID := fmt.Sprintf("b%d", i)
		slotIDs = append(slotIDs, slotID)

		blockID := strings.TrimSpace(b.ID)
		if blockID == "" {
			blockID = slotID
		}
		kind := strings.TrimSpace(b.Kind)
		if kind == "" {
			kind = "context_block"
		}
		fullTok := b.FullTokens
		if fullTok < 0 {
			fullTok = 0
		}
		stubTok := b.StubTokens
		if stubTok < 0 {
			stubTok = 0
		}
		if stubTok > fullTok {
			stubTok = fullTok
		}

		inst := fmt.Sprintf("Select retention level for block %s (%s, full=%d tok, stub=%d tok).", blockID, kind, fullTok, stubTok)
		if summary := strings.TrimSpace(b.Summary); summary != "" {
			inst = fmt.Sprintf("Select retention level for block %s (%s, full=%d tok, stub=%d tok): %s", blockID, kind, fullTok, stubTok, summary)
		}

		questions[slotID] = SystemOneQuestion{
			Type:         "choice",
			Instructions: inst,
			Criteria: map[string]string{
				ContextActionFull: "Retain complete block text verbatim.",
				ContextActionStub: "Replace full body with compact summary or signature stub.",
				ContextActionDrop: "Omit block from active context window.",
			},
			Order: []string{ContextActionFull, ContextActionStub, ContextActionDrop},
		}
	}
	return questions, slotIDs, nil
}

func normalizeActionTriplet(raw map[string]float64) (pFull, pStub, pDrop float64) {
	clean := func(v float64) float64 {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return 0
		}
		return v
	}
	if len(raw) > 0 {
		pFull = clean(raw[ContextActionFull])
		pStub = clean(raw[ContextActionStub])
		pDrop = clean(raw[ContextActionDrop])
	}
	sum := pFull + pStub + pDrop
	if sum <= 0 {
		return 1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0
	}
	return pFull / sum, pStub / sum, pDrop / sum
}

// SelectContextPruningActions solves the exact Multiple-Choice Knapsack Problem (MCKP)
// across up to MaxSlotsPerPass (8) blocks (at most 3^8 = 6,561 joint states) under cfg.TokenBudget,
// preserving Pinned blocks at "full" and favoring "stub" over irreversible "drop" on high-hesitation slots.
func SelectContextPruningActions(
	blocks []ContextBlockSpec,
	slotProbs map[string]map[string]float64,
	cfg ContextPruneConfig,
) (ContextPruneResult, error) {
	if len(blocks) > MaxSlotsPerPass {
		return ContextPruneResult{}, fmt.Errorf("%w: got %d blocks", ErrTooManyContextBlocks, len(blocks))
	}
	if cfg.TokenBudget < 0 {
		return ContextPruneResult{}, fmt.Errorf("%w: got %d", ErrNegativeTokenBudget, cfg.TokenBudget)
	}
	if len(blocks) == 0 {
		return ContextPruneResult{
			Decisions:       []ContextBlockDecision{},
			TokenBudget:     cfg.TokenBudget,
			BudgetSatisfied: true,
		}, nil
	}

	rho := cfg.StubUtilityRatio
	if math.IsNaN(rho) || math.IsInf(rho, 0) || rho <= 0 || rho >= 1 {
		rho = 0.65
	}
	beta := cfg.UncertaintyStubBonus
	if math.IsNaN(beta) || math.IsInf(beta, 0) {
		beta = 0.25
	} else if beta == 0 {
		beta = 0.25
	} else if beta < 0 {
		beta = 0
	}

	m := len(blocks)
	prep := make([]preparedBlock, m)
	origTokens := 0
	pinnedTokens := 0

	for i, b := range blocks {
		slotID := fmt.Sprintf("b%d", i)
		blockID := b.ID
		if blockID == "" {
			blockID = slotID
		}
		fullTok := b.FullTokens
		if fullTok < 0 {
			fullTok = 0
		}
		stubTok := b.StubTokens
		if stubTok < 0 {
			stubTok = 0
		}
		if stubTok > fullTok {
			stubTok = fullTok
		}
		origTokens += fullTok
		if b.Pinned {
			pinnedTokens += fullTok
		}

		var rawMap map[string]float64
		if b.ID != "" {
			if pm, ok := slotProbs[b.ID]; ok {
				rawMap = pm
			}
		}
		if rawMap == nil {
			rawMap = slotProbs[slotID]
		}

		pFull, pStub, pDrop := normalizeActionTriplet(rawMap)
		hNorm := NormalizedShannonEntropy([]float64{pFull, pStub, pDrop})

		w := b.ImportanceWeight
		if math.IsNaN(w) || math.IsInf(w, 0) || w <= 0 {
			w = 1.0
		}

		uFull := w * (1.0*pFull + rho*pStub)
		uStub := w * (rho*pFull + 1.0*pStub + beta*hNorm*(1.0-pDrop))
		uDrop := w * (0.15 * pDrop)

		prep[i] = preparedBlock{
			slotID:  slotID,
			blockID: blockID,
			costs:   [3]int{fullTok, stubTok, 0},
			utils:   [3]float64{uFull, uStub, uDrop},
			probs:   [3]float64{pFull, pStub, pDrop},
			hNorm:   hNorm,
			pinned:  b.Pinned,
		}
	}

	var bestActions [MaxSlotsPerPass]int
	var curActions [MaxSlotsPerPass]int
	bestFound := false
	bestUtil := math.Inf(-1)
	bestTokens := 0

	// If pinned blocks alone exceed TokenBudget, keep pinned blocks at "full" and drop unpinned non-zero-cost blocks.
	if pinnedTokens > cfg.TokenBudget {
		totalUtil := 0.0
		for i := 0; i < m; i++ {
			if prep[i].pinned {
				bestActions[i] = 0
			} else if prep[i].costs[1] == 0 && prep[i].utils[1] >= prep[i].utils[2] {
				bestActions[i] = 1
			} else {
				bestActions[i] = 2
			}
			totalUtil += prep[i].utils[bestActions[i]]
		}
		return buildContextPruneResult(blocks, prep, bestActions[:m], origTokens, pinnedTokens, cfg.TokenBudget, false, totalUtil), nil
	}

	// Suffix upper bound on remaining achievable utility for fast exact branch-and-bound over <= 6,561 tuples.
	var suffixMaxUtil [MaxSlotsPerPass + 1]float64
	for i := m - 1; i >= 0; i-- {
		maxU := prep[i].utils[0]
		if !prep[i].pinned {
			if prep[i].utils[1] > maxU {
				maxU = prep[i].utils[1]
			}
			if prep[i].utils[2] > maxU {
				maxU = prep[i].utils[2]
			}
		}
		suffixMaxUtil[i] = suffixMaxUtil[i+1] + maxU
	}

	betterAssignment := func(candUtil float64, candTok int) bool {
		if !bestFound {
			return true
		}
		if candUtil > bestUtil+1e-12 {
			return true
		}
		if math.Abs(candUtil-bestUtil) > 1e-12 {
			return false
		}
		if candTok != bestTokens {
			return candTok < bestTokens
		}
		// Prefer retaining more context (0="full" < 1="stub" < 2="drop") in source block order.
		for i := 0; i < m; i++ {
			if curActions[i] != bestActions[i] {
				return curActions[i] < bestActions[i]
			}
		}
		return false
	}

	var search func(idx int, usedTok int, accUtil float64)
	search = func(idx int, usedTok int, accUtil float64) {
		if usedTok > cfg.TokenBudget {
			return
		}
		if bestFound && accUtil+suffixMaxUtil[idx] < bestUtil-1e-12 {
			return
		}
		if idx == m {
			if betterAssignment(accUtil, usedTok) {
				bestFound = true
				bestUtil = accUtil
				bestTokens = usedTok
				for i := 0; i < m; i++ {
					bestActions[i] = curActions[i]
				}
			}
			return
		}
		if prep[idx].pinned {
			curActions[idx] = 0
			search(idx+1, usedTok+prep[idx].costs[0], accUtil+prep[idx].utils[0])
			return
		}
		for act := 0; act < 3; act++ {
			curActions[idx] = act
			search(idx+1, usedTok+prep[idx].costs[act], accUtil+prep[idx].utils[act])
		}
	}

	search(0, 0, 0)
	return buildContextPruneResult(blocks, prep, bestActions[:m], origTokens, bestTokens, cfg.TokenBudget, true, bestUtil), nil
}

// PruneAgentContextBlocks is an alias for SelectContextPruningActions for callers using the verb form.
func PruneAgentContextBlocks(
	blocks []ContextBlockSpec,
	slotProbs map[string]map[string]float64,
	cfg ContextPruneConfig,
) (ContextPruneResult, error) {
	return SelectContextPruningActions(blocks, slotProbs, cfg)
}

type preparedBlock struct {
	slotID  string
	blockID string
	costs   [3]int
	utils   [3]float64
	probs   [3]float64
	hNorm   float64
	pinned  bool
}

func buildContextPruneResult(
	blocks []ContextBlockSpec,
	meta []preparedBlock,
	actions []int,
	origTokens int,
	retainedTokens int,
	budget int,
	satisfied bool,
	expectedUtil float64,
) ContextPruneResult {
	actionNames := [3]string{ContextActionFull, ContextActionStub, ContextActionDrop}
	decisions := make([]ContextBlockDecision, len(blocks))
	for i, b := range blocks {
		actIdx := actions[i]
		actName := actionNames[actIdx]
		retTok := meta[i].costs[actIdx]
		var retText string
		switch actIdx {
		case 0:
			retText = b.FullText
		case 1:
			retText = b.StubText
		default:
			retText = ""
		}
		decisions[i] = ContextBlockDecision{
			BlockID:           meta[i].blockID,
			SlotID:            meta[i].slotID,
			Action:            actName,
			RetainedText:      retText,
			RetainedTokens:    retTok,
			SavedTokens:       meta[i].costs[0] - retTok,
			NormalizedEntropy: meta[i].hNorm,
			Probabilities: map[string]float64{
				ContextActionFull: meta[i].probs[0],
				ContextActionStub: meta[i].probs[1],
				ContextActionDrop: meta[i].probs[2],
			},
		}
	}
	return ContextPruneResult{
		Decisions:       decisions,
		OriginalTokens:  origTokens,
		RetainedTokens:  retainedTokens,
		SavedTokens:     origTokens - retainedTokens,
		TokenBudget:     budget,
		BudgetSatisfied: satisfied,
		ExpectedUtility: expectedUtil,
	}
}
