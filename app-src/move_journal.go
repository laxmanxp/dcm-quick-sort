package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MoveJournalState string
type MoveJournalStage string

const (
	MoveJournalPending   MoveJournalState = "pending"
	MoveJournalCompleted MoveJournalState = "completed"
	MoveJournalFailed    MoveJournalState = "failed"
	MoveJournalAttention MoveJournalState = "attention"
	MoveJournalUndone    MoveJournalState = "undone"
	MoveJournalResolved  MoveJournalState = "resolved"
)

const (
	MoveJournalStageStarted                   MoveJournalStage = "started"
	MoveJournalStageReplaceBackupPlanned      MoveJournalStage = "replace-backup-planned"
	MoveJournalStageReplaceBackupCreated      MoveJournalStage = "replace-backup-created"
	MoveJournalStageReplaceDestinationRemoved MoveJournalStage = "replace-destination-removed"
	MoveJournalStageReplaceCommitted          MoveJournalStage = "replace-committed"
	MoveJournalStageDuplicateVerified         MoveJournalStage = "duplicate-verified"
	MoveJournalStageDuplicateSourceRemoved    MoveJournalStage = "duplicate-source-removed"
	MoveJournalStageRecoveryResumeStarted     MoveJournalStage = "recovery-resume-started"
	MoveJournalStageRecoveryRestartStarted    MoveJournalStage = "recovery-restart-started"
	MoveJournalStageRecoveryPartialRemoved    MoveJournalStage = "recovery-partial-removed"
	MoveJournalStageRecoveryFinishStarted     MoveJournalStage = "recovery-finish-started"
)

