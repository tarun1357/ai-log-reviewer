package correlator

import (
	"fmt"
	"sort"

	"ai-log-analyzer/internal/models"
)

// Correlate reads LogEntry items from the input channel, groups them by
// RequestID, and returns a slice of RequestGroups.
func Correlate(entries <-chan models.LogEntry) []models.RequestGroup {
	groups := make(map[string]*models.RequestGroup)

	count := 0
	for entry := range entries {
		count++
		g, ok := groups[entry.RequestID]
		if !ok {
			g = &models.RequestGroup{
				RequestID: entry.RequestID,
				StartTime: entry.Timestamp,
				EndTime:   entry.Timestamp,
			}
			groups[entry.RequestID] = g
		}

		// Deduplicate: check if we already have an entry with the exact same Raw string
		isDuplicate := false
		for _, existing := range g.Entries {
			if existing.Raw == entry.Raw {
				isDuplicate = true
				break
			}
		}

		if isDuplicate {
			continue // skip adding this entry
		}

		g.Entries = append(g.Entries, entry)

		if entry.Timestamp.Before(g.StartTime) {
			g.StartTime = entry.Timestamp
		}
		if entry.Timestamp.After(g.EndTime) {
			g.EndTime = entry.Timestamp
		}
	}

	result := make([]models.RequestGroup, 0, len(groups))
	for _, g := range groups {
		// Sort entries within each group by timestamp.
		sort.Slice(g.Entries, func(i, j int) bool {
			return g.Entries[i].Timestamp.Before(g.Entries[j].Timestamp)
		})
		result = append(result, *g)
	}

	// Sort groups by start time.
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime.Before(result[j].StartTime)
	})

	fmt.Printf("🔗 Correlator: grouped %d log entries into %d request groups\n", count, len(result))
	return result
}
