package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultNamingTemplateConfig(t *testing.T) {
	cfg := defaultNamingTemplateConfig()
	if len(cfg.Templates) != 1 {
		t.Fatalf("expected one default template, got %d", len(cfg.Templates))
	}
	if cfg.Active != "Default" {
		t.Fatalf("active=%q", cfg.Active)
	}
	if cfg.Templates[0].Pattern != DefaultRenameTemplate {
		t.Fatalf("pattern=%q", cfg.Templates[0].Pattern)
	}
}

func TestNamingTemplateSupportsUserVariables(t *testing.T) {
	tpl := NamingTemplate{
		Name:    "Government Voucher",
		Pattern: "{office}-{scheme}-{fy}-{name}",
		Variables: []NamingVariable{
			{Name: "office", Value: "Gurundia"},
			{Name: "scheme", Value: "PMAY-G"},
			{Name: "fy", Value: "2026-27"},
		},
	}
	if err := validateNamingTemplate(tpl); err != nil {
		t.Fatal(err)
	}
	vars := namingVariableMap(tpl.Variables)
	if vars["scheme"] != "PMAY-G" || vars["fy"] != "2026-27" {
		t.Fatalf("variables=%v", vars)
	}
}

func TestNamingTemplateRejectsUnknownVariable(t *testing.T) {
	tpl := NamingTemplate{Name: "Broken", Pattern: "{department}-{name}"}
	if err := validateNamingTemplate(tpl); err == nil {
		t.Fatal("expected unknown variable validation error")
	}
}

func TestNamingVariableRejectsBuiltInName(t *testing.T) {
	if err := validateNamingVariable(NamingVariable{Name: "date", Value: "x"}); err == nil {
		t.Fatal("expected built-in variable name rejection")
	}
}

func TestUpsertNamingTemplateAndSetActiveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "naming-templates.json")
	cfg := defaultNamingTemplateConfig()
	tpl := NamingTemplate{Name: "Voucher", Pattern: "{office}-{name}", Variables: []NamingVariable{{Name: "office", Value: "BDO"}}}
	var idx int
	var err error
	cfg, idx, err = upsertNamingTemplate(cfg, tpl)
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 {
		t.Fatalf("idx=%d", idx)
	}
	cfg.Active = "Voucher"
	if err := saveNamingTemplateConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadNamingTemplateConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	active, ok := activeNamingTemplateFromConfig(loaded)
	if !ok || active.Name != "Voucher" || active.Pattern != "{office}-{name}" {
		t.Fatalf("active=%+v ok=%v", active, ok)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteActiveNamingTemplateFallsBack(t *testing.T) {
	cfg := NamingTemplateConfig{Active: "Second", Templates: []NamingTemplate{
		{Name: "First", Pattern: "{name}"},
		{Name: "Second", Pattern: "{date}-{name}"},
	}}
	cfg = deleteNamingTemplate(cfg, 1)
	if cfg.Active != "First" {
		t.Fatalf("active=%q", cfg.Active)
	}
}
