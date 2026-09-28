package cmd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ghchinoy/dgem/pkg/client"
)

func TestSplitBalanced(t *testing.T) {
	opts := make([]string, 30)
	for i := range opts {
		opts[i] = fmt.Sprintf("o%02d", i)
	}
	g := splitBalanced(opts, 20)
	if len(g) != 2 || len(g[0]) != 15 || len(g[1]) != 15 {
		t.Fatalf("30/20 -> %d groups %v", len(g), []int{len(g[0]), len(g[1])})
	}
	if g := splitBalanced(opts[:77%30], 20); len(g) != 1 {
		t.Fatalf("17 options -> %d groups", len(g))
	}
}

// fake decide: picks the option named "o21" when present, otherwise the first option; rejects > 26 options.
func fakeDecide(calls *[]int) intentDecideFunc {
	return func(v map[string]interface{}) (*client.StructuredDecisionResponse, *client.RequestStats, error) {
		opts := v["options"].([]string)
		*calls = append(*calls, len(opts))
		if len(opts) > maxChoiceOptions {
			return nil, nil, fmt.Errorf("at most 26 alternatives")
		}
		pick := opts[0]
		for _, o := range opts {
			if o == "o21" {
				pick = o
			}
		}
		probs := map[string]float64{}
		for _, o := range opts {
			probs[o] = 0.01
		}
		probs[pick] = 0.9
		return &client.StructuredDecisionResponse{Answers: map[string]client.QuestionAnswer{
				"intent": {Type: "choice", Label: pick, Choice: pick, Probabilities: probs}}},
			&client.RequestStats{WallTime: 10 * time.Millisecond}, nil
	}
}

func TestDecideIntentBracketed(t *testing.T) {
	opts := make([]string, 30)
	for i := range opts {
		opts[i] = fmt.Sprintf("o%02d", i)
	}
	var calls []int
	resp, stats, err := decideIntentBracketed(context.Background(), "", map[string]interface{}{"text": "x"}, opts, fakeDecide(&calls))
	if err != nil {
		t.Fatal(err)
	}
	if got := intentAnswer(resp).DisplayValue(); got != "o21" {
		t.Fatalf("final answer %q, want o21", got)
	}
	if fmt.Sprint(calls) != "[15 15 10]" {
		t.Fatalf("calls %v, want two groups of 15 then 10 finalists", calls)
	}
	if stats.WallTime != 30*time.Millisecond {
		t.Fatalf("wall time %v, want summed 30ms", stats.WallTime)
	}

	calls = nil
	if _, _, err := decideIntentBracketed(context.Background(), "", nil, opts[:26], fakeDecide(&calls)); err != nil || fmt.Sprint(calls) != "[26]" {
		t.Fatalf("26 options should be one call: %v %v", calls, err)
	}
}
