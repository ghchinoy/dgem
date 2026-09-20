package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChatCompletionRequest is the OpenAI-compatible payload format.
type ChatCompletionRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Logprobs    bool          `json:"logprobs,omitempty"`
	TopLogprobs int           `json:"top_logprobs,omitempty"`
}

// ChatMessage represents a single chat turn.
type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ContentPart represents an element in a multimodal content array.
type ContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *ImageURLPart `json:"image_url,omitempty"`
}

// ImageURLPart holds an image URL or base64 data URI.
type ImageURLPart struct {
	URL string `json:"url"`
}

// ChatCompletionResponse is the standard OpenAI-compatible response.
type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

// ChatChoice contains the assistant message and optional token logprobs.
type ChatChoice struct {
	Index        int             `json:"index"`
	Message      ChatMessage     `json:"message"`
	FinishReason string          `json:"finish_reason"`
	Logprobs     *ChoiceLogprobs `json:"logprobs,omitempty"`
}

// ChoiceLogprobs holds OpenAI/vLLM token-level log-probabilities.
type ChoiceLogprobs struct {
	Content []TokenLogprob `json:"content"`
}

// TokenLogprob represents the log-probability and top-k alternatives for a single output token.
type TokenLogprob struct {
	Token       string           `json:"token"`
	Logprob     float64          `json:"logprob"`
	TopLogprobs []TopLogprobItem `json:"top_logprobs,omitempty"`
}

// TopLogprobItem represents a single candidate token and its logprob at a given position.
type TopLogprobItem struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
}

// Usage reports token statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StructuredDecisionResponse represents the parsed Jev-style content.
type StructuredDecisionResponse struct {
	Answers     map[string]QuestionAnswer `json:"answers"`
	Diagnostics Diagnostics               `json:"diagnostics"`
}

// QuestionAnswer holds the evaluated result for a single question.
type QuestionAnswer struct {
	Type          string             `json:"type"`
	Label         string             `json:"label"`
	Confidence    float64            `json:"confidence"`
	Logprob       float64            `json:"logprob,omitempty"`
	Entropy       float64            `json:"entropy,omitempty"`
	Stderr        float64            `json:"stderr"`
	Agreement     float64            `json:"agreement"`
	Probabilities map[string]float64 `json:"probabilities"`

	// Type-specific values:
	Choice string  `json:"choice,omitempty"` // For choice type
	Score  float64 `json:"score,omitempty"`  // For score type
	Level  string  `json:"level,omitempty"`  // For score type
	Noul   float64 `json:"noul,omitempty"`   // For boolean/noul type (probability of true/yes)
}

// DisplayType returns a human-friendly type name, replacing "noul" with "boolean".
func (qa QuestionAnswer) DisplayType() string {
	switch qa.Type {
	case "noul", "bool", "boolean":
		return "boolean"
	default:
		return qa.Type
	}
}

// DisplayValue returns the primary value (e.g. "yes", "engineering", "2 (frustrated)").
func (qa QuestionAnswer) DisplayValue() string {
	switch qa.Type {
	case "noul", "bool", "boolean":
		return qa.Label
	case "choice":
		return qa.Choice
	case "score":
		if qa.Level != "" {
			return qa.Level
		}
		return qa.Label
	default:
		return qa.Label
	}
}

// Diagnostics contains server-side timing, sampling, and slot entropy data.
type Diagnostics struct {
	Hole      string                       `json:"hole"`
	Steps     int                          `json:"steps"`
	Timing    TimingStats                  `json:"timing"`
	Samples   SampleStats                  `json:"samples"`
	Questions map[string]QuestionDiagnostic `json:"questions"`
}

// TimingStats details execution breakdown on Metal.
type TimingStats struct {
	DenoiseMs    float64 `json:"denoise_ms"`
	PrefillMs    float64 `json:"prefill_ms"`
	PromptTokens int     `json:"prompt_tokens"`
	ReusedTokens int     `json:"reused_tokens"`
	Rounds       int     `json:"rounds"`
	Samples      int     `json:"samples"`
	StepsRun     int     `json:"steps_run"`
}

