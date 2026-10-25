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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ghchinoy/dgem/pkg/client"
	"github.com/ghchinoy/dgem/pkg/template"
)

func TestCalibrateConformalQuantileValidation(t *testing.T) {
	if _, err := CalibrateConformalQuantile(nil, nil, 0.10, ConformalConfig{}); !errors.Is(err, ErrEmptyCalibration) {
		t.Fatalf("expected ErrEmptyCalibration, got %v", err)
	}

	sampleProb := []map[string]float64{{"a": 0.7, "b": 0.3}}
	if _, err := CalibrateConformalQuantile(sampleProb, []string{"a", "b"}, 0.10, ConformalConfig{}); !errors.Is(err, ErrCalibrationLenMismatch) {
		t.Fatalf("expected ErrCalibrationLenMismatch, got %v", err)
	}

	for _, badAlpha := range []float64{0.0, 1.0, -0.1, 1.2, math.NaN(), math.Inf(1)} {
		if _, err := CalibrateConformalQuantile(sampleProb, []string{"a"}, badAlpha, ConformalConfig{}); !errors.Is(err, ErrInvalidAlpha) {
			t.Fatalf("alpha=%v: expected ErrInvalidAlpha, got %v", badAlpha, err)
		}
	}

	if _, err := CalibrateConformalQuantile(sampleProb, []string{"missing"}, 0.10, ConformalConfig{}); !errors.Is(err, ErrUnknownCalibrationKey) {
		t.Fatalf("expected ErrUnknownCalibrationKey, got %v", err)
	}
	if _, err := ConformalScore(sampleProb[0], "missing", ConformalConfig{}); !errors.Is(err, ErrUnknownCalibrationKey) {
		t.Fatalf("ConformalScore: expected ErrUnknownCalibrationKey, got %v", err)
	}

	badCfg := ConformalConfig{Method: ConformalMethod("invalid_mode")}
	if _, err := CalibrateConformalQuantile(sampleProb, []string{"a"}, 0.10, badCfg); !errors.Is(err, ErrInvalidConformalMethod) {
		t.Fatalf("expected ErrInvalidConformalMethod, got %v", err)
	}
	if _, err := ConformalScore(sampleProb[0], "a", badCfg); !errors.Is(err, ErrInvalidConformalMethod) {
		t.Fatalf("ConformalScore: expected ErrInvalidConformalMethod, got %v", err)
	}

	// When ceil((n+1)*(1-alpha)) > n (for example n=5, alpha=0.05 -> ceil(5.7)=6 > 5),
	// QHat must be +Inf so finite-sample marginal coverage >= 1-alpha holds unconditionally.
	smallProbs := make([]map[string]float64, 5)
	smallLabels := make([]string, 5)
	for i := 0; i < 5; i++ {
		smallProbs[i] = map[string]float64{"opt_0": 0.6, "opt_1": 0.4}
		smallLabels[i] = "opt_0"
	}
	calInf, err := CalibrateConformalQuantile(smallProbs, smallLabels, 0.05, ConformalConfig{Method: ConformalLAC})
	if err != nil {
		t.Fatalf("unexpected error on small n calibration: %v", err)
	}
	if !math.IsInf(calInf.QHat, 1) || calInf.OrderIndex != 6 || calInf.EmpiricalCover != 1.0 {
		t.Fatalf("expected +Inf QHat and OrderIndex=6, got QHat=%v OrderIndex=%d Cover=%v", calInf.QHat, calInf.OrderIndex, calInf.EmpiricalCover)
	}
	setInf := ConformalPredictionSet(map[string]float64{"opt_0": 0.9, "opt_1": 0.1}, calInf.QHat, ConformalConfig{Method: ConformalLAC})
	if setInf.SetSize != 2 || !setInf.ShouldEscalate {
		t.Fatalf("expected full 2-option set under +Inf QHat, got %+v", setInf)
	}
}

