package decisionindex

import "testing"

func TestNoulCriteriaKept(t *testing.T) {
	got := noulCriteria(map[string]string{"true": " The response is unsupported. ", "false": "All content is supported."})
	if got["true"] != "The response is unsupported." || got["false"] != "All content is supported." {
		t.Fatalf("got %v", got)
	}
	if got := noulCriteria(map[string]string{"Yes": "a", "no": "b"}); got["true"] != "a" || got["false"] != "b" {
		t.Fatalf("yes/no synonyms: %v", got)
	}
	if noulCriteria(nil) != nil || noulCriteria(map[string]string{"maybe": "x"}) != nil {
		t.Fatal("expected nil")
	}
}