// SampleStats details the multi-read sampling policy.
type SampleStats struct {
	N      int          `json:"n"`
	Policy SamplePolicy `json:"policy"`
}

// SamplePolicy reveals whether the auto-sampling threshold was triggered.
type SamplePolicy struct {
	Extended            bool    `json:"extended"`
	FirstReadMaxEntropy float64 `json:"first_read_max_entropy"`
	Max                 int     `json:"max"`
	Mode                string  `json:"mode"`
	PinnedSlots         int     `json:"pinned_slots"`
	Threshold           float64 `json:"threshold"`
}

// QuestionDiagnostic holds raw token entropy and probability mass.
type QuestionDiagnostic struct {
	ArgmaxIsLabel bool    `json:"argmax_is_label"`
	ArgmaxToken   string  `json:"argmax_token"`
	Entropy       float64 `json:"entropy"`
	LabelMass     float64 `json:"label_mass"`
}

// RequestStats summarizes the full lifecycle of a query for CLI display.
type RequestStats struct {
	Model        string
	Endpoint     string
	WallTime     time.Duration
	PrefillMs    float64
	DenoiseMs    float64
	PromptTokens int
	ReusedTokens int
	OutputTokens int
	TotalTokens  int
	DenoiseSteps int
	SamplesN     int
	Extended     bool
	Diagnostics  *Diagnostics
}

// ParseStructuredContent attempts to unmarshal the raw assistant text as a StructuredDecisionResponse.
func ParseStructuredContent(content string) (*StructuredDecisionResponse, error) {
	return ParseStructuredContentWithLogprobs(content, nil)
}

// ParseStructuredContentWithLogprobs unmarshals the assistant text and enriches slot confidence,
// top-k candidate probabilities, and Shannon entropy using OpenAI/vLLM token logprobs when present.
func ParseStructuredContentWithLogprobs(content string, logprobs *ChoiceLogprobs) (*StructuredDecisionResponse, error) {
	var structured StructuredDecisionResponse
	if err := json.Unmarshal([]byte(content), &structured); err == nil && len(structured.Answers) > 0 {
		return &structured, nil
	}

	cleaned := cleanJSON(content)
	if err := json.Unmarshal([]byte(cleaned), &structured); err == nil && len(structured.Answers) > 0 {
		return &structured, nil
	}

	// Fallback: Check if response is a direct JSON key-value map (e.g. from vLLM completions)
	var rawMap map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &rawMap); err == nil && len(rawMap) > 0 {
		structured.Answers = make(map[string]QuestionAnswer)
		if structured.Diagnostics.Questions == nil {
			structured.Diagnostics.Questions = make(map[string]QuestionDiagnostic)
		}
		for k, v := range rawMap {
			label := fmt.Sprintf("%v", v)
			conf, lp, ent, probs := extractSlotLogprobTelemetry(label, logprobs)
			structured.Answers[k] = QuestionAnswer{
				Type:          "auto",
				Label:         label,
				Confidence:    conf,
				Logprob:       lp,
				Entropy:       ent,
				Probabilities: probs,
			}
			structured.Diagnostics.Questions[k] = QuestionDiagnostic{
				ArgmaxIsLabel: true,
				ArgmaxToken:   label,
				Entropy:       ent,
				LabelMass:     conf,
			}
		}
		return &structured, nil
	}

	return nil, fmt.Errorf("failed to parse structured decision output from: %s", content)
}

