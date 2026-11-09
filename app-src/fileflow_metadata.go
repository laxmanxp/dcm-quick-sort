package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const fileFlowDraftStoreVersion = 1

type FileFlowMetadataDraft struct {
	Tags  []string `json:"tags,omitempty"`
	Todos []string `json:"todos,omitempty"`
	Notes []string `json:"notes,omitempty"`
}

func (d FileFlowMetadataDraft) Empty() bool {
	return len(d.Tags) == 0 && len(d.Todos) == 0 && len(d.Notes) == 0
}

func cleanMetadataValue(v string) string { return strings.TrimSpace(v) }

func addDistinctFold(dst []string, value string) []string {
	value = cleanMetadataValue(value)
	if value == "" {
		return dst
	}
	for _, existing := range dst {
		if strings.EqualFold(strings.TrimSpace(existing), value) {
			return dst
		}
	}
	return append(dst, value)
}

func (d *FileFlowMetadataDraft) AddTag(value string)  { d.Tags = addDistinctFold(d.Tags, value) }
func (d *FileFlowMetadataDraft) AddTodo(value string) { d.Todos = addDistinctFold(d.Todos, value) }
func (d *FileFlowMetadataDraft) AddNote(value string) { d.Notes = addDistinctFold(d.Notes, value) }

func (d FileFlowMetadataDraft) Normalized() FileFlowMetadataDraft {
	var out FileFlowMetadataDraft
	for _, v := range d.Tags {
		out.AddTag(v)
	}
	for _, v := range d.Todos {
		out.AddTodo(v)
	}
	for _, v := range d.Notes {
		out.AddNote(v)
	}
	return out
}

type fileFlowDraftStore struct {
	Version int                               `json:"version"`
	Drafts  map[string]fileFlowDraftStoreItem `json:"drafts"`
}

type fileFlowDraftStoreItem struct {
	Path  string                `json:"path"`
	Draft FileFlowMetadataDraft `json:"draft"`
}

func fileFlowDraftPathKey(path string) string {
	return strings.ToLower(filepath.Clean(strings.TrimSpace(path)))
}

func fileFlowDraftStorePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "fileflow-drafts.json")
	}
	return filepath.Join(os.TempDir(), "DCM Quick Sort", "fileflow-drafts.json")
}

func readFileFlowDraftStore(path string) (fileFlowDraftStore, error) {
	store := fileFlowDraftStore{Version: fileFlowDraftStoreVersion, Drafts: map[string]fileFlowDraftStoreItem{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return store, nil
	}
	if err != nil {
		return store, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return store, errors.New("FileFlow metadata draft store is corrupt: " + err.Error())
	}
	if store.Version != fileFlowDraftStoreVersion {
		return store, errors.New("FileFlow metadata draft store has an unsupported version")
	}
	if store.Drafts == nil {
		store.Drafts = map[string]fileFlowDraftStoreItem{}
	}
	return store, nil
}

func writeFileFlowDraftStore(path string, store fileFlowDraftStore) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("FileFlow metadata draft store path is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	store.Version = fileFlowDraftStoreVersion
	if store.Drafts == nil {
		store.Drafts = map[string]fileFlowDraftStoreItem{}
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func loadFileFlowDraft(storePath, filePath string) (FileFlowMetadataDraft, bool, error) {
	store, err := readFileFlowDraftStore(storePath)
	if err != nil {
		return FileFlowMetadataDraft{}, false, err
	}
	item, ok := store.Drafts[fileFlowDraftPathKey(filePath)]
	if !ok {
		return FileFlowMetadataDraft{}, false, nil
	}
	draft := item.Draft.Normalized()
	if draft.Empty() {
		return FileFlowMetadataDraft{}, false, nil
	}
	return draft, true, nil
}

func saveFileFlowDraft(storePath, filePath string, draft FileFlowMetadataDraft) error {
	filePath = filepath.Clean(strings.TrimSpace(filePath))
	if filePath == "." || filePath == "" {
		return errors.New("cannot stage FileFlow metadata without a file path")
	}
	store, err := readFileFlowDraftStore(storePath)
	if err != nil {
		return err
	}
	draft = draft.Normalized()
	key := fileFlowDraftPathKey(filePath)
	if draft.Empty() {
		delete(store.Drafts, key)
	} else {
		store.Drafts[key] = fileFlowDraftStoreItem{Path: filePath, Draft: draft}
	}
	return writeFileFlowDraftStore(storePath, store)
}

func clearFileFlowDraft(storePath, filePath string) error {
	store, err := readFileFlowDraftStore(storePath)
	if err != nil {
		return err
	}
	delete(store.Drafts, fileFlowDraftPathKey(filePath))
	return writeFileFlowDraftStore(storePath, store)
}

func migrateFileFlowDraft(storePath, from, to string) error {
	store, err := readFileFlowDraftStore(storePath)
	if err != nil {
		return err
	}
	fromKey := fileFlowDraftPathKey(from)
	item, ok := store.Drafts[fromKey]
	if !ok {
		return nil
	}
	delete(store.Drafts, fromKey)
	to = filepath.Clean(strings.TrimSpace(to))
	item.Path = to
	store.Drafts[fileFlowDraftPathKey(to)] = item
	return writeFileFlowDraftStore(storePath, store)
}
