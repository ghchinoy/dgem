package cmd

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ghchinoy/dgem/pkg/template"
	"github.com/spf13/cobra"
)

var (
	intentsCorpus  string
	intentsDataset string
	intentsFull    bool
	intentsOutput  string
	intentsOffset  int
	intentsLimit   int
	intentsWorkers int
	intentsSamples string
)

var benchIntentsCmd = &cobra.Command{
	Use:   "bench-intents",
	Short: "Benchmark DiffusionGemma high-cardinality intent classification and OOS detection (Banking77 & CLINC150)",
	Long: `bench-intents evaluates DiffusionGemma's discrete slot readout on high-cardinality intent
datasets such as PolyAI/banking77 (77 intents, 3,080 test items) and DeepPavlov/clinc150
(150 intents across 10 domains + Out-of-Scope 'oos', 5,500 test items).

Use --full (-F) with --dataset banking77 or --dataset clinc150 to automatically fetch and run
the complete upstream evaluation split (3,080 items for Banking77; 5,500 items for CLINC150).
Use --workers (-w) to parallelize evaluation across vLLM continuous batching.`,
	Example: `  # Run curated 30-case Banking77 collision cluster
  dgem bench-intents -c benchmarks/intents/banking77_eval.jsonl

  # Run the FULL 3,080-case PolyAI/banking77 test set across all 77 intents with 16 concurrent workers
  dgem bench-intents --dataset banking77 --full --workers 16 -o benchmarks/results_banking77_full.json

  # Run the FULL 5,500-case DeepPavlov/clinc150 test set (4,500 in-domain + 1,000 OOS)
  dgem bench-intents --dataset clinc150 --full --workers 16 -o benchmarks/results_clinc150_full.json`,
	RunE: runBenchIntents,
}

func init() {
	benchIntentsCmd.Flags().StringVarP(&intentsCorpus, "corpus", "c", "benchmarks/intents/banking77_eval.jsonl", "Path to intent evaluation JSONL file")
	benchIntentsCmd.Flags().StringVarP(&intentsDataset, "dataset", "d", "", "Preset dataset to target ('banking77' or 'clinc150')")
	benchIntentsCmd.Flags().BoolVarP(&intentsFull, "full", "F", false, "Download (if needed) and evaluate the complete official test dataset (3,080 for banking77; 5,500 for clinc150)")
	benchIntentsCmd.Flags().StringVarP(&intentsOutput, "output", "o", "", "Export JSON report to file")
	benchIntentsCmd.Flags().IntVar(&intentsOffset, "offset", 0, "Start offset index in dataset (0 = beginning)")
	benchIntentsCmd.Flags().IntVarP(&intentsLimit, "limit", "n", 0, "Limit number of cases to evaluate (0 = all)")
	benchIntentsCmd.Flags().IntVarP(&intentsWorkers, "workers", "w", 1, "Number of concurrent evaluation workers for vLLM continuous batching")
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
	ID               string             `json:"id"`
	Dataset          string             `json:"dataset"`
	Text             string             `json:"text"`
	IsOOS            bool               `json:"is_oos,omitempty"`
	ExpectedDomain   string             `json:"expected_domain,omitempty"`
	ActualDomain     string             `json:"actual_domain,omitempty"`
	DomainAccurate   bool               `json:"domain_accurate,omitempty"`
	ExpectedIntent   string             `json:"expected_intent"`
	ActualIntent     string             `json:"actual_intent"`
	IntentAccurate   bool               `json:"intent_accurate"`
	Confidence       float64            `json:"confidence"`
	Logprob          float64            `json:"logprob,omitempty"`
	Entropy          float64            `json:"entropy,omitempty"`
	TopProbabilities map[string]float64 `json:"top_probabilities,omitempty"`
	WallTimeMs       float64            `json:"wall_time_ms"`
}

