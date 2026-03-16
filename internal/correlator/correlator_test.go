package correlator

import (
	"testing"
	"time"

	"ai-log-analyzer/internal/models"
)

func TestCorrelateGroupsDuplicateFiltering(t *testing.T) {
	// Simulate input channel
	entriesChan := make(chan models.LogEntry, 5)

	t1 := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 13, 10, 0, 1, 0, time.UTC)

	// Add 3 log entries: e1, e2, and a duplicate of e1
	e1 := models.LogEntry{RequestID: "req-1", Timestamp: t1, Message: "m1", Raw: "raw1"}
	e2 := models.LogEntry{RequestID: "req-1", Timestamp: t2, Message: "m2", Raw: "raw2"}
	e3 := models.LogEntry{RequestID: "req-1", Timestamp: t1, Message: "m1", Raw: "raw1"}

	entriesChan <- e1
	entriesChan <- e2
	entriesChan <- e3
	close(entriesChan)

	groups := Correlate(entriesChan)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	g := groups[0]
	if g.RequestID != "req-1" {
		t.Errorf("expected request ID req-1, got %s", g.RequestID)
	}
	if len(g.Entries) != 2 {
		t.Fatalf("expected 2 entries after deduplication, got %d", len(g.Entries))
	}
	if g.Entries[0].Message != "m1" {
		t.Errorf("expected m1 to be the first entry, got %s", g.Entries[0].Message)
	}
	if g.Entries[1].Message != "m2" {
		t.Errorf("expected m2 to be the second entry, got %s", g.Entries[1].Message)
	}
}

func TestCorrelateGroupsTimestampSorting(t *testing.T) {
	entriesChan := make(chan models.LogEntry, 3)

	t1 := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 13, 10, 0, 1, 0, time.UTC)
	t3 := time.Date(2026, 3, 13, 10, 0, 2, 0, time.UTC)

	// Add entries out of order
	e3 := models.LogEntry{RequestID: "req-1", Timestamp: t3, Message: "m3", Raw: "raw3"}
	e1 := models.LogEntry{RequestID: "req-1", Timestamp: t1, Message: "m1", Raw: "raw1"}
	e2 := models.LogEntry{RequestID: "req-1", Timestamp: t2, Message: "m2", Raw: "raw2"}

	entriesChan <- e3
	entriesChan <- e1
	entriesChan <- e2
	close(entriesChan)

	groups := Correlate(entriesChan)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	g := groups[0]
	if len(g.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(g.Entries))
	}
	if g.Entries[0].Message != "m1" {
		t.Errorf("expected m1, got %s", g.Entries[0].Message)
	}
	if g.Entries[2].Message != "m3" {
		t.Errorf("expected m3, got %s", g.Entries[2].Message)
	}
}

func TestCorrelateMultipleGroups(t *testing.T) {
	entriesChan := make(chan models.LogEntry, 3)

	t1 := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 13, 10, 0, 1, 0, time.UTC)

	e1 := models.LogEntry{RequestID: "req-1", Timestamp: t2, Message: "m1", Raw: "raw1"}
	e2 := models.LogEntry{RequestID: "req-2", Timestamp: t1, Message: "m2", Raw: "raw2"}

	entriesChan <- e1
	entriesChan <- e2
	close(entriesChan)

	groups := Correlate(entriesChan)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// Groups should be sorted by overall StartTime
	if groups[0].RequestID != "req-2" {
		t.Errorf("expected first group to be req-2 (earlier time), got %s", groups[0].RequestID)
	}
	if groups[1].RequestID != "req-1" {
		t.Errorf("expected second group to be req-1, got %s", groups[1].RequestID)
	}
}
