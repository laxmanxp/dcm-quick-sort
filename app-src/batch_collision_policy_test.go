package main

import "testing"

func TestBatchCollisionChoice(t *testing.T) {
	cases := []struct { button int; want batchCollisionDecision }{
		{6, batchCollisionRename},
		{7, batchCollisionSkip},
		{2, batchCollisionCancel},
		{0, batchCollisionCancel},
		{-1, batchCollisionCancel},
	}
	for _, tc := range cases {
		if got := batchCollisionChoice(tc.button); got != tc.want {
			t.Errorf("button %d: got %d want %d", tc.button, got, tc.want)
		}
	}
}

func TestBatchCancelledRemainderPreservesCompletedMoves(t *testing.T) {
	// Three moves already succeeded; the conflicting item is handled
	// separately, leaving two unstarted items marked as cancelled.
	completed, issues := batchCancelledRemainder(4, 1, 2)
	if completed != 6 || issues != 3 || completed-issues != 3 {
		t.Fatalf("completed=%d issues=%d moved=%d", completed, issues, completed-issues)
	}
}

func TestBatchCancelledRemainderEmpty(t *testing.T) {
	completed, issues := batchCancelledRemainder(1, 0, 0)
	if completed != 1 || issues != 0 {
		t.Fatalf("completed=%d issues=%d", completed, issues)
	}
}
