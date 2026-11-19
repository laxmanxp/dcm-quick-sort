package main

import "testing"

func TestD21EmptyStateMessages(t *testing.T) {
	for _, tc := range []struct {
		name          string
		filter        InboxFilter
		listed        int
		heading, hint string
	}{
		{"default-empty", InboxFilter{}, 0, "No files remaining", "Folder complete"},
		{"all-filters-cleared", InboxFilter{Category: CategoryAll, Extension: "All extensions", Age: InboxAgeAll, Size: InboxSizeAll}, 0, "No files remaining", "Folder complete"},
		{"whitespace-name", InboxFilter{NameContains: " \t "}, 0, "No files remaining", "Folder complete"},
		{"sort-is-not-filter", InboxFilter{Sort: InboxSortName}, 0, "No files remaining", "Folder complete"},
		{"name", InboxFilter{NameContains: "absent"}, 0, "No files match the current filters", "Clear or change filters to show files"},
		{"category", InboxFilter{Category: CategoryImages}, 0, "No files match the current filters", "Clear or change filters to show files"},
		{"extension", InboxFilter{Extension: ".pdf"}, 0, "No files match the current filters", "Clear or change filters to show files"},
		{"age", InboxFilter{Age: InboxAgeLast24Hours}, 0, "No files match the current filters", "Clear or change filters to show files"},
		{"size", InboxFilter{Size: InboxSize100MiBPlus}, 0, "No files match the current filters", "Clear or change filters to show files"},
		{"matches-exist-no-selection", InboxFilter{NameContains: "alpha"}, 2, "No file selected", "Select a file from the list"},
		{"unfiltered-list-exhausted", InboxFilter{}, 2, "No file selected", "Select a file from the list"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			heading, hint := inboxEmptyState(tc.filter, tc.listed)
			if heading != tc.heading || hint != tc.hint {
				t.Fatalf("got (%q,%q), want (%q,%q)", heading, hint, tc.heading, tc.hint)
			}
		})
	}
}
