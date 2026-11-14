//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

func populateSourceAliases() {
	if app.sourceAliasCombo == 0 {
		return
	}
	app.updatingSourceAlias = true
	procSendMessageW.Call(app.sourceAliasCombo, CB_RESETCONTENT, 0, 0)
	comboAddItem(app.sourceAliasCombo, "Custom path")
	for _, alias := range app.sourceAliases {
		comboAddItem(app.sourceAliasCombo, alias.Name)
	}
	app.updatingSourceAlias = false
	syncSourceAliasSelection(strings.TrimSpace(getText(app.sourceEdit)))
}

func syncSourceAliasSelection(path string) {
	if app.sourceAliasCombo == 0 || app.updatingSourceAlias {
		return
	}
	idx := sourceAliasIndexForPath(app.sourceAliases, path)
	selection := uintptr(0)
	if idx >= 0 {
		selection = uintptr(idx + 1)
	}
	app.updatingSourceAlias = true
	procSendMessageW.Call(app.sourceAliasCombo, CB_SETCURSEL, selection, 0)
	app.updatingSourceAlias = false
}

func selectedSourceAlias() (SourceAlias, bool) {
	if app.sourceAliasCombo == 0 {
		return SourceAlias{}, false
	}
	idx, _, _ := procSendMessageW.Call(app.sourceAliasCombo, CB_GETCURSEL, 0, 0)
	if idx == ^uintptr(0) || idx == 0 {
		return SourceAlias{}, false
	}
	ai := int(idx) - 1
	if ai < 0 || ai >= len(app.sourceAliases) {
		return SourceAlias{}, false
	}
	return app.sourceAliases[ai], true
}

func reloadSourceAliases() {
	defaultPath := strings.TrimSpace(getText(app.sourceEdit))
	aliases, err := loadSourceAliases(defaultPath)
	if err != nil {
		setText(app.statusLabel, "Could not load saved sources: "+err.Error())
		return
	}
	app.sourceAliases = aliases
	populateSourceAliases()
	_, _ = backupCurrentConfiguration("configuration-loaded")
}

func setSourceText(path string) {
	app.updatingSourceSuggestions = true
	setText(app.sourceEdit, path)
	app.updatingSourceSuggestions = false
	moveEditCaretToEnd(app.sourceEdit)
	hideSourceSuggestions()
	syncSourceAliasSelection(path)
}

func activateSourcePath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		setText(app.statusLabel, "Source folder is required.")
		return false
	}
	cleaned := filepath.Clean(path)
	st, err := os.Stat(cleaned)
	if err != nil || !st.IsDir() {
		setText(app.statusLabel, "Source folder does not exist or is not a folder: "+path)
		return false
	}
	app.sourceDir = cleaned
	setSourceText(cleaned)
	reloadFiles()
	setText(app.statusLabel, "Source: "+cleaned)
	return true
}

func hideSourceSuggestions() {
	if app.sourceSuggestionList != 0 {
		procShowWindow.Call(app.sourceSuggestionList, 0)
	}
	app.sourceSuggestionsVisible = false
}

func updateSourceSuggestions() {
	typed := strings.TrimSpace(getText(app.sourceEdit))
	app.sourceSuggestions = nil
	procSendMessageW.Call(app.sourceSuggestionList, LB_RESETCONTENT, 0, 0)
	if typed == "" {
		hideSourceSuggestions()
		return
	}
	seen := map[string]bool{}
	low := strings.ToLower(typed)
	for _, alias := range app.sourceAliases {
		if strings.HasPrefix(strings.ToLower(alias.Name), low) || strings.HasPrefix(strings.ToLower(alias.Path), low) {
			key := strings.ToLower(filepath.Clean(alias.Path))
			if !seen[key] {
				seen[key] = true
				app.sourceSuggestions = append(app.sourceSuggestions, alias.Path)
				procSendMessageW.Call(app.sourceSuggestionList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(alias.Name+"    →    "+alias.Path))))
			}
		}
	}
	if len(app.sourceSuggestions) < 12 {
		for _, suggestion := range folderSuggestions(typed, 12-len(app.sourceSuggestions)) {
			key := strings.ToLower(filepath.Clean(suggestion))
			if seen[key] {
				continue
			}
			seen[key] = true
			app.sourceSuggestions = append(app.sourceSuggestions, suggestion)
			procSendMessageW.Call(app.sourceSuggestionList, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(suggestion))))
		}
	}
	if len(app.sourceSuggestions) == 0 {
		hideSourceSuggestions()
		return
	}
	procSendMessageW.Call(app.sourceSuggestionList, LB_SETCURSEL, ^uintptr(0), 0)
	procShowWindow.Call(app.sourceSuggestionList, SW_SHOW)
	app.sourceSuggestionsVisible = true
}

func focusWithinSource() bool {
	focus, _, _ := procGetFocus.Call()
	return focus == app.sourceEdit || focus == app.sourceSuggestionList
}

func selectedSourceSuggestion() (string, bool) {
	idx, _, _ := procSendMessageW.Call(app.sourceSuggestionList, LB_GETCURSEL, 0, 0)
	if idx == LB_ERR || int(idx) < 0 || int(idx) >= len(app.sourceSuggestions) {
		return "", false
	}
	return app.sourceSuggestions[int(idx)], true
}

func acceptSourceSuggestion() bool {
	p, ok := selectedSourceSuggestion()
	if !ok {
		return false
	}
	return activateSourcePath(p)
}

func moveSourceSuggestionSelection(delta int) bool {
	if !app.sourceSuggestionsVisible || len(app.sourceSuggestions) == 0 {
		return false
	}
	idx, _, _ := procSendMessageW.Call(app.sourceSuggestionList, LB_GETCURSEL, 0, 0)
	current := -1
	if idx != LB_ERR {
		current = int(idx)
	}
	if delta > 0 {
		current++
	} else {
		current--
	}
	if current < 0 {
		current = 0
	}
	if current >= len(app.sourceSuggestions) {
		current = len(app.sourceSuggestions) - 1
	}
	procSendMessageW.Call(app.sourceSuggestionList, LB_SETCURSEL, uintptr(current), 0)
	return true
}

func sourceStatus(alias SourceAlias) string { return fmt.Sprintf("%s → %s", alias.Name, alias.Path) }
