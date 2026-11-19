package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type StartupSelection struct {
	SourceDir     string
	PreferredFile string
}

func resolveStartupSelection(argument string) (StartupSelection, error) {
	argument = strings.TrimSpace(argument)
	if argument == "" {
		return StartupSelection{}, nil
	}
	abs, err := filepath.Abs(argument)
	if err != nil {
		return StartupSelection{}, err
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return StartupSelection{}, err
	}
	if info.IsDir() {
		return StartupSelection{SourceDir: abs}, nil
	}
	if !info.Mode().IsRegular() {
		return StartupSelection{}, errors.New("startup path is neither a regular file nor a directory")
	}
	return StartupSelection{SourceDir: filepath.Dir(abs), PreferredFile: abs}, nil
}

func preferredFileIndex(files []string, preferred string) int {
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		return -1
	}
	preferred = filepath.Clean(preferred)
	for i, path := range files {
		if strings.EqualFold(filepath.Clean(path), preferred) {
			return i
		}
	}
	return -1
}
