package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type RecoveryMoveAction string

const (
	RecoveryMoveResume        RecoveryMoveAction = "resume"
	RecoveryMoveRestart       RecoveryMoveAction = "restart"
	RecoveryMoveRemovePartial RecoveryMoveAction = "remove-partial"
	RecoveryMoveFinish        RecoveryMoveAction = "finish"
)

type InterruptedMoveAnalysis struct {
	SourceExists     bool
	TargetExists     bool
	SourceSize       int64
	TargetSize       int64
	Partial          bool
	PrefixMatches    bool
	ExactMatch       bool
	CanResume        bool
	CanRestart       bool
	CanRemovePartial bool
	CanFinish        bool
	Detail           string
}

type InterruptedMoveRecoveryResult struct {
	Action   RecoveryMoveAction
	Record   MoveRecord
	Analysis InterruptedMoveAnalysis
	Err      error
}

type InterruptedMoveRecoveryAvailability struct {
	CanAttemptPartial bool
	CanAttemptFinish  bool
	SourceSize        int64
	TargetSize        int64
}

func interruptedMoveRecoveryAvailability(source, target string) InterruptedMoveRecoveryAvailability {
	var out InterruptedMoveRecoveryAvailability
	src, err1 := os.Stat(source)
	dst, err2 := os.Stat(target)
	if err1 != nil || err2 != nil || !src.Mode().IsRegular() || !dst.Mode().IsRegular() {
		return out
	}
	out.SourceSize = src.Size()
	out.TargetSize = dst.Size()
	if dst.Size() < src.Size() {
		out.CanAttemptPartial = true
	} else if dst.Size() == src.Size() {
		out.CanAttemptFinish = true
	}
	return out
}

func fileMatchesSourcePrefix(source, target string) (bool, error) {
	src, err := os.Open(source)
	if err != nil {
		return false, err
	}
	defer src.Close()
	dst, err := os.Open(target)
	if err != nil {
		return false, err
	}
	defer dst.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return false, err
	}
	dstInfo, err := dst.Stat()
	if err != nil {
		return false, err
	}
	if !srcInfo.Mode().IsRegular() || !dstInfo.Mode().IsRegular() {
		return false, errors.New("source and destination must be regular files")
	}
	if dstInfo.Size() > srcInfo.Size() {
		return false, nil
	}

	remaining := dstInfo.Size()
	srcBuf := make([]byte, 1024*1024)
	dstBuf := make([]byte, 1024*1024)
	for remaining > 0 {
		want := int64(len(srcBuf))
		if remaining < want {
			want = remaining
		}
		n1, err1 := io.ReadFull(src, srcBuf[:want])
		n2, err2 := io.ReadFull(dst, dstBuf[:want])
		if err1 != nil && err1 != io.EOF && err1 != io.ErrUnexpectedEOF {
			return false, err1
		}
		if err2 != nil && err2 != io.EOF && err2 != io.ErrUnexpectedEOF {
			return false, err2
		}
		if n1 != n2 || !bytes.Equal(srcBuf[:n1], dstBuf[:n2]) {
			return false, nil
		}
		if int64(n1) != want {
			return false, nil
		}
		remaining -= int64(n1)
	}
	return true, nil
}