func drawExchangeableMiscalibratedBatch(rng *rand.Rand, n int, numClasses int, tempMiscal float64) ([]map[string]float64, []string) {
	probs := make([]map[string]float64, n)
	labels := make([]string, n)

	classNames := make([]string, numClasses)
	for c := 0; c < numClasses; c++ {
		classNames[c] = fmt.Sprintf("class_%02d", c)
	}

	for i := 0; i < n; i++ {
		trueLogits := make([]float64, numClasses)
		maxTrue := math.Inf(-1)
		for c := 0; c < numClasses; c++ {
			z := rng.NormFloat64() * 1.35
			trueLogits[c] = z
			if z > maxTrue {
				maxTrue = z
			}
		}
		trueProbs := make([]float64, numClasses)
		sumTrue := 0.0
		for c := 0; c < numClasses; c++ {
			v := math.Exp(trueLogits[c] - maxTrue)
			trueProbs[c] = v
			sumTrue += v
		}
		u := rng.Float64() * sumTrue
		acc := 0.0
		goldIdx := numClasses - 1
		for c := 0; c < numClasses; c++ {
			acc += trueProbs[c]
			if u <= acc {
				goldIdx = c
				break
			}
		}
		labels[i] = classNames[goldIdx]

		// Model observes noisy, overconfident (tempMiscal < 1) logits.
		modelLogits := make([]float64, numClasses)
		maxModel := math.Inf(-1)
		for c := 0; c < numClasses; c++ {
			ml := (trueLogits[c] + rng.NormFloat64()*0.45) / tempMiscal
			modelLogits[c] = ml
			if ml > maxModel {
				maxModel = ml
			}
		}
		pMap := make(map[string]float64, numClasses)
		sumModel := 0.0
		for c := 0; c < numClasses; c++ {
			v := math.Exp(modelLogits[c] - maxModel)
			pMap[classNames[c]] = v
			sumModel += v
		}
		for c := 0; c < numClasses; c++ {
			pMap[classNames[c]] /= sumModel
		}
		probs[i] = pMap
	}
	return probs, labels
}

func setContainsLabel(set []string, target string) bool {
	for _, s := range set {
		if s == target {
			return true
		}
	}
	return false
}

func TestSplitConformalFiniteSampleCoverage(t *testing.T) {
	configs := []struct {
		name string
		cfg  ConformalConfig
	}{
		{name: "lac", cfg: ConformalConfig{Method: ConformalLAC}},
		{name: "aps_det", cfg: ConformalConfig{Method: ConformalAPS, Randomized: false}},
		{name: "aps_rand", cfg: ConformalConfig{Method: ConformalAPS, Randomized: true, Seed: 42}},
		{name: "raps_det", cfg: ConformalConfig{Method: ConformalRAPS, KReg: 2, LambdaReg: 0.05, Randomized: false}},
		{name: "raps_rand", cfg: ConformalConfig{Method: ConformalRAPS, KReg: 2, LambdaReg: 0.05, Randomized: true, Seed: 42}},
	}

	for _, alpha := range []float64{0.05, 0.10} {
		for _, tc := range configs {
			rng := rand.New(rand.NewSource(20260417 + int64(alpha*1000)))
			// 1. Single large exchangeable evaluation with N_test = 1,500 (>= 1,000 trials).
			calProbs, calLabels := drawExchangeableMiscalibratedBatch(rng, 500, 8, 0.55)
			testProbs, testLabels := drawExchangeableMiscalibratedBatch(rng, 1500, 8, 0.55)

			cal, err := CalibrateConformalQuantile(calProbs, calLabels, alpha, tc.cfg)
			if err != nil {
				t.Fatalf("%s alpha=%.2f: calibration failed: %v", tc.name, alpha, err)
			}

			hits := 0
			for j := 0; j < len(testProbs); j++ {
				res := ConformalPredictionSet(testProbs[j], cal.QHat, tc.cfg)
				if setContainsLabel(res.PredictionSet, testLabels[j]) {
					hits++
				}
			}
			empCover := float64(hits) / float64(len(testProbs))
			minRequired := 1.0 - alpha - 0.025
			if empCover < minRequired {
				t.Fatalf("%s alpha=%.2f: 1500-trial empirical coverage %.4f < %.4f (q_hat=%.4f)",
					tc.name, alpha, empCover, minRequired, cal.QHat)
			}

			// 2. Repeated split-conformal coverage across R=30 independent calibration/test splits.
			const numSplits = 30
			sumCover := 0.0
			for s := 0; s < numSplits; s++ {
				cP, cL := drawExchangeableMiscalibratedBatch(rng, 250, 8, 0.55)
				tP, tL := drawExchangeableMiscalibratedBatch(rng, 250, 8, 0.55)
				cRes, err := CalibrateConformalQuantile(cP, cL, alpha, tc.cfg)
				if err != nil {
					t.Fatalf("%s split %d: %v", tc.name, s, err)
				}
				splitHits := 0
				for j := 0; j < len(tP); j++ {
					pSet := ConformalPredictionSet(tP[j], cRes.QHat, tc.cfg)
					if setContainsLabel(pSet.PredictionSet, tL[j]) {
						splitHits++
					}
				}
				sumCover += float64(splitHits) / float64(len(tP))
			}
			meanCover := sumCover / float64(numSplits)
			if meanCover < 1.0-alpha-0.015 {
				t.Fatalf("%s alpha=%.2f: mean coverage across %d splits = %.4f < %.4f",
					tc.name, alpha, numSplits, meanCover, 1.0-alpha-0.015)
			}
		}
	}
}

