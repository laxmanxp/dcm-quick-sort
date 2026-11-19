package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultJournalCompactionThreshold = 500

type JournalCompactionSummary struct {
	Compacted    bool
	BeforeEvents int
	AfterEvents  int
}

func restoreCompactionBackupIfNeeded(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("move journal path is unavailable")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	matches, err := filepath.Glob(path + ".compactbak.*")
	if err != nil || len(matches) == 0 {
		return err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	for _, backup := range matches {
		data, readErr := os.ReadFile(backup)
		if readErr != nil || !journalJSONLLooksValid(data) {
			continue
		}
		if err := os.Rename(backup, path); err != nil {
			return fmt.Errorf("could not restore interrupted journal compaction backup: %w", err)
		}
		return nil
	}
	return errors.New("move journal is missing and no valid compaction backup could be restored")
}

func journalJSONLLooksValid(data []byte) bool {
	for _, raw := range bytes.Split(data, []byte{'\n'}) {
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 {
			continue
		}
		var entry MoveJournalEntry
		if json.Unmarshal(raw, &entry) != nil || strings.TrimSpace(entry.ID) == "" || entry.State == "" {
			return false
		}
	}
	return true
}

func countMoveJournalEvents(path string) (int, error) {
	if err := restoreCompactionBackupIfNeeded(path); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	count := 0
	lines := bytes.Split(data, []byte{'\n'})
	for i, raw := range lines {
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 {
			continue
		}
		var entry MoveJournalEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			isTail := i == len(lines)-1 && len(data) > 0 && data[len(data)-1] != '\n'
			if isTail && count > 0 {
				break
			}
			return 0, fmt.Errorf("move journal is corrupt at line %d: %w", i+1, err)
		}
		count++
	}
	return count, nil
}

func writeCompactedJournal(path string, entries []MoveJournalEntry) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	enc := json.NewEncoder(f)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			return err
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func compactMoveJournal(path string, minEvents int) (JournalCompactionSummary, error) {
	var summary JournalCompactionSummary
	if minEvents < 1 {
		minEvents = 1
	}
	before, err := countMoveJournalEvents(path)
	if err != nil {
		return summary, err
	}
	summary.BeforeEvents = before
	if before < minEvents {
		summary.AfterEvents = before
		return summary, nil
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		return summary, err
	}
	if before <= len(journal.Entries) {
		summary.AfterEvents = before
		return summary, nil
	}
	stamp := time.Now().UTC().UnixNano()
	tmp := fmt.Sprintf("%s.compacttmp.%d", path, stamp)
	backup := fmt.Sprintf("%s.compactbak.%d", path, stamp)
	if err := writeCompactedJournal(tmp, journal.Entries); err != nil {
		return summary, err
	}
	// Verify the compacted file before swapping it into service.
	if _, err := loadMoveJournal(tmp); err != nil {
		_ = os.Remove(tmp)
		return summary, fmt.Errorf("compacted journal verification failed: %w", err)
	}
	if err := os.Rename(path, backup); err != nil {
		_ = os.Remove(tmp)
		return summary, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Rename(backup, path)
		_ = os.Remove(tmp)
		return summary, fmt.Errorf("could not activate compacted journal: %w", err)
	}
	verified, err := loadMoveJournal(path)
	if err != nil || len(verified.Entries) != len(journal.Entries) {
		_ = os.Remove(path)
		_ = os.Rename(backup, path)
		if err != nil {
			return summary, fmt.Errorf("compacted journal failed final verification: %w", err)
		}
		return summary, errors.New("compacted journal changed entry count")
	}
	_ = os.Remove(backup)
	summary.Compacted = true
	summary.AfterEvents = len(journal.Entries)
	return summary, nil
}
