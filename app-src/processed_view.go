package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const processedUnknownDateKey = "unknown"

type processedViewRow struct {
	Header       bool
	GroupKey     string
	Label        string
	HistoryIndex int
}

func journalIDForProcessedTime(at time.Time) string {
	return fmt.Sprintf("%d-000001", at.UTC().UnixNano())
}

func processedRecordTime(rec MoveRecord) time.Time {
	id := strings.TrimSpace(rec.JournalID)
	if id == "" {
		return time.Time{}
	}
	part, _, ok := strings.Cut(id, "-")
	if !ok || part == "" {
		return time.Time{}
	}
	ns, err := strconv.ParseInt(part, 10, 64)
	if err != nil || ns <= 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

func processedDateKey(rec MoveRecord, now time.Time) string {
	at := processedRecordTime(rec)
	if at.IsZero() {
		return processedUnknownDateKey
	}
	loc := now.Location()
	if loc == nil {
		loc = time.Local
	}
	return at.In(loc).Format("2006-01-02")
}

func processedDateLabel(key string, now time.Time) string {
	if key == processedUnknownDateKey || strings.TrimSpace(key) == "" {
		return "Earlier / unknown date"
	}
	at, err := time.ParseInLocation("2006-01-02", key, now.Location())
	if err != nil {
		return key
	}
	return at.Format("02 Jan 2006")
}

func ensureProcessedGroupState(history []MoveRecord, collapsed map[string]bool) {
	if collapsed == nil {
		return
	}
	now := time.Now()
	for _, rec := range history {
		key := processedDateKey(rec, now)
		if _, exists := collapsed[key]; !exists {
			collapsed[key] = true
		}
	}
}

func buildProcessedView(history []MoveRecord, query string, collapsed map[string]bool, now time.Time) []processedViewRow {
	type group struct {
		key     string
		label   string
		at      time.Time
		indices []int
	}
	query = strings.ToLower(strings.TrimSpace(query))
	groups := map[string]*group{}
	for i := len(history) - 1; i >= 0; i-- {
		rec := history[i]
		key := processedDateKey(rec, now)
		label := processedDateLabel(key, now)
		if query != "" {
			haystack := strings.ToLower(strings.Join([]string{
				processedRecordLabel(rec), rec.From, rec.To, label,
			}, " "))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		g := groups[key]
		if g == nil {
			g = &group{key: key, label: label}
			if key != processedUnknownDateKey {
				g.at, _ = time.ParseInLocation("2006-01-02", key, now.Location())
			}
			groups[key] = g
		}
		g.indices = append(g.indices, i)
	}
	ordered := make([]*group, 0, len(groups))
	for _, g := range groups {
		ordered = append(ordered, g)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].key == processedUnknownDateKey {
			return false
		}
		if ordered[j].key == processedUnknownDateKey {
			return true
		}
		return ordered[i].at.After(ordered[j].at)
	})
	rows := make([]processedViewRow, 0, len(history)+len(ordered))
	for _, g := range ordered {
		isCollapsed := query == "" && collapsed != nil && collapsed[g.key]
		marker := "▼"
		if isCollapsed {
			marker = "▶"
		}
		rows = append(rows, processedViewRow{
			Header:       true,
			GroupKey:     g.key,
			Label:        fmt.Sprintf("%s %s (%d)", marker, g.label, len(g.indices)),
			HistoryIndex: -1,
		})
		if isCollapsed {
			continue
		}
		for _, idx := range g.indices {
			rows = append(rows, processedViewRow{
				GroupKey:     g.key,
				Label:        "    " + processedRecordLabel(history[idx]),
				HistoryIndex: idx,
			})
		}
	}
	return rows
}
