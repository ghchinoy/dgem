package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ghchinoy/dgem/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	serverURL string
	modelName string
	timeout   time.Duration
	showStats bool
	authToken string
	gcpAuth   bool
)

// RootCmd is the base command for dgem.
var RootCmd = &cobra.Command{
	Use:   "dgem",
	Short: "dgem is a CLI assistant and evaluation tool for DiffusionGemma",
	Long: `dgem interacts with local or remote DiffusionGemma inference engines.
It provides high-performance discrete diffusion slot readout decisions (slot-evaluated
in a single forward pass without conversational text overhead, also known as Jev-style),
generative prompt execution, Go template rendering, and comprehensive request/response telemetry.`,
}

// Execute runs the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Inference & Template Commands:"},
		&cobra.Group{ID: "eval", Title: "Benchmark & Calibration Suites:"},
	)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/dgem/config.yaml or .dgem.yaml)")
	RootCmd.PersistentFlags().StringVarP(&serverURL, "url", "u", "http://127.0.0.1:8080/v1", "Base URL for the DiffusionGemma server")
	RootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "diffgemma-26b-a4b-it-q4", "Model ID to target")
	RootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 120*time.Second, "Client timeout")
	RootCmd.PersistentFlags().BoolVarP(&showStats, "stats", "s", false, "Display execution timing, token breakdown, and inference telemetry")
	RootCmd.PersistentFlags().StringVarP(&authToken, "token", "k", "", "Authorization Bearer token / API key")
	RootCmd.PersistentFlags().BoolVar(&gcpAuth, "gcp-auth", false, "Automatically obtain GCP IAM identity token via gcloud auth print-identity-token")

	viper.BindPFlag("url", RootCmd.PersistentFlags().Lookup("url"))
	viper.BindPFlag("model", RootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("timeout", RootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("stats", RootCmd.PersistentFlags().Lookup("stats"))
	viper.BindPFlag("token", RootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("gcp_auth", RootCmd.PersistentFlags().Lookup("gcp-auth"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home + "/.config/dgem")
		}
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".dgem")
	}

	viper.SetEnvPrefix("DGEM")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()
}

// GetClient returns a configured API client using Viper values.
func GetClient() *client.Client {
	baseURL := viper.GetString("url")
	model := viper.GetString("model")
	to := viper.GetDuration("timeout")
	token := viper.GetString("token")

	if token == "" && viper.GetBool("gcp_auth") {
		// Attempt to fetch identity token using gcloud CLI
		out, err := exec.Command("gcloud", "auth", "print-identity-token").Output()
		if err == nil {
			token = strings.TrimSpace(string(out))
		}
	}

	c := client.NewClient(baseURL, model, to)
	if token != "" {
		c.WithAuthToken(token)
	}
	return c
}

// PrintStats prints formatted telemetry from the request and response.
func PrintStats(stats *client.RequestStats) {
	if stats == nil {
		return
	}

	fmt.Println()
	fmt.Println("──────────────────────────────── STATS ────────────────────────────────")
	fmt.Printf("  Model:             %s\n", stats.Model)
	fmt.Printf("  Endpoint:          %s\n", stats.Endpoint)

	fmt.Println("\n  Timing:")
	fmt.Printf("    • Total Wall Time:     %v\n", stats.WallTime.Round(time.Millisecond))
	if stats.PrefillMs > 0 {
		fmt.Printf("    • Server Prefill:      %.0f ms\n", stats.PrefillMs)
	}
	if stats.DenoiseMs > 0 {
		fmt.Printf("    • Server Denoise:      %.0f ms\n", stats.DenoiseMs)
	}

	fmt.Println("\n  Token Breakdown:")
	fmt.Printf("    • Prompt Tokens:       %d tokens\n", stats.PromptTokens)
	if stats.ReusedTokens > 0 {
		hitPct := float64(stats.ReusedTokens) / float64(stats.PromptTokens) * 100
		fmt.Printf("    • KV Cache Reused:     %d tokens (%.1f%% cache hit rate)\n", stats.ReusedTokens, hitPct)
	}
	fmt.Printf("    • Completion Tokens:   %d tokens\n", stats.OutputTokens)
	fmt.Printf("    • Total Tokens:        %d tokens\n", stats.TotalTokens)

	if stats.Diagnostics != nil {
		fmt.Println("\n  Inference Mechanics:")
		if stats.DenoiseSteps > 0 {
			fmt.Printf("    • Denoise Steps:       %d step(s)\n", stats.DenoiseSteps)
		}
		if stats.SamplesN > 0 {
			policy := stats.Diagnostics.Samples.Policy
			fmt.Printf("    • Noise Samples (N):   %d sample(s) (policy: %s, threshold: %.2f)\n",
				stats.SamplesN, policy.Mode, policy.Threshold)
			fmt.Printf("    • Multi-Read Extended: %v (first-read max entropy: %.4f nats)\n",
				stats.Extended, policy.FirstReadMaxEntropy)
		}

		if len(stats.Diagnostics.Questions) > 0 {
			fmt.Println("\n  Question Diagnostics:")
			for qID, diag := range stats.Diagnostics.Questions {
				fmt.Printf("    • %-12s: argmax='%s' (entropy=%.4f nats, label_mass=%.1f%%)\n",
					qID, strings.TrimSpace(diag.ArgmaxToken), diag.Entropy, diag.LabelMass*100)
			}
		}
	}
	fmt.Println("───────────────────────────────────────────────────────────────────────")
}
