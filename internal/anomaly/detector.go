package anomaly

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ai-log-analyzer/internal/models"
	"ai-log-analyzer/internal/simulator"
)

// Detect scans each RequestGroup for anomalies and returns reports for
// groups that triggered at least one rule.
func Detect(groups []models.RequestGroup, latencyThreshold time.Duration) []models.AnomalyReport {
	var reports []models.AnomalyReport

	for _, g := range groups {
		rules := runRules(g, latencyThreshold)
		if len(rules) > 0 {
			reports = append(reports, models.AnomalyReport{
				Group:   g,
				Rules:   rules,
				Summary: buildSummary(g, rules),
			})
		}
	}

	fmt.Printf("🚨 Anomaly Detector: found %d anomalous request groups out of %d total\n", len(reports), len(groups))
	return reports
}

func runRules(g models.RequestGroup, latencyThreshold time.Duration) []string {
	var triggered []string

	// Rule 1: Error spike — any ERROR-level entry.
	for _, e := range g.Entries {
		if e.Level == "ERROR" {
			triggered = append(triggered, "error_detected")
			break
		}
	}

	// Rule 2: High latency — any entry exceeds threshold.
	for _, e := range g.Entries {
		if e.Latency > latencyThreshold {
			triggered = append(triggered, fmt.Sprintf("high_latency(%s in %s)", e.Latency, e.Service))
			break
		}
	}

	// Rule 3: Missing hops — expected service chain is incomplete.
	seen := make(map[string]bool)
	for _, e := range g.Entries {
		seen[e.Service] = true
	}
	for _, svc := range simulator.ServiceChain {
		if !seen[svc] {
			triggered = append(triggered, fmt.Sprintf("missing_hop(%s)", svc))
		}
	}

	// Rule 4: Timeout pattern — message contains timeout keywords.
	for _, e := range g.Entries {
		lower := strings.ToLower(e.Message)
		if strings.Contains(lower, "timeout") || strings.Contains(lower, "context deadline exceeded") {
			triggered = append(triggered, "timeout_pattern")
			break
		}
	}

	return triggered
}

func buildSummary(g models.RequestGroup, rules []string) string {
	services := make(map[string]bool)
	for _, e := range g.Entries {
		services[e.Service] = true
	}
	svcList := make([]string, 0, len(services))
	for s := range services {
		svcList = append(svcList, s)
	}

	return fmt.Sprintf("Request %s touched %d services [%s], triggered rules: %s",
		g.RequestID, len(svcList), strings.Join(svcList, ", "), strings.Join(rules, ", "))
}

// DetectGlobalErrorSpike evaluates all RequestGroups globally to find sliding windows
// where the exact number of logged ERRORs exceeds the threshold within a given span.
func DetectGlobalErrorSpike(groups []models.RequestGroup) []models.Diagnosis {
	var globalDiagnoses []models.Diagnosis

	// Extract all ERROR entries into a flat chronological list.
	var allErrors []models.LogEntry
	for _, g := range groups {
		for _, e := range g.Entries {
			if e.Level == "ERROR" {
				allErrors = append(allErrors, e)
			}
		}
	}

	if len(allErrors) == 0 {
		return globalDiagnoses
	}

	sort.Slice(allErrors, func(i, j int) bool {
		return allErrors[i].Timestamp.Before(allErrors[j].Timestamp)
	})

	const threshold = 10
	const windowSize = 1 * time.Minute

	// Detect if ANY sliding 1-minute window has >10 errors
	// (Check each error as the potential 'start' of a 1-minute window)
	spikeDetected := false
	var spikeStart time.Time

	for i := 0; i < len(allErrors); i++ {
		start := allErrors[i].Timestamp
		end := start.Add(windowSize)

		count := 0
		for j := i; j < len(allErrors); j++ {
			if allErrors[j].Timestamp.Before(end) {
				count++
			} else {
				break
			}
		}

		if count > threshold {
			spikeDetected = true
			spikeStart = start
			break // Only need to detect one global spike per run for now
		}
	}

	if spikeDetected {
		globalDiagnoses = append(globalDiagnoses, models.Diagnosis{
			RequestID:      "GLOBAL_SYSTEM_SPIKE",
			RootCause:      fmt.Sprintf("Catastrophic failure detected: > %d errors occurred globally within 1 minute starting at %s.", threshold, spikeStart.Format(time.RFC3339)),
			Recommendation: "Immediate SEV-1. Check infrastructure health, recent deployments, or database connectivity immediately.",
			Severity:       "critical",
		})
	}

	return globalDiagnoses
}
