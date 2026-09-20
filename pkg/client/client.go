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
	AuthToken  string
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

// WithAuthToken sets the Authorization Bearer / IAM token.
func (c *Client) WithAuthToken(token string) *Client {
	c.AuthToken = strings.TrimSpace(token)
	return c
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

	if c.AuthToken != "" {
		token := c.AuthToken
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = "Bearer " + token
		}
		httpReq.Header.Set("Authorization", token)
	}

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
		content := chatResp.Choices[0].Message.RawContent()
		if structured, err := ParseStructuredContent(content); err == nil && (structured.Diagnostics.Steps > 0 || structured.Diagnostics.Timing.TotalMs > 0 || len(structured.Answers) > 0) {
			stats.Diagnostics = &structured.Diagnostics
			stats.PrefillMs = structured.Diagnostics.Timing.PrefillMs
			if structured.Diagnostics.Timing.DenoiseMs > 0 {
				stats.DenoiseMs = structured.Diagnostics.Timing.DenoiseMs
			} else {
				stats.DenoiseMs = structured.Diagnostics.Timing.TotalMs
			}
			stats.ReusedTokens = structured.Diagnostics.Timing.ReusedTokens
			stats.DenoiseSteps = structured.Diagnostics.Timing.StepsRun
			stats.SamplesN = structured.Diagnostics.Samples.N
			stats.Extended = structured.Diagnostics.Samples.Policy.Extended
		}
	}

	return &chatResp, stats, nil
}

// Decide executes a discrete diffusion slot readout decision query with optional multimodal images.
func (c *Client) Decide(ctx context.Context, schemaContent, userStateContent string, images ...string) (*StructuredDecisionResponse, *RequestStats, error) {
	userPayload, err := BuildMultimodalContent(userStateContent, images)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process multimodal content: %w", err)
	}

	req := ChatCompletionRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: schemaContent},
			{Role: "user", Content: userPayload},
		},
		Logprobs:    true,
		TopLogprobs: 5,
	}

	chatResp, stats, err := c.Complete(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, stats, fmt.Errorf("no response choices returned by model")
	}

	rawText := chatResp.Choices[0].Message.RawContent()
	structured, err := ParseStructuredContentWithLogprobs(rawText, chatResp.Choices[0].Logprobs)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to parse structured decision output: %w (raw content: %s)", err, rawText)
	}

	return structured, stats, nil
}
