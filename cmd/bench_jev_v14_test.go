package cmd

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestJevBenchV14CompositeHarmonicScore(t *testing.T) {
	testCases := []struct {
		name          string
		intel         float64
		cal           float64
		speed         float64
		cost          float64
		expectedScore float64
	}{
		{
			name:          "plumb-4b (v1.4.2.1 rank 1)",
			intel:         52.97892668438094,
			cal:           75.48544394158392,
			speed:         93.49040479933188,
			cost:          55.76977914062651,
			expectedScore: 65.843448,
		},
		{
			name:          "decider-4b-v2 (v1.4.2.1 rank 2, intel gate <50)",
			intel:         49.35289661364111,
			cal:           74.99659054834055,
			speed:         92.93237918931897,
			cost:          60.92496476683517,
			expectedScore: 64.128895,
		},
		{
			name:          "jev-1.13.0 (v1.4.2.1 rank 3, API endpoint)",
			intel:         53.05904597275748,
			cal:           76.33898315040740,
			speed:         83.26811926100174,
			cost:          51.96616538951724,
			expectedScore: 63.292057,
		},
		{
			name:          "djev (v1.4.2.1 rank 9, diffusion-gemma reference)",
			intel:         46.99529771527977,
			cal:           55.355844103478105,
			speed:         91.35677310946093,
			cost:          57.57524920597589,
			expectedScore: 52.228492,
		},
		{
			name:          "openjev-razorback16 (v1.4.2.1 rank 28, NVFP4 dual gate <50)",
			intel:         45.43523560052183,
			cal:           54.993853529501855,
			speed:         83.17789847863520,
			cost:          45.49198106171689,
			expectedScore: 36.850709,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			calc := computeJevV14CompositeScore(tc.intel, tc.cal, tc.speed, tc.cost)
			if math.Abs(calc-tc.expectedScore) > 1e-4 {
				t.Errorf("%s: expected score %.6f, got %.6f (diff: %.6f)",
					tc.name, tc.expectedScore, calc, math.Abs(calc-tc.expectedScore))
			}
		})
	}
}

func TestJevBenchV14Gates(t *testing.T) {
	// Zero axis yields zero composite
	if s := computeJevV14CompositeScore(0, 50, 50, 50); s != 0.0 {
		t.Errorf("expected 0 for zero intelligence, got %f", s)
	}

	// Below 50 penalty on intelligence
	baseH := 4.0 / (1.0/40.0 + 1.0/60.0 + 1.0/60.0 + 1.0/60.0) // ~53.33
	expected := baseH * (40.0 / 50.0) * (40.0 / 50.0)          // ~34.13
	actual := computeJevV14CompositeScore(40.0, 60.0, 60.0, 60.0)
	if math.Abs(actual-expected) > 1e-6 {
		t.Errorf("expected %f with intel penalty, got %f", expected, actual)
	}
}

func TestJevBenchV14SpeedAdjustment(t *testing.T) {
	p50 := 0.12
	p95 := 0.39

	// API kind: no load factor adjustment
	adjAPI, rawAPI := computeJevSpeedScoreWithKind(p50, p95, "api")
	if adjAPI != rawAPI {
		t.Errorf("expected API kind to be unadjusted, got adj=%f raw=%f", adjAPI, rawAPI)
	}
	if math.Abs(rawAPI-93.2975) > 1e-3 {
		t.Errorf("expected raw speed ~93.30, got %f", rawAPI)
	}

	// GPU kind: load factor 2.0x + 0.15s
	adjGPU, rawGPU := computeJevSpeedScoreWithKind(p50, p95, "gpu")
	if adjGPU >= rawGPU {
		t.Errorf("expected GPU adjusted speed to be lower than raw, got adj=%f raw=%f", adjGPU, rawGPU)
	}
	// p50 adj = 0.12*2 + 0.15 = 0.39s -> 100 - 20*log10(3.9) = 88.178
	// p95 adj = 0.39*2 + 0.15 = 0.93s -> 100 - 20*log10(9.3) = 80.629
	// expected mean = ~84.40
	if math.Abs(adjGPU-84.4039) > 1e-3 {
		t.Errorf("expected GPU adjusted speed ~84.40, got %f", adjGPU)
	}
}

func TestJevBenchV14IntelligenceGapPenalty(t *testing.T) {
	publicIntel := 76.77
	publicAcc := 0.818 // 81.8%

	// 1. Unknown sealed accuracy (public only)
	intelPub, penPub, isEst := computeJevV14Intelligence(publicIntel, publicAcc, 0.0)
	if !isEst {
		t.Errorf("expected isEst=true for zero sealed acc")
	}
	if intelPub != publicIntel {
		t.Errorf("expected intel to equal publicIntel, got %f", intelPub)
	}
	if penPub != 0.0 {
		t.Errorf("expected zero gap penalty, got %f", penPub)
	}

	// 2. High sealed accuracy within 25 pt gap: no penalty
	// sealedAcc = 0.65 -> gap = 81.8 - 65.0 = 16.8 pts <= 25.0
	intelNoPen, penNo, _ := computeJevV14Intelligence(publicIntel, publicAcc, 0.65)
	if penNo != 0.0 {
		t.Errorf("expected 0 gap penalty when gap <= 25, got %f", penNo)
	}
	if intelNoPen <= 0 {
		t.Errorf("expected positive intel, got %f", intelNoPen)
	}

	// 3. Sealed accuracy with large gap: penalty applied
	// sealedAcc = 0.40 -> gap = 81.8 - 40.0 = 41.8 pts -> penalty = (41.8 - 25.0) = 16.8%
	_, penLarge, _ := computeJevV14Intelligence(publicIntel, publicAcc, 0.40)
	if math.Abs(penLarge-16.8) > 0.1 {
		t.Errorf("expected gap penalty ~16.8%%, got %f%%", penLarge)
	}
}

func TestLeaderboardJSONFile(t *testing.T) {
	b, err := os.ReadFile("../benchmarks/jevbench/leaderboard_v14.json")
	if err != nil {
		b, err = os.ReadFile("benchmarks/jevbench/leaderboard_v14.json")
	}
	if err != nil {
		t.Fatalf("failed to read leaderboard_v14.json: %v", err)
	}

	var lb LeaderboardFile
	if err := json.Unmarshal(b, &lb); err != nil {
		t.Fatalf("invalid json in leaderboard_v14.json: %v", err)
	}

	if lb.Revision != "v1.4.2.1" {
		t.Errorf("expected revision v1.4.2.1, got %q", lb.Revision)
	}
	if len(lb.Systems) < 5 {
		t.Errorf("expected at least 5 systems, got %d", len(lb.Systems))
	}

	top1 := lb.Systems[0]
	if top1.Key != "plumb-4b" || top1.Rank != 1 {
		t.Errorf("expected plumb-4b at rank 1, got %q rank %d", top1.Key, top1.Rank)
	}

	hasDjev := false
	hasOpenjev := false
	for _, s := range lb.Systems {
		if s.Key == "djev" {
			hasDjev = true
		}
		if s.Key == "openjev-razorback16" {
			hasOpenjev = true
		}
	}
	if !hasDjev {
		t.Errorf("missing djev in leaderboard systems")
	}
	if !hasOpenjev {
		t.Errorf("missing openjev-razorback16 in leaderboard systems")
	}
}
