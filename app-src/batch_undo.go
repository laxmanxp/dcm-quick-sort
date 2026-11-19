package main

import (
	"errors"
	"fmt"
	"strings"
)

func latestUndoableBatchID(history []MoveRecord) string {
	for i := len(history) - 1; i >= 0; i-- {
		if id := strings.TrimSpace(history[i].BatchID); id != "" {
			return id
		}
	}
	return ""
}

func batchHistoryIndexes(history []MoveRecord, batchID string) []int {
	batchID = strings.TrimSpace(batchID)
	if batchID == "" {
		return nil
	}
	indexes := make([]int, 0)
	for i, rec := range history {
		if strings.TrimSpace(rec.BatchID) == batchID {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func undoHistoryBatch(history []MoveRecord, batchID string) ([]MoveRecord, []MoveRecord, error) {
	indexes := batchHistoryIndexes(history, batchID)
	if len(indexes) == 0 {
		return history, nil, errors.New("there is no completed batch available to undo")
	}
	for _, idx := range indexes {
		if err := validateUndoMove(history[idx]); err != nil {
			return history, nil, fmt.Errorf("batch undo preflight failed for %s: %w", processedRecordLabel(history[idx]), err)
		}
	}

	removed := make(map[int]bool, len(indexes))
	undone := make([]MoveRecord, 0, len(indexes))
	for i := len(indexes) - 1; i >= 0; i-- {
		idx := indexes[i]
		rec := history[idx]
		if err := undoMove(rec); err != nil {
			remaining := make([]MoveRecord, 0, len(history)-len(removed))
			for j, candidate := range history {
				if !removed[j] {
					remaining = append(remaining, candidate)
				}
			}
			return remaining, undone, fmt.Errorf("batch undo stopped at %s: %w", processedRecordLabel(rec), err)
		}
		removed[idx] = true
		undone = append(undone, rec)
	}

	remaining := make([]MoveRecord, 0, len(history)-len(removed))
	for i, rec := range history {
		if !removed[i] {
			remaining = append(remaining, rec)
		}
	}
	return remaining, undone, nil
}
