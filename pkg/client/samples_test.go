package client

import (
	"encoding/json"
	"testing"
)

func TestNormalizeSchemaSamples(t *testing.T) {
	cases := map[string]any{
		`{"questions":[],"samples":"4"}`:    float64(4),
		`{"questions":[],"samples":" 2 "}`:  float64(2),
		`{"questions":[],"samples":4}`:      float64(4),
		`{"questions":[],"samples":"auto"}`: "auto",
		`{"questions":[],"samples":"0"}`:    "0",
	}
	for in, want := range cases {
		var m map[string]any
		if err := json.Unmarshal([]byte(NormalizeSchemaSamples(in)), &m); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if m["samples"] != want {
			t.Errorf("%s: samples = %#v, want %#v", in, m["samples"], want)
		}
	}
	if got := NormalizeSchemaSamples(`{"questions":[]}`); got != `{"questions":[]}` {
		t.Errorf("no samples: changed to %s", got)
	}
	if got := NormalizeSchemaSamples(`not json`); got != `not json` {
		t.Errorf("invalid json changed: %s", got)
	}
}
