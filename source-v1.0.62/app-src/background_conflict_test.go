package main

import (
	"errors"
	"testing"
	"time"
)

func TestBackgroundConflictInspectionReturnsBeforeInspectorFinishes(t *testing.T) {
	gate := make(chan struct{})
	started := make(chan struct{})
	results := startBackgroundConflictInspection("a", "b", func(source, target string) (DestinationConflictInfo, error) {
		close(started)
		<-gate
		return DestinationConflictInfo{Kind: ConflictDifferent, Source: source, Target: target}, nil
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("inspector never started")
	}
	select {
	case <-results:
		t.Fatal("background inspection completed before gate was released")
	default:
	}
	close(gate)
	select {
	case got := <-results:
		if got.Err != nil || got.Info.Kind != ConflictDifferent || got.Source != "a" || got.Target != "b" {
			t.Fatalf("unexpected result: %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("background inspection did not complete")
	}
}

func TestBackgroundConflictInspectionPropagatesError(t *testing.T) {
	want := errors.New("hash failed")
	got := <-startBackgroundConflictInspection("a", "b", func(string, string) (DestinationConflictInfo, error) {
		return DestinationConflictInfo{}, want
	})
	if !errors.Is(got.Err, want) {
		t.Fatalf("err=%v want %v", got.Err, want)
	}
}

func TestBackgroundConflictActionReturnsBeforeActionFinishes(t *testing.T) {
	gate := make(chan struct{})
	started := make(chan struct{})
	results := startBackgroundConflictAction(BackgroundConflictReplace, "a", "b", func() (MoveRecord, error) {
		close(started)
		<-gate
		return MoveRecord{From: "a", To: "b", Kind: MoveKindReplace}, nil
	})
	<-started
	select {
	case <-results:
		t.Fatal("background action completed before gate was released")
	default:
	}
	close(gate)
	got := <-results
	if got.Err != nil || got.Record.Kind != MoveKindReplace || got.Kind != BackgroundConflictReplace {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestBackgroundConflictActionPreservesFailure(t *testing.T) {
	want := errors.New("replace failed")
	got := <-startBackgroundConflictAction(BackgroundConflictReplace, "a", "b", func() (MoveRecord, error) {
		return MoveRecord{From: "a", To: "b", Kind: MoveKindReplace}, want
	})
	if !errors.Is(got.Err, want) || got.Record.Kind != MoveKindReplace {
		t.Fatalf("unexpected result: %#v", got)
	}
}
