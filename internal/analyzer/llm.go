package analyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ai-log-analyzer/internal/models"
)

// openAIRequest / openAIResponse model the Chat Completions API.
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float32         `json:"temperature"`
}
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Analyze sends each anomaly report to the LLM and returns a Diagnosis.
// If apiKey is empty, it returns a fallback diagnosis without calling the LLM.
func Analyze(reports []models.AnomalyReport, apiKey, model string) []models.Diagnosis {
	if apiKey == "" {
		fmt.Println("🤖 LLM Analyzer: no API key provided — generating rule-based diagnoses")
		return fallbackAnalyze(reports)
	}

	fmt.Printf("🤖 LLM Analyzer: sending %d anomaly reports to %s\n", len(reports), model)

	var diagnoses []models.Diagnosis
	for i, report := range reports {
		fmt.Printf("   ↳ Analyzing report %d/%d (request %s)...\n", i+1, len(reports), report.Group.RequestID)
		d, err := callLLM(report, apiKey, model)
		if err != nil {
			fmt.Printf("   ⚠️  LLM call failed: %v — using fallback\n", err)
			d = fallbackDiagnosis(report)
		}
		diagnoses = append(diagnoses, d)
	}
	return diagnoses
}

func callLLM(report models.AnomalyReport, apiKey, model string) (models.Diagnosis, error) {
	prompt := buildPrompt(report)

	reqBody := openAIRequest{
		Model: model,
		Messages: []openAIMessage{
			{Role: "system", Content: "You are an expert distributed systems engineer performing root cause analysis on microservice logs. Analyze the logs and deduce which service caused the fault. Respond ONLY with valid JSON in this format: {\"root_cause\": \"...\", \"recommendation\": \"...\", \"severity\": \"critical|warning|info\"}. Do not wrap the JSON in markdown code blocks or add introductory text."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return models.Diagnosis{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://integrate.api.nvidia.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return models.Diagnosis{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.Diagnosis{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Diagnosis{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return models.Diagnosis{}, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return models.Diagnosis{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(oaiResp.Choices) == 0 {
		return models.Diagnosis{}, fmt.Errorf("no choices in response")
	}

	// Parse the JSON from the LLM response.
	content := oaiResp.Choices[0].Message.Content
	// Strip markdown fences if present.
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// In case the model ignored directions and prepended/appended text, attempt regex match.
	// Prevented possible crash if model adds extra text
	re := regexp.MustCompile(`(?s)\{.*\}`)
	if match := re.FindString(content); match != "" {
		content = match
	}

	var d models.Diagnosis
	if err := json.Unmarshal([]byte(content), &d); err != nil {
		// Fallback parse approach if still broken
		return models.Diagnosis{}, fmt.Errorf("parse LLM JSON: %w (raw: %s)", err, content)
	}
	d.RequestID = report.Group.RequestID
	return d, nil
}

func buildPrompt(report models.AnomalyReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Analyze the following anomalous request flow (request_id: %s).\n\n", report.Group.RequestID))
	sb.WriteString(fmt.Sprintf("Triggered anomaly rules: %s\n\n", strings.Join(report.Rules, ", ")))
	sb.WriteString("Log entries (chronological):\n")

	entries := report.Group.Entries
	if len(entries) > 50 {
		// Token limit handling - Keep first 20 and last 30 logs (the critical error sequence context)
		for _, e := range entries[:20] {
			sb.WriteString(fmt.Sprintf("  [%s] %s | %s | %s | latency=%dms\n",
				e.Timestamp.Format(time.RFC3339), e.Service, e.Level, e.Message, e.Latency.Milliseconds()))
		}

		sb.WriteString(fmt.Sprintf("\n  ... [%d logs omitted to fit token limits] ...\n\n", len(entries)-50))

		for _, e := range entries[len(entries)-30:] {
			sb.WriteString(fmt.Sprintf("  [%s] %s | %s | %s | latency=%dms\n",
				e.Timestamp.Format(time.RFC3339), e.Service, e.Level, e.Message, e.Latency.Milliseconds()))
		}
	} else {
		for _, e := range entries {
			sb.WriteString(fmt.Sprintf("  [%s] %s | %s | %s | latency=%dms\n",
				e.Timestamp.Format(time.RFC3339), e.Service, e.Level, e.Message, e.Latency.Milliseconds()))
		}
	}

	sb.WriteString("\nProvide root cause analysis, a recommended fix, and severity level.")
	return sb.String()
}

// fallbackAnalyze produces rule-based diagnoses when no LLM is available.
func fallbackAnalyze(reports []models.AnomalyReport) []models.Diagnosis {
	diagnoses := make([]models.Diagnosis, 0, len(reports))
	for _, r := range reports {
		diagnoses = append(diagnoses, fallbackDiagnosis(r))
	}
	return diagnoses
}

func fallbackDiagnosis(r models.AnomalyReport) models.Diagnosis {
	severity := "warning"
	for _, rule := range r.Rules {
		if strings.Contains(rule, "error") || strings.Contains(rule, "timeout") {
			severity = "critical"
			break
		}
	}

	return models.Diagnosis{
		RequestID:      r.Group.RequestID,
		RootCause:      fmt.Sprintf("Rule-based detection: %s", strings.Join(r.Rules, ", ")),
		Recommendation: "Investigate the flagged services. Consider adding circuit breakers and retries with backoff.",
		Severity:       severity,
	}
}
