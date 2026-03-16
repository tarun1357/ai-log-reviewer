package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-log-analyzer/internal/models"
)

// Alert dispatches diagnoses to all configured sinks.
func Alert(diagnoses []models.Diagnosis, alertFile, webhookURL string) {
	printDashboard(diagnoses)

	for _, d := range diagnoses {
		printConsole(d)
	}

	fmt.Printf("\n🔔 Alerter: dispatching %d alerts to external sinks...\n\n", len(diagnoses))

	if alertFile != "" {
		writeJSONFile(diagnoses, alertFile)
	}

	if webhookURL != "" {
		postWebhook(diagnoses, webhookURL)
	}
}

// ANSI Color constants
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorBgRed  = "\033[41;37m"
)

func printDashboard(diagnoses []models.Diagnosis) {
	total := len(diagnoses)
	if total == 0 {
		return
	}

	critical := 0
	warning := 0
	info := 0
	for _, d := range diagnoses {
		switch d.Severity {
		case "critical":
			critical++
		case "warning":
			warning++
		case "info":
			info++
		}
	}

	fmt.Printf("\n%s╭─────────────────────────────────────────────────────────────╮%s\n", colorCyan, colorReset)
	fmt.Printf("%s│                 🚨 INCIDENT DASHBOARD                       │%s\n", colorCyan, colorReset)
	fmt.Printf("%s├─────────────────────────────────────────────────────────────┤%s\n", colorCyan, colorReset)
	fmt.Printf("%s│ Total Incidents Detected: %-33d │%s\n", colorCyan, total, colorReset)
	fmt.Printf("%s│ %s  Critical: %-42d %s│%s\n", colorCyan, colorRed, critical, colorCyan, colorReset)
	fmt.Printf("%s│ %s  Warning : %-42d %s│%s\n", colorCyan, colorYellow, warning, colorCyan, colorReset)
	fmt.Printf("%s│   Info    : %-42d │%s\n", colorCyan, info, colorReset)
	fmt.Printf("%s╰─────────────────────────────────────────────────────────────╯%s\n\n", colorCyan, colorReset)
}

func printConsole(d models.Diagnosis) {
	if d.RequestID == "GLOBAL_SYSTEM_SPIKE" {
		fmt.Printf("%s🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑%s\n", colorBgRed, colorReset)
		fmt.Printf("%s🔥 GLOBAL SYSTEM SPIKE DETECTED [%s]%s\n", colorBold, strings.ToUpper(d.Severity), colorReset)
		fmt.Printf("   Issue      :%s %s%s\n", colorRed, d.RootCause, colorReset)
		fmt.Printf("   Action     :%s %s%s\n", colorBold, d.Recommendation, colorReset)
		fmt.Printf("%s🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑🛑%s\n", colorBgRed, colorReset)
		fmt.Println()
		return
	}

	icon := "ℹ️ "
	colorTheme := colorReset

	switch d.Severity {
	case "critical":
		icon = "🔴"
		colorTheme = colorRed
	case "warning":
		icon = "🟡"
		colorTheme = colorYellow
	}

	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorTheme, colorReset)
	fmt.Printf("%s%s  ALERT [%s]%s\n", colorBold, icon, strings.ToUpper(d.Severity), colorReset)
	fmt.Printf("%s   Request ID : %s%s\n", colorCyan, d.RequestID, colorReset)
	fmt.Printf("   Root Cause : %s%s%s\n", colorTheme, d.RootCause, colorReset)
	fmt.Printf("   Recommend  : %s%s%s\n", colorBold, d.Recommendation, colorReset)
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorTheme, colorReset)
	fmt.Println()
}

func writeJSONFile(diagnoses []models.Diagnosis, path string) {
	data, err := json.MarshalIndent(diagnoses, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Alerter: marshal JSON: %v\n", err)
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Alerter: write file: %v\n", err)
		return
	}
	fmt.Printf("💾 Alerter: alerts written to %s\n", path)
}

func postWebhook(diagnoses []models.Diagnosis, url string) {
	data, err := json.Marshal(diagnoses)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Alerter: marshal webhook payload: %v\n", err)
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Alerter: webhook POST failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("📡 Alerter: webhook POST to %s → %d\n", url, resp.StatusCode)
}
