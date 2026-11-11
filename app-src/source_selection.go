package main

// activeSourceSelectionIndex chooses a selected file for the preview/draft.
// The native keyboard caret may remain on a Ctrl-deselected row; it must not
// turn that row into an operation target. No selection means no current file.
func activeSourceSelectionIndex(caret int, selected []int32, total int) int {
	first := -1
	for _, raw := range selected {
		index := int(raw)
		if index < 0 || index >= total {
			continue
		}
		if index == caret {
			return index
		}
		if first < 0 {
			first = index
		}
	}
	return first
}
