package cmd

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// geminiSafeInputSchema infers the JSON Schema for an MCP tool input type T and
// normalizes it so Gemini / Vertex AI function calling accepts it.
//
// jsonschema-go marks Go slices and pointers as nullable (`"type": ["null",
// "array"]`). Gemini rewrites a type array into `anyOf`, then rejects the
// declaration because `anyOf` sits next to sibling keys such as `description`
// and `items` ("When using any_of, it must be the only field set"). A single
// invalid declaration fails every request from the MCP client, not just calls
// to this tool, so every tool registers its input schema through this helper.
//
// Output schemas are intentionally not normalized: Gemini does not validate
// them, and go-sdk validates handler results against them, where nil
// slices/pointers legitimately serialize as null.
func geminiSafeInputSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](nil)
	if err != nil {
		// Input types are static Go structs; failure is a programming error.
		panic(fmt.Sprintf("dgem mcp: inferring input schema for %T: %v", *new(T), err))
	}
	stripNullability(s)
	return s
}

// stripNullability recursively collapses `["null", X]` type arrays to `X` and
// drops null-only branches from `anyOf`/`oneOf`.
func stripNullability(s *jsonschema.Schema) {
	if s == nil {
		return
	}

	if len(s.Types) > 0 {
		nonNull := make([]string, 0, len(s.Types))
		for _, t := range s.Types {
			if t != "null" {
				nonNull = append(nonNull, t)
			}
		}
		if len(nonNull) == 1 {
			s.Type, s.Types = nonNull[0], nil
		} else {
			s.Types = nonNull
		}
	}
	if s.Type == "null" && len(s.Types) == 0 {
		// A bare null-only schema carries no useful constraint for Gemini.
		s.Type = ""
	}

	s.AnyOf = dropNullBranches(s.AnyOf)
	s.OneOf = dropNullBranches(s.OneOf)

	for _, c := range s.Properties {
		stripNullability(c)
	}
	for _, c := range s.PatternProperties {
		stripNullability(c)
	}
	for _, c := range s.Defs {
		stripNullability(c)
	}
	for _, c := range s.Definitions {
		stripNullability(c)
	}
	for _, c := range s.PrefixItems {
		stripNullability(c)
	}
	for _, c := range s.ItemsArray {
		stripNullability(c)
	}
	for _, c := range append(append(append([]*jsonschema.Schema{}, s.AllOf...), s.AnyOf...), s.OneOf...) {
		stripNullability(c)
	}
	for _, c := range []*jsonschema.Schema{
		s.Items, s.AdditionalItems, s.Contains, s.UnevaluatedItems,
		s.AdditionalProperties, s.PropertyNames, s.UnevaluatedProperties,
		s.Not, s.If, s.Then, s.Else,
	} {
		stripNullability(c)
	}
}

// dropNullBranches removes `{"type":"null"}` branches from an anyOf/oneOf
// list. jsonschema-go does not emit anyOf/oneOf for struct inference today;
// TestMCPInputSchemasAreGeminiSafe fails if a future type introduces one.
func dropNullBranches(branches []*jsonschema.Schema) []*jsonschema.Schema {
	var kept []*jsonschema.Schema
	for _, b := range branches {
		if b != nil && b.Type == "null" && len(b.Types) == 0 {
			continue
		}
		kept = append(kept, b)
	}
	return kept
}
