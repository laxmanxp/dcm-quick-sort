package main

import (
	"reflect"
	"testing"
)

func TestPreviewQuarterTurnNormalization(t *testing.T) {
	for _, tc := range []struct{ current, delta, want int }{
		{0, -1, 3}, {0, 1, 1}, {3, 1, 0}, {1, -1, 0},
		{2, 6, 0}, {0, -5, 3},
	} {
		if got := rotatePreviewQuarterTurns(tc.current, tc.delta); got != tc.want {
			t.Fatalf("rotate(%d,%d)=%d; want %d", tc.current, tc.delta, got, tc.want)
		}
	}
}

func TestRotatedPreviewDimensions(t *testing.T) {
	for _, tc := range []struct{ turns int; w, h int32 }{
		{0, 600, 900}, {1, 900, 600}, {2, 600, 900}, {3, 900, 600},
	} {
		w, h := rotatedPreviewDimensions(600, 900, tc.turns)
		if w != tc.w || h != tc.h {
			t.Fatalf("rotation %d dimensions = %dx%d; want %dx%d", tc.turns, w, h, tc.w, tc.h)
		}
	}
}

func TestPreviewRotationPointMapping(t *testing.T) {
	for _, tc := range []struct{ turns int; want [3]previewRotationPoint }{
		{0, [3]previewRotationPoint{{10, 20}, {310, 20}, {10, 220}}},
		{1, [3]previewRotationPoint{{310, 20}, {310, 220}, {10, 20}}},
		{2, [3]previewRotationPoint{{310, 220}, {10, 220}, {310, 20}}},
		{3, [3]previewRotationPoint{{10, 220}, {10, 20}, {310, 220}}},
	} {
		if got := previewRotationPoints(10, 20, 300, 200, tc.turns); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("rotation %d points = %#v; want %#v", tc.turns, got, tc.want)
		}
	}
}
