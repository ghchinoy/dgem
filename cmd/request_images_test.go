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

package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestCheckRequestImages(t *testing.T) {
	ctx := context.Background()
	defer func() { allowLocalImagePaths = true }()

	allowLocalImagePaths = true
	if err := checkRequestImages(ctx, []string{"fixtures/x.png", "data:image/png;base64,AA=="}); err != nil {
		t.Fatalf("CLI must allow local paths and data URIs: %v", err)
	}

	allowLocalImagePaths = false
	for _, bad := range []string{"/etc/passwd", "../../secret.json", "http://127.0.0.1/x.png", "http://169.254.169.254/computeMetadata/v1/",
		"http://10.0.0.5/a.png", "http://[::1]/a.png", "http://localhost/a.png"} {
		if err := checkRequestImages(ctx, []string{bad}); err == nil {
			t.Fatalf("server must reject %q", bad)
		}
	}
	if err := checkRequestImages(ctx, []string{"data:image/png;base64,AA==", ""}); err != nil {
		t.Fatalf("server must allow data URIs: %v", err)
	}
}

func TestCheckSystemOneImages(t *testing.T) {
	ctx := context.Background()
	defer func() { allowLocalImagePaths = true }()
	allowLocalImagePaths = false
	if err := checkSystemOneImages(ctx, []string{"data:image/png;base64,AA==", ""}); err != nil {
		t.Fatalf("data URIs must pass: %v", err)
	}
	for _, bad := range []string{"/etc/passwd", "http://169.254.169.254/computeMetadata/v1/", "http://localhost/a.png"} {
		if err := checkSystemOneImages(ctx, []string{bad}); err == nil {
			t.Fatalf("gateway must reject %q", bad)
		}
	}
	// The adapter never reads local paths, even where the CLI would allow them.
	allowLocalImagePaths = true
	if err := checkSystemOneImages(ctx, []string{"fixtures/x.png"}); err == nil {
		t.Fatal("systemone images must not be local paths")
	}
}

// SVG references are refused at the gateway with a 400-able error before any model call (#117).
func TestCheckRequestImagesRejectsSVG(t *testing.T) {
	ctx := context.Background()
	for _, ref := range []string{"data:image/svg+xml;base64,PHN2Zy8+", "fixtures/ui_component.svg"} {
		if err := checkRequestImages(ctx, []string{ref}); err == nil || !strings.Contains(err.Error(), "SVG") {
			t.Errorf("%s: err = %v, want an SVG refusal", ref, err)
		}
	}
	if err := checkSystemOneImages(ctx, []string{"data:image/svg+xml,<svg/>"}); err == nil {
		t.Error("systemone: SVG data URI accepted")
	}
}
