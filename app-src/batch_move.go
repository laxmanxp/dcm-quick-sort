package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type batchMovePlanItem struct {
	Source string
	Target string
}

func buildBatchMovePlan(selected []string, dest string) ([]batchMovePlanItem, error) {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return nil, fmt.Errorf("choose or type a destination folder first")
	}
	dest = filepath.Clean(dest)
	seen := make(map[string]struct{}, len(selected))
	plan := make([]batchMovePlanItem, 0, len(selected))
	for _, raw := range selected {
		source := strings.TrimSpace(raw)
		if source == "" {
			continue
		}
		source = filepath.Clean(source)
		key := strings.ToLower(source)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if sameMovePath(filepath.Dir(source), dest) {
			return nil, fmt.Errorf("the selected destination is the current folder for %s", filepath.Base(source))
		}
		stem := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		target, err := targetFor(source, dest, stem)
		if err != nil {
			return nil, err
		}
		plan = append(plan, batchMovePlanItem{Source: source, Target: target})
	}
	if len(plan) == 0 {
		return nil, fmt.Errorf("select at least one file to move")
	}
	return plan, nil
}
