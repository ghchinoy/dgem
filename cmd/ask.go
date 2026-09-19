package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghchinoy/dgem/pkg/client"
	"github.com/ghchinoy/dgem/pkg/template"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	askTemplate  string
	askVars      []string
	askMaxTokens int
	askThink     bool
)

var askCmd = &cobra.Command{
	Use:   "ask [PROMPT]",
	Short: "Send a generative prompt or rendered template to DiffusionGemma",
	Long: `ask sends a natural-language prompt or template to DiffusionGemma
for standard generative completion. You can enable or disable thinking mode
and inspect generation statistics.`,
	RunE: runAsk,
}

func init() {
	askCmd.Flags().StringVarP(&askTemplate, "template", "t", "", "Path to Go prompt template file (.txt.tmpl)")
	askCmd.Flags().StringArrayVarP(&askVars, "var", "v", nil, "Template variables in key=value format")
	askCmd.Flags().IntVar(&askMaxTokens, "max-tokens", 256, "Maximum tokens to generate")
	askCmd.Flags().BoolVar(&askThink, "think", false, "Enable thinking mode (<|think|>)")

	RootCmd.AddCommand(askCmd)
}

func runAsk(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c := GetClient()

	var promptText string

	if askTemplate != "" {
		vars := make(map[string]interface{})
		for _, kv := range askVars {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		engine := template.NewEngine()
		rendered, err := engine.RenderFile(askTemplate, vars)
		if err != nil {
			return fmt.Errorf("failed to render prompt template: %w", err)
		}
		promptText = strings.TrimSpace(rendered)
	} else if len(args) > 0 {
		promptText = strings.Join(args, " ")
	} else {
		return fmt.Errorf("must provide either a prompt as arguments or a --template (-t)")
	}

	// Model target selection for thinking mode
	targetModel := c.Model
	if !askThink && !strings.Contains(targetModel, ":think=false") {
		targetModel = targetModel + ":think=false"
	} else if askThink && !strings.Contains(targetModel, ":think") {
		targetModel = targetModel + ":think"
	}

	req := client.ChatCompletionRequest{
		Model: targetModel,
		Messages: []client.ChatMessage{
			{Role: "user", Content: promptText},
		},
		MaxTokens: askMaxTokens,
	}

	resp, stats, err := c.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("completion request failed: %w", err)
	}

	if len(resp.Choices) > 0 {
		fmt.Println(resp.Choices[0].Message.Content)
	}

	if viper.GetBool("stats") {
		PrintStats(stats)
	}

	return nil
}
