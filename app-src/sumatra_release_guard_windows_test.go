//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

// Faults are injected only into test-local session state: an invalid wait handle,
// or an unsignalled event plus invalid job. No real viewer/ACL/runtime is changed.
func TestSumatraReleaseFailurePreservesOperations(t *testing.T) {
	for _, fault := range []string{"wait-failed", "termination-failed"} {
		for _, operation := range []string{"rename", "move", "undo"} {
			t.Run(fault+"/"+operation, func(t *testing.T) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				oldApp, oldViewer := app, sumatraView
				defer func() { app = oldApp; sumatraView = oldViewer }()
				root := t.TempDir()
				t.Setenv("LOCALAPPDATA", filepath.Join(root, "profile"))
				src, dst := filepath.Join(root, "source.pdf"), filepath.Join(root, "target.pdf")
				data := []byte("test-local bytes; must not change")
				present, absent := src, dst
				if operation == "undo" {
					present, absent = dst, src
				}
				if err := os.WriteFile(present, data, 0600); err != nil {
					t.Fatal(err)
				}
				rec := MoveRecord{From: src, To: dst, Kind: MoveKindMove, JournalID: "release-guard"}
				entry := MoveJournalEntry{ID: rec.JournalID, State: MoveJournalCompleted, Record: rec}
				raw, err := json.Marshal(entry)
				if err != nil {
					t.Fatal(err)
				}
				raw = append(raw, '\n')
				journal := moveJournalFilePath()
				if err = os.MkdirAll(filepath.Dir(journal), 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(journal, raw, 0600); err != nil {
					t.Fatal(err)
				}
				app = appState{index: -1, history: []MoveRecord{rec}}
				v := &sumatraSession{profile: filepath.Join(root, "owned-profile")}
				if err = os.Mkdir(v.profile, 0700); err != nil {
					t.Fatal(err)
				}
				if fault == "termination-failed" {
					h, _, e := kernel32.NewProc("CreateEventW").Call(0, 1, 0, 0)
					if h == 0 {
						t.Fatalf("create test event: %v", e)
					}
					defer spClose.Call(h)
					v.process = h // waits out; job=0 makes termination fail closed
				}
				sumatraView = v
				if operation == "undo" {
					list := createControl("LISTBOX", "", 0, 0, 0, 200, 100, 0, 0)
					if list == 0 {
						t.Fatal("hidden test list could not be created")
					}
					defer user32.NewProc("DestroyWindow").Call(list)
					app.processedList = list
					app.processedRows = []processedViewRow{{HistoryIndex: 0, Label: "owned test"}}
					procSendMessageW.Call(list, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr("owned test"))))
					procSendMessageW.Call(list, LB_SETCURSEL, 0, 0)
					if selectedProcessedIndex() != 0 {
						t.Fatal("test Undo record not selected")
					}
					undoSelectedProcessed()
				} else {
					mode := fileOperationRename
					if operation == "move" {
						mode = fileOperationMove
					}
					beginFileOperation(mode, src, dst)
				}
				if sumatraView != v {
					t.Fatal("unconfirmed session discarded")
				}
				if app.pendingFileOperation != fileOperationNone || app.backgroundMoveBusy {
					t.Fatal("operation admitted despite release failure")
				}
				if !strings.Contains(app.fallback, "PDF preview unavailable") {
					t.Fatal("release error not surfaced")
				}
				if !reflect.DeepEqual(app.history, []MoveRecord{rec}) {
					t.Fatal("Undo history changed")
				}
				got, err := os.ReadFile(present)
				if err != nil || !bytes.Equal(got, data) {
					t.Fatalf("file changed: %v", err)
				}
				if _, err = os.Stat(absent); !os.IsNotExist(err) {
					t.Fatal("operation changed destination/source")
				}
				got, err = os.ReadFile(journal)
				if err != nil || !bytes.Equal(got, raw) {
					t.Fatal("journal changed")
				}
				if _, err = os.Stat(v.profile); err != nil {
					t.Fatal("unconfirmed profile removed")
				}
			})
		}
	}
}
