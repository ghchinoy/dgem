package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func TestIsLoopbackURL(t *testing.T) {
	loopbacks := []string{
		"http://127.0.0.1:8080/v1",
		"http://localhost:8090/api/decide",
		"http://[::1]:8080",
		"http://0.0.0.0:8080",
		"127.0.0.1:8080",
	}
	for _, u := range loopbacks {
		if !isLoopbackURL(u) {
			t.Errorf("expected %q to be recognized as loopback URL", u)
		}
	}

	nonLoopbacks := []string{
		"https://dgemma-example-uc.a.run.app/v1",
		"https://gateway.example.com/v1",
		"https://us-central1-aiplatform.googleapis.com/v1/projects/foo/locations/us-central1/endpoints/123:rawPredict",
	}
	for _, u := range nonLoopbacks {
		if isLoopbackURL(u) {
			t.Errorf("expected %q NOT to be recognized as loopback URL", u)
		}
	}
}

func TestResolveBackendTargetLocal(t *testing.T) {
	backendConfigMu.Lock()
	origDefB := serveDefaultBackend
	origLocURL := serveLocalURL
	serveDefaultBackend = "vertex_first"
	serveLocalURL = "http://127.0.0.1:8080/v1"
	backendConfigMu.Unlock()
	defer func() {
		backendConfigMu.Lock()
		serveDefaultBackend = origDefB
		serveLocalURL = origLocURL
		backendConfigMu.Unlock()
	}()

	ctx := context.Background()

	// 1. Explicit "local" request
	target, targetURL, err := resolveBackendTargetFromParams(ctx, "local", "")
	if err != nil {
		t.Fatalf("unexpected error resolving local backend: %v", err)
	}
	if target != "local" {
		t.Errorf("expected target 'local', got %q", target)
	}
	if targetURL != "http://127.0.0.1:8080/v1" {
		t.Errorf("expected targetURL 'http://127.0.0.1:8080/v1', got %q", targetURL)
	}

	// 2. Default backend set to "local"
	backendConfigMu.Lock()
	serveDefaultBackend = "local"
	backendConfigMu.Unlock()

	target, targetURL, err = resolveBackendTargetFromParams(ctx, "", "")
	if err != nil {
		t.Fatalf("unexpected error resolving empty backend with local default: %v", err)
	}
	if target != "local" {
		t.Errorf("expected target 'local' from default, got %q", target)
	}
	if targetURL != "http://127.0.0.1:8080/v1" {
		t.Errorf("expected targetURL 'http://127.0.0.1:8080/v1', got %q", targetURL)
	}

	// 3. Fallback to Cloud Run when unknown requested mode and default is cloudrun
	backendConfigMu.Lock()
	serveDefaultBackend = "cloudrun"
	backendConfigMu.Unlock()
	viper.Set("url", "https://dgemma-test.run.app/v1")

	target, targetURL, err = resolveBackendTargetFromParams(ctx, "cloudrun", "")
	if err != nil {
		t.Fatalf("unexpected error resolving cloudrun: %v", err)
	}
	if target != "cloudrun" {
		t.Errorf("expected target 'cloudrun', got %q", target)
	}
	if targetURL != "https://dgemma-test.run.app/v1" {
		t.Errorf("expected cloudrun targetURL, got %q", targetURL)
	}
}

func TestCheckHealthAndGPUStatusForBackendLocal(t *testing.T) {
	// Start mock local diffgemma server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" || r.URL.Path == "/models" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{{"id": "diffgemma-26b-a4b-it-q4"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	backendConfigMu.Lock()
	origDefB := serveDefaultBackend
	origLocURL := serveLocalURL
	serveDefaultBackend = "local"
	serveLocalURL = mockServer.URL + "/v1"
	backendConfigMu.Unlock()
	defer func() {
		backendConfigMu.Lock()
		serveDefaultBackend = origDefB
		serveLocalURL = origLocURL
		backendConfigMu.Unlock()
	}()

	gpuStateMu.Lock()
	initialWarm := lastWarmTimestamp
	gpuStateMu.Unlock()

	out := CheckHealthAndGPUStatusForBackend(context.Background(), "user@example.com", "local")

	if !out.GPUAvailable {
		t.Errorf("expected GPUAvailable true, got false")
	}
	if out.GPUState != "warm_and_ready" {
		t.Errorf("expected GPUState 'warm_and_ready', got %q", out.GPUState)
	}
	if out.ActiveBackend != "local" {
		t.Errorf("expected ActiveBackend 'local', got %q", out.ActiveBackend)
	}
	if out.GPUTier != "Apple Silicon Metal (Unified Memory)" {
		t.Errorf("expected Apple Silicon Metal tier, got %q", out.GPUTier)
	}

	// Verify Cloud Run warm timestamp was NOT modified by checking local backend
	gpuStateMu.Lock()
	afterWarm := lastWarmTimestamp
	gpuStateMu.Unlock()
	if !afterWarm.Equal(initialWarm) {
		t.Errorf("expected Cloud Run lastWarmTimestamp to remain %v, but was mutated to %v", initialWarm, afterWarm)
	}
}

func TestRecordLocalReadoutLatency(t *testing.T) {
	RecordLocalReadoutLatency(678)
	gpuStateMu.RLock()
	val := lastLocalReadoutLatencyMs
	gpuStateMu.RUnlock()
	if val != 678 {
		t.Errorf("expected lastLocalReadoutLatencyMs 678, got %d", val)
	}

	// Negative or zero shouldn't overwrite
	RecordLocalReadoutLatency(0)
	gpuStateMu.RLock()
	val2 := lastLocalReadoutLatencyMs
	gpuStateMu.RUnlock()
	if val2 != 678 {
		t.Errorf("expected lastLocalReadoutLatencyMs to stay 678, got %d", val2)
	}
}