func TestRAPSReducesPredictionSetSizeOnWideVocabulary(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	const K = 20

	drawDiffuseWide := func(n int) ([]map[string]float64, []string) {
		probs := make([]map[string]float64, n)
		labels := make([]string, n)
		for i := 0; i < n; i++ {
			perm := rng.Perm(K)
			topA := fmt.Sprintf("opt_%02d", perm[0])
			topB := fmt.Sprintf("opt_%02d", perm[1])

			// Heterogeneous top-2 concentration across items (ranging from 0.72 to 0.94, mean ~0.83)
			// with remaining mass spread thinly across 18 tail options (~0.003 to ~0.015 each).
			topMass := 0.72 + rng.Float64()*0.22
			if i%3 != 0 {
				// Overconfident tail leakage: model spreads 12% mass across 18 irrelevant tail classes
				// while true label is drawn from a sharper distribution concentrated on top-2.
				topMass = 0.88
			}

			pMap := make(map[string]float64, K)
			pA := topMass * (0.58 + rng.Float64()*0.12)
			pB := topMass - pA
			pMap[topA] = pA
			pMap[topB] = pB

			tailMass := 1.0 - topMass
			tailRaw := make([]float64, K-2)
			tailSum := 0.0
			for j := 0; j < K-2; j++ {
				w := 0.5 + rng.Float64()
				tailRaw[j] = w
				tailSum += w
			}
			for j := 0; j < K-2; j++ {
				optName := fmt.Sprintf("opt_%02d", perm[2+j])
				pMap[optName] = tailMass * (tailRaw[j] / tailSum)
			}

			// True data-generating distribution: on 2/3 of items the model's tail is noisy leakage
			// (true tail probability 1.5%), and on 1/3 of items the label is drawn directly from pMap.
			if i%3 != 0 {
				r := rng.Float64()
				switch {
				case r < 0.64:
					labels[i] = topA
				case r < 0.985:
					labels[i] = topB
				default:
					labels[i] = fmt.Sprintf("opt_%02d", perm[2+rng.Intn(K-2)])
				}
			} else {
				u := rng.Float64()
				acc := 0.0
				chosen := topA
				for c := 0; c < K; c++ {
					optName := fmt.Sprintf("opt_%02d", perm[c])
					acc += pMap[optName]
					if u <= acc {
						chosen = optName
						break
					}
				}
				labels[i] = chosen
			}
			probs[i] = pMap
		}
		return probs, labels
	}

	calProbs, calLabels := drawDiffuseWide(500)
	testProbs, testLabels := drawDiffuseWide(1000)

	apsCfg := ConformalConfig{Method: ConformalAPS}
	rapsCfg := ConformalConfig{Method: ConformalRAPS, KReg: 2, LambdaReg: 0.08}

	apsCal, err := CalibrateConformalQuantile(calProbs, calLabels, 0.05, apsCfg)
	if err != nil {
		t.Fatalf("APS calibration failed: %v", err)
	}
	rapsCal, err := CalibrateConformalQuantile(calProbs, calLabels, 0.05, rapsCfg)
	if err != nil {
		t.Fatalf("RAPS calibration failed: %v", err)
	}

	apsSizeSum, rapsSizeSum := 0, 0
	rapsHits := 0
	for i := 0; i < len(testProbs); i++ {
		apsRes := ConformalPredictionSet(testProbs[i], apsCal.QHat, apsCfg)
		rapsRes := ConformalPredictionSet(testProbs[i], rapsCal.QHat, rapsCfg)
		apsSizeSum += apsRes.SetSize
		rapsSizeSum += rapsRes.SetSize
		if setContainsLabel(rapsRes.PredictionSet, testLabels[i]) {
			rapsHits++
		}
	}

	meanAPS := float64(apsSizeSum) / float64(len(testProbs))
	meanRAPS := float64(rapsSizeSum) / float64(len(testProbs))
	rapsCover := float64(rapsHits) / float64(len(testProbs))

	if meanRAPS >= meanAPS {
		t.Fatalf("expected RAPS mean set size (%.3f) < APS mean set size (%.3f)", meanRAPS, meanAPS)
	}
	if rapsCover < 0.94 {
		t.Fatalf("expected RAPS empirical coverage >= 0.94 at alpha=0.05, got %.4f", rapsCover)
	}
}

