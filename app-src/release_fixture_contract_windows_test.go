//go:build windows

package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestReleaseWindowsFixtureContracts(t *testing.T) {
	t.Run("source-line-endings", func(t *testing.T) {
		data, err := os.ReadFile("main_windows.go")
		if err != nil {
			t.Fatal(err)
		}
		want := strings.ReplaceAll(string(data), "\r\n", "\n")
		t.Logf("checkout CRLF lines=%d; LF-normalized Clear Filters wiring present=%t", strings.Count(string(data), "\r\n"), strings.Contains(want, "case ID_FILTER_CLEAR:\n\t\t\t\tclearInboxFilters()"))
		if got := windowsMainSource(t); got != want {
			t.Fatal("source fixture must normalize checkout CRLF without changing any other source bytes")
		}
	})
	t.Run("explicit-created-time", func(t *testing.T) {
		want := time.Date(2026, 9, 22, 8, 0, 0, 0, time.Local)
		path := writeInboxFixture(t, t.TempDir(), "created.txt", 1, want)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		created := fileCreatedTime(path, info)
		t.Logf("creation=%s modification=%s requested=%s", created.Format(time.RFC3339Nano), info.ModTime().Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
		if !created.Equal(want) || !info.ModTime().Equal(want) {
			t.Fatal("date-sorted fixture must establish and verify creation and modification timestamps independently")
		}
	})
}
