package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func ansiAccent(s string) string { return "\033[1;38;5;208m" + s + "\033[0m" }
func ansiPass(s string) string   { return "\033[1;38;5;42m" + s + "\033[0m" }
func ansiWarn(s string) string   { return "\033[1;38;5;214m" + s + "\033[0m" }
func ansiMuted(s string) string  { return "\033[38;5;245m" + s + "\033[0m" }

var (
	rerankDatasetPath string
	rerankReceiptPath string
	rerankFromReceipt string
	rerankJSONOutput  bool
	rerankVerifyMath  bool
)

// RerankMethodMetrics represents the IR & governance metrics for a single reranking strategy.
type RerankMethodMetrics struct {
	Label                  string  `json:"label"`
	NDCG3                  float64 `json:"nDCG@3"`
	NDCG5                  float64 `json:"nDCG@5"`
	NDCG10                 float64 `json:"nDCG@10"`
	MRR10                  float64 `json:"MRR@10"`
	MAP10                  float64 `json:"MAP@10"`
	SpearmanRho            float64 `json:"Spearman_rho"`
	ExactTieRatePct        float64 `json:"Exact_Tie_Rate_pct"`
	NevIRPairwiseAccPct    float64 `json:"S1_NevIR_Pairwise_Acc_pct"`
	HotpotQARecall2Pct     float64 `json:"S3_HotpotQA_Recall@2_pct"`
	FollowIRPMRR           float64 `json:"S4_FollowIR_p_MRR"`
	PoisonQuarantinePct    float64 `json:"S5_Poison_Quarantine_pct"`
	AbstentionAccPct       float64 `json:"S5_Abstention_Acc_pct"`
}

// RerankSuiteReceipt represents the serialized EXP-10 reranking benchmark receipt.
type RerankSuiteReceipt struct {
	Experiment             string                         `json:"experiment"`
	ExecutionMode          string                         `json:"execution_mode"`
	MeanWallLatencyMs      *float64                       `json:"mean_wall_latency_ms"`
	DatasetPath            string                         `json:"dataset_path"`
	TemplatePath           string                         `json:"template_path,omitempty"`
	TotalQueries           int                            `json:"total_queries"`
	CandidatesPerQuery     int                            `json:"candidates_per_query"`
	TotalQueryPassagePairs int                            `json:"total_query_passage_pairs"`
	SummaryMetrics         map[string]RerankMethodMetrics `json:"summary_metrics"`
}

var benchRerankCmd = &cobra.Command{
	Use:     "bench-rerank",
	GroupID: "eval",
	Short:   "Evaluate Listwise Diffusion Canvas Reranking, Softmax Expectation & RAG Poison Quarantine (EXP-10)",
	Long: `bench-rerank runs or inspects the EXP-10 Listwise Diffusion Canvas Reranking evaluation suite
(benchmarks/rerank_suite.jsonl: 30 queries x 10 passages = 300 pairs across 5 public benchmark slices:
NevIR, TREC-DL19, HotpotQA, FollowIR, and MuSiQue + AgentDrift).

It compares four reranking regimes across standard IR metrics (nDCG@10, MRR@10, MAP@10, Exact Tie Rate,
NevIR negation inversion, FollowIR p-MRR policy steerability, and RAG prompt-injection quarantine):
  1. Stage-1 Bi-Encoder Baseline (Dot Product)
  2. Pointwise Cross-Encoder (Isolated scalar s(q, d_i))
  3. dgem Listwise Canvas (Discrete mode argmax 0..3)
  4. dgem Listwise Decision Canvas (Continuous restricted-softmax expectation r_hat_i = Sum_{g=0..3} g * p_{i,g})`,
	Example: `  # Render the saved live Cloud Run L4 telemetry receipt in a semantic table
  dgem bench-rerank --from-receipt benchmarks/results_rerank_cloudrun.json

  # Output machine-readable JSON summary from the Cloud Run L4 receipt
  dgem bench-rerank --from-receipt benchmarks/results_rerank_cloudrun.json --json

  # Run offline mathematical & fixture verification (zero GPU calls)
  dgem bench-rerank --verify-math

  # Run live 12-slot listwise reranking against Serverless Cloud Run L4
  dgem bench-rerank -u "https://dgemma-xxxxxx-uc.a.run.app/v1" --gcp-auth`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if rerankFromReceipt != "" {
			return renderRerankReceipt(rerankFromReceipt, rerankJSONOutput)
		}

		scriptPath := "scripts/eval_rerank_validation.py"
		if _, err := os.Stat(scriptPath); err != nil {
			return fmt.Errorf("cannot locate %s (%v). Hint: run from the repository root or pass --from-receipt benchmarks/results_rerank_cloudrun.json", scriptPath, err)
		}

		pyArgs := []string{scriptPath}
		if rerankVerifyMath {
			pyArgs = append(pyArgs, "--verify-math")
		} else {
			c := GetClient()
			pyArgs = append(pyArgs, "--endpoint", c.BaseURL)
			if rerankReceiptPath != "" {
				pyArgs = append(pyArgs, "--output", rerankReceiptPath)
			}
			if gcpAuth {
				pyArgs = append(pyArgs, "--gcp-auth")
			}
		}

		proc := exec.Command("python3", pyArgs...)
		proc.Stdout = os.Stdout
		proc.Stderr = os.Stderr
		if err := proc.Run(); err != nil {
			return fmt.Errorf("reranking evaluation script failed: %w\nHint: pass --from-receipt benchmarks/results_rerank_cloudrun.json to inspect saved Cloud Run L4 telemetry or --verify-math for offline validation", err)
		}
		return nil
	},
}