func inspectInterruptedStandardMove(source, target string) (InterruptedMoveAnalysis, error) {
	var out InterruptedMoveAnalysis
	srcInfo, srcErr := os.Stat(source)
	dstInfo, dstErr := os.Stat(target)
	out.SourceExists = srcErr == nil
	out.TargetExists = dstErr == nil
	if srcErr != nil && !os.IsNotExist(srcErr) {
		return out, srcErr
	}
	if dstErr != nil && !os.IsNotExist(dstErr) {
		return out, dstErr
	}
	if !out.SourceExists || !out.TargetExists {
		out.Detail = "resume actions require both source and destination to exist"
		return out, nil
	}
	if !srcInfo.Mode().IsRegular() || !dstInfo.Mode().IsRegular() {
		return out, errors.New("recovery actions require regular files")
	}
	out.SourceSize = srcInfo.Size()
	out.TargetSize = dstInfo.Size()
	if out.TargetSize > out.SourceSize {
		out.Detail = "destination is larger than source; automatic recovery is unsafe"
		return out, nil
	}
	prefix, err := fileMatchesSourcePrefix(source, target)
	if err != nil {
		return out, err
	}
	out.PrefixMatches = prefix
	if !prefix {
		out.Detail = "destination bytes do not match the beginning of the source; DCM will not overwrite or delete either file"
		return out, nil
	}
	if out.TargetSize < out.SourceSize {
		out.Partial = true
		out.CanResume = true
		out.CanRestart = true
		out.CanRemovePartial = true
		out.Detail = fmt.Sprintf("verified partial copy: %d of %d bytes match the source prefix", out.TargetSize, out.SourceSize)
		return out, nil
	}
	// Equal size plus prefix equality means every byte is equal. Re-hash before
	// the destructive Finish action as a second, independent safety check.
	out.ExactMatch = true
	out.CanFinish = true
	out.Detail = "destination is a complete byte-identical copy; DCM can finish the move without copying again"
	return out, nil
}

