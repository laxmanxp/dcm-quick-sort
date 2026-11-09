package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLearningFeaturesUseAggregatesNotWholeFilename(t *testing.T) {
	features := learningFeaturesForPath(filepath.Join(`C:\Users\Test\Downloads`, "5th_SFC_GIA_2026.pdf"))
	want := map[string]bool{
		"all":            true,
		"ext:.pdf":       true,
		"type:documents": true,
		"token:5th":      true,
		"token:sfc":      true,
		"token:gia":      true,
		"token:2026":     true,
		"pair:5th+sfc":   true,
	}
	got := map[string]bool{}
	for _, f := range features {
		got[f] = true
	}
	for f := range want {
		if !got[f] {
			t.Fatalf("missing feature %q in %#v", f, features)
		}
	}
	for _, f := range features {
		if f == "5th_sfc_gia_2026" || f == "5th_sfc_gia_2026.pdf" {
			t.Fatalf("whole filename should not be stored as a learning feature: %q", f)
		}
	}
}

func TestDetectRenameStyle(t *testing.T) {
	date := time.Date(2026, 9, 22, 0, 0, 0, 0, time.Local)
	cases := []struct{ original, target, want string }{
		{"invoice_report", "invoice_report", RenameStyleUnchanged},
		{"invoice_report", "invoice report", RenameStyleCleanSpacing},
		{"invoice report", "Invoice Report", RenameStyleTitleCase},
		{"Invoice Report", "invoice report", RenameStyleLower},
		{"Invoice Report", "INVOICE REPORT", RenameStyleUpper},
		{"Invoice Report", "2026-09-22 - Invoice Report", RenameStyleDatePrefix},
		{"Invoice Report", "Quarterly Accounts", RenameStyleCustom},
	}
	for _, tc := range cases {
		if got := detectRenameStyle(tc.original, tc.target, date); got != tc.want {
			t.Fatalf("detectRenameStyle(%q,%q)=%q want %q", tc.original, tc.target, got, tc.want)
		}
	}
}

func TestLearningObservationPersistsDestinationAndRenamePatterns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "learning.json")
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	rec := MoveRecord{From: filepath.Join(dir, "5th_SFC_order.pdf"), To: filepath.Join(dir, "Office", "5th SFC", "2026-09-22 - 5th SFC Order.pdf"), Kind: MoveKindMove}

	if err := observeCompletedMoveLearning(path, rec, now); err != nil {
		t.Fatal(err)
	}
	model, err := loadLearningModel(path)
	if err != nil {
		t.Fatal(err)
	}
	if model.TotalObservations != 1 {
		t.Fatalf("TotalObservations=%d", model.TotalObservations)
	}
	dest := filepath.Clean(filepath.Dir(rec.To))
	if model.DestinationCounts[dest] != 1 {
		t.Fatalf("destination count=%d", model.DestinationCounts[dest])
	}
	if model.FeatureDestinations["token:sfc"][dest] != 1 {
		t.Fatalf("token destination count missing: %#v", model.FeatureDestinations)
	}
	if model.RenameStyleCounts[RenameStyleDatePrefix] != 1 {
		t.Fatalf("rename style counts=%#v", model.RenameStyleCounts)
	}
}

func TestDuplicateRemovalDoesNotTrainLearning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning.json")
	rec := MoveRecord{From: "duplicate.pdf", To: "kept.pdf", Kind: MoveKindDuplicateRemoved}
	if err := observeCompletedMoveLearning(path, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	model, err := loadLearningModel(path)
	if err != nil {
		t.Fatal(err)
	}
	if model.TotalObservations != 0 {
		t.Fatalf("duplicate removal trained model: %#v", model)
	}
}

func TestUndoUnlearnsObservation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning.json")
	rec := MoveRecord{From: filepath.Join("inbox", "invoice_report.pdf"), To: filepath.Join("Accounts", "Invoice Report.pdf"), Kind: MoveKindMove}
	now := time.Now()
	if err := observeCompletedMoveLearning(path, rec, now); err != nil {
		t.Fatal(err)
	}
	if err := unobserveCompletedMoveLearning(path, rec, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	model, err := loadLearningModel(path)
	if err != nil {
		t.Fatal(err)
	}
	if model.TotalObservations != 0 {
		t.Fatalf("TotalObservations=%d", model.TotalObservations)
	}
	if len(model.DestinationCounts) != 0 || len(model.FeatureDestinations) != 0 || len(model.RenameStyleCounts) != 0 {
		t.Fatalf("expected empty aggregate maps after unlearning, got %#v", model)
	}
}

func TestPausedLearningDoesNotObserve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning.json")
	model := newLearningModel()
	model.Paused = true
	if err := saveLearningModel(path, model); err != nil {
		t.Fatal(err)
	}
	rec := MoveRecord{From: "invoice.pdf", To: filepath.Join("Accounts", "invoice.pdf"), Kind: MoveKindMove}
	if err := observeCompletedMoveLearning(path, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, err := loadLearningModel(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalObservations != 0 {
		t.Fatalf("paused model observed a move: %#v", got)
	}
}

func TestCorruptLearningFileIsNotSilentlyOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := MoveRecord{From: "invoice.pdf", To: filepath.Join("Accounts", "invoice.pdf"), Kind: MoveKindMove}
	if err := observeCompletedMoveLearning(path, rec, time.Now()); err == nil {
		t.Fatal("expected corrupt learning model error")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "{broken" {
		t.Fatalf("corrupt learning file was overwritten: %q", data)
	}
}

func TestLearningIgnoresSameDirectoryRenameTransaction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "learning.json")
	rec := MoveRecord{
		From: filepath.Join(dir, "scan.pdf"),
		To:   filepath.Join(dir, "Road Estimate.pdf"),
		Kind: MoveKindMove,
	}
	if err := observeCompletedMoveLearning(path, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
	model, err := loadLearningModel(path)
	if err != nil {
		t.Fatal(err)
	}
	if model.TotalObservations != 0 {
		t.Fatalf("rename-only transaction must not train move suggestions; got %d observations", model.TotalObservations)
	}
}
