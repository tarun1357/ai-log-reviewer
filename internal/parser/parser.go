package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-log-analyzer/internal/models"
)

// Parse converts a raw log line into a structured LogEntry.
// Expected format:
//
//	2026-03-08T00:01:23.123Z | order-service | INFO | req-abc-123 | Order created | latency=45ms
func Parse(raw string) (models.LogEntry, error) {
	parts := strings.SplitN(raw, " | ", 6)
	if len(parts) < 6 {
		return models.LogEntry{}, fmt.Errorf("expected 6 pipe-delimited fields, got %d", len(parts))
	}

	ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(parts[0]))
	if err != nil {
		return models.LogEntry{}, fmt.Errorf("parse timestamp: %w", err)
	}

	service := strings.TrimSpace(parts[1])
	if service == "" {
		return models.LogEntry{}, fmt.Errorf("missing service name")
	}

	level := strings.TrimSpace(parts[2])
	if level != "INFO" && level != "WARN" && level != "ERROR" && level != "DEBUG" {
		return models.LogEntry{}, fmt.Errorf("invalid log level: %s", level)
	}

	requestID := strings.TrimSpace(parts[3])
	if requestID == "" {
		return models.LogEntry{}, fmt.Errorf("missing request ID")
	}

	message := strings.TrimSpace(parts[4])
	if message == "" {
		return models.LogEntry{}, fmt.Errorf("missing message")
	}

	latency := parseLatency(parts[5])

	return models.LogEntry{
		Timestamp: ts,
		Service:   service,
		Level:     level,
		RequestID: requestID,
		Message:   message,
		Latency:   latency,
		Raw:       raw,
	}, nil

	// Old : possible code break
	// return models.LogEntry{
	// 	Service:   strings.TrimSpace(parts[1]),
	// 	Level:     strings.TrimSpace(parts[2]),
	// 	RequestID: strings.TrimSpace(parts[3]),
	// 	Message:   strings.TrimSpace(parts[4]),
	// 	Latency:   latency,
	// 	Raw:       raw,
	// }, nil
}

// ParseStream reads raw lines from a channel, parses each one, and sends
// the structured entries to the returned channel.
func ParseStream(lines <-chan string) <-chan models.LogEntry {
	entries := make(chan models.LogEntry, 500)
	go func() {
		defer close(entries)
		errCount := 0
		total := 0
		for line := range lines {
			total++
			entry, err := Parse(line)
			if err != nil {
				errCount++
				if errCount <= 5 {
					fmt.Printf("⚠️  Parser: skipping malformed line: %v\n", err)
				}
				continue
			}
			entries <- entry
		}
		fmt.Printf("📝 Parser: processed %d lines (%d errors)\n", total, errCount)
	}()
	return entries
}

func parseLatency(s string) time.Duration {
	s = strings.TrimSpace(s)
	// latency=45ms
	s = strings.TrimPrefix(s, "latency=")
	s = strings.TrimSuffix(s, "ms")
	ms, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
