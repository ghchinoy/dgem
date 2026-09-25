package permutation

import (
	"bytes"
	"strings"
	"testing"
)

func TestDualMirrorWarning(t *testing.T) {
	var buf bytes.Buffer
	WarnWriter = &buf
	letterCollisionSeen.Store(false)

	boolSchema := `{"questions":[{"id":"q","type":"boolean","instructions":"x","options":[{"name":"yes"},{"name":"no"}]}]}`
	if _, _, ok := InjectDualMirrorSchema(boolSchema); ok && DualMirrorLetterCollisionSeen() {
		t.Fatalf("boolean slot should not trigger the letter-collision warning")
	}

	choiceSchema := `{"questions":[{"id":"q","type":"choice","instructions":"x","options":[{"name":"a","description":"A"},{"name":"b","description":"B"}]}]}`
	if _, _, ok := InjectDualMirrorSchema(choiceSchema); !ok {
		t.Fatalf("expected mirror injection for choice slot")
	}
	if !DualMirrorLetterCollisionSeen() {
		t.Fatalf("choice slot should trigger the letter-collision warning")
	}
	if !strings.Contains(buf.String(), "EXP-14") {
		t.Fatalf("warning not written: %q", buf.String())
	}
}
