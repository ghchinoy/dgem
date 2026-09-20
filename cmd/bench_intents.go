package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ghchinoy/dgem/pkg/template"
	"github.com/spf13/cobra"
)

var (
	intentsCorpus  string
	intentsOutput  string
	intentsLimit   int
	intentsSamples string
)

var benchIntentsCmd = &cobra.Command{
	Use:   "bench-intents",
	Short: "Benchmark DiffusionGemma high-cardinality intent classification and OOS detection (Banking77 & CLINC150)",
	Long: `bench-intents evaluates DiffusionGemma's discrete slot readout on high-cardinality intent
datasets such as PolyAI/banking77 (fine-grained shared-prefix collisions) and DeepPavlov/clinc150
(joint domain + intent hierarchical routing and Out-of-Scope 'oos' rejection).`,
	RunE: runBenchIntents,
}

func init() {
	benchIntentsCmd.Flags().StringVarP(&intentsCorpus, "corpus", "c", "benchmarks/intents/banking77_eval.jsonl", "Path to intent evaluation JSONL file")
	benchIntentsCmd.Flags().StringVarP(&intentsOutput, "output", "o", "", "Export JSON report to file")
	benchIntentsCmd.Flags().IntVarP(&intentsLimit, "limit", "n", 0, "Limit number of cases to evaluate (0 = all)")
	benchIntentsCmd.Flags().StringVar(&intentsSamples, "samples", "1", "DiffusionGemma samples policy ('1', '2', '4', or 'auto')")

	RootCmd.AddCommand(benchIntentsCmd)
}

type IntentCase struct {
	ID             string   `json:"id"`
	Dataset        string   `json:"dataset"`
	Text           string   `json:"text"`
	ExpectedDomain string   `json:"expected_domain,omitempty"`
	ExpectedIntent string   `json:"expected_intent"`
	IsOOS          bool     `json:"is_oos,omitempty"`
	Domains        []string `json:"domains,omitempty"`
	Options        []string `json:"options"`
}

type IntentResult struct {
	ID             string  `json:"id"`
	Dataset        string  `json:"dataset"`
	Text           string  `json:"text"`
	IsOOS          bool    `json:"is_oos,omitempty"`
	ExpectedDomain string  `json:"expected_domain,omitempty"`
	ActualDomain   string  `json:"actual_domain,omitempty"`
	DomainAccurate bool    `json:"domain_accurate,omitempty"`
	ExpectedIntent string  `json:"expected_intent"`
	ActualIntent   string  `json:"actual_intent"`
	IntentAccurate bool    `json:"intent_accurate"`
	Confidence     float64 `json:"confidence"`
	WallTimeMs     float64 `json:"wall_time_ms"`
}

func loadIntentCorpus(path string) ([]IntentCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open intent corpus %s: %w", path, err)
	}
	defer file.Close()

	var cases []IntentCase
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var ic IntentCase
		if err := json.Unmarshal([]byte(line), &ic); err != nil {
			return nil, fmt.Errorf("invalid line %q: %w", line, err)
		}
		cases = append(cases, ic)
	}
	return cases, scanner.Err()
}

