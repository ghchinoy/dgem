package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghchinoy/dgem/pkg/template"
	"github.com/spf13/cobra"
)

var (
	renderTemplate string
	renderVars     []string
	renderDataFile string
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Inspect, list, and render Go template definition files",
	Long:  `template provides utilities to manage and preview prompt and schema templates.`,
}

var templateListCmd = &cobra.Command{
	Use:   "list [DIRECTORY]",
	Short: "List available template files in a directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "templates"
		if len(args) > 0 {
			dir = args[0]
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		fmt.Printf("\nAvailable Templates in %s/:\n", dir)
		fmt.Println(strings.Repeat("-", 60))
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".tmpl") {
				fmt.Printf("  • %-30s\n", filepath.Join(dir, e.Name()))
			}
		}
		fmt.Println()
		return nil
	},
}

var templateRenderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render a template definition with provided variables to inspect output",
	RunE: func(cmd *cobra.Command, args []string) error {
		if renderTemplate == "" {
			return fmt.Errorf("must specify --template (-t)")
		}

		vars := make(map[string]interface{})
		for _, kv := range renderVars {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if (strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]")) || (strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}")) {
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
		}

		engine := template.NewEngine()
		rendered, err := engine.RenderFile(renderTemplate, vars)
		if err != nil {
			return fmt.Errorf("render failed: %w", err)
		}

		fmt.Println("───────────────────────── RENDERED TEMPLATE ─────────────────────────")
		fmt.Println(rendered)
		fmt.Println("───────────────────────────────────────────────────────────────────────")
		return nil
	},
}

func init() {
	templateRenderCmd.Flags().StringVarP(&renderTemplate, "template", "t", "", "Path to template file (.tmpl)")
	templateRenderCmd.Flags().StringArrayVarP(&renderVars, "var", "v", nil, "Template variables in key=value format")
	templateRenderCmd.Flags().StringVarP(&renderDataFile, "data", "d", "", "JSON data file")

	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateRenderCmd)
	RootCmd.AddCommand(templateCmd)
}
