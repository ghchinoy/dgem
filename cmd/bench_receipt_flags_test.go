package cmd

import "testing"

// Bench commands must not write receipts unless asked: their old defaults pointed at committed reference
// receipts, so every live run silently overwrote them.
func TestBenchReceiptFlagsDefaultEmpty(t *testing.T) {
	for _, c := range []struct {
		name string
		flag string
		get  func() string
	}{
		{"bench-permutation", "output", func() string { return benchPermutationCmd.Flags().Lookup("output").DefValue }},
		{"bench-permutation", "out", func() string { return benchPermutationCmd.Flags().Lookup("out").DefValue }},
		{"bench-decision-index", "output", func() string { return benchDecisionIndexCmd.Flags().Lookup("output").DefValue }},
		{"bench-decision-index", "out", func() string { return benchDecisionIndexCmd.Flags().Lookup("out").DefValue }},
		{"bench-rerank", "output", func() string { return benchRerankCmd.Flags().Lookup("output").DefValue }},
	} {
		if got := c.get(); got != "" {
			t.Errorf("%s --%s default = %q, want empty", c.name, c.flag, got)
		}
	}
	if benchPermutationCmd.Flags().ShorthandLookup("o") == nil || benchDecisionIndexCmd.Flags().ShorthandLookup("o") == nil {
		t.Error("bench-permutation and bench-decision-index need -o")
	}
}
