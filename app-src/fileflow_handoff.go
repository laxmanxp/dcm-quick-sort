package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	fileFlowHandoffVersion       = 2
	fileFlowActionHandoff        = "handoff"
	fileFlowActionEnsureMetadata = "ensure_file_metadata"
)

type FileFlowHandoffEvent struct {
	Version      int       `json:"version"`
	Action       string    `json:"action,omitempty"`
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	OriginalPath string    `json:"original_path"`
	CurrentPath  string    `json:"current_path"`
	JournalID    string    `json:"journal_id,omitempty"`
	Kind         MoveKind  `json:"kind"`
	DisplayName  string    `json:"display_name"`
	Tags         []string  `json:"tags,omitempty"`
	Todos        []string  `json:"todos,omitempty"`
	Notes        []string  `json:"notes,omitempty"`
}

func fileFlowHandoffQueuePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "handoff", "fileflow.jsonl")
	}
	return filepath.Join(os.TempDir(), "DCM Quick Sort", "handoff", "fileflow.jsonl")
}

func fileFlowHandoffID(rec MoveRecord) string {
	if id := strings.TrimSpace(rec.JournalID); id != "" {
		return "journal:" + id
	}
	sum := sha256.Sum256([]byte(string(rec.Kind) + "\x00" + filepath.Clean(rec.From) + "\x00" + filepath.Clean(rec.To)))
	return "move:" + hex.EncodeToString(sum[:16])
}

func validateFileFlowHandoffRecord(rec MoveRecord) error {
	if rec.Kind == MoveKindDuplicateRemoved {
		return errors.New("exact-duplicate cleanup does not create a new FileFlow handoff")
	}
	if strings.TrimSpace(rec.To) == "" {
		return errors.New("completed target path is unavailable")
	}
	st, err := os.Stat(rec.To)
	if err != nil {
		return fmt.Errorf("completed target is unavailable: %w", err)
	}
	if !st.Mode().IsRegular() {
		return errors.New("completed target is not a regular file")
	}
	return nil
}

func loadFileFlowHandoffs(path string) ([]FileFlowHandoffEvent, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []FileFlowHandoffEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event FileFlowHandoffEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("FileFlow handoff queue is corrupt: %w", err)
		}
		if (event.Version != 1 && event.Version != fileFlowHandoffVersion) || strings.TrimSpace(event.ID) == "" {
			return nil, errors.New("FileFlow handoff queue contains an unsupported event")
		}
		if strings.TrimSpace(event.Action) == "" {
			event.Action = fileFlowActionHandoff
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func queueFileFlowHandoff(path string, rec MoveRecord, now time.Time) (bool, error) {
	if err := validateFileFlowHandoffRecord(rec); err != nil {
		return false, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return false, errors.New("FileFlow handoff queue path is unavailable")
	}
	id := fileFlowHandoffID(rec)
	existing, err := loadFileFlowHandoffs(path)
	if err != nil {
		return false, err
	}
	for _, event := range existing {
		if event.ID == id {
			return false, nil
		}
	}
	event := FileFlowHandoffEvent{
		Version:      fileFlowHandoffVersion,
		Action:       fileFlowActionHandoff,
		ID:           id,
		CreatedAt:    now,
		OriginalPath: filepath.Clean(rec.From),
		CurrentPath:  filepath.Clean(rec.To),
		JournalID:    strings.TrimSpace(rec.JournalID),
		Kind:         rec.Kind,
		DisplayName:  filepath.Base(rec.To),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return false, err
	}
	if _, err = f.Write(append(data, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func validateFileFlowMetadataTarget(rec MoveRecord) error {
	if strings.TrimSpace(rec.To) == "" {
		return errors.New("completed target path is unavailable")
	}
	st, err := os.Stat(rec.To)
	if err != nil {
		return fmt.Errorf("completed target is unavailable: %w", err)
	}
	if !st.Mode().IsRegular() {
		return errors.New("completed target is not a regular file")
	}
	return nil
}

func fileFlowMetadataUpdateID(rec MoveRecord) string {
	if id := strings.TrimSpace(rec.JournalID); id != "" {
		return "metadata:journal:" + id
	}
	sum := sha256.Sum256([]byte("metadata\x00" + filepath.Clean(rec.From) + "\x00" + filepath.Clean(rec.To)))
	return "metadata:move:" + hex.EncodeToString(sum[:16])
}

func queueFileFlowMetadataUpdate(path string, rec MoveRecord, draft FileFlowMetadataDraft, now time.Time) (bool, error) {
	draft = draft.Normalized()
	if draft.Empty() {
		return false, errors.New("no FileFlow metadata is staged")
	}
	if err := validateFileFlowMetadataTarget(rec); err != nil {
		return false, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return false, errors.New("FileFlow handoff queue path is unavailable")
	}
	id := fileFlowMetadataUpdateID(rec)
	existing, err := loadFileFlowHandoffs(path)
	if err != nil {
		return false, err
	}
	for _, event := range existing {
		if event.ID == id {
			return false, nil
		}
	}
	event := FileFlowHandoffEvent{
		Version:      fileFlowHandoffVersion,
		Action:       fileFlowActionEnsureMetadata,
		ID:           id,
		CreatedAt:    now,
		OriginalPath: filepath.Clean(rec.From),
		CurrentPath:  filepath.Clean(rec.To),
		JournalID:    strings.TrimSpace(rec.JournalID),
		Kind:         rec.Kind,
		DisplayName:  filepath.Base(rec.To),
		Tags:         append([]string(nil), draft.Tags...),
		Todos:        append([]string(nil), draft.Todos...),
		Notes:        append([]string(nil), draft.Notes...),
	}
	data, err := json.Marshal(event)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return false, err
	}
	if _, err = f.Write(append(data, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
