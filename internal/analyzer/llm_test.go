package analyzer

import (
	"strings"
	"testing"
	"time"

	"ai-log-analyzer/internal/models"
)

func TestBuildPromptTokenLimitTrimming(t *testing.T) {
	// Create an anomaly report with 60 entries (over the 50 limit)
	report := models.AnomalyReport{
		Group: models.RequestGroup{
			RequestID: "req-huge",
		},
		Rules: []string{"error_avalanche"},
	}

	for i := 0; i < 60; i++ {
		report.Group.Entries = append(report.Group.Entries, models.LogEntry{
			Timestamp: time.Now(),
			Service:   "test-svc",
			Level:     "INFO",
			Message:   "Test log",
			Latency:   10 * time.Millisecond,
		})
	}

	prompt := buildPrompt(report)

	if !strings.Contains(prompt, "10 logs omitted to fit token limits") {
		t.Errorf("Expected omission message for 10 omitted logs, but prompt did not contain it")
	}

	// Calculate rough structure: intro + 20 logs + omission line + 30 logs + footer
	// Count occurrence of "test-svc | INFO | Test log"
	count := strings.Count(prompt, "test-svc | INFO | Test log")
	if count != 50 {
		t.Errorf("Expected exactly 50 instances of the log string outputted due to trimming, got %d", count)
	}
}

func TestBuildPromptNoTrimmingUnderLimit(t *testing.T) {
	report := models.AnomalyReport{
		Group: models.RequestGroup{
			RequestID: "req-small",
		},
		Rules: []string{"slow_request"},
	}

	for i := 0; i < 40; i++ {
		report.Group.Entries = append(report.Group.Entries, models.LogEntry{
			Timestamp: time.Now(),
			Service:   "test-svc",
			Level:     "WARN",
			Message:   "Test warning",
			Latency:   100 * time.Millisecond,
		})
	}

	prompt := buildPrompt(report)

	if strings.Contains(prompt, "logs omitted to fit token limits") {
		t.Errorf("Did not expect omission message for under 50 logs")
	}

	count := strings.Count(prompt, "test-svc | WARN | Test warning")
	if count != 40 {
		t.Errorf("Expected exactly 40 instances of the log string outputted, got %d", count)
	}
}
