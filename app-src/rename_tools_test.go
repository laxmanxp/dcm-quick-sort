package main

import (
	"strings"
	"testing"
	"time"
)

func TestApplyRenameToolCleanSpacing(t *testing.T) {
	got, err := applyRenameTool("  invoice__final\t2026  ", "original", RenameToolClean)
	if err != nil {
		t.Fatal(err)
	}
	if got != "invoice final 2026" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRenameToolTitleCasePreservesShortAcronym(t *testing.T) {
	got, err := applyRenameTool("PMAY gurundia verification", "original", RenameToolTitleCase)
	if err != nil {
		t.Fatal(err)
	}
	if got != "PMAY Gurundia Verification" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRenameToolTitleCaseHandlesHyphenatedWords(t *testing.T) {
	got, err := applyRenameTool("mayurbhanj-road estimate", "original", RenameToolTitleCase)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Mayurbhanj-Road Estimate" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRenameToolCaseConversions(t *testing.T) {
	lower, err := applyRenameTool("PMAY Report 2026", "original", RenameToolLower)
	if err != nil {
		t.Fatal(err)
	}
	if lower != "pmay report 2026" {
		t.Fatalf("lower=%q", lower)
	}
	upper, err := applyRenameTool("Pmay Report 2026", "original", RenameToolUpper)
	if err != nil {
		t.Fatal(err)
	}
	if upper != "PMAY REPORT 2026" {
		t.Fatalf("upper=%q", upper)
	}
}

func TestApplyRenameToolOriginalRestoresSourceStem(t *testing.T) {
	got, err := applyRenameTool("changed", "Adobe Scan 01", RenameToolOriginal)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Adobe Scan 01" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRenameToolRejectsUnknownTool(t *testing.T) {
	if _, err := applyRenameTool("name", "original", "Mystery Tool"); err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestApplyRenameToolDatePrefixUsesFileDate(t *testing.T) {
	ctx := RenameToolContext{FileDate: time.Date(2026, 9, 22, 4, 0, 0, 0, time.FixedZone("IST", 5*60*60+30*60))}
	got, err := applyRenameToolWithContext("Invoice September", "IMG_0042", RenameToolDatePrefix, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-09-22 Invoice September" {
		t.Fatalf("got %q", got)
	}
}

func TestDatePrefixIsIdempotentForSameDate(t *testing.T) {
	ctx := RenameToolContext{FileDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}
	got, err := applyRenameToolWithContext("2026-09-22 Invoice", "IMG_0042", RenameToolDatePrefix, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-09-22 Invoice" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRenameToolTemplateExpandsSupportedTokens(t *testing.T) {
	ctx := RenameToolContext{
		FileDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		Template: "{date} - {name} - source {original}",
	}
	got, err := applyRenameToolWithContext("September Invoice", "IMG_0042", RenameToolTemplate, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-09-22 - September Invoice - source IMG_0042" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRenameToolTemplateRejectsUnknownToken(t *testing.T) {
	ctx := RenameToolContext{
		FileDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		Template: "{date} - {project} - {name}",
	}
	if _, err := applyRenameToolWithContext("Invoice", "IMG_0042", RenameToolTemplate, ctx); err == nil {
		t.Fatal("expected unsupported template token error")
	}
}

func TestApplyRenameToolTemplateRequiresPattern(t *testing.T) {
	ctx := RenameToolContext{FileDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}
	if _, err := applyRenameToolWithContext("Invoice", "IMG_0042", RenameToolTemplate, ctx); err == nil {
		t.Fatal("expected empty template error")
	}
}

func TestTemplateNeverAddsAnExtensionImplicitly(t *testing.T) {
	ctx := RenameToolContext{
		FileDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		Template: "{date}_{name}",
	}
	got, err := applyRenameToolWithContext("report", "scan", RenameToolTemplate, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-09-22_report" {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, ".pdf") {
		t.Fatalf("template unexpectedly added extension: %q", got)
	}
}

func TestTemplateExpandsUserDefinedVariables(t *testing.T) {
	ctx := RenameToolContext{
		FileDate:  time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC),
		Template:  "{office} - {scheme} - {date} - {name}",
		Variables: map[string]string{"office": "Gurundia", "scheme": "PMAY-G"},
	}
	got, err := applyRenameToolWithContext("Sanction", "IMG_0042", RenameToolTemplate, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Gurundia - PMAY-G - 2026-09-23 - Sanction" {
		t.Fatalf("got %q", got)
	}
}

func TestTemplateVariableNamesAreCaseInsensitive(t *testing.T) {
	ctx := RenameToolContext{
		FileDate:  time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC),
		Template:  "{Office}-{name}",
		Variables: map[string]string{"office": "Gurundia"},
	}
	got, err := applyRenameToolWithContext("Sanction", "IMG_0042", RenameToolTemplate, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Gurundia-Sanction" {
		t.Fatalf("got %q", got)
	}
}