func resumeVerifiedPartialCopyWithProgress(source, target string, progress CopyProgressFunc) error {
	analysis, err := inspectInterruptedStandardMove(source, target)
	if err != nil {
		return err
	}
	if !analysis.CanResume {
		return errors.New("destination is not a verified partial prefix of the source")
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if _, err := in.Seek(analysis.TargetSize, io.SeekStart); err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = out.Close()
		}
	}()

	copied := analysis.TargetSize
	total := analysis.SourceSize
	if progress != nil {
		progress(copied, total)
	}
	buf := make([]byte, 1024*1024)
	nextReport := copied + 4*1024*1024
	for {
		n, readErr := in.Read(buf)
		if n > 0 {
			wn, writeErr := out.Write(buf[:n])
			copied += int64(wn)
			if writeErr != nil {
				return writeErr
			}
			if wn != n {
				return io.ErrShortWrite
			}
			if progress != nil && (copied >= nextReport || copied == total) {
				progress(copied, total)
				for nextReport <= copied {
					nextReport += 4 * 1024 * 1024
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if copied != total {
		return fmt.Errorf("resumed copy wrote %d of %d bytes", copied, total)
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	closed = true
	if !sameFileContent(source, target) {
		return errors.New("SHA-256 verification failed after resumed copy")
	}
	return nil
}

func moveJournalEntryByID(path, id string) (MoveJournalEntry, error) {
	journal, err := loadMoveJournal(path)
	if err != nil {
		return MoveJournalEntry{}, err
	}
	id = strings.TrimSpace(id)
	for _, entry := range journal.Entries {
		if entry.ID == id {
			return entry, nil
		}
	}
	return MoveJournalEntry{}, fmt.Errorf("move journal entry %q was not found", id)
}

func completeRecoveredMoveAfterVerifiedDestination(journalPath string, rec MoveRecord, detail string) error {
	if err := os.Remove(rec.From); err != nil && !os.IsNotExist(err) {
		_ = failMoveJournal(journalPath, rec, MoveJournalAttention, detail+"; destination is verified but source could not be removed: "+err.Error(), time.Now())
		return fmt.Errorf("destination is complete and verified, but source could not be removed: %w", err)
	}
	if err := completeMoveJournal(journalPath, rec, time.Now()); err != nil {
		return fmt.Errorf("move completed, but durable journal finalization failed: %w", err)
	}
	return nil
}

func executeInterruptedMoveRecovery(journalPath, id string, action RecoveryMoveAction, progress CopyProgressFunc) InterruptedMoveRecoveryResult {
	result := InterruptedMoveRecoveryResult{Action: action}
	entry, err := moveJournalEntryByID(journalPath, id)
	if err != nil {
		result.Err = err
		return result
	}
	result.Record = entry.Record
	if entry.Record.Kind != MoveKindMove {
		result.Err = errors.New("resume/restart recovery actions apply only to ordinary moves")
		return result
	}
	if entry.State != MoveJournalAttention && entry.State != MoveJournalFailed {
		result.Err = fmt.Errorf("journal state %q is not eligible for interrupted-move recovery", entry.State)
		return result
	}

	analysis, err := inspectInterruptedStandardMove(entry.Record.From, entry.Record.To)
	result.Analysis = analysis
	if err != nil {
		result.Err = err
		return result
	}
	rec := entry.Record
	now := time.Now()

	failAttention := func(err error, detail string) InterruptedMoveRecoveryResult {
		_ = failMoveJournal(journalPath, rec, MoveJournalAttention, detail+": "+err.Error(), time.Now())
		result.Err = err
		return result
	}

	switch action {
	case RecoveryMoveResume:
		if !analysis.CanResume {
			result.Err = errors.New("resume is unsafe because the destination is not a verified partial prefix")
			return result
		}
		if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageRecoveryResumeStarted, "user requested verified resume after interrupted copy", now); err != nil {
			result.Err = err
			return result
		}
		if err := resumeVerifiedPartialCopyWithProgress(rec.From, rec.To, progress); err != nil {
			return failAttention(err, "resume did not complete; source was retained")
		}
		if err := completeRecoveredMoveAfterVerifiedDestination(journalPath, rec, "resumed destination verified"); err != nil {
			result.Err = err
			return result
		}
	case RecoveryMoveRestart:
		if !analysis.CanRestart {
			result.Err = errors.New("restart is unsafe because the destination is not a verified partial prefix")
			return result
		}
		if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageRecoveryRestartStarted, "user requested restart of verified partial copy", now); err != nil {
			result.Err = err
			return result
		}
		if err := os.Remove(rec.To); err != nil {
			return failAttention(err, "could not remove verified partial destination before restart")
		}
		if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageRecoveryPartialRemoved, "verified partial destination removed; source remains authoritative", time.Now()); err != nil {
			result.Err = fmt.Errorf("partial destination removed safely, but journal update failed: %w", err)
			return result
		}
		if err := copyFileVerifiedWithProgress(rec.From, rec.To, progress); err != nil {
			_ = failMoveJournal(journalPath, rec, MoveJournalFailed, "restart copy failed; source remains authoritative", time.Now())
			result.Err = err
			return result
		}
		if err := completeRecoveredMoveAfterVerifiedDestination(journalPath, rec, "restarted destination verified"); err != nil {
			result.Err = err
			return result
		}
	case RecoveryMoveRemovePartial:
		if !analysis.CanRemovePartial {
			result.Err = errors.New("destination is not a verified partial copy and will not be removed")
			return result
		}
		if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageRecoveryRestartStarted, "user requested removal of verified partial destination", now); err != nil {
			result.Err = err
			return result
		}
		if err := os.Remove(rec.To); err != nil {
			return failAttention(err, "could not remove verified partial destination")
		}
		if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageRecoveryPartialRemoved, "verified partial destination removed; source retained", time.Now()); err != nil {
			result.Err = err
			return result
		}
		if err := updateMoveJournal(journalPath, rec.JournalID, MoveJournalResolved, rec, "verified partial destination removed by user; source retained and interrupted move cancelled", time.Now()); err != nil {
			result.Err = err
			return result
		}
	case RecoveryMoveFinish:
		if !analysis.CanFinish || !sameFileContent(rec.From, rec.To) {
			result.Err = errors.New("finish is unsafe because source and destination are not exact SHA-256 matches")
			return result
		}
		if err := updateMoveJournalStage(journalPath, rec, MoveJournalStageRecoveryFinishStarted, "user requested finish after exact destination verification", now); err != nil {
			result.Err = err
			return result
		}
		if err := completeRecoveredMoveAfterVerifiedDestination(journalPath, rec, "exact destination verified"); err != nil {
			result.Err = err
			return result
		}
	default:
		result.Err = fmt.Errorf("unknown recovery action %q", action)
		return result
	}

	return result
}
