package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteBackgroundCopyMoveSuccess(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "incoming.bin")
	target := filepath.Join(dstDir, "sorted.bin")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := executeBackgroundCopyMove(src, target, copyFileVerified, retireVerifiedSource)
	if got.Status != BackgroundMoveSucceeded {
		t.Fatalf("status=%q err=%v", got.Status, got.Err)
	}
	if got.Record.Kind != MoveKindMove || got.Record.From != src || got.Record.To != target {
		t.Fatalf("record=%#v", got.Record)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be removed, err=%v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "payload" {
		t.Fatalf("target data=%q err=%v", data, err)
	}
}

func TestExecuteBackgroundCopyMoveRemoveFailureKeepsSafeCopy(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "incoming.bin")
	target := filepath.Join(dstDir, "sorted.bin")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	removeErr := errors.New("source is locked")
	got := executeBackgroundCopyMove(src, target, copyFileVerified, func(string, string, sourceSnapshot) error { return removeErr })
	if got.Status != BackgroundMoveCopiedSourceRetained {
		t.Fatalf("status=%q err=%v", got.Status, got.Err)
	}
	if !errors.Is(got.Err, removeErr) {
		t.Fatalf("err=%v", got.Err)
	}
	if got.Record != (MoveRecord{}) {
		t.Fatalf("a retained source must not create an undo move record: %#v", got.Record)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source must remain: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "payload" {
		t.Fatalf("safe copied target missing data=%q err=%v", data, err)
	}
}

func TestExecuteBackgroundCopyMoveCopyFailureLeavesSource(t *testing.T) {
	d := t.TempDir()
	src := filepath.Join(d, "incoming.bin")
	target := filepath.Join(d, "sorted.bin")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	copyErr := errors.New("copy failed")
	removed := false
	got := executeBackgroundCopyMove(src, target, func(string, string) error { return copyErr }, func(string, string, sourceSnapshot) error {
		removed = true
		return nil
	})
	if got.Status != BackgroundMoveFailed || !errors.Is(got.Err, copyErr) {
		t.Fatalf("status=%q err=%v", got.Status, got.Err)
	}
	if removed {
		t.Fatal("source removal must not run after a failed copy")
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source must remain: %v", err)
	}
}

func TestStartBackgroundCopyMoveReturnsBeforeWorkCompletes(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
	if err := os.WriteFile(source, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	copyFn := func(source, target string) error {
		close(entered)
		<-release
		return copyFileVerified(source, target)
	}

	resultCh := startBackgroundCopyMove(source, target, copyFn, retireVerifiedSource)

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("background worker did not start")
	}

	select {
	case result := <-resultCh:
		t.Fatalf("worker completed before release: %#v", result)
	default:
		// Expected: caller is free while the worker is still blocked.
	}

	close(release)
	select {
	case result := <-resultCh:
		if result.Status != BackgroundMoveSucceeded {
			t.Fatalf("status=%q err=%v", result.Status, result.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("background worker did not complete")
	}
}

func TestBackgroundCopyMoveWithProgressForwardsProgress(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.bin"), filepath.Join(root, "target.bin")
	if err := os.WriteFile(source, make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	var points [][2]int64
	results := startBackgroundCopyMoveWithProgress(source, target, func(source, target string, progress CopyProgressFunc) error {
		progress(0, 100)
		progress(50, 100)
		progress(100, 100)
		return copyFileVerified(source, target)
	}, retireVerifiedSource, func(copied, total int64) {
		points = append(points, [2]int64{copied, total})
	})
	got := <-results
	if got.Status != BackgroundMoveSucceeded {
		t.Fatalf("status=%s err=%v", got.Status, got.Err)
	}
	if len(points) != 3 || points[1] != [2]int64{50, 100} || points[2] != [2]int64{100, 100} {
		t.Fatalf("progress=%v", points)
	}
}
