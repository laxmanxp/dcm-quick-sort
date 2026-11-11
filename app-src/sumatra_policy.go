package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const sumatraRuntimeSHA256 = "719f689b34f47be8ca105ce8484948474dafde0e106bab599e4a89326070c3d0"
const sumatraPreviewPolicy = `[Policies]
InternetAccess = 0
DiskAccess = 0
SavePreferences = 0
RegistryAccess = 0
PrinterAccess = 0
CopySelection = 0
FullscreenAccess = 0
LinkProtocols =
SafeFileTypes =
`

// Deliberately opt-in while provider/lifecycle and distribution gates are open.
func useSumatraProvider() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("DCM_PDF_PROVIDER")), "sumatra")
}

func sumatraArguments(parent uintptr, document, profile string) ([]string, error) {
	if parent == 0 || !filepath.IsAbs(document) || !filepath.IsAbs(profile) || strings.ContainsRune(document, 0) || strings.ContainsRune(profile, 0) {
		return nil, errors.New("owned parent and absolute PDF/profile paths required")
	}
	if !strings.EqualFold(filepath.Ext(document), ".pdf") {
		return nil, errors.New("Sumatra provider accepts PDF only")
	}
	return []string{"-restrict", "-appdata", profile, "-view", "single page", "-zoom", "fit page", "-plugin", strconv.FormatUint(uint64(parent), 10), document}, nil
}

// No PATH search or user's installed viewer: require a pinned private runtime.
func validateSumatraRuntime(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("set DCM_SUMATRA_EXE to the pinned private runtime")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("Sumatra runtime must be a regular file")
	}
	policy := filepath.Join(filepath.Dir(path), "sumatrapdfrestrict.ini")
	st, err = os.Lstat(policy)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() != int64(len(sumatraPreviewPolicy)) {
		return "", errors.New("exact no-write/no-print Sumatra policy required")
	}
	data, err := os.ReadFile(policy)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(data, []byte(sumatraPreviewPolicy)) {
		return "", errors.New("Sumatra restriction policy differs from the approved policy")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	if hex.EncodeToString(h.Sum(nil)) != sumatraRuntimeSHA256 {
		return "", errors.New("Sumatra runtime hash is not the pinned unmodified 3.6.1 executable")
	}
	return filepath.Clean(path), nil
}

func validateSumatraDocument(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return "", errors.New("PDF extension required")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() < 8 {
		return "", errors.New("PDF must be a nonempty regular file")
	}
	header := make([]byte, 1024)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		return "", err
	}
	if !bytes.Contains(header[:n], []byte("%PDF-")) {
		return "", fmt.Errorf("%s has no PDF header", filepath.Base(path))
	}
	return path, nil
}
