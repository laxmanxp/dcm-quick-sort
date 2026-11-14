package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const learningModelVersion = 1

const (
	RenameStyleUnchanged    = "unchanged"
	RenameStyleCleanSpacing = "clean-spacing"
	RenameStyleTitleCase    = "title-case"
	RenameStyleLower        = "lowercase"
	RenameStyleUpper        = "uppercase"
	RenameStyleDatePrefix   = "date-prefix"
	RenameStyleCustom       = "custom"
)

type LearningModel struct {
	Version             int                       `json:"version"`
	Paused              bool                      `json:"paused,omitempty"`
	TotalObservations   int                       `json:"total_observations"`
	UpdatedAt           time.Time                 `json:"updated_at,omitempty"`
	DestinationCounts   map[string]int            `json:"destination_counts"`
	RenameStyleCounts   map[string]int            `json:"rename_style_counts"`
	FeatureDestinations map[string]map[string]int `json:"feature_destinations"`
	FeatureRenameStyles map[string]map[string]int `json:"feature_rename_styles"`
}

func newLearningModel() LearningModel {
	return LearningModel{
		Version:             learningModelVersion,
		DestinationCounts:   make(map[string]int),
		RenameStyleCounts:   make(map[string]int),
		FeatureDestinations: make(map[string]map[string]int),
		FeatureRenameStyles: make(map[string]map[string]int),
	}
}

func normalizeLearningModel(m *LearningModel) {
	if m.Version == 0 {
		m.Version = learningModelVersion
	}
	if m.DestinationCounts == nil {
		m.DestinationCounts = make(map[string]int)
	}
	if m.RenameStyleCounts == nil {
		m.RenameStyleCounts = make(map[string]int)
	}
	if m.FeatureDestinations == nil {
		m.FeatureDestinations = make(map[string]map[string]int)
	}
	if m.FeatureRenameStyles == nil {
		m.FeatureRenameStyles = make(map[string]map[string]int)
	}
}

func loadLearningModel(path string) (LearningModel, error) {
	m := newLearningModel()
	path = strings.TrimSpace(path)
	if path == "" {
		return m, errors.New("learning file path is unavailable")
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return newLearningModel(), fmt.Errorf("learning model is corrupt: %w", err)
	}
	if m.Version != learningModelVersion {
		return newLearningModel(), fmt.Errorf("unsupported learning model version %d", m.Version)
	}
	normalizeLearningModel(&m)
	return m, nil
}

