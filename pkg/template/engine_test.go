package template

import (
	"testing"
)

func TestRenderString(t *testing.T) {
	e := NewEngine()
	data := map[string]interface{}{
		"Name": "DiffusionGemma",
		"Active": true,
	}

	tmpl := `Model: {{ .Name }}, Active: {{ .Active }}, Upper: {{ upper .Name }}`
	out, err := e.RenderString("test", tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Model: DiffusionGemma, Active: true, Upper: DIFFUSIONGEMMA"
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestParseStructuredPayload(t *testing.T) {
	input := `{
		"schema": {
			"instructions": "Test",
			"questions": [{"id": "q1", "type": "boolean"}]
		},
		"state": {
			"ticket": "sample ticket"
		}
	}`

	schema, state, err := ParseStructuredPayload(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schema == "" || state == "" {
		t.Fatalf("expected non-empty schema and state, got schema=%q state=%q", schema, state)
	}
}
