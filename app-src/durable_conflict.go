package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

func replaceMoveWithRecoveryJournaled(journalPath, source, target, recoveryDir string, batchID ...string) (MoveRecord, error) {
	return replaceMoveWithRecoveryJournaledUsing(journalPath, source, target, recoveryDir, moveFile, batchID...)
}

func replaceMoveWithRecoveryJournaledUsing(journalPath, source, target, recoveryDir string, moveFn func(string, string) error, batchID ...string) (MoveRecord, error) {
	rec := MoveRecord{From: source, To: target, Kind: MoveKindReplace}
	if len(batchID) > 0 {
		rec.BatchID = batchID[0]
	}
	info, err := inspectDestinationConflict(source, target)
	if err != nil {
		return rec, err
	}
	if info.Kind == ConflictNone {
		return rec, errors.New("replace requested but destination does not exist")
	}

	rec, err = beginMoveJournal(journalPath, rec, time.Now())
	if err != nil {
		return rec, fmt.Errorf("durable journal unavailable; replacement not started: %w", err)
	}

	backup, err := replacementBackupPath(recoveryDir, target)
	if err != nil {
		_ = failMoveJournal(journalPath, rec, MoveJournalFailed, "could not allocate replacement recovery path", time.Now())
		return rec, err
	}
	rec.ReplacedBackup = backup
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageReplaceBackupPlanned, "replacement recovery path reserved before backup copy", time.Now()); err != nil {
		return rec, fmt.Errorf("could not persist replacement backup plan; no destination bytes were changed: %w", err)
	}

	if err := copyFileVerifiedHash(target, backup); err != nil {
		_ = failMoveJournal(journalPath, rec, MoveJournalFailed, "could not create verified replacement recovery copy", time.Now())
		return rec, fmt.Errorf("could not back up existing destination: %w", err)
	}
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageReplaceBackupCreated, "previous destination copied and SHA-256 verified", time.Now()); err != nil {
		return rec, fmt.Errorf("replacement recovery copy is safe at %s, but journal stage could not be persisted; destination was not removed: %w", backup, err)
	}

	if err := os.Remove(target); err != nil {
		_ = failMoveJournal(journalPath, rec, MoveJournalFailed, "could not remove previous destination; recovery copy retained", time.Now())
		return rec, fmt.Errorf("could not prepare destination for replacement: %w", err)
	}
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageReplaceDestinationRemoved, "previous destination removed; recovery copy retained", time.Now()); err != nil {
		// Do not proceed to the source move without a durable stage recording that
		// the previous destination is gone. Restore it immediately if possible.
		restoreErr := copyFileVerifiedHash(backup, target)
		if restoreErr == nil {
			_ = failMoveJournal(journalPath, rec, MoveJournalFailed, "journal stage failed after destination removal; previous destination restored", time.Now())
			return rec, fmt.Errorf("journal update failed after preparing replacement; previous destination was restored safely: %w", err)
		}
		_ = failMoveJournal(journalPath, rec, MoveJournalAttention, "journal stage failed after destination removal and automatic restoration failed", time.Now())
		return rec, fmt.Errorf("journal update failed after destination removal: %v; previous destination remains recoverable at %s but could not be restored automatically: %v", err, backup, restoreErr)
	}

	if err := moveFn(source, target); err != nil {
		// Roll back when possible. If target exists because a copy succeeded but
		// source deletion failed, never overwrite it; the recovery copy remains
		// durable and the journal is marked attention for manual review.
		restoreErr := error(nil)
		if !pathExists(target) {
			restoreErr = copyFileVerifiedHash(backup, target)
		} else if sameFileContent(target, backup) {
			restoreErr = nil
		} else {
			restoreErr = errors.New("replacement target contains different bytes and was preserved")
		}
		if restoreErr == nil {
			_ = failMoveJournal(journalPath, rec, MoveJournalFailed, "replacement move failed; previous destination restored and source retained", time.Now())
			return rec, err
		}
		_ = failMoveJournal(journalPath, rec, MoveJournalAttention, "replacement move failed; recovery copy retained for manual recovery", time.Now())
		return rec, fmt.Errorf("replacement failed: %v; previous destination is preserved at %s; automatic restoration was not safe: %v", err, backup, restoreErr)
	}

	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageReplaceCommitted, "replacement committed; recovery copy retained for Undo", time.Now()); err != nil {
		return rec, fmt.Errorf("replacement completed safely, but durable journal commit stage is pending recovery on next launch: %w", err)
	}
	if err := completeMoveJournal(journalPath, rec, time.Now()); err != nil {
		return rec, fmt.Errorf("replacement completed safely, but durable journal finalization is pending recovery on next launch: %w", err)
	}
	return rec, nil
}

func removeExactDuplicateSourceJournaled(journalPath, source, target string, batchID ...string) (MoveRecord, error) {
	rec := MoveRecord{From: source, To: target, Kind: MoveKindDuplicateRemoved}
	if len(batchID) > 0 {
		rec.BatchID = batchID[0]
	}
	info, err := inspectDestinationConflict(source, target)
	if err != nil {
		return rec, err
	}
	if info.Kind != ConflictExactDuplicate {
		return rec, errors.New("source is not an exact duplicate of the destination")
	}

	rec, err = beginMoveJournal(journalPath, rec, time.Now())
	if err != nil {
		return rec, fmt.Errorf("durable journal unavailable; duplicate source was not removed: %w", err)
	}
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageDuplicateVerified, "source and destination verified identical by size and SHA-256", time.Now()); err != nil {
		return rec, fmt.Errorf("duplicate was verified but journal stage could not be persisted; source was not removed: %w", err)
	}
	if err := os.Remove(source); err != nil {
		_ = failMoveJournal(journalPath, rec, MoveJournalFailed, "verified duplicate source could not be removed", time.Now())
		return rec, err
	}
	if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageDuplicateSourceRemoved, "verified duplicate source removed; destination retained", time.Now()); err != nil {
		return rec, fmt.Errorf("duplicate source was removed safely, but durable journal stage is pending recovery on next launch: %w", err)
	}
	if err := completeMoveJournal(journalPath, rec, time.Now()); err != nil {
		return rec, fmt.Errorf("duplicate source was removed safely, but durable journal finalization is pending recovery on next launch: %w", err)
	}
	return rec, nil
}
