package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Engine parses and executes Go templates with rich helper functions.
type Engine struct {
	funcMap template.FuncMap
}

// NewEngine creates a new template engine with default helper functions.
func NewEngine() *Engine {
	e := &Engine{}
	e.funcMap = template.FuncMap{
		"toJson": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"toPrettyJson": func(v interface{}) (string, error) {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"default": func(def, val interface{}) interface{} {
			if val == nil {
				return def
			}
			switch v := val.(type) {
			case string:
				if strings.TrimSpace(v) == "" {
					return def
				}
			case bool:
				return v
			}
			return val
		},
		"indent": func(spaces int, s string) string {
			pad := strings.Repeat(" ", spaces)
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = pad + line
				}
			}
			return strings.Join(lines, "\n")
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
	}
	return e
}

// RenderFile loads a template file from disk and executes it with the given data.
func (e *Engine) RenderFile(filePath string, data map[string]interface{}) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}

	name := filepath.Base(filePath)
	return e.RenderString(name, string(content), data)
}

// RenderString parses and executes an in-memory template string.
func (e *Engine) RenderString(name, tmplStr string, data map[string]interface{}) (string, error) {
	if data == nil {
		data = make(map[string]interface{})
	}

	tmpl, err := template.New(name).Funcs(e.funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// StructuredPayload represents the rendered JSON package containing a schema and state.
type StructuredPayload struct {
	Schema json.RawMessage `json:"schema"`
	State  json.RawMessage `json:"state"`
}

// ParseStructuredPayload parses rendered output that has separate "schema" and "state" envelopes,
// or treats the full payload as the schema if not wrapped.
func ParseStructuredPayload(rendered string, fallbackState map[string]interface{}) (string, string, error) {
	trimmed := strings.TrimSpace(rendered)
	if strings.HasPrefix(trimmed, "{") {
		var envelope StructuredPayload
		if err := json.Unmarshal([]byte(trimmed), &envelope); err == nil && len(envelope.Schema) > 0 {
			// Schema is present in envelope
			schemaStr := string(envelope.Schema)

			var stateStr string
			if len(envelope.State) > 0 && string(envelope.State) != "null" {
				stateStr = string(envelope.State)
			} else {
				// Fallback to variables provided via CLI
				sb, _ := json.Marshal(fallbackState)
				stateStr = string(sb)
			}
			return schemaStr, stateStr, nil
		}
	}

	// Not wrapped in schema/state envelope: treat rendered as the schema itself
	stateBytes, err := json.Marshal(fallbackState)
	if err != nil {
		return "", "", fmt.Errorf("failed to serialize fallback state: %w", err)
	}

	return trimmed, string(stateBytes), nil
}
