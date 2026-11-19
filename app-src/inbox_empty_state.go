package main

// inboxEmptyState describes the current view, without claiming that filtered
// files have been processed. Sorting alone does not narrow the view. No extra
// filesystem scan is needed, so this does not add work to the preview/UI thread.
func inboxEmptyState(filter InboxFilter, listedFiles int) (heading, hint string) {
	if listedFiles > 0 {
		return "No file selected", "Select a file from the list"
	}
	filter = normalizedInboxFilter(filter)
	if filter.Category != CategoryAll || filter.Extension != "All extensions" ||
		filter.NameContains != "" || filter.Age != InboxAgeAll || filter.Size != InboxSizeAll {
		return "No files match the current filters", "Clear or change filters to show files"
	}
	return "No files remaining", "Folder complete"
}
