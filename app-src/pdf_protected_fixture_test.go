package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const dcmProtectedHash = "cd167e6404f23dec6015d2897a9f7e462d0fec82f046c35f59afc7e9e774c24c"

// Reuse the versioned synthetic fixture, never a user's encrypted document.
func dcmProtectedFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "tests", "fixtures", "sumatra-password.json"))
	if err != nil {
		t.Fatal(err)
	}
	var definition struct {
		Schema               int
		File, SHA256, Base64 string
	}
	if err := json.Unmarshal(raw, &definition); err != nil {
		t.Fatal(err)
	}
	data, err := base64.StdEncoding.Strict().DecodeString(definition.Base64)
	if err != nil {
		t.Fatal(err)
	}
	if definition.Schema != 1 || definition.File != "12-protected.pdf" || definition.SHA256 != dcmProtectedHash || fmt.Sprintf("%x", sha256.Sum256(data)) != dcmProtectedHash {
		t.Fatal("protected fixture identity mismatch")
	}
	return data
}

func TestDCM34PDFProtectedFixture(t *testing.T) {
	data := dcmProtectedFixture(t)
	for _, token := range []string{"%PDF-1.3", "/Encrypt 6 0 R", "/Filter /Standard", "/Length 128", "%%EOF"} {
		if !bytes.Contains(data, []byte(token)) {
			t.Fatalf("pinned encrypted fixture lacks %q", token)
		}
	}
	if bytes.Contains(data, []byte("CORRECT PASSWORD ACCEPTED")) {
		t.Fatal("protected page text is unexpectedly plaintext")
	}
	t.Logf("encrypted fixture bytes=%d sha256=%s; decryption and native refusal are separate checks", len(data), dcmProtectedHash)
}

func TestDCM34PDFProtectedProtocol(t *testing.T) {
	for _, s := range []string{"", "Password required", "Exception calling LoadFromFileAsync: The specified network password is not correct."} {
		meta, err := parsePDFRendererOutput(s)
		if err == nil || meta != (PDFRenderMetadata{}) {
			t.Fatalf("refusal became success: metadata=%+v err=%v", meta, err)
		}
	}
	// Production has no password argument; this qualification must not silently unlock.
	args := pdfRendererArgs("fixture-renderer.ps1", PDFRenderRequest{InputPath: "12-protected.pdf", OutputPath: "output.png", Fit: true})
	for _, a := range args {
		if strings.Contains(strings.ToLower(a), "password") {
			t.Fatal("test unexpectedly submits a password")
		}
	}
}
