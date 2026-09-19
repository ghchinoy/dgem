package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client communicates with the local or remote diffgemma server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Model      string
}

// NewClient creates a new DiffGemma client.
func NewClient(baseURL, defaultModel string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: timeout},
		Model:      defaultModel,
	}
}

// Complete executes an OpenAI-compatible chat completion.
func (c *Client) Complete(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, *RequestStats, error) {
	if req.Model == "" {
		req.Model = c.Model
	}

	endpoint := fmt.Sprintf("%s/chat/completions", c.BaseURL)
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("http request failed to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	wallTime := time.Since(start)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("server returned error HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	stats := &RequestStats{
		Model:        chatResp.Model,
		Endpoint:     endpoint,
		WallTime:     wallTime,
		PromptTokens: chatResp.Usage.PromptTokens,
		OutputTokens: chatResp.Usage.CompletionTokens,
		TotalTokens:  chatResp.Usage.TotalTokens,
	}

	// If the response contains structured diagnostics, extract timing telemetry
	if len(chatResp.Choices) > 0 {
		content := chatResp.Choices[0].Message.Content
		if structured, err := ParseStructuredContent(content); err == nil && structured.Diagnostics.Steps > 0 {
			stats.Diagnostics = &structured.Diagnostics
			stats.PrefillMs = structured.Diagnostics.Timing.PrefillMs
			stats.DenoiseMs = structured.Diagnostics.Timing.DenoiseMs
			stats.ReusedTokens = structured.Diagnostics.Timing.ReusedTokens
			stats.DenoiseSteps = structured.Diagnostics.Timing.StepsRun
			stats.SamplesN = structured.Diagnostics.Samples.N
			stats.Extended = structured.Diagnostics.Samples.Policy.Extended
		}
	}

	return &chatResp, stats, nil
}

// Decide executes a Jev-style structured decision query.
func (c *Client) Decide(ctx context.Context, schemaContent, userStateContent string) (*StructuredDecisionResponse, *RequestStats, error) {
	req := ChatCompletionRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: schemaContent},
			{Role: "user", Content: userStateContent},
		},
	}

	chatResp, stats, err := c.Complete(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, stats, fmt.Errorf("no response choices returned by model")
	}

	structured, err := ParseStructuredContent(chatResp.Choices[0].Message.Content)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to parse structured decision output: %w (raw content: %s)", err, chatResp.Choices[0].Message.Content)
	}

	return structured, stats, nil
}
