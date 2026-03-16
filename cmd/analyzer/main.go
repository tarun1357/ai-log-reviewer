package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"ai-log-analyzer/internal/alerter"
	"ai-log-analyzer/internal/analyzer"
	"ai-log-analyzer/internal/anomaly"
	"ai-log-analyzer/internal/collector"
	"ai-log-analyzer/internal/correlator"
	"ai-log-analyzer/internal/models"
	"ai-log-analyzer/internal/parser"
	"ai-log-analyzer/internal/simulator"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         AI-Powered Log Analyzer for Distributed Systems     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	cfg, err := loadConfig("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ── Step 1: Simulate microservice logs ────────────────────────
	fmt.Println("── Step 1: Generating simulated logs ──────────────────────────")
	if err := simulator.Run(cfg.LogDir, cfg.Services, cfg.SimDuration, cfg.SimRatePerSec); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Simulator failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()

	// ── Step 2: Collect log files ─────────────────────────────────
	fmt.Println("── Step 2: Collecting log files ───────────────────────────────")
	lines, err := collector.Collect(cfg.LogDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Collector failed: %v\n", err)
		os.Exit(1)
	}

	// ── Step 3: Parse log lines ───────────────────────────────────
	fmt.Println("── Step 3: Parsing log lines ──────────────────────────────────")
	entries := parser.ParseStream(lines)

	// ── Step 4: Correlate by request ID ───────────────────────────
	fmt.Println("── Step 4: Correlating by request ID ──────────────────────────")
	groups := correlator.Correlate(entries)

	// ── Step 5: Detect anomalies ──────────────────────────────────
	fmt.Println("── Step 5: Detecting anomalies ────────────────────────────────")
	reports := anomaly.Detect(groups, cfg.LatencyThreshold)
	if len(reports) == 0 {
		fmt.Println("✅ No anomalies detected. All request flows look healthy!")
		return
	}

	// ── Step 6: LLM root-cause analysis ───────────────────────────
	fmt.Println("── Step 6: Running root-cause analysis ────────────────────────")
	diagnoses := analyzer.Analyze(reports, cfg.NvidiaKey, cfg.NvidiaModel)

	// Evaluate global spikes independent of specific Request ID chains
	globalDiagnoses := anomaly.DetectGlobalErrorSpike(groups)
	diagnoses = append(globalDiagnoses, diagnoses...) // Prepend global spikes so they appear first

	// ── Step 7: Generate alerts ───────────────────────────────────
	fmt.Println("── Step 7: Generating alerts ──────────────────────────────────")
	alerter.Alert(diagnoses, cfg.AlertFile, cfg.WebhookURL)

	fmt.Println("\n✅ Analysis complete!")
}

func loadConfig(path string) (models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.Config{}, fmt.Errorf("read config: %w", err)
	}

	// Parse raw YAML to handle duration strings manually.
	var raw struct {
		LogDir            string   `yaml:"log_dir"`
		Services          []string `yaml:"services"`
		SimDuration       string   `yaml:"sim_duration"`
		SimRatePerSec     int      `yaml:"sim_rate_per_sec"`
		CorrelationWindow string   `yaml:"correlation_window"`
		LatencyThreshold  string   `yaml:"latency_threshold"`
		NvidiaKey         string   `yaml:"nvidia_key"`
		NvidiaModel       string   `yaml:"nvidia_model"`
		AlertFile         string   `yaml:"alert_file"`
		WebhookURL        string   `yaml:"webhook_url"`
	}

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return models.Config{}, fmt.Errorf("parse config: %w", err)
	}

	simDur, _ := time.ParseDuration(raw.SimDuration)
	if simDur == 0 {
		simDur = 15 * time.Second
	}
	corrWin, _ := time.ParseDuration(raw.CorrelationWindow)
	if corrWin == 0 {
		corrWin = 10 * time.Second
	}
	latThresh, _ := time.ParseDuration(raw.LatencyThreshold)
	if latThresh == 0 {
		latThresh = 500 * time.Millisecond
	}
	rps := raw.SimRatePerSec
	if rps == 0 {
		rps = 5
	}
	model := raw.NvidiaModel
	if model == "" {
		model = "meta/llama3-70b-instruct"
	}

	// Allow override from environment variable.
	apiKey := raw.NvidiaKey
	if envKey := os.Getenv("NVIDIA_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	return models.Config{
		LogDir:            raw.LogDir,
		Services:          raw.Services,
		SimDuration:       simDur,
		SimRatePerSec:     rps,
		CorrelationWindow: corrWin,
		LatencyThreshold:  latThresh,
		NvidiaKey:         apiKey,
		NvidiaModel:       model,
		AlertFile:         raw.AlertFile,
		WebhookURL:        raw.WebhookURL,
	}, nil
}
