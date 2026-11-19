package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const namingTemplatesFileName = "naming-templates.json"

var namingVariableNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

type NamingVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type NamingTemplate struct {
	Name      string           `json:"name"`
	Pattern   string           `json:"pattern"`
	Variables []NamingVariable `json:"variables,omitempty"`
}

type NamingTemplateConfig struct {
	Active    string           `json:"active"`
	Templates []NamingTemplate `json:"templates"`
}

func namingTemplatesFilePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, namingTemplatesFileName)
	}
	return filepath.Join(os.TempDir(), "DCM Quick Sort", namingTemplatesFileName)
}

func defaultNamingTemplateConfig() NamingTemplateConfig {
	return NamingTemplateConfig{
		Active: "Default",
		Templates: []NamingTemplate{{
			Name:    "Default",
			Pattern: DefaultRenameTemplate,
		}},
	}
}

func normalizeNamingVariableName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func builtInNamingVariable(name string) bool {
	switch normalizeNamingVariableName(name) {
	case "date", "name", "original":
		return true
	default:
		return false
	}
}

func validateNamingVariable(v NamingVariable) error {
	name := strings.TrimSpace(v.Name)
	if name == "" {
		return fmt.Errorf("variable name is required")
	}
	if !namingVariableNamePattern.MatchString(name) {
		return fmt.Errorf("variable %q must start with a letter and contain only letters, numbers, or underscore", name)
	}
	if builtInNamingVariable(name) {
		return fmt.Errorf("%q is built in and cannot be redefined", name)
	}
	return nil
}

func namingVariableMap(vars []NamingVariable) map[string]string {
	out := make(map[string]string, len(vars))
	for _, v := range vars {
		name := normalizeNamingVariableName(v.Name)
		if name != "" {
			out[name] = v.Value
		}
	}
	return out
}

func validateNamingTemplate(t NamingTemplate) error {
	name := strings.TrimSpace(t.Name)
	pattern := strings.TrimSpace(t.Pattern)
	if name == "" {
		return fmt.Errorf("template name is required")
	}
	if strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("template name cannot contain line breaks")
	}
	if pattern == "" {
		return fmt.Errorf("template pattern is required")
	}
	seen := map[string]bool{}
	for _, v := range t.Variables {
		if err := validateNamingVariable(v); err != nil {
			return err
		}
		key := normalizeNamingVariableName(v.Name)
		if seen[key] {
			return fmt.Errorf("variable %q is defined more than once", v.Name)
		}
		seen[key] = true
	}
	return validateFilenameTemplateTokens(pattern, namingVariableMap(t.Variables))
}

func normalizeNamingTemplate(t NamingTemplate) NamingTemplate {
	t.Name = strings.TrimSpace(t.Name)
	t.Pattern = strings.TrimSpace(t.Pattern)
	for i := range t.Variables {
		t.Variables[i].Name = strings.TrimSpace(t.Variables[i].Name)
		t.Variables[i].Value = strings.TrimSpace(t.Variables[i].Value)
	}
	sort.SliceStable(t.Variables, func(i, j int) bool {
		return strings.ToLower(t.Variables[i].Name) < strings.ToLower(t.Variables[j].Name)
	})
	return t
}

func validateNamingTemplateConfig(cfg NamingTemplateConfig) error {
	if len(cfg.Templates) == 0 {
		return fmt.Errorf("at least one naming template is required")
	}
	seen := map[string]bool{}
	for _, t := range cfg.Templates {
		if err := validateNamingTemplate(t); err != nil {
			return fmt.Errorf("template %q: %w", t.Name, err)
		}
		key := strings.ToLower(strings.TrimSpace(t.Name))
		if seen[key] {
			return fmt.Errorf("template name %q is duplicated", t.Name)
		}
		seen[key] = true
	}
	if strings.TrimSpace(cfg.Active) == "" {
		return fmt.Errorf("active template is required")
	}
	if !seen[strings.ToLower(strings.TrimSpace(cfg.Active))] {
		return fmt.Errorf("active template %q does not exist", cfg.Active)
	}
	return nil
}