func saveLearningModel(path string, m LearningModel) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("learning file path is unavailable")
	}
	normalizeLearningModel(&m)
	m.Version = learningModelVersion
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	// Keep the current model intact until the replacement is fully written.
	// os.Rename is atomic on the supported local filesystems for this same-dir temp file.
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func learningTokensForPath(path string) []string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	fields := strings.FieldsFunc(strings.ToLower(stem), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := make(map[string]bool)
	out := make([]string, 0, len(fields))
	for _, token := range fields {
		token = strings.TrimSpace(token)
		if len([]rune(token)) < 2 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func learningFeaturesForPath(path string) []string {
	features := []string{"all"}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		features = append(features, "ext:"+ext)
	}
	if category := categoryForPath(path); category != "" {
		features = append(features, "type:"+strings.ToLower(string(category)))
	} else {
		features = append(features, "type:other / unknown")
	}
	srcDir := filepath.Clean(filepath.Dir(path))
	if srcDir != "." && strings.TrimSpace(srcDir) != "" {
		features = append(features, "source:"+strings.ToLower(srcDir))
	}
	tokens := learningTokensForPath(path)
	for _, token := range tokens {
		features = append(features, "token:"+token)
	}
	// Co-occurrence pairs are more discriminating than a single broad token.
	// Cap the pair source to avoid quadratic growth from pathological filenames.
	pairTokens := tokens
	if len(pairTokens) > 8 {
		pairTokens = pairTokens[:8]
	}
	for i := 0; i < len(pairTokens); i++ {
		for j := i + 1; j < len(pairTokens); j++ {
			a, b := pairTokens[i], pairTokens[j]
			if b < a {
				a, b = b, a
			}
			features = append(features, "pair:"+a+"+"+b)
		}
	}
	return features
}

var learningDatePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(?:\s*-\s*|\s+)`)

func detectRenameStyle(original, target string, date time.Time) string {
	original = strings.TrimSpace(original)
	target = strings.TrimSpace(target)
	if target == original {
		return RenameStyleUnchanged
	}
	if learningDatePrefix.MatchString(target) {
		return RenameStyleDatePrefix
	}
	if target == cleanRenameSpacing(original) {
		return RenameStyleCleanSpacing
	}
	if target == titleCaseFilename(original) || target == titleCaseFilename(cleanRenameSpacing(original)) {
		return RenameStyleTitleCase
	}
	if target == strings.ToLower(original) {
		return RenameStyleLower
	}
	if target == strings.ToUpper(original) {
		return RenameStyleUpper
	}
	return RenameStyleCustom
}

func learningObservation(rec MoveRecord, now time.Time) (features []string, destination, style string, ok bool) {
	if rec.Kind == MoveKindDuplicateRemoved {
		return nil, "", "", false
	}
	if strings.TrimSpace(rec.From) == "" || strings.TrimSpace(rec.To) == "" {
		return nil, "", "", false
	}
	// A Rename button action is represented by the same durable move journal
	// record as an ordinary filesystem move, but it stays in the same folder.
	// Do not let these rename-only transactions train destination suggestions.
	if filepath.Clean(filepath.Dir(rec.From)) == filepath.Clean(filepath.Dir(rec.To)) {
		return nil, "", "", false
	}
	destination = filepath.Clean(filepath.Dir(rec.To))
	if destination == "." || strings.TrimSpace(destination) == "" {
		return nil, "", "", false
	}
	features = learningFeaturesForPath(rec.From)
	originalStem := strings.TrimSuffix(filepath.Base(rec.From), filepath.Ext(rec.From))
	targetStem := strings.TrimSuffix(filepath.Base(rec.To), filepath.Ext(rec.To))
	style = detectRenameStyle(originalStem, targetStem, now)
	return features, destination, style, true
}

func incrementNested(m map[string]map[string]int, feature, choice string, delta int) {
	if delta == 0 {
		return
	}
	inner := m[feature]
	if inner == nil {
		if delta < 0 {
			return
		}
		inner = make(map[string]int)
		m[feature] = inner
	}
	inner[choice] += delta
	if inner[choice] <= 0 {
		delete(inner, choice)
	}
	if len(inner) == 0 {
		delete(m, feature)
	}
}

func incrementFlat(m map[string]int, key string, delta int) {
	if delta == 0 {
		return
	}
	m[key] += delta
	if m[key] <= 0 {
		delete(m, key)
	}
}

func observeCompletedMoveLearning(path string, rec MoveRecord, now time.Time) error {
	model, err := loadLearningModel(path)
	if err != nil {
		return err
	}
	if model.Paused {
		return nil
	}
	features, dest, style, ok := learningObservation(rec, now)
	if !ok {
		return nil
	}
	model.TotalObservations++
	model.UpdatedAt = now
	incrementFlat(model.DestinationCounts, dest, 1)
	incrementFlat(model.RenameStyleCounts, style, 1)
	for _, feature := range features {
		incrementNested(model.FeatureDestinations, feature, dest, 1)
		incrementNested(model.FeatureRenameStyles, feature, style, 1)
	}
	return saveLearningModel(path, model)
}

func unobserveCompletedMoveLearning(path string, rec MoveRecord, now time.Time) error {
	model, err := loadLearningModel(path)
	if err != nil {
		return err
	}
	features, dest, style, ok := learningObservation(rec, now)
	if !ok || model.TotalObservations <= 0 {
		return nil
	}
	model.TotalObservations--
	model.UpdatedAt = now
	incrementFlat(model.DestinationCounts, dest, -1)
	incrementFlat(model.RenameStyleCounts, style, -1)
	for _, feature := range features {
		incrementNested(model.FeatureDestinations, feature, dest, -1)
		incrementNested(model.FeatureRenameStyles, feature, style, -1)
	}
	return saveLearningModel(path, model)
}

func setLearningPaused(path string, paused bool, now time.Time) error {
	model, err := loadLearningModel(path)
	if err != nil {
		return err
	}
	model.Paused = paused
	model.UpdatedAt = now
	return saveLearningModel(path, model)
}

func resetLearningModel(path string, now time.Time) error {
	existing, err := loadLearningModel(path)
	if err != nil {
		return err
	}
	fresh := newLearningModel()
	fresh.Paused = existing.Paused
	fresh.UpdatedAt = now
	return saveLearningModel(path, fresh)
}

func learningFeatureTotal(choices map[string]int) int {
	total := 0
	for _, n := range choices {
		total += n
	}
	return total
}

func sortedLearningKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
