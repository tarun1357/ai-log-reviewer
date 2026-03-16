package parser

import (
	"strings"
	"testing"
	"time"
)

func TestParseValidLog(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z | order-service   | INFO  | req-abc-123 | Order created | latency=45ms"
	entry, err := Parse(raw)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if entry.Service != "order-service" {
		t.Errorf("expected service 'order-service', got '%s'", entry.Service)
	}
	if entry.Level != "INFO" {
		t.Errorf("expected level 'INFO', got '%s'", entry.Level)
	}
	if entry.RequestID != "req-abc-123" {
		t.Errorf("expected requestID 'req-abc-123', got '%s'", entry.RequestID)
	}
	if entry.Message != "Order created" {
		t.Errorf("expected message 'Order created', got '%s'", entry.Message)
	}
	if entry.Latency != 45*time.Millisecond {
		t.Errorf("expected latency 45ms, got %v", entry.Latency)
	}
}

func TestParseInvalidFormat(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z | order-service | INFO"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for malformed short line, got nil")
	}
	if !strings.Contains(err.Error(), "expected 6 pipe-delimited fields") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseInvalidTimestamp(t *testing.T) {
	raw := "bad-timestamp | order-service | INFO | req-abc-123 | Order created | latency=45ms"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for invalid timestamp, got nil")
	}
	if !strings.Contains(err.Error(), "parse timestamp") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseMissingService(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z |    | INFO | req-abc-123 | Order created | latency=45ms"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for missing service, got nil")
	}
	if !strings.Contains(err.Error(), "missing service name") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseInvalidLogLevel(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z | order-service | FOOBAR | req-abc-123 | Order created | latency=45ms"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log level: FOOBAR") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseMissingRequestID(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z | order-service | INFO |   | Order created | latency=45ms"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for missing request ID, got nil")
	}
	if !strings.Contains(err.Error(), "missing request ID") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseMissingMessage(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z | order-service | INFO | req-abc-123 |   | latency=45ms"
	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for missing message, got nil")
	}
	if !strings.Contains(err.Error(), "missing message") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseMissingLatency(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z | order-service | INFO | req-abc-123 | Order created |  "
	entry, err := Parse(raw)
	if err != nil {
		t.Fatalf("expected nil error for missing latency, got %v", err)
	}
	if entry.Latency != 0 {
		t.Errorf("expected 0 latency, got %v", entry.Latency)
	}
}

func TestParseInvalidLatencyFormat(t *testing.T) {
	raw := "2026-03-08T00:01:23.123Z | order-service | INFO | req-abc-123 | Order created | latency=abc"
	entry, err := Parse(raw)
	if err != nil {
		t.Fatalf("expected nil error for invalid latency format, got %v", err)
	}
	if entry.Latency != 0 {
		t.Errorf("expected 0 latency, got %v", entry.Latency)
	}
}
