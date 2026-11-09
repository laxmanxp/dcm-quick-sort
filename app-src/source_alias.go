package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SourceAlias struct {
	Name string
	Path string
}

func sourceConfigFilePath() string {
	if dir := settingsDirectory(); dir != "" {
		return filepath.Join(dir, "sources.conf")
	}
	return ""
}

func defaultSourceAliasName(path string) string {
	name := strings.TrimSpace(filepath.Base(filepath.Clean(path)))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "Source"
	}
	return name
}

func ensureSourceConfig(defaultPath string) error {
	path := sourceConfigFilePath()
	if path == "" {
		return fmt.Errorf("user configuration directory is unavailable")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# DCM - Quick Sort named sources\n")
	b.WriteString("# Managed by DCM. Use Manage Sources in the app.\n")
	b.WriteString("# Format: Name=Full path\n")
	defaultPath = strings.TrimSpace(defaultPath)
	if defaultPath != "" {
		b.WriteString(defaultSourceAliasName(defaultPath))
		b.WriteString("=")
		b.WriteString(defaultPath)
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func loadSourceAliases(defaultPath string) ([]SourceAlias, error) {
	if err := ensureSourceConfig(defaultPath); err != nil {
		return nil, err
	}
	f, err := os.Open(sourceConfigFilePath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	aliases := make([]SourceAlias, 0, 8)
	byName := map[string]int{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		name, path, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		alias := SourceAlias{Name: strings.TrimSpace(name), Path: strings.TrimSpace(path)}
		if validateSourceAlias(alias) != nil {
			continue
		}
		key := strings.ToLower(alias.Name)
		if idx, exists := byName[key]; exists {
			aliases[idx] = alias
			continue
		}
		byName[key] = len(aliases)
		aliases = append(aliases, alias)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return aliases, nil
}

func normalizeSourcePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimRight(path, "\\/")
	return path
}

func sourceAliasIndexForPath(aliases []SourceAlias, path string) int {
	want := normalizeSourcePath(path)
	if want == "" {
		return -1
	}
	for i, alias := range aliases {
		if strings.EqualFold(normalizeSourcePath(alias.Path), want) {
			return i
		}
	}
	return -1
}

func validateSourceAlias(alias SourceAlias) error {
	alias.Name = strings.TrimSpace(alias.Name)
	alias.Path = strings.TrimSpace(alias.Path)
	if alias.Name == "" {
		return fmt.Errorf("source name is required")
	}
	if alias.Path == "" {
		return fmt.Errorf("source folder is required")
	}
	if strings.ContainsAny(alias.Name, "=\r\n") {
		return fmt.Errorf("source name cannot contain = or line breaks")
	}
	return nil
}

func upsertSourceAlias(aliases []SourceAlias, alias SourceAlias) ([]SourceAlias, int, error) {
	alias.Name = strings.TrimSpace(alias.Name)
	alias.Path = strings.TrimSpace(alias.Path)
	if err := validateSourceAlias(alias); err != nil {
		return aliases, -1, err
	}
	out := append([]SourceAlias(nil), aliases...)
	for i := range out {
		if strings.EqualFold(strings.TrimSpace(out[i].Name), alias.Name) {
			out[i] = alias
			return out, i, nil
		}
	}
	out = append(out, alias)
	return out, len(out) - 1, nil
}

func deleteSourceAlias(aliases []SourceAlias, index int) []SourceAlias {
	if index < 0 || index >= len(aliases) {
		return append([]SourceAlias(nil), aliases...)
	}
	out := make([]SourceAlias, 0, len(aliases)-1)
	out = append(out, aliases[:index]...)
	out = append(out, aliases[index+1:]...)
	return out
}

func sourceAliasesData(aliases []SourceAlias) ([]byte, error) {
	var b strings.Builder
	b.WriteString("# DCM - Quick Sort named sources\n")
	b.WriteString("# Managed by DCM. Use Manage Sources in the app.\n")
	b.WriteString("# Format: Name=Full path\n")
	seen := map[string]bool{}
	for _, alias := range aliases {
		alias.Name = strings.TrimSpace(alias.Name)
		alias.Path = strings.TrimSpace(alias.Path)
		if err := validateSourceAlias(alias); err != nil {
			return nil, err
		}
		key := strings.ToLower(alias.Name)
		if seen[key] {
			return nil, fmt.Errorf("duplicate source alias %q", alias.Name)
		}
		seen[key] = true
		b.WriteString(alias.Name)
		b.WriteString("=")
		b.WriteString(alias.Path)
		b.WriteString("\n")
	}
	return []byte(b.String()), nil
}

func writeSourceAliases(aliases []SourceAlias) error {
	path := sourceConfigFilePath()
	if path == "" {
		return fmt.Errorf("user configuration directory is unavailable")
	}
	data, err := sourceAliasesData(aliases)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func countSourceAliasConflicts(existing, imported []SourceAlias) int {
	names := make(map[string]bool, len(existing))
	for _, alias := range existing {
		names[strings.ToLower(strings.TrimSpace(alias.Name))] = true
	}
	conflicts := 0
	for _, alias := range imported {
		if names[strings.ToLower(strings.TrimSpace(alias.Name))] {
			conflicts++
		}
	}
	return conflicts
}