// extractSlotLogprobTelemetry locates the token(s) corresponding to the slot value in ChoiceLogprobs
// and computes calibrated probability exp(logprob), Shannon entropy H = -sum(p * ln(p)), and top-k probabilities.
func extractSlotLogprobTelemetry(label string, lp *ChoiceLogprobs) (confidence float64, logprob float64, entropy float64, probs map[string]float64) {
	confidence = 1.0
	if lp == nil || len(lp.Content) == 0 {
		return confidence, 0, 0, nil
	}

	normLabel := strings.ToLower(strings.TrimSpace(label))
	var matched *TokenLogprob

	// Search backwards (after "thought" preamble and JSON key) for the token matching the start of label
	for i := len(lp.Content) - 1; i >= 0; i-- {
		tokClean := strings.ToLower(strings.Trim(lp.Content[i].Token, " \t\n\r\"',:{}[]"))
		if tokClean == "" {
			continue
		}
		if tokClean == normLabel || strings.HasPrefix(normLabel, tokClean) || strings.HasSuffix(normLabel, tokClean) {
			matched = &lp.Content[i]
			break
		}
	}
	if matched == nil {
		// Fallback to the highest-entropy content token inside the JSON body
		for i := len(lp.Content) - 1; i >= 0; i-- {
			tokClean := strings.Trim(lp.Content[i].Token, " \t\n\r\"',:{}[]")
			if len(tokClean) > 0 {
				matched = &lp.Content[i]
				break
			}
		}
	}
	if matched == nil {
		return 1.0, 0, 0, nil
	}

	logprob = matched.Logprob
	confidence = math.Exp(logprob)
	if confidence > 1.0 {
		confidence = 1.0
	}

	if len(matched.TopLogprobs) > 0 {
		probs = make(map[string]float64, len(matched.TopLogprobs))
		var h float64
		for _, item := range matched.TopLogprobs {
			p := math.Exp(item.Logprob)
			tKey := strings.TrimSpace(item.Token)
			if tKey != "" {
				probs[tKey] = p
			}
			if p > 0 {
				h -= p * math.Log(p)
			}
		}
		entropy = h
	}

	return confidence, logprob, entropy, probs
}

func cleanJSON(content string) string {
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "```json"); idx != -1 {
		content = content[idx+7:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
	} else if idx := strings.Index(content, "```"); idx != -1 {
		content = content[idx+3:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
	} else if idx := strings.Index(content, "{"); idx != -1 {
		if end := strings.LastIndex(content, "}"); end != -1 && end > idx {
			content = content[idx : end+1]
		}
	}
	return strings.TrimSpace(content)
}

// RawContent returns the content as a string regardless of whether it was deserialized as string or map.
func (m ChatMessage) RawContent() string {
	if s, ok := m.Content.(string); ok {
		return s
	}
	b, _ := json.Marshal(m.Content)
	return string(b)
}

// BuildMultimodalContent formats user message content with optional image parts.
// If images are provided, it encodes local files to base64 data URIs and places
// image parts BEFORE the text content, conforming to DiffusionGemma best practices.
func BuildMultimodalContent(textContent string, imageInputs []string) (interface{}, error) {
	if len(imageInputs) == 0 {
		return textContent, nil
	}

	var parts []ContentPart

	for _, img := range imageInputs {
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}

		var imageURI string
		if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") || strings.HasPrefix(img, "data:") {
			imageURI = img
		} else {
			// Read local file
			fileData, err := os.ReadFile(img)
			if err != nil {
				return nil, fmt.Errorf("failed to read local image file %s: %w", img, err)
			}

			mimeType := detectImageMime(img, fileData)
			encoded := base64.StdEncoding.EncodeToString(fileData)
			imageURI = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
		}

		parts = append(parts, ContentPart{
			Type:     "image_url",
			ImageURL: &ImageURLPart{URL: imageURI},
		})
	}

	// Place text content after images
	if textContent != "" {
		parts = append(parts, ContentPart{
			Type: "text",
			Text: textContent,
		})
	}

	return parts, nil
}

func detectImageMime(path string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		contentType := http.DetectContentType(data)
		if strings.HasPrefix(contentType, "image/") {
			return contentType
		}
		return "image/jpeg"
	}
}