func TestConformalCascadeEscalationAndPreFusionGate(t *testing.T) {
	cfg := ConformalConfig{Method: ConformalLAC}

	// 1. Singleton non-abstain set -> fast-path (ShouldEscalate == false).
	confident := map[string]float64{"approve": 0.95, "manual_review": 0.03, "abstain": 0.02}
	resSingle := ConformalPredictionSet(confident, 0.20, cfg)
	if !resSingle.Singleton || resSingle.ContainsAbstain || resSingle.ShouldEscalate {
		t.Fatalf("expected singleton non-abstain set without escalation, got %+v", resSingle)
	}
	if resSingle.TopOption != "approve" || resSingle.SetSize != 1 {
		t.Fatalf("unexpected top option or set size: %+v", resSingle)
	}

	// 2. Singleton abstain set -> must escalate despite Singleton == true.
	for _, absOpt := range []string{"abstain", "insufficient_evidence"} {
		absProbs := map[string]float64{absOpt: 0.94, "approve": 0.06}
		resAbs := ConformalPredictionSet(absProbs, 0.20, cfg)
		if !resAbs.Singleton || !resAbs.ContainsAbstain || !resAbs.ShouldEscalate {
			t.Fatalf("abstain option %q: expected Singleton && ContainsAbstain && ShouldEscalate, got %+v", absOpt, resAbs)
		}
	}

	// 3. Multi-option ambiguous set and empty set -> both must escalate.
	ambig := map[string]float64{"approve": 0.52, "manual_review": 0.48}
	resMulti := ConformalPredictionSet(ambig, 0.55, cfg)
	if resMulti.SetSize != 2 || resMulti.Singleton || !resMulti.ShouldEscalate {
		t.Fatalf("expected 2-option set to escalate, got %+v", resMulti)
	}

	resEmpty := ConformalPredictionSet(ambig, 0.10, cfg)
	if resEmpty.SetSize != 0 || resEmpty.Singleton || !resEmpty.ShouldEscalate {
		t.Fatalf("expected empty prediction set to escalate safely, got %+v", resEmpty)
	}

	// 4. Pre-prior-fusion gate must evaluate RawProbabilities rather than post-fusion Probabilities.
	qr := QuestionResult{
		Choice:           "opt_a",
		RawProbabilities: map[string]float64{"opt_a": 0.51, "opt_b": 0.49},
		Probabilities:    map[string]float64{"opt_a": 0.96, "opt_b": 0.04},
		PriorFused:       true,
	}
	postOnly := ConformalPredictionSet(qr.Probabilities, 0.52, cfg)
	if postOnly.ShouldEscalate {
		t.Fatalf("sanity check: post-fusion distribution {0.96, 0.04} should be singleton at qHat=0.52")
	}
	preGate := EvaluatePreFusionConformalGate(qr, 0.52, cfg)
	if preGate.SetSize != 2 || !preGate.ShouldEscalate {
		t.Fatalf("expected EvaluatePreFusionConformalGate to use RawProbabilities and escalate, got %+v", preGate)
	}

	// 5. Verify SystemOneAnswer pre-fusion recovery from FusePriorWithGateMode("always").
	rawInput := map[string]float64{"opt_a": 0.51, "opt_b": 0.49}
	priors := map[string]float64{"opt_a": 3.5, "opt_b": -3.5}
	fusedProbs, winner, h, normH, rawH, rawNormH, fused := FusePriorWithGateMode(
		rawInput, priors, []string{"opt_a", "opt_b"}, 1.0, 0.5, 0.16, "always",
	)
	if !fused {
		t.Fatalf("expected FusePriorWithGateMode('always') to fuse priors")
	}
	ans := SystemOneAnswer{
		Type:           "choice",
		Choice:         winner,
		Probabilities:  fusedProbs,
		Entropy:        h,
		NormalizedH:    normH,
		RawEntropy:     rawH,
		RawNormalizedH: rawNormH,
		PriorFused:     fused,
	}
	ansGate := EvaluateSystemOnePreFusionGate(ans, priors, 0.5, 0.52, cfg)
	if ansGate.SetSize != 2 || !ansGate.ShouldEscalate {
		t.Fatalf("expected EvaluateSystemOnePreFusionGate to invert prior fusion and escalate, got %+v", ansGate)
	}
}

