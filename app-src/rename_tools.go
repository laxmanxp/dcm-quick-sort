package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	RenameToolClean      = "Clean spacing"
	RenameToolTitleCase  = "Title Case"
	RenameToolLower      = "lowercase"
	RenameToolUpper      = "UPPERCASE"
	RenameToolDatePrefix = "Date prefix"
	RenameToolTemplate   = "Naming Template"
	RenameToolOriginal   = "Original filename"

	DefaultRenameTemplate = "{date} - {name}"
)

var renameToolOptions = []string{
	RenameToolClean,
	RenameToolTitleCase,
	RenameToolLower,
	RenameToolUpper,
	RenameToolDatePrefix,
	RenameToolTemplate,
	RenameToolOriginal,
}

type RenameToolContext struct {
	FileDate  time.Time
	Template  string
	Variables map[string]string
}

var renameTemplateTokenPattern = regexp.MustCompile(`\{[^{}]+\}`)

func applyRenameTool(current, original, tool string) (string, error) {
	return applyRenameToolWithContext(current, original, tool, RenameToolContext{})
}

func applyRenameToolWithContext(current, original, tool string, ctx RenameToolContext) (string, error) {
	var result string
	current = strings.TrimSpace(current)
	original = strings.TrimSpace(original)
	switch tool {
	case RenameToolClean:
		result = cleanRenameSpacing(current)
	case RenameToolTitleCase:
		result = titleCaseFilename(current)
	case RenameToolLower:
		result = strings.ToLower(current)
	case RenameToolUpper:
		result = strings.ToUpper(current)
	case RenameToolDatePrefix:
		result = addDatePrefix(current, original, ctx.FileDate)
	case RenameToolTemplate:
		var err error
		result, err = applyFilenameTemplateWithVariables(ctx.Template, current, original, ctx.FileDate, ctx.Variables)
		if err != nil {
			return "", err
		}
	case RenameToolOriginal:
		result = original
	default:
		return "", fmt.Errorf("unknown rename tool: %s", tool)
	}
	if _, err := sanitizeBaseName(result); err != nil {
		return "", err
	}
	return result, nil
}

func renameDateText(date time.Time) string {
	if date.IsZero() {
		date = time.Now()
	}
	return date.Format("2006-01-02")
}

func addDatePrefix(current, original string, date time.Time) string {
	name := strings.TrimSpace(current)
	if name == "" {
		name = strings.TrimSpace(original)
	}
	prefix := renameDateText(date) + " "
	if strings.HasPrefix(name, prefix) {
		return name
	}
	return strings.TrimSpace(prefix + name)
}

func applyFilenameTemplate(pattern, current, original string, date time.Time) (string, error) {
	return applyFilenameTemplateWithVariables(pattern, current, original, date, nil)
}

func validateFilenameTemplateTokens(pattern string, variables map[string]string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("template is empty")
	}
	for _, token := range renameTemplateTokenPattern.FindAllString(pattern, -1) {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}"))
		key := strings.ToLower(name)
		switch key {
		case "date", "name", "original":
		default:
			if _, ok := variables[key]; !ok {
				return fmt.Errorf("unsupported template token %s; add it as a custom variable or use {date}, {name}, or {original}", token)
			}
		}
	}
	return nil
}

func applyFilenameTemplateWithVariables(pattern, current, original string, date time.Time, variables map[string]string) (string, error) {
	if err := validateFilenameTemplateTokens(pattern, variables); err != nil {
		return "", err
	}
	name := strings.TrimSpace(current)
	if name == "" {
		name = strings.TrimSpace(original)
	}
	builtins := map[string]string{
		"date":     renameDateText(date),
		"name":     name,
		"original": strings.TrimSpace(original),
	}
	return strings.TrimSpace(renameTemplateTokenPattern.ReplaceAllStringFunc(pattern, func(token string) string {
		key := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")))
		if value, ok := builtins[key]; ok {
			return value
		}
		return variables[key]
	})), nil
}

func cleanRenameSpacing(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '_' || unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

func titleCaseFilename(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	for i, field := range fields {
		segments := strings.Split(field, "-")
		for j, segment := range segments {
			segments[j] = titleCaseSegment(segment)
		}
		fields[i] = strings.Join(segments, "-")
	}
	return strings.Join(fields, " ")
}

func titleCaseSegment(s string) string {
	if s == "" || isShortAcronym(s) {
		return s
	}
	runes := []rune(strings.ToLower(s))
	for i, r := range runes {
		if unicode.IsLetter(r) {
			runes[i] = unicode.ToUpper(r)
			break
		}
	}
	return string(runes)
}

func isShortAcronym(s string) bool {
	letters := 0
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if unicode.IsLower(r) {
			return false
		}
	}
	return letters >= 2 && letters <= 5
}
