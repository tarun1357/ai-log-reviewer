package models

import "time"

// LogEntry represents a single parsed log line from a microservice.
type LogEntry struct {
	Timestamp time.Time     `json:"timestamp"`
	Service   string        `json:"service"`
	Level     string        `json:"level"` // INFO, WARN, ERROR
	RequestID string        `json:"request_id"`
	Message   string        `json:"message"`
	Latency   time.Duration `json:"latency_ms"`
	Raw       string        `json:"-"`
}

// RequestGroup holds all log entries that share the same request ID.
type RequestGroup struct {
	RequestID string     `json:"request_id"`
	Entries   []LogEntry `json:"entries"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
}

// AnomalyReport is produced when a RequestGroup triggers one or more anomaly rules.
type AnomalyReport struct {
	Group   RequestGroup `json:"group"`
	Rules   []string     `json:"triggered_rules"`
	Summary string       `json:"summary"`
}

// Diagnosis is the output of the LLM root-cause analysis.
type Diagnosis struct {
	RequestID      string `json:"request_id"`
	RootCause      string `json:"root_cause"`
	Recommendation string `json:"recommendation"`
	Severity       string `json:"severity"` // critical, warning, info
}

// Config holds all runtime configuration.
type Config struct {
	LogDir            string        `yaml:"log_dir"`
	Services          []string      `yaml:"services"`
	SimDuration       time.Duration `yaml:"sim_duration"`
	SimRatePerSec     int           `yaml:"sim_rate_per_sec"`
	CorrelationWindow time.Duration `yaml:"correlation_window"`
	LatencyThreshold  time.Duration `yaml:"latency_threshold"`
	NvidiaKey         string        `yaml:"nvidia_key"`
	NvidiaModel       string        `yaml:"nvidia_model"`
	AlertFile         string        `yaml:"alert_file"`
	WebhookURL        string        `yaml:"webhook_url"`
}
