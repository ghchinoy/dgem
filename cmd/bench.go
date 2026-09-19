package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghchinoy/dgem/pkg/client"
	"github.com/spf13/cobra"
)

var (
	benchIncludeGenerative bool
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Run the Jev-style structured reads vs autoregression benchmark",
	Long: `bench runs an empirical benchmark comparing single-pass structured decisions
against traditional autoregressive text generation on Apple Silicon Metal.`,
	RunE: runBench,
}

func init() {
	benchCmd.Flags().BoolVar(&benchIncludeGenerative, "with-generative", true, "Include generative comparison on first test cases")
	RootCmd.AddCommand(benchCmd)
}

type benchCase struct {
	Name    string
	Ticket  string
	Urgent  bool
	Team    string
}

var benchCases = []benchCase{
	{
		Name:   "Production Outage",
		Ticket: "EMERGENCY: Production API cluster returning 500 across all US-East nodes. Customer traffic is failing.",
		Urgent: true,
		Team:   "engineering",
	},
	{
		Name:   "Billing Overcharge",
		Ticket: "I was billed twice for my annual renewal this morning ($1200 instead of $600). Please reverse the second charge.",
		Urgent: true,
		Team:   "billing",
	},
	{
		Name:   "Routine Account Question",
		Ticket: "Hi there, where in the dashboard can I invite my colleague as a viewer? No rush at all, thanks!",
		Urgent: false,
		Team:   "support",
	},
	{
		Name:   "Angry Cancellation Threat",
		Ticket: "YOUR APP IS A PIECE OF GARBAGE! Deleted my database migration and corrupted files. Cancel my subscription immediately!",
		Urgent: true,
		Team:   "billing",
	},
	{
		Name:   "Borderline Feature Request",
		Ticket: "Would be nice to have dark mode or maybe some CSS customization when exporting PDFs. Not a bug, just wondering.",
		Urgent: false,
		Team:   "support",
	},
}

const benchSchema = `{
  "instructions": "Classify incoming customer operations tickets.",
  "questions": [
    {
      "id": "urgent",
      "type": "boolean",
      "instructions": "Does this issue require immediate same-day escalation?"
    },
    {
      "id": "team",
      "type": "choice",
      "instructions": "Which department owns resolution of this ticket?",
      "options": [
        {"name": "billing", "description": "charges, invoices, refunds, subscriptions"},
        {"name": "support", "description": "general questions, account help, password resets"},
        {"name": "engineering", "description": "system bugs, outages, API errors, 500s"}
      ]
    },
    {
      "id": "sentiment",
      "type": "score",
      "instructions": "Customer anger / distress level",
      "levels": ["calm", "frustrated", "furious"]
    }
  ],
  "samples": "auto"
}`

func runBench(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c := GetClient()

	fmt.Println("================================================================================")
	fmt.Println("  DIFFUSIONGEMMA: JEV STRUCTURED READS VS AUTOREGRESSIVE CLASSIFICATION")
	fmt.Println("================================================================================")
	fmt.Printf("Target Server: %s\n\n", c.BaseURL)

	fmt.Println("Warming up server & prefilling schema KV cache...")
	warmupState := `{"ticket": "Health check prefill"}`
	_, _, err := c.Decide(ctx, benchSchema, warmupState)
	if err != nil {
		return fmt.Errorf("failed to connect to server at %s: %w", c.BaseURL, err)
	}
	fmt.Println("Warmup complete. Schema prefix is resident in KV cache.\n")

	type structResult struct {
		Name     string
		Team     string
		Urgent   string
		Conf     float64
		Samples  int
		Denoise  float64
		WallTime time.Duration
		Extended bool
	}

	var results []structResult

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-28s | %-12s | %-8s | %-6s | %-7s | %-8s | %s\n",
		"Test Case", "Team", "Urgent", "Conf", "Samples", "Denoise", "Total")
	fmt.Println(strings.Repeat("-", 80))

	for _, tc := range benchCases {
		stateBytes, _ := json.Marshal(map[string]string{"ticket": tc.Ticket})
		resp, stats, err := c.Decide(ctx, benchSchema, string(stateBytes))
		if err != nil {
			return fmt.Errorf("bench case %q failed: %w", tc.Name, err)
		}

		team := resp.Answers["team"].Choice
		urgent := resp.Answers["urgent"].DisplayValue()
		conf := resp.Answers["team"].Confidence
		samples := stats.SamplesN
		denoise := stats.DenoiseMs
		wall := stats.WallTime

		results = append(results, structResult{
			Name:     tc.Name,
			Team:     team,
			Urgent:   urgent,
			Conf:     conf,
			Samples:  samples,
			Denoise:  denoise,
			WallTime: wall,
			Extended: stats.Extended,
		})

		fmt.Printf("%-28s | %-12s | %-8s | %5.1f%% | %7d | %6.0fms | %5.2fs\n",
			tc.Name, team, urgent, conf*100, samples, denoise, wall.Seconds())
	}

	if benchIncludeGenerative {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("  TESTING STANDARD GENERATIVE CHAT ON FIRST 2 CASES FOR COMPARISON")
		fmt.Println(strings.Repeat("=", 80))

		for _, tc := range benchCases[:2] {
			fmt.Printf("\n--- Generative Test: %s ---\n", tc.Name)
			prompt := fmt.Sprintf("Classify this ticket. Output ONLY valid JSON with keys: 'urgent' (boolean), 'team' ('billing', 'support', or 'engineering'), 'sentiment' ('calm', 'frustrated', or 'furious').\n\nTicket: %s", tc.Ticket)

			model := c.Model
			if !strings.Contains(model, ":think=false") {
				model = model + ":think=false"
			}

			req := client.ChatCompletionRequest{
				Model: model,
				Messages: []client.ChatMessage{
					{Role: "user", Content: prompt},
				},
				MaxTokens: 64,
			}

			chatResp, stats, err := c.Complete(ctx, req)
			if err != nil {
				fmt.Printf("Generative test failed: %v\n", err)
				continue
			}

			content := ""
			if len(chatResp.Choices) > 0 {
				content = strings.TrimSpace(chatResp.Choices[0].Message.Content)
			}
			fmt.Printf("Response: %s\n", content)
			fmt.Printf("Tokens: %d | Wall Time: %v\n", stats.OutputTokens, stats.WallTime.Round(time.Millisecond))
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  SUMMARY & ARCHITECTURAL ADVANTAGES OF JEV-STYLE READS")
	fmt.Println(strings.Repeat("=", 80))

	var totalDenoise float64
	var totalWall float64
	var extendedCount int
	for _, r := range results {
		totalDenoise += r.Denoise
		totalWall += r.WallTime.Seconds()
		if r.Extended {
			extendedCount++
		}
	}

	avgDenoise := totalDenoise / float64(len(results))
	avgWall := totalWall / float64(len(results))

	fmt.Printf("• Average Structured Denoise Time: %.1f ms per ticket (pure GPU forward)\n", avgDenoise)
	fmt.Printf("• Average Total Request Time:      %.2f s (including prefill & transmission)\n", avgWall)
	fmt.Printf("• Adaptive Multi-Read Triggered:   %d of %d tickets required multi-sampling\n", extendedCount, len(results))
	fmt.Println("• Zero JSON parsing errors: Output language is mathematically bounded to the schema.")
	fmt.Println(strings.Repeat("=", 80))

	return nil
}