func TestSelectContextPruningActionsExactKnapsackAndStubRescue(t *testing.T) {
	nineBlocks := make([]ContextBlockSpec, 9)
	if _, _, err := BuildContextPruningQuestions(nineBlocks); !errors.Is(err, ErrTooManyContextBlocks) {
		t.Fatalf("expected ErrTooManyContextBlocks on 9 blocks, got %v", err)
	}
	if _, err := SelectContextPruningActions(nineBlocks, nil, ContextPruneConfig{TokenBudget: 1000}); !errors.Is(err, ErrTooManyContextBlocks) {
		t.Fatalf("expected ErrTooManyContextBlocks from SelectContextPruningActions, got %v", err)
	}
	if _, err := PruneAgentContextBlocks(nil, nil, ContextPruneConfig{TokenBudget: -5}); !errors.Is(err, ErrNegativeTokenBudget) {
		t.Fatalf("expected ErrNegativeTokenBudget, got %v", err)
	}

	// Verify Pinned == true invariance and high-hesitation stub rescue over irreversible drop.
	blocks := []ContextBlockSpec{
		{
			ID:         "sys_invariant",
			Kind:       "system_rules",
			FullText:   "PINNED SYSTEM INVARIANTS",
			StubText:   "PINNED STUB",
			FullTokens: 300,
			StubTokens: 60,
			Pinned:     true,
		},
		{
			ID:         "uncertain_trace",
			Kind:       "test_trace",
			FullText:   "FULL STACK TRACE BODY",
			StubText:   "panic at worker.go:84 (race on map write)",
			FullTokens: 600,
			StubTokens: 80,
		},
		{
			ID:         "stale_build_log",
			Kind:       "tool_output",
			FullText:   "VERBOSE OLD BUILD LOG",
			StubText:   "build succeeded 4 turns ago",
			FullTokens: 500,
			StubTokens: 70,
		},
	}
	probs := map[string]map[string]float64{
		// Even if the model assigns high drop probability to a pinned block, Pinned forces "full".
		"b0": {"full": 0.05, "stub": 0.05, "drop": 0.90},
		// High-hesitation block under tight 420-token budget (300 used by pinned b0 -> 120 left, so full=600 cannot fit).
		"b1": {"full": 0.42, "stub": 0.38, "drop": 0.20},
		"b2": {"full": 0.05, "stub": 0.10, "drop": 0.85},
	}

	res, err := PruneAgentContextBlocks(blocks, probs, ContextPruneConfig{TokenBudget: 420})
	if err != nil {
		t.Fatalf("PruneAgentContextBlocks failed: %v", err)
	}
	if !res.BudgetSatisfied || res.RetainedTokens > 420 {
		t.Fatalf("expected budget<=420 satisfied, got %+v", res)
	}
	if res.Decisions[0].Action != ContextActionFull || res.Decisions[0].RetainedText != "PINNED SYSTEM INVARIANTS" {
		t.Fatalf("expected pinned block b0 to remain full, got %+v", res.Decisions[0])
	}
	if res.Decisions[1].Action != ContextActionStub || res.Decisions[1].RetainedTokens != 80 {
		t.Fatalf("expected high-hesitation block b1 to be rescued as stub (80 tok), got %+v", res.Decisions[1])
	}
	if res.Decisions[2].Action != ContextActionDrop || res.Decisions[2].RetainedTokens != 0 {
		t.Fatalf("expected stale block b2 to be dropped, got %+v", res.Decisions[2])
	}

	// Verify unsatisfied budget behavior when Pinned blocks alone exceed TokenBudget.
	resOver, err := SelectContextPruningActions(blocks, probs, ContextPruneConfig{TokenBudget: 200})
	if err != nil {
		t.Fatalf("unexpected error when pinned exceeds budget: %v", err)
	}
	if resOver.BudgetSatisfied || resOver.Decisions[0].Action != ContextActionFull || resOver.RetainedTokens != 300 {
		t.Fatalf("expected BudgetSatisfied=false with only 300-token pinned block retained, got %+v", resOver)
	}

	// Verify exact MCKP optimality against exhaustive 3^8 enumeration across 50 random 8-block instances.
	rng := rand.New(rand.NewSource(12345))
	for trial := 0; trial < 50; trial++ {
		rBlocks := make([]ContextBlockSpec, MaxSlotsPerPass)
		rProbs := make(map[string]map[string]float64, MaxSlotsPerPass)
		totalFull := 0
		for i := 0; i < MaxSlotsPerPass; i++ {
			fullT := 150 + rng.Intn(1800)
			stubT := 20 + rng.Intn(fullT/2)
			totalFull += fullT
			rBlocks[i] = ContextBlockSpec{
				ID:               fmt.Sprintf("blk_%d", i),
				FullTokens:       fullT,
				StubTokens:       stubT,
				Pinned:           i == 0 && trial%3 == 0,
				ImportanceWeight: 0.5 + rng.Float64()*1.5,
			}
			a, b, c := rng.Float64()+0.05, rng.Float64()+0.05, rng.Float64()+0.05
			rProbs[fmt.Sprintf("b%d", i)] = map[string]float64{"full": a, "stub": b, "drop": c}
		}
		budget := rBlocks[0].FullTokens + rng.Intn(totalFull)
		got, err := SelectContextPruningActions(rBlocks, rProbs, ContextPruneConfig{
			TokenBudget:          budget,
			StubUtilityRatio:     0.65,
			UncertaintyStubBonus: 0.25,
		})
		if err != nil {
			t.Fatalf("trial %d: %v", trial, err)
		}
		if !got.BudgetSatisfied || got.RetainedTokens > budget {
			t.Fatalf("trial %d: budget invariant violated: retained=%d > budget=%d", trial, got.RetainedTokens, budget)
		}

		// Independent brute-force check over all 3^8 = 6561 states.
		bestBrute := math.Inf(-1)
		const totalStates = 6561
		for mask := 0; mask < totalStates; mask++ {
			tmp := mask
			tok := 0
			util := 0.0
			valid := true
			for i := 0; i < MaxSlotsPerPass; i++ {
				act := tmp % 3
				tmp /= 3
				if rBlocks[i].Pinned && act != 0 {
					valid = false
					break
				}
				d := got.Decisions[i]
				pF := d.Probabilities[ContextActionFull]
				pS := d.Probabilities[ContextActionStub]
				pD := d.Probabilities[ContextActionDrop]
				w := rBlocks[i].ImportanceWeight
				switch act {
				case 0:
					tok += rBlocks[i].FullTokens
					util += w * (pF + 0.65*pS)
				case 1:
					tok += rBlocks[i].StubTokens
					util += w * (0.65*pF + pS + 0.25*d.NormalizedEntropy*(1.0-pD))
				case 2:
					util += w * (0.15 * pD)
				}
			}
			if valid && tok <= budget && util > bestBrute {
				bestBrute = util
			}
		}
		if math.Abs(got.ExpectedUtility-bestBrute) > 1e-9 {
			t.Fatalf("trial %d: solver utility %.9f != brute-force optimum %.9f", trial, got.ExpectedUtility, bestBrute)
		}
	}
}

