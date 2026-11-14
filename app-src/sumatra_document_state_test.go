package main

import (
	"strings"
	"testing"
)

func TestSumatraDocumentStatusDoesNotClaimUnloadedPages(t *testing.T) {
	for _, tc := range []struct {
		known bool
		label string
		ready bool
	}{
		{false, "", false}, {false, "1", false}, {true, "", false}, {true, " \t", false}, {true, "0", false}, {true, "1", true}, {true, "12", true}, {true, "iv", true}, {true, "A-1", true},
	} {
		status := sumatraDocumentStatus(tc.known, tc.label)
		if strings.HasPrefix(status, "Sumatra PDF preview") != tc.ready {
			t.Errorf("known=%v label=%q: wrong readiness %q", tc.known, tc.label, status)
		}
		if !tc.ready && !strings.Contains(status, "viewer") {
			t.Errorf("missing recovery guidance: %q", status)
		}
	}
}
