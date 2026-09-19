package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ghchinoy/dgem/pkg/client"
	"github.com/ghchinoy/dgem/pkg/template"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	decideTemplate string
	decideVars     []string
	decideDataFile string
	decideFormat   string
	decideSchema   string
	decideState    string
	decideImages   []string
)

var decideCmd = &cobra.Command{
	Use:   "decide",
	Short: "Execute a discrete diffusion slot readout decision using single-forward evaluation",
	Long: `decide evaluates propositions, categorical choices, and ordered scales
via discrete diffusion slot readout in a single forward pass without autoregressive
text generation overhead (popularized by TypeSafe AI's Jev evaluations and vLLM PR #57250).
You can specify a template definition file (-t), key-value pairs (-v key=val),
or raw schema/state payloads.`,
	RunE: runDecide,
}

func init() {
	decideCmd.Flags().StringVarP(&decideTemplate, "template", "t", "", "Path to Go template definition file (.json.tmpl)")
	decideCmd.Flags().StringArrayVarP(&decideVars, "var", "v", nil, "Template variables in key=value format (can be specified multiple times)")
	decideCmd.Flags().StringVarP(&decideDataFile, "data", "d", "", "JSON file containing template variables")
	decideCmd.Flags().StringVarP(&decideFormat, "format", "f", "table", "Output format: 'table' or 'json'")
	decideCmd.Flags().StringVar(&decideSchema, "schema", "", "Raw JSON schema file path (skips template engine)")
	decideCmd.Flags().StringVar(&decideState, "state", "", "Raw JSON state string or file path")
	decideCmd.Flags().StringArrayVarP(&decideImages, "image", "I", nil, "Attach local image file path or remote image URL (can be specified multiple times)")

	RootCmd.AddCommand(decideCmd)
}

func parseVariables() (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	// Load from data file if provided
	if decideDataFile != "" {
		dataBytes, err := os.ReadFile(decideDataFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read data file %s: %w", decideDataFile, err)
		}
		if err := json.Unmarshal(dataBytes, &vars); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from %s: %w", decideDataFile, err)
		}
	}

	// Override or populate with CLI -v key=val pairs
	for _, kv := range decideVars {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid variable format %q (expected key=value)", kv)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Auto-parse booleans, numbers, or JSON arrays/objects if applicable
		if val == "true" {
			vars[key] = true
		} else if val == "false" {
			vars[key] = false
		} else if (strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]")) || (strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}")) {
			var parsedJSON interface{}
			if err := json.Unmarshal([]byte(val), &parsedJSON); err == nil {
				vars[key] = parsedJSON
			} else {
				vars[key] = val
			}
		} else {
			vars[key] = val
		}
	}

	return vars, nil
}

func runDecide(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c := GetClient()

	var schemaContent, stateContent string
	var err error

	vars, err := parseVariables()
	if err != nil {
		return err
	}

	if decideTemplate != "" {
		engine := template.NewEngine()
		rendered, err := engine.RenderFile(decideTemplate, vars)
		if err != nil {
			return fmt.Errorf("template render failed: %w", err)
		}

		schemaContent, stateContent, err = template.ParseStructuredPayload(rendered, vars)
		if err != nil {
			return fmt.Errorf("failed to process rendered template: %w", err)
		}
	} else if decideSchema != "" {
		sb, err := os.ReadFile(decideSchema)
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}
		schemaContent = string(sb)

		if decideState != "" {
			if strings.HasPrefix(strings.TrimSpace(decideState), "{") {
				stateContent = decideState
			} else {
				stb, err := os.ReadFile(decideState)
				if err != nil {
					return fmt.Errorf("failed to read state file: %w", err)
				}
				stateContent = string(stb)
			}
		} else {
			stb, _ := json.Marshal(vars)
			stateContent = string(stb)
		}
	} else {
		return fmt.Errorf("must specify either --template (-t) or --schema")
	}

	resp, stats, err := c.Decide(ctx, schemaContent, stateContent, decideImages...)
	if err != nil {
		return fmt.Errorf("decision query failed: %w", err)
	}

	if decideFormat == "json" {
		pretty, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(pretty))
	} else {
		printDecisionTable(resp)
	}

	if viper.GetBool("stats") {
		PrintStats(stats)
	}

	return nil
}

func printDecisionTable(resp *client.StructuredDecisionResponse) {
	fmt.Println()
	fmt.Printf("%-16s | %-10s | %-20s | %-10s | %-10s | %-10s\n",
		"QUESTION", "TYPE", "VALUE / CHOICE", "CONFIDENCE", "STDERR", "AGREEMENT")
	fmt.Println(strings.Repeat("-", 88))

	keys := make([]string, 0, len(resp.Answers))
	for k := range resp.Answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		ans := resp.Answers[k]
		qType := ans.DisplayType()
		val := ans.DisplayValue()
		confStr := fmt.Sprintf("%.1f%%", ans.Confidence*100)
		stderrStr := fmt.Sprintf("±%.4f", ans.Stderr)
		agreeStr := fmt.Sprintf("%.2f", ans.Agreement)

		fmt.Printf("%-16s | %-10s | %-20s | %-10s | %-10s | %-10s\n",
			k, qType, val, confStr, stderrStr, agreeStr)
	}
}