func TestAgentContextPruningTemplateAndSinglePassExecution(t *testing.T) {
	tmplPath := filepath.Join("..", "..", "templates", "agent_context_pruning.json.tmpl")
	eng := template.NewEngine()

	// 1. Empty variable map renders valid JSON with defaults.
	emptyRendered, err := eng.RenderFile(tmplPath, map[string]interface{}{})
	if err != nil {
		t.Fatalf("RenderFile with empty data failed: %v", err)
	}
	var emptyParsed map[string]any
	if err := json.Unmarshal([]byte(emptyRendered), &emptyParsed); err != nil {
		t.Fatalf("rendered template with empty data is not valid JSON: %v", err)
	}

	// 2. Render with 8 trajectory blocks and validate via template.ParseStructuredPayload.
	blocks := make([]ContextBlockSpec, MaxSlotsPerPass)
	blockPayload := make([]map[string]any, MaxSlotsPerPass)
	for i := 0; i < MaxSlotsPerPass; i++ {
		blocks[i] = ContextBlockSpec{
			ID:         fmt.Sprintf("turn_block_%d", i),
			Kind:       "tool_output",
			Summary:    fmt.Sprintf("Tool output digest %d", i),
			FullText:   fmt.Sprintf("Complete content for block %d", i),
			StubText:   fmt.Sprintf("Stub digest %d", i),
			FullTokens: 400,
			StubTokens: 60,
			Pinned:     i == 0,
		}
		blockPayload[i] = map[string]any{
			"slot_id":     fmt.Sprintf("b%d", i),
			"id":          blocks[i].ID,
			"kind":        blocks[i].Kind,
			"full_tokens": blocks[i].FullTokens,
			"stub_tokens": blocks[i].StubTokens,
			"summary":     blocks[i].Summary,
		}
	}

	rendered, err := eng.RenderFile(tmplPath, map[string]interface{}{
		"task_goal":    "fix data race in worker pool shutdown",
		"turn_index":   4,
		"token_budget": 1200,
		"blocks":       blockPayload,
	})
	if err != nil {
		t.Fatalf("RenderFile with 8 blocks failed: %v", err)
	}

	schemaJSON, stateJSON, err := template.ParseStructuredPayload(rendered, nil)
	if err != nil {
		t.Fatalf("ParseStructuredPayload failed: %v", err)
	}

	var schemaObj struct {
		Instructions string `json:"instructions"`
		Samples      int    `json:"samples"`
		Questions    []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Options []struct {
				Name string `json:"name"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaObj); err != nil {
		t.Fatalf("unmarshaling normalized schema failed: %v", err)
	}
	if schemaObj.Samples != 1 {
		t.Fatalf("expected samples=1, got %d", schemaObj.Samples)
	}
	if len(schemaObj.Questions) != MaxSlotsPerPass {
		t.Fatalf("expected %d questions in template, got %d", MaxSlotsPerPass, len(schemaObj.Questions))
	}
	for i, q := range schemaObj.Questions {
		wantID := fmt.Sprintf("b%d", i)
		if q.ID != wantID || q.Type != "choice" {
			t.Fatalf("question %d: want id=%q type=choice, got id=%q type=%q", i, wantID, q.ID, q.Type)
		}
		if len(q.Options) != 3 || q.Options[0].Name != "full" || q.Options[1].Name != "stub" || q.Options[2].Name != "drop" {
			t.Fatalf("question %s: unexpected options %+v", q.ID, q.Options)
		}
	}

	var stateObj map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &stateObj); err != nil {
		t.Fatalf("unmarshaling state JSON failed: %v", err)
	}
	if stateObj["task_goal"] != "fix data race in worker pool shutdown" {
		t.Fatalf("unexpected state task_goal: %v", stateObj["task_goal"])
	}

	// 3. Execute all 8 slots through ExecuteSystemOne against an in-process mock server and assert ForwardPasses == 1.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var sysPayload struct {
			Questions []struct {
				ID string `json:"id"`
			} `json:"questions"`
		}
		for _, m := range reqBody.Messages {
			if m.Role == "system" {
				_ = json.Unmarshal([]byte(m.Content), &sysPayload)
			}
		}
		if len(sysPayload.Questions) > MaxSlotsPerPass {
			http.Error(w, "the canvas holds at most 8 slots per pass", http.StatusUnprocessableEntity)
			return
		}

		answers := make(map[string]client.QuestionAnswer, len(sysPayload.Questions))
		for idx, q := range sysPayload.Questions {
			var probs map[string]float64
			var chosen string
			switch {
			case idx < 2:
				chosen = "full"
				probs = map[string]float64{"full": 0.85, "stub": 0.10, "drop": 0.05}
			case idx < 5:
				chosen = "full"
				probs = map[string]float64{"full": 0.44, "stub": 0.40, "drop": 0.16}
			default:
				chosen = "drop"
				probs = map[string]float64{"full": 0.05, "stub": 0.10, "drop": 0.85}
			}
			answers[q.ID] = client.QuestionAnswer{
				Choice:        chosen,
				Probabilities: probs,
			}
		}
		envBytes, _ := json.Marshal(client.StructuredDecisionResponse{Answers: answers})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": string(envBytes)}},
			},
		})
	}))
	defer srv.Close()

	qs, slotIDs, err := BuildContextPruningQuestions(blocks)
	if err != nil {
		t.Fatalf("BuildContextPruningQuestions failed: %v", err)
	}
	if len(slotIDs) != MaxSlotsPerPass {
		t.Fatalf("expected %d slotIDs, got %d", MaxSlotsPerPass, len(slotIDs))
	}

	cli := client.NewClient(srv.URL, "dgemma", 10*time.Second)
	sysResp, err := ExecuteSystemOne(context.Background(), cli, SystemOneRequest{
		State:     stateObj,
		Questions: qs,
	}, DefaultEngineOptions())
	if err != nil {
		t.Fatalf("ExecuteSystemOne failed: %v", err)
	}
	if sysResp.EvaluationTrace == nil || sysResp.EvaluationTrace.ForwardPasses != 1 {
		t.Fatalf("expected ForwardPasses == 1 for 8-slot context pruning, got %+v", sysResp.EvaluationTrace)
	}

	slotProbs := make(map[string]map[string]float64, len(sysResp.Answers))
	for slotID, ans := range sysResp.Answers {
		slotProbs[slotID] = ans.Probabilities
	}
	pruned, err := PruneAgentContextBlocks(blocks, slotProbs, ContextPruneConfig{TokenBudget: 1200})
	if err != nil {
		t.Fatalf("PruneAgentContextBlocks failed: %v", err)
	}
	if !pruned.BudgetSatisfied || pruned.RetainedTokens > 1200 || pruned.SavedTokens <= 0 {
		t.Fatalf("unexpected pruning summary: %+v", pruned)
	}
}
