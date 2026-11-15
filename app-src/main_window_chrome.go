package main

// Keep the normal Windows caption, system menu, minimize, maximize and sizing
// frame. Only the main window uses this style; tool dialogs remain fixed-size.
func mainWindowChromeStyle(overlapped, visible, clipChildren uint32) uint32 {
	return overlapped | visible | clipChildren
}

// The layout requires a usable preview and two sidebars. These are outer-window
// pixels (not client pixels); the headed test verifies the work-area boundary.
func minimumMainWindowTrackSize() (int32, int32) {
	return 1100, 740
}