// ensureFullDataset downloads and caches the complete 3,080-row Banking77 or 5,500-row CLINC150 dataset on demand.
func ensureFullDataset(dataset string) (string, error) {
	if err := os.MkdirAll("benchmarks/intents", 0755); err != nil {
		return "", err
	}
	ds := strings.ToLower(strings.TrimSpace(dataset))
	if ds == "" {
		if strings.Contains(strings.ToLower(intentsCorpus), "clinc") {
			ds = "clinc150"
		} else {
			ds = "banking77"
		}
	}

	if ds == "banking77" {
		targetPath := "benchmarks/intents/banking77_full_test.jsonl"
		if info, err := os.Stat(targetPath); err == nil && info.Size() > 100000 {
			return targetPath, nil
		}
		fmt.Println("==> Fetching full PolyAI/banking77 test split (3,080 utterances, 77 intents)...")
		catResp, err := http.Get("https://raw.githubusercontent.com/PolyAI-LDN/task-specific-datasets/master/banking_data/categories.json")
		if err != nil {
			return "", fmt.Errorf("failed to download banking77 categories: %w", err)
		}
		defer catResp.Body.Close()
		var categories []string
		if err := json.NewDecoder(catResp.Body).Decode(&categories); err != nil {
			return "", fmt.Errorf("failed to parse banking77 categories: %w", err)
		}

		csvResp, err := http.Get("https://raw.githubusercontent.com/PolyAI-LDN/task-specific-datasets/master/banking_data/test.csv")
		if err != nil {
			return "", fmt.Errorf("failed to download banking77 test.csv: %w", err)
		}
		defer csvResp.Body.Close()

		reader := csv.NewReader(csvResp.Body)
		rows, err := reader.ReadAll()
		if err != nil {
			return "", fmt.Errorf("failed to read banking77 CSV: %w", err)
		}

		outFile, err := os.Create(targetPath)
		if err != nil {
			return "", err
		}
		defer outFile.Close()

		count := 0
		for idx, r := range rows {
			if idx == 0 || len(r) < 2 {
				continue
			}
			count++
			ic := IntentCase{
				ID:             fmt.Sprintf("b77-%04d", count),
				Dataset:        "PolyAI/banking77 (Full 77-Intent Test Set)",
				Text:           strings.TrimSpace(r[0]),
				ExpectedIntent: strings.TrimSpace(r[1]),
				Options:        categories,
			}
			b, _ := json.Marshal(ic)
			outFile.Write(b)
			outFile.WriteString("\n")
		}
		fmt.Printf("==> Cached %d Banking77 test cases to %s\n", count, targetPath)
		return targetPath, nil
	}

	if ds == "clinc150" {
		targetPath := "benchmarks/intents/clinc150_full_test.jsonl"
		if info, err := os.Stat(targetPath); err == nil && info.Size() > 100000 {
			return targetPath, nil
		}
		fmt.Println("==> Fetching full DeepPavlov/clinc150 test + OOS split (5,500 utterances, 151 intents)...")
		resp, err := http.Get("https://raw.githubusercontent.com/clinc/oos-eval/master/data/data_full.json")
		if err != nil {
			return "", fmt.Errorf("failed to download clinc150 data_full.json: %w", err)
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		var rawData struct {
			Test    [][]string `json:"test"`
			OOSTest [][]string `json:"oos_test"`
		}
		if err := json.Unmarshal(bodyBytes, &rawData); err != nil {
			return "", fmt.Errorf("failed to parse clinc150 json: %w", err)
		}

		seenIntents := make(map[string]bool)
		var allIntents []string
		for _, pair := range rawData.Test {
			if len(pair) >= 2 && !seenIntents[pair[1]] {
				seenIntents[pair[1]] = true
				allIntents = append(allIntents, pair[1])
			}
		}
		allIntents = append(allIntents, "oos")

		outFile, err := os.Create(targetPath)
		if err != nil {
			return "", err
		}
		defer outFile.Close()

		count := 0
		for _, pair := range rawData.Test {
			if len(pair) < 2 {
				continue
			}
			count++
			ic := IntentCase{
				ID:             fmt.Sprintf("c150-%04d", count),
				Dataset:        "DeepPavlov/clinc150 (Full 151-Intent + OOS Test Set)",
				Text:           strings.TrimSpace(pair[0]),
				ExpectedIntent: strings.TrimSpace(pair[1]),
				IsOOS:          false,
				Options:        allIntents,
			}
			b, _ := json.Marshal(ic)
			outFile.Write(b)
			outFile.WriteString("\n")
		}
		for _, pair := range rawData.OOSTest {
			if len(pair) < 2 {
				continue
			}
			count++
			ic := IntentCase{
				ID:             fmt.Sprintf("c150-%04d", count),
				Dataset:        "DeepPavlov/clinc150 (Full 151-Intent + OOS Test Set)",
				Text:           strings.TrimSpace(pair[0]),
				ExpectedIntent: "oos",
				IsOOS:          true,
				Options:        allIntents,
			}
			b, _ := json.Marshal(ic)
			outFile.Write(b)
			outFile.WriteString("\n")
		}
		fmt.Printf("==> Cached %d CLINC150 test cases (4,500 in-domain + 1,000 OOS) to %s\n", count, targetPath)
		return targetPath, nil
	}

	return "", fmt.Errorf("unsupported preset dataset %q (use 'banking77' or 'clinc150')", dataset)
}

func loadIntentCorpus(path string) ([]IntentCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open intent corpus %s: %w", path, err)
	}
	defer file.Close()

	var cases []IntentCase
	scanner := bufio.NewScanner(file)
	// Increase scanner buffer for rows with 151 options
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
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

	targetCorpus := intentsCorpus
	if intentsFull {
		resolvedPath, err := ensureFullDataset(intentsDataset)
		if err != nil {
			return err
		}
		targetCorpus = resolvedPath
	} else if intentsDataset != "" {
		ds := strings.ToLower(strings.TrimSpace(intentsDataset))
		if ds == "clinc150" {
			targetCorpus = filepath.FromSlash("benchmarks/intents/clinc150_eval.jsonl")
		} else if ds == "banking77" {
			targetCorpus = filepath.FromSlash("benchmarks/intents/banking77_eval.jsonl")
		}
	}

	cases, err := loadIntentCorpus(targetCorpus)
	if err != nil {
		return err
	}
	if intentsOffset > 0 && intentsOffset < len(cases) {
		cases = cases[intentsOffset:]
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

	workers := intentsWorkers
	if workers < 1 {
		workers = 1
	}

	fmt.Println("==========================================================================================")
	fmt.Println("  DIFFUSIONGEMMA HIGH-CARDINALITY INTENT & OOS BENCHMARK (BANKING77 / CLINC150)")
	fmt.Println("==========================================================================================")
	fmt.Printf("Corpus:         %s (%d test cases, %d candidate intents)\n", targetCorpus, len(cases), len(cases[0].Options))
	fmt.Printf("Template:       %s (Hierarchical Domain+Intent: %v)\n", tmplPath, hasDomains)
	fmt.Printf("DiffusionGemma: %s (%s) [samples=%v, workers=%d, logprobs=true]\n\n", c.Model, c.BaseURL, samplesParam, workers)

	fmt.Println(strings.Repeat("-", 126))
	fmt.Printf("%-8s | %-32s | %-24s | %-24s | %-5s | %-6s | %-6s | %-6s\n",
		"ID", "Utterance", "Expected Intent", "dgem Slot Output", "Match", "Conf", "Ent", "Latency")
	fmt.Println(strings.Repeat("-", 126))

	results := make([]IntentResult, len(cases))
	var mu sync.Mutex
	var intentCorrect, domainCorrect, inDomainCorrect, inDomainTotal, oosCorrect, oosTotal int
	var sumWall, sumConf, sumEntropy float64

	type jobItem struct {
		idx int
		tc  IntentCase
	}
	jobs := make(chan jobItem, len(cases))
	for idx, tc := range cases {
		jobs <- jobItem{idx: idx, tc: tc}
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				tc := j.tc
				vars := map[string]interface{}{
					"text":    tc.Text,
					"options": tc.Options,
					"domains": tc.Domains,
					"samples": samplesParam,
				}

				rendered, err := engine.RenderFile(tmplPath, vars)
				if err != nil {
					continue
				}
				schemaContent, stateContent, err := template.ParseStructuredPayload(rendered, vars)
				if err != nil {
					continue
				}

				resp, stats, err := c.Decide(ctx, schemaContent, stateContent)
				if err != nil {
					continue
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
				wallMs := float64(stats.WallTime.Milliseconds())

				ir := IntentResult{
					ID:               tc.ID,
					Dataset:          tc.Dataset,
					Text:             tc.Text,
					IsOOS:            tc.IsOOS,
					ExpectedDomain:   tc.ExpectedDomain,
					ActualDomain:     actualDomain,
					DomainAccurate:   dAcc,
					ExpectedIntent:   tc.ExpectedIntent,
					ActualIntent:     actualIntent,
					IntentAccurate:   iAcc,
					Confidence:       ansIntent.Confidence,
					Logprob:          ansIntent.Logprob,
					Entropy:          ansIntent.Entropy,
					TopProbabilities: ansIntent.Probabilities,
					WallTimeMs:       wallMs,
				}

				mu.Lock()
				results[j.idx] = ir
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
				sumWall += wallMs
				sumConf += ansIntent.Confidence
				sumEntropy += ansIntent.Entropy

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

				fmt.Printf("%-8s | %-32s | %-24s | %-24s | %-5s | %5.3f  | %5.3f  | %5.0fms\n",
					tc.ID,
					truncateStr(tc.Text, 32),
					truncateStr(expLabel, 24),
					truncateStr(actLabel, 24),
					mark,
					ansIntent.Confidence,
					ansIntent.Entropy,
					wallMs,
				)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	n := float64(len(results))
	intentAccPct := float64(intentCorrect) / n * 100
	domainAccPct := float64(domainCorrect) / n * 100

	fmt.Println("\n" + strings.Repeat("=", 88))
	fmt.Printf("  SUMMARY: %s (%d Cases)\n", cases[0].Dataset, len(results))
	fmt.Println(strings.Repeat("=", 88))
	fmt.Printf("• Fine-Grained Intent Accuracy:       %.1f%% (%d of %d correct)\n", intentAccPct, intentCorrect, len(results))
	if hasDomains {
		fmt.Printf("• Hierarchical Domain Accuracy:       %.1f%% (%d of %d correct)\n", domainAccPct, domainCorrect, len(results))
	}
	if inDomainTotal > 0 && oosTotal > 0 {
		fmt.Printf("• In-Domain Intent Accuracy:          %.1f%% (%d of %d correct)\n", float64(inDomainCorrect)/float64(inDomainTotal)*100, inDomainCorrect, inDomainTotal)
		fmt.Printf("• Out-of-Scope (OOS) Rejection Rate:  %.1f%% (%d of %d correct)\n", float64(oosCorrect)/float64(oosTotal)*100, oosCorrect, oosTotal)
	}
	fmt.Printf("• Mean Slot Confidence exp(logprob):  %.4f (Mean Entropy: %.4f nats)\n", sumConf/n, sumEntropy/n)
	fmt.Printf("• Average Slot Readout Latency:       %.1f ms\n", sumWall/n)
	fmt.Println(strings.Repeat("=", 88))

	if intentsOutput != "" {
		outData := map[string]interface{}{
			"timestamp":           time.Now().UTC().Format(time.RFC3339),
			"dataset":             cases[0].Dataset,
			"corpus":              targetCorpus,
			"target_url":          c.BaseURL,
			"target_model":        c.Model,
			"samples_policy":      samplesParam,
			"workers":             workers,
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
			"mean_confidence":     sumConf / n,
			"mean_entropy_nats":   sumEntropy / n,
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
