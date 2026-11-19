package main

import "testing"

func TestActiveSourceSelectionIndex(t *testing.T) {
	cases := []struct {
		name         string
		caret, total int
		selected     []int32
		want         int
	}{
		{"single", 2, 6, []int32{2}, 2},
		{"ctrl-remove-other-to-single", 0, 6, []int32{2}, 2},
		{"ctrl-remove-last", 2, 6, nil, -1},
		{"multi-caret-selected", 4, 6, []int32{1, 3, 4}, 4},
		{"multi-caret-deselected", 2, 6, []int32{1, 3, 4}, 1},
		{"missing-caret", -1, 6, []int32{2, 4}, 2},
		{"outside-caret", 6, 6, []int32{0, 5}, 0},
		{"empty-list", 0, 0, nil, -1},
		{"invalid-indexes", 8, 6, []int32{-1, 8}, -1},
		{"skip-invalid-indexes", 8, 6, []int32{-1, 8, 3}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := activeSourceSelectionIndex(tc.caret, tc.selected, tc.total); got != tc.want {
				t.Fatalf("current index=%d want %d; caret=%d selected=%v", got, tc.want, tc.caret, tc.selected)
			}
		})
	}
}
