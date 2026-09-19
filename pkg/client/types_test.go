package client

import (
	"testing"
)

func TestParseStructuredContent(t *testing.T) {
	raw := `{
		"answers": {
			"urgent": {
				"type": "noul",
				"label": "yes",
				"confidence": 0.985,
				"stderr": 0.012,
				"agreement": 1.0,
				"probabilities": {
					"yes": 0.985,
					"no": 0.015
				}
			},
			"team": {
				"type": "choice",
				"choice": "engineering",
				"label": "C",
				"confidence": 0.999,
				"stderr": 0.0001,
				"agreement": 1.0,
				"probabilities": {
					"engineering": 0.999
				}
			}
		},
		"diagnostics": {
			"hole": "noise",
			"steps": 1,
			"timing": {
				"denoise_ms": 850.5,
				"prefill_ms": 1200.0,
				"prompt_tokens": 160,
				"reused_tokens": 120,
				"rounds": 1,
				"samples": 1,
				"steps_run": 1
			},
			"samples": {
				"n": 1,
				"policy": {
					"extended": false,
					"first_read_max_entropy": 0.02,
					"max": 4,
					"mode": "auto",
					"pinned_slots": 0,
					"threshold": 0.1
				}
			}
		}
	}`

	parsed, err := ParseStructuredContent(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing structured content: %v", err)
	}

	if len(parsed.Answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(parsed.Answers))
	}

	urgent := parsed.Answers["urgent"]
	if urgent.DisplayType() != "boolean" {
		t.Errorf("expected DisplayType 'boolean', got %q", urgent.DisplayType())
	}
	if urgent.DisplayValue() != "yes" {
		t.Errorf("expected DisplayValue 'yes', got %q", urgent.DisplayValue())
	}

	if parsed.Diagnostics.Timing.DenoiseMs != 850.5 {
		t.Errorf("expected DenoiseMs 850.5, got %f", parsed.Diagnostics.Timing.DenoiseMs)
	}
}