func runBenchIntents(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c := GetClient()
	engine := template.NewEngine()

	cases, err := loadIntentCorpus(intentsCorpus)
	if err != nil {
		return err
	}
	if intentsLimit > 0 && intentsLimit < len(cases) {
		cases = cases[:intentsLimit]
	}

	var samplesParam interface{} = "auto"
	if n, err := strconv.Atoi(intentsSamples); err == nil && n > 0 {
		samplesParam = n
	} else if intentsSamples != "" {
		samplesParam = intentsSamples
	}

	hasDomains := len(cases) > 0 && len(cases[0].Domains) > 0
	tmplPath := "templates/intent_banking77.json.tmpl"
	if hasDomains {
		tmplPath = "templates/intent_clinc150.json.tmpl"
	}

	fmt.Println("==========================================================================================")
	fmt.Println("  DIFFUSIONGEMMA HIGH-CARDINALITY INTENT & OOS BENCHMARK (BANKING77 / CLINC150)")
	fmt.Println("==========================================================================================")
	fmt.Printf("Corpus:         %s (%d test cases, %d candidate intents)\n", intentsCorpus, len(cases), len(cases[0].Options))
	fmt.Printf("Template:       %s (Hierarchical Domain+Intent: %v)\n", tmplPath, hasDomains)
	fmt.Printf("DiffusionGemma: %s (%s) [samples=%v]\n\n", c.Model, c.BaseURL, samplesParam)

	var results []IntentResult
	var intentCorrect, domainCorrect, inDomainCorrect, inDomainTotal, oosCorrect, oosTotal int
	var sumWall float64

	fmt.Println(strings.Repeat("-", 116))
	fmt.Printf("%-7s | %-34s | %-26s | %-26s | %-6s | %-7s\n",
		"ID", "Utterance", "Expected Intent", "dgem Slot Output", "Match", "Latency")
	fmt.Println(strings.Repeat("-", 116))

	for _, tc := range cases {
		vars := map[string]interface{}{
			"text":    tc.Text,
			"options": tc.Options,
			"domains": tc.Domains,
			"samples": samplesParam,
		}

		rendered, err := engine.RenderFile(tmplPath, vars)
		if err != nil {
			return fmt.Errorf("template render failed for %s: %w", tc.ID, err)
		}
		schemaContent, stateContent, err := template.ParseStructuredPayload(rendered, vars)
		if err != nil {
			return fmt.Errorf("payload parse failed for %s: %w", tc.ID, err)
		}

		resp, stats, err := c.Decide(ctx, schemaContent, stateContent)
		if err != nil {
			return fmt.Errorf("decide query failed for %s: %w", tc.ID, err)
		}

		ansIntent := resp.Answers["intent"]
		if ansIntent.DisplayValue() == "" {
			if a, ok := resp.Answers["answer"]; ok && a.DisplayValue() != "" {
				ansIntent = a
			} else if len(resp.Answers) == 1 {
				for _, v := range resp.Answers {
					ansIntent = v
				}
			}
		}
		actualIntent := strings.TrimSpace(ansIntent.DisplayValue())
		actualDomain := strings.TrimSpace(resp.Answers["domain"].DisplayValue())
		if actualDomain == "" && actualIntent == "oos" {
			actualDomain = "oos"
		}

		iAcc := strings.EqualFold(actualIntent, tc.ExpectedIntent)
		dAcc := !hasDomains || strings.EqualFold(actualDomain, tc.ExpectedDomain)

		if iAcc {
			intentCorrect++
		}
		if dAcc {
			domainCorrect++
		}
		if tc.IsOOS {
			oosTotal++
			if iAcc {
				oosCorrect++
			}
		} else {
			inDomainTotal++
			if iAcc {
				inDomainCorrect++
			}
		}

		wallMs := float64(stats.WallTime.Milliseconds())
		sumWall += wallMs

		ir := IntentResult{
			ID:             tc.ID,
			Dataset:        tc.Dataset,
			Text:           tc.Text,
			IsOOS:          tc.IsOOS,
			ExpectedDomain: tc.ExpectedDomain,
			ActualDomain:   actualDomain,
			DomainAccurate: dAcc,
			ExpectedIntent: tc.ExpectedIntent,
			ActualIntent:   actualIntent,
			IntentAccurate: iAcc,
			Confidence:     ansIntent.Confidence,
			WallTimeMs:     wallMs,
		}
		results = append(results, ir)

		mark := "PASS"
		if !iAcc {
			mark = "FAIL"
		}
		expLabel := tc.ExpectedIntent
		actLabel := actualIntent
		if hasDomains {
			expLabel = fmt.Sprintf("%s/%s", tc.ExpectedDomain, tc.ExpectedIntent)
			actLabel = fmt.Sprintf("%s/%s", actualDomain, actualIntent)
		}

		fmt.Printf("%-7s | %-34s | %-26s | %-26s | %-6s | %5.0fms\n",
			tc.ID,
			truncateStr(tc.Text, 34),
			truncateStr(expLabel, 26),
			truncateStr(actLabel, 26),
			mark,
			wallMs,
		)
	}

	n := float64(len(results))
	intentAccPct := float64(intentCorrect) / n * 100
	domainAccPct := float64(domainCorrect) / n * 100

	fmt.Println("\n" + strings.Repeat("=", 88))
	fmt.Printf("  SUMMARY: %s (%d Cases)\n", cases[0].Dataset, len(results))
	fmt.Println(strings.Repeat("=", 88))
	fmt.Printf("• Fine-Grained Intent Accuracy:       %.1f%% (%d of %d correct)\n", intentAccPct, intentCorrect, len(results))
	if hasDomains {
		fmt.Printf("• Hierarchical Domain Accuracy:       %.1f%% (%d of %d correct)\n", domainAccPct, domainCorrect, len(results))
		if inDomainTotal > 0 {
			fmt.Printf("• In-Domain Intent Accuracy:          %.1f%% (%d of %d correct)\n", float64(inDomainCorrect)/float64(inDomainTotal)*100, inDomainCorrect, inDomainTotal)
		}
		if oosTotal > 0 {
			fmt.Printf("• Out-of-Scope (OOS) Rejection Rate:  %.1f%% (%d of %d correct)\n", float64(oosCorrect)/float64(oosTotal)*100, oosCorrect, oosTotal)
		}
	}
	fmt.Printf("• Average Slot Readout Latency:       %.1f ms\n", sumWall/n)
	fmt.Println(strings.Repeat("=", 88))

	if intentsOutput != "" {
		outData := map[string]interface{}{
			"timestamp":           time.Now().UTC().Format(time.RFC3339),
			"dataset":             cases[0].Dataset,
			"corpus":              intentsCorpus,
			"target_url":          c.BaseURL,
			"target_model":        c.Model,
			"samples_policy":      samplesParam,
			"candidate_intents":   len(cases[0].Options),
			"total_cases":         len(results),
			"intent_accuracy_pct": intentAccPct,
			"intent_correct":      intentCorrect,
			"domain_accuracy_pct": domainAccPct,
			"domain_correct":      domainCorrect,
			"in_domain_correct":   inDomainCorrect,
			"in_domain_total":     inDomainTotal,
			"oos_correct":         oosCorrect,
			"oos_total":           oosTotal,
			"avg_wall_time_ms":    sumWall / n,
			"results":             results,
		}
		b, _ := json.MarshalIndent(outData, "", "  ")
		if err := os.WriteFile(intentsOutput, b, 0644); err != nil {
			return fmt.Errorf("failed to write output file %s: %w", intentsOutput, err)
		}
		fmt.Printf("\nStructured report written to: %s\n", intentsOutput)
	}

	return nil
}
