package decisionindex

import (
	"fmt"
	"testing"
)

func TestBalancedBracketsNoSingletons(t *testing.T) {
	for k := 2; k <= 255; k++ {
		keys := make([]string, k)
		for i := range keys {
			keys[i] = fmt.Sprintf("o%d", i)
		}
		bs := balancedBrackets(keys, BracketSize)
		total := 0
		for _, b := range bs {
			if len(b) > BracketSize || (k > 1 && len(b) < 2) {
				t.Fatalf("K=%d: bracket of size %d", k, len(b))
			}
			total += len(b)
		}
		if total != k {
			t.Fatalf("K=%d: covered %d", k, total)
		}
	}
}
