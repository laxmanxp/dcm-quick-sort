//go:build !windows

package main

import (
	"testing"
	"time"
)

// Non-Windows fileCreatedTime currently uses the fixture's modification time.
func setInboxFixtureCreatedTime(t *testing.T, path string, created time.Time) {
	t.Helper()
}
