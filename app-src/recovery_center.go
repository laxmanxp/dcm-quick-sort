package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultResolvedRecoveryRetention = 30 * 24 * time.Hour

type RecoveryCenterItem struct {
	ID              string
	State           MoveJournalState
	Stage           MoveJournalStage
	Kind            MoveKind
	From            string
	To              string
	Backup          string
	Detail          string
	UpdatedAt       time.Time
	Reviewed        bool
	CanMarkReviewed bool
}

type RecoveryCleanupSummary struct {
	Removed int
	Skipped int
	Missing int
}

func loadRecoveryCenterItems(path string) ([]RecoveryCenterItem, error) {
	journal, err := loadMoveJournal(path)
	if err != nil {
		return nil, err
	}
	items := make([]RecoveryCenterItem, 0)
	for _, entry := range journal.Entries {
		if entry.State != MoveJournalAttention && entry.State != MoveJournalFailed && entry.State != MoveJournalResolved {
			continue
		}
		items = append(items, RecoveryCenterItem{
			ID:              entry.ID,
			State:           entry.State,
			Stage:           entry.Stage,
			Kind:            entry.Record.Kind,
			From:            entry.Record.From,
			To:              entry.Record.To,
			Backup:          entry.Record.ReplacedBackup,
			Detail:          entry.Detail,
			UpdatedAt:       entry.UpdatedAt,
			Reviewed:        entry.State == MoveJournalResolved,
			CanMarkReviewed: entry.State == MoveJournalAttention || entry.State == MoveJournalFailed,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func markRecoveryEntryReviewed(path, id string, now time.Time) error {
	journal, err := loadMoveJournal(path)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	for _, entry := range journal.Entries {
		if entry.ID != id {
			continue
		}
		if entry.State == MoveJournalResolved {
			return nil
		}
		if entry.State != MoveJournalAttention && entry.State != MoveJournalFailed {
			return fmt.Errorf("journal entry %q is not a reviewable recovery item", id)
		}
		detail := strings.TrimSpace(entry.Detail)
		if detail == "" {
			detail = "reviewed by user; no filesystem changes were made"
		} else {
			detail = "reviewed by user; no filesystem changes were made | " + detail
		}
		return updateMoveJournal(path, id, MoveJournalResolved, entry.Record, detail, now)
	}
	return fmt.Errorf("move journal entry %q was not found", id)
}

func pathWithinRoot(path, root string) bool {
	path = strings.TrimSpace(path)
	root = strings.TrimSpace(root)
	if path == "" || root == "" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func cleanupResolvedRecoveryArtifacts(journalPath, recoveryRoot string, retention time.Duration, now time.Time) (RecoveryCleanupSummary, error) {
	var summary RecoveryCleanupSummary
	if retention < 0 {
		return summary, errors.New("recovery retention cannot be negative")
	}
	journal, err := loadMoveJournal(journalPath)
	if err != nil {
		return summary, err
	}
	cutoff := now.Add(-retention)
	for _, entry := range journal.Entries {
		if entry.State != MoveJournalResolved || entry.Record.Kind != MoveKindReplace || strings.TrimSpace(entry.Record.ReplacedBackup) == "" {
			continue
		}
		if entry.UpdatedAt.After(cutoff) {
			continue
		}
		backup := entry.Record.ReplacedBackup
		if !pathWithinRoot(backup, recoveryRoot) {
			summary.Skipped++
			continue
		}
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			summary.Missing++
			entry.Record.ReplacedBackup = ""
			_ = updateMoveJournal(journalPath, entry.ID, MoveJournalResolved, entry.Record, "reviewed recovery artifact was already absent", now)
			continue
		} else if err != nil {
			return summary, err
		}
		// Automatic cleanup is intentionally stricter than user acknowledgement.
		// Remove only a reviewed backup that is provably redundant because the
		// current destination contains exactly the same bytes.
		if !pathExists(entry.Record.To) || !sameFileContent(backup, entry.Record.To) {
			summary.Skipped++
			continue
		}
		if err := os.Remove(backup); err != nil {
			return summary, err
		}
		summary.Removed++
		entry.Record.ReplacedBackup = ""
		if err := updateMoveJournal(journalPath, entry.ID, MoveJournalResolved, entry.Record, "reviewed redundant recovery artifact cleaned after retention period", now); err != nil {
			return summary, err
		}
	}
	return summary, nil
}