func renderRerankReceipt(path string, asJSON bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read rerank receipt %s: %w\nHint: verify benchmarks/results_rerank_cloudrun.json exists", path, err)
	}
	var rec RerankSuiteReceipt
	if err := json.Unmarshal(raw, &rec); err != nil {
		return fmt.Errorf("failed to parse rerank receipt JSON: %w", err)
	}

	if asJSON {
		out, _ := json.MarshalIndent(rec, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Println(ansiAccent("════════════════════════════════════════════════════════════════════════════════════════════════════════════════════"))
	fmt.Printf(" %s · %s\n", ansiAccent(rec.Experiment), rec.ExecutionMode)
	if rec.MeanWallLatencyMs != nil {
		fmt.Printf(" %s %.1f ms (12 simultaneous slots / 10 passages in 1 forward pass = ~%.1f ms/passage)\n",
			ansiMuted("Mean Live Wall Latency:"), *rec.MeanWallLatencyMs, *rec.MeanWallLatencyMs/10.0)
	}
	fmt.Printf(" %s %s (%d queries × %d passages = %d pairs)\n",
		ansiMuted("Dataset:"), rec.DatasetPath, rec.TotalQueries, rec.CandidatesPerQuery, rec.TotalQueryPassagePairs)
	fmt.Println(ansiAccent("════════════════════════════════════════════════════════════════════════════════════════════════════════════════════"))
	fmt.Printf("%-46s │ %7s │ %7s │ %7s │ %8s │ %9s │ %7s │ %8s\n",
		"Reranking Strategy", "nDCG@10", "MRR@10", "MAP@10", "NevIR", "FollowIR", "TieRate", "PoisonQ")
	fmt.Println(ansiMuted("───────────────────────────────────────────────┼─────────┼─────────┼─────────┼──────────┼───────────┼─────────┼─────────"))

	orderedKeys := []string{
		"1_bi_encoder",
		"2_pointwise_rerank",
		"3_dgem_listwise_argmax",
		"4_dgem_listwise_expectation",
	}
	for _, k := range orderedKeys {
		m, ok := rec.SummaryMetrics[k]
		if !ok {
			continue
		}
		label := m.Label
		if len(label) > 46 {
			label = label[:46]
		}
		tieStr := fmt.Sprintf("%6.1f%%", m.ExactTieRatePct)
		if m.ExactTieRatePct > 20.0 {
			tieStr = ansiWarn(tieStr)
		} else if k == "4_dgem_listwise_expectation" {
			tieStr = ansiPass(tieStr)
		}
		ndcgStr := fmt.Sprintf("%7.4f", m.NDCG10)
		if k == "4_dgem_listwise_expectation" {
			ndcgStr = ansiPass(ndcgStr)
		}
		fmt.Printf("%-46s │ %s │ %7.4f │ %7.4f │ %7.1f%% │ %+9.4f │ %s │ %7.1f%%\n",
			label, ndcgStr, m.MRR10, m.MAP10, m.NevIRPairwiseAccPct, m.FollowIRPMRR, tieStr, m.PoisonQuarantinePct)
	}
	fmt.Println(ansiAccent("════════════════════════════════════════════════════════════════════════════════════════════════════════════════════"))
	return nil
}

func init() {
	benchRerankCmd.Flags().StringVarP(&rerankDatasetPath, "dataset", "d", "benchmarks/rerank_suite.jsonl", "Path to the reranking JSONL dataset")
	benchRerankCmd.Flags().StringVarP(&rerankReceiptPath, "output", "o", "benchmarks/results_rerank_cloudrun.json", "Output path for the JSON telemetry receipt")
	benchRerankCmd.Flags().StringVar(&rerankFromReceipt, "from-receipt", "", "Render formatted summary from an existing JSON receipt file without invoking GPU")
	benchRerankCmd.Flags().BoolVar(&rerankVerifyMath, "verify-math", false, "Run offline mathematical & fixture verification (no GPU required)")
	benchRerankCmd.Flags().BoolVar(&rerankJSONOutput, "json", false, "Emit machine-readable JSON when inspecting a receipt")
	RootCmd.AddCommand(benchRerankCmd)
}