type MoveJournalEntry struct {
	ID        string           `json:"id"`
	State     MoveJournalState `json:"state"`
	Stage     MoveJournalStage `json:"stage,omitempty"`
	Record    MoveRecord       `json:"record"`
	StartedAt time.Time        `json:"started_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Detail    string           `json:"detail,omitempty"`
}

type MoveJournal struct {
	Entries []MoveJournalEntry
}

type MoveJournalRecovery struct {
	History       []MoveRecord
	Attention     []MoveJournalEntry
	AutoFinalized int
	Interrupted   int
}

// loadMoveJournal replays the append-only journal and returns the latest state
// for every operation in first-seen order. A truncated final event is tolerated
// only when at least one earlier valid event exists; in that case recovery falls
// back to the earlier durable state and reconciles it conservatively.
func loadMoveJournal(path string) (MoveJournal, error) {
	var out MoveJournal
	path = strings.TrimSpace(path)
	if path == "" {
		return out, errors.New("move journal path is unavailable")
	}
	if err := restoreCompactionBackupIfNeeded(path); err != nil {
		if !os.IsNotExist(err) {
			return out, err
		}
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return out, err
	}

	byID := make(map[string]int)
	lines := bytes.Split(data, []byte{'\n'})
	valid := 0
	for i, rawBytes := range lines {
		raw := strings.TrimSpace(string(rawBytes))
		if raw == "" {
			continue
		}
		var entry MoveJournalEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			isPhysicalTail := i == len(lines)-1 && len(data) > 0 && data[len(data)-1] != '\n'
			if isPhysicalTail && valid > 0 {
				// A crash may tear only the last append. The previous fsynced state
				// remains authoritative; appendMoveJournalEntry repairs this tail
				// before writing another event.
				break
			}
			return MoveJournal{}, fmt.Errorf("move journal is corrupt at line %d: %w", i+1, err)
		}
		if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.Record.From) == "" || strings.TrimSpace(entry.Record.To) == "" {
			return MoveJournal{}, fmt.Errorf("move journal contains an incomplete event at line %d", i+1)
		}
		if entry.State == "" {
			return MoveJournal{}, fmt.Errorf("move journal contains an event without state at line %d", i+1)
		}
		valid++
		if idx, ok := byID[entry.ID]; ok {
			out.Entries[idx] = entry
		} else {
			byID[entry.ID] = len(out.Entries)
			out.Entries = append(out.Entries, entry)
		}
	}
	return out, nil
}

func repairMoveJournalTail(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) || len(data) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	if data[len(data)-1] == '\n' {
		return nil
	}
	lastNL := bytes.LastIndexByte(data, '\n')
	tail := bytes.TrimSpace(data[lastNL+1:])
	if len(tail) > 0 && json.Valid(tail) {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if _, err := f.Write([]byte{'\n'}); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Sync(); err != nil {
			_ = f.Close()
			return err
		}
		return f.Close()
	}
	if lastNL < 0 {
		return errors.New("move journal contains only an invalid truncated event")
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Truncate(int64(lastNL + 1)); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func appendMoveJournalEntry(path string, entry MoveJournalEntry) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("move journal path is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := repairMoveJournalTail(path); err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			// Never delete an existing journal because one append failed.
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
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

func beginMoveJournal(path string, rec MoveRecord, now time.Time) (MoveRecord, error) {
	journal, err := loadMoveJournal(path)
	if err != nil {
		return rec, err
	}
	if rec.Kind == "" {
		rec.Kind = MoveKindMove
	}
	rec.JournalID = fmt.Sprintf("%d-%06d", now.UTC().UnixNano(), len(journal.Entries)+1)
	entry := MoveJournalEntry{
		ID: rec.JournalID, State: MoveJournalPending, Stage: MoveJournalStageStarted, Record: rec,
		StartedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if err := appendMoveJournalEntry(path, entry); err != nil {
		rec.JournalID = ""
		return rec, err
	}
	return rec, nil
}

func updateMoveJournal(path, id string, state MoveJournalState, rec MoveRecord, detail string, now time.Time) error {
	journal, err := loadMoveJournal(path)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	for _, existing := range journal.Entries {
		if existing.ID != id {
			continue
		}
		if strings.TrimSpace(rec.JournalID) == "" {
			rec.JournalID = id
		}
		entry := existing
		entry.State = state
		entry.Record = rec
		entry.UpdatedAt = now.UTC()
		entry.Detail = strings.TrimSpace(detail)
		return appendMoveJournalEntry(path, entry)
	}
	return fmt.Errorf("move journal entry %q was not found", id)
}

func updateMoveJournalStage(path string, rec MoveRecord, stage MoveJournalStage, detail string, now time.Time) error {
	if strings.TrimSpace(rec.JournalID) == "" {
		return errors.New("move record has no journal id")
	}
	journal, err := loadMoveJournal(path)
	if err != nil {
		return err
	}
	for _, existing := range journal.Entries {
		if existing.ID != rec.JournalID {
			continue
		}
		entry := existing
		entry.State = MoveJournalPending
		entry.Stage = stage
		entry.Record = rec
		entry.UpdatedAt = now.UTC()
		entry.Detail = strings.TrimSpace(detail)
		return appendMoveJournalEntry(path, entry)
	}
	return fmt.Errorf("move journal entry %q was not found", rec.JournalID)
}

func completeMoveJournal(path string, rec MoveRecord, now time.Time) error {
	if strings.TrimSpace(rec.JournalID) == "" {
		return errors.New("move record has no journal id")
	}
	return updateMoveJournal(path, rec.JournalID, MoveJournalCompleted, rec, "", now)
}

func failMoveJournal(path string, rec MoveRecord, state MoveJournalState, detail string, now time.Time) error {
	if state != MoveJournalFailed && state != MoveJournalAttention {
		return errors.New("invalid failure journal state")
	}
	if strings.TrimSpace(rec.JournalID) == "" {
		return errors.New("move record has no journal id")
	}
	return updateMoveJournal(path, rec.JournalID, state, rec, detail, now)
}

func laterJournalConsumer(entries []MoveJournalEntry, index int) (MoveJournalEntry, bool) {
	if index < 0 || index >= len(entries) {
		return MoveJournalEntry{}, false
	}
	target := entries[index].Record.To
	for i := index + 1; i < len(entries); i++ {
		next := entries[i]
		if next.State == MoveJournalUndone {
			continue
		}
		if sameMovePath(next.Record.From, target) {
			return next, true
		}
	}
	return MoveJournalEntry{}, false
}

// reconcileConsumedRenameChain recognizes the split-button workflow:
// Rename A -> B, then Move B -> C. Once B has been consumed by the later
// transaction, neither endpoint of the Rename exists. That is expected and is
// not a recovery failure. Legacy v1.0.54/v1.0.55 journals may already have
// changed such a Rename to attention; repair it back to completed.
func reconcileConsumedRenameChain(path string, entries []MoveJournalEntry, index int, now time.Time) (MoveJournalEntry, bool, error) {
	if index < 0 || index >= len(entries) {
		return MoveJournalEntry{}, false, nil
	}
	entry := entries[index]
	if !isRenameOnlyRecord(entry.Record) {
		return entry, false, nil
	}
	if pathExists(entry.Record.From) || pathExists(entry.Record.To) {
		return entry, false, nil
	}
	successor, ok := laterJournalConsumer(entries, index)
	if !ok {
		return entry, false, nil
	}
	if entry.State != MoveJournalCompleted {
		entry.State = MoveJournalCompleted
		entry.Detail = "rename output was consumed by later DCM transaction " + successor.ID
		entry.UpdatedAt = now.UTC()
		if err := appendMoveJournalEntry(path, entry); err != nil {
			return entry, false, err
		}
	}
	return entry, true, nil
}

func markMoveJournalUndone(path, id string, now time.Time) error {
	journal, err := loadMoveJournal(path)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	for _, entry := range journal.Entries {
		if entry.ID == id {
			return updateMoveJournal(path, id, MoveJournalUndone, entry.Record, "undone by user", now)
		}
	}
	return fmt.Errorf("move journal entry %q was not found", id)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sameFileContent(a, b string) bool {
	if !pathExists(a) || !pathExists(b) {
		return false
	}
	ah, err := fileSHA256(a)
	if err != nil {
		return false
	}
	bh, err := fileSHA256(b)
	return err == nil && ah == bh
}

func appendRecoveredEntry(path string, entry MoveJournalEntry, now time.Time) error {
	entry.UpdatedAt = now.UTC()
	return appendMoveJournalEntry(path, entry)
}

func recoverPendingStandardMove(path string, entry MoveJournalEntry, now time.Time) (MoveJournalEntry, bool, bool, error) {
	srcExists := pathExists(entry.Record.From)
	dstExists := pathExists(entry.Record.To)
	switch {
	case !srcExists && dstExists:
		entry.State = MoveJournalCompleted
		entry.Detail = "finalized after restart: destination exists and source is absent"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, false, false, err
		}
		return entry, true, false, nil
	case srcExists && !dstExists:
		entry.State = MoveJournalFailed
		entry.Detail = "interrupted before destination commit; source remains unchanged"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, false, false, err
		}
		return entry, false, true, nil
	default:
		entry.State = MoveJournalAttention
		if srcExists && dstExists {
			entry.Detail = "both source and destination exist after an interrupted move; DCM preserved both. Recovery Center can verify whether the destination is a resumable partial copy or an already-complete exact copy"
		} else {
			entry.Detail = "neither source nor destination exists; manual review required"
		}
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, false, false, err
		}
		return entry, false, false, nil
	}
}

func recoverPendingReplace(path string, entry MoveJournalEntry, now time.Time) (MoveJournalEntry, bool, bool, error) {
	srcExists := pathExists(entry.Record.From)
	dstExists := pathExists(entry.Record.To)
	backup := strings.TrimSpace(entry.Record.ReplacedBackup)
	backupExists := backup != "" && pathExists(backup)

	// Once the old destination may have been removed, source absent + destination
	// present + verified backup is the durable signature of a committed replace.
	if !srcExists && dstExists && backupExists &&
		(entry.Stage == MoveJournalStageReplaceDestinationRemoved || entry.Stage == MoveJournalStageReplaceCommitted) {
		entry.State = MoveJournalCompleted
		entry.Stage = MoveJournalStageReplaceCommitted
		entry.Detail = "finalized replacement after restart; recovery copy retained for Undo"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, false, false, err
		}
		return entry, true, false, nil
	}

	// If the source still exists and the old destination is absent, DCM can safely
	// restore the verified recovery copy without touching the source.
	if srcExists && !dstExists && backupExists &&
		(entry.Stage == MoveJournalStageReplaceBackupCreated || entry.Stage == MoveJournalStageReplaceDestinationRemoved) {
		if err := copyFileVerifiedHash(backup, entry.Record.To); err != nil {
			entry.State = MoveJournalAttention
			entry.Detail = "could not restore previous destination from the durable recovery copy: " + err.Error()
			if appendErr := appendRecoveredEntry(path, entry, now); appendErr != nil {
				return entry, false, false, appendErr
			}
			return entry, false, false, nil
		}
		entry.State = MoveJournalFailed
		entry.Detail = "interrupted replacement rolled back after restart; source preserved and previous destination restored"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, false, false, err
		}
		return entry, false, true, nil
	}

	// Before destructive replacement starts, both originals still existing means
	// the operation can be closed as interrupted.  We keep any recovery artifact;
	// a future retention policy may clean redundant failed-operation backups.
	if srcExists && dstExists {
		if entry.Stage == MoveJournalStageStarted || entry.Stage == MoveJournalStageReplaceBackupPlanned || entry.Stage == MoveJournalStageReplaceBackupCreated {
			entry.State = MoveJournalFailed
			entry.Detail = "replacement interrupted before commit; source and destination remain present"
			if err := appendRecoveredEntry(path, entry, now); err != nil {
				return entry, false, false, err
			}
			return entry, false, true, nil
		}
		if backupExists && sameFileContent(entry.Record.To, backup) {
			entry.State = MoveJournalFailed
			entry.Detail = "replacement rollback is already present on disk; source preserved"
			if err := appendRecoveredEntry(path, entry, now); err != nil {
				return entry, false, false, err
			}
			return entry, false, true, nil
		}
	}

	entry.State = MoveJournalAttention
	entry.Detail = "replacement filesystem state is ambiguous; DCM preserved all remaining files and requires review"
	if err := appendRecoveredEntry(path, entry, now); err != nil {
		return entry, false, false, err
	}
	return entry, false, false, nil
}

func recoverPendingDuplicate(path string, entry MoveJournalEntry, now time.Time) (MoveJournalEntry, bool, bool, error) {
	srcExists := pathExists(entry.Record.From)
	dstExists := pathExists(entry.Record.To)
	if !srcExists && dstExists && verifyDuplicateOriginal(entry.Record.To, entry.Record) == nil && (entry.Stage == MoveJournalStageDuplicateVerified || entry.Stage == MoveJournalStageDuplicateSourceRemoved) {
		entry.State = MoveJournalCompleted
		entry.Stage = MoveJournalStageDuplicateSourceRemoved
		entry.Detail = "finalized exact-duplicate removal after restart"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, false, false, err
		}
		return entry, true, false, nil
	}
	if srcExists && dstExists {
		entry.State = MoveJournalFailed
		entry.Detail = "duplicate removal interrupted before source deletion; both files remain"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, false, false, err
		}
		return entry, false, true, nil
	}
	entry.State = MoveJournalAttention
	entry.Detail = "duplicate-removal filesystem state is ambiguous; DCM did not delete any remaining file"
	if err := appendRecoveredEntry(path, entry, now); err != nil {
		return entry, false, false, err
	}
	return entry, false, false, nil
}

func reconcileCompletedMove(path string, entry MoveJournalEntry, now time.Time) (MoveJournalEntry, error) {
	srcExists := pathExists(entry.Record.From)
	dstExists := pathExists(entry.Record.To)
	if srcExists && !dstExists {
		entry.State = MoveJournalUndone
		entry.Detail = "reconciled after restart: filesystem shows move was undone"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, err
		}
	} else if !srcExists && !dstExists {
		entry.State = MoveJournalAttention
		entry.Detail = "completed journal entry no longer has source or destination; manual review required"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, err
		}
	}
	return entry, nil
}

func reconcileCompletedReplace(path string, entry MoveJournalEntry, now time.Time) (MoveJournalEntry, error) {
	srcExists := pathExists(entry.Record.From)
	dstExists := pathExists(entry.Record.To)
	backup := strings.TrimSpace(entry.Record.ReplacedBackup)
	backupExists := backup != "" && pathExists(backup)

	switch {
	case !srcExists && dstExists && backupExists:
		return entry, nil // normal completed replacement; remains undoable
	case srcExists && !dstExists && backupExists:
		// Undo moved the replacement back to the source but crashed before the old
		// destination was restored. Finish the already-requested Undo safely.
		if err := copyFileVerifiedHash(backup, entry.Record.To); err != nil {
			entry.State = MoveJournalAttention
			entry.Detail = "partial replacement Undo detected; previous destination could not be restored: " + err.Error()
			if appendErr := appendRecoveredEntry(path, entry, now); appendErr != nil {
				return entry, appendErr
			}
			return entry, nil
		}
		_ = os.Remove(backup)
		entry.State = MoveJournalUndone
		entry.Detail = "completed interrupted replacement Undo after restart"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, err
		}
		return entry, nil
	case srcExists && dstExists && backupExists:
		if sameFileContent(entry.Record.To, backup) {
			_ = os.Remove(backup)
			entry.State = MoveJournalUndone
			entry.Detail = "reconciled replacement Undo after restart; previous destination already restored"
			if err := appendRecoveredEntry(path, entry, now); err != nil {
				return entry, err
			}
			return entry, nil
		}
	case srcExists && dstExists && !backupExists:
		entry.State = MoveJournalUndone
		entry.Detail = "reconciled after restart: replacement Undo completed before journal finalization"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, err
		}
		return entry, nil
	}

	entry.State = MoveJournalAttention
	entry.Detail = "completed replacement no longer has a safe undoable filesystem state; manual review required"
	if err := appendRecoveredEntry(path, entry, now); err != nil {
		return entry, err
	}
	return entry, nil
}

func reconcileCompletedDuplicate(path string, entry MoveJournalEntry, now time.Time) (MoveJournalEntry, error) {
	srcExists := pathExists(entry.Record.From)
	dstExists := pathExists(entry.Record.To)
	if !srcExists && dstExists && verifyDuplicateOriginal(entry.Record.To, entry.Record) == nil {
		return entry, nil
	}
	if srcExists && dstExists && verifyDuplicateOriginal(entry.Record.From, entry.Record) == nil && verifyDuplicateOriginal(entry.Record.To, entry.Record) == nil {
		entry.State = MoveJournalUndone
		entry.Detail = "reconciled after restart: exact-duplicate removal was undone"
		if err := appendRecoveredEntry(path, entry, now); err != nil {
			return entry, err
		}
		return entry, nil
	}
	entry.State = MoveJournalAttention
	entry.Detail = "completed duplicate removal lacks a verified original fingerprint or its files changed; manual review required"
	if err := appendRecoveredEntry(path, entry, now); err != nil {
		return entry, err
	}
	return entry, nil
}

func recoverMoveJournal(path string, now time.Time) (MoveJournalRecovery, error) {
	journal, err := loadMoveJournal(path)
	if err != nil {
		return MoveJournalRecovery{}, err
	}
	var recovery MoveJournalRecovery
	for i := range journal.Entries {
		entry := journal.Entries[i]
		var renameConsumed bool
		entry, renameConsumed, err = reconcileConsumedRenameChain(path, journal.Entries, i, now)
		if err != nil {
			return recovery, err
		}
		if !renameConsumed && entry.State == MoveJournalPending {
			var autoFinalized, interrupted bool
			switch entry.Record.Kind {
			case MoveKindMove:
				entry, autoFinalized, interrupted, err = recoverPendingStandardMove(path, entry, now)
			case MoveKindReplace:
				entry, autoFinalized, interrupted, err = recoverPendingReplace(path, entry, now)
			case MoveKindDuplicateRemoved:
				entry, autoFinalized, interrupted, err = recoverPendingDuplicate(path, entry, now)
			default:
				entry.State = MoveJournalAttention
				entry.Detail = "unfinished operation kind requires manual review"
				err = appendRecoveredEntry(path, entry, now)
			}
			if err != nil {
				return recovery, err
			}
			if autoFinalized {
				recovery.AutoFinalized++
			}
			if interrupted {
				recovery.Interrupted++
			}
		}

		if entry.State == MoveJournalCompleted && !renameConsumed {
			switch entry.Record.Kind {
			case MoveKindMove:
				entry, err = reconcileCompletedMove(path, entry, now)
			case MoveKindReplace:
				entry, err = reconcileCompletedReplace(path, entry, now)
			case MoveKindDuplicateRemoved:
				entry, err = reconcileCompletedDuplicate(path, entry, now)
			}
			if err != nil {
				return recovery, err
			}
		}

		switch entry.State {
		case MoveJournalCompleted:
			recovery.History = append(recovery.History, entry.Record)
		case MoveJournalAttention:
			recovery.Attention = append(recovery.Attention, entry)
		}
	}
	return recovery, nil
}