func loadNamingTemplateConfig(path string) (NamingTemplateConfig, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return NamingTemplateConfig{}, fmt.Errorf("naming-template path is unavailable")
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultNamingTemplateConfig(), nil
	}
	if err != nil {
		return NamingTemplateConfig{}, err
	}
	var cfg NamingTemplateConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return NamingTemplateConfig{}, fmt.Errorf("read naming templates: %w", err)
	}
	for i := range cfg.Templates {
		cfg.Templates[i] = normalizeNamingTemplate(cfg.Templates[i])
	}
	cfg.Active = strings.TrimSpace(cfg.Active)
	if err := validateNamingTemplateConfig(cfg); err != nil {
		return NamingTemplateConfig{}, err
	}
	return cfg, nil
}

func saveNamingTemplateConfig(path string, cfg NamingTemplateConfig) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("naming-template path is unavailable")
	}
	for i := range cfg.Templates {
		cfg.Templates[i] = normalizeNamingTemplate(cfg.Templates[i])
	}
	cfg.Active = strings.TrimSpace(cfg.Active)
	if err := validateNamingTemplateConfig(cfg); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWriteFile(path, append(data, '\n'), 0o644)
}

func upsertNamingTemplate(cfg NamingTemplateConfig, t NamingTemplate) (NamingTemplateConfig, int, error) {
	t = normalizeNamingTemplate(t)
	if err := validateNamingTemplate(t); err != nil {
		return cfg, -1, err
	}
	out := NamingTemplateConfig{Active: strings.TrimSpace(cfg.Active), Templates: append([]NamingTemplate(nil), cfg.Templates...)}
	for i := range out.Templates {
		if strings.EqualFold(out.Templates[i].Name, t.Name) {
			out.Templates[i] = t
			if strings.EqualFold(out.Active, t.Name) {
				out.Active = t.Name
			}
			return out, i, nil
		}
	}
	out.Templates = append(out.Templates, t)
	if strings.TrimSpace(out.Active) == "" {
		out.Active = t.Name
	}
	return out, len(out.Templates) - 1, nil
}

func deleteNamingTemplate(cfg NamingTemplateConfig, index int) NamingTemplateConfig {
	if index < 0 || index >= len(cfg.Templates) {
		return cfg
	}
	removed := cfg.Templates[index]
	out := NamingTemplateConfig{Active: cfg.Active, Templates: make([]NamingTemplate, 0, len(cfg.Templates)-1)}
	out.Templates = append(out.Templates, cfg.Templates[:index]...)
	out.Templates = append(out.Templates, cfg.Templates[index+1:]...)
	if len(out.Templates) == 0 {
		return defaultNamingTemplateConfig()
	}
	if strings.EqualFold(strings.TrimSpace(out.Active), strings.TrimSpace(removed.Name)) {
		out.Active = out.Templates[0].Name
	}
	return out
}

func activeNamingTemplateFromConfig(cfg NamingTemplateConfig) (NamingTemplate, bool) {
	for _, t := range cfg.Templates {
		if strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(cfg.Active)) {
			return t, true
		}
	}
	return NamingTemplate{}, false
}

func activeNamingTemplate() (NamingTemplate, error) {
	cfg, err := loadNamingTemplateConfig(namingTemplatesFilePath())
	if err != nil {
		return NamingTemplate{}, err
	}
	t, ok := activeNamingTemplateFromConfig(cfg)
	if !ok {
		return NamingTemplate{}, fmt.Errorf("active naming template is unavailable")
	}
	return t, nil
}

func upsertNamingVariable(vars []NamingVariable, v NamingVariable) ([]NamingVariable, int, error) {
	v.Name = strings.TrimSpace(v.Name)
	v.Value = strings.TrimSpace(v.Value)
	if err := validateNamingVariable(v); err != nil {
		return vars, -1, err
	}
	out := append([]NamingVariable(nil), vars...)
	for i := range out {
		if strings.EqualFold(out[i].Name, v.Name) {
			out[i] = v
			return out, i, nil
		}
	}
	out = append(out, v)
	return out, len(out) - 1, nil
}

func deleteNamingVariable(vars []NamingVariable, index int) []NamingVariable {
	if index < 0 || index >= len(vars) {
		return append([]NamingVariable(nil), vars...)
	}
	out := make([]NamingVariable, 0, len(vars)-1)
	out = append(out, vars[:index]...)
	out = append(out, vars[index+1:]...)
	return out
}
