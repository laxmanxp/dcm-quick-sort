//go:build windows

package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func TestDCM27ImageLoaderFailureAndRecovery(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if app.image != 0 || app.imageStream != nil || app.gpToken != 0 {
		t.Fatal("image loader test requires idle global image state")
	}
	hr, _, _ := procCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	if hresultFailed(hr) {
		t.Fatalf("COM initialization failed: %#x", hr)
	}
	defer procCoUninitialize.Call()
	if !initGDIPlus() {
		t.Fatal("GDI+ startup failed")
	}
	defer func() { releasePreviewImage(); procGdiplusShutdown.Call(app.gpToken); app.gpToken = 0 }()
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.png")
	bad := filepath.Join(dir, "damaged.jpg")
	badBytes := []byte{255, 216, 255, 0, 1, 2, 3, 4}
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, badBytes, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, reason string }{
		{filepath.Join(dir, "missing.png"), "missing.png"},
		{empty, "empty image file"},
		{bad, "image decoder"},
	} {
		err := loadImageWithError(tc.path)
		if err == nil || !strings.Contains(err.Error(), tc.reason) {
			t.Fatalf("%s: want reason %q, got %v", tc.path, tc.reason, err)
		}
		if app.image != 0 || app.imageStream != nil {
			t.Fatal("failed load retained an image/stream")
		}
	}
	after, err := os.ReadFile(bad)
	if err != nil || !bytes.Equal(after, badBytes) {
		t.Fatalf("failed decoding changed source: %v", err)
	}
	var buf bytes.Buffer
	fixture := image.NewRGBA(image.Rect(0, 0, 16, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 16; x++ {
			fixture.Set(x, y, color.RGBA{20, 160, 190, 255})
		}
	}
	if err := png.Encode(&buf, fixture); err != nil {
		t.Fatal(err)
	}
	valid := filepath.Join(dir, "valid.png")
	if err := os.WriteFile(valid, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadImageWithError(valid); err != nil {
		t.Fatalf("valid image did not recover: %v", err)
	}
	var w, h uint32
	procGdipGetImageWidth.Call(app.image, uintptr(unsafe.Pointer(&w)))
	procGdipGetImageHeight.Call(app.image, uintptr(unsafe.Pointer(&h)))
	if w != 16 || h != 12 || app.imageStream == nil {
		t.Fatalf("invalid loaded image: %dx%d", w, h)
	}
	path, err := syscall.UTF16PtrFromString(valid)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(path, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("preview locked source: %v", err)
	}
	syscall.CloseHandle(handle)
	releasePreviewImage()
	if app.image != 0 || app.imageStream != nil {
		t.Fatal("image cleanup incomplete")
	}
	if !loadImage(valid) {
		t.Fatal("existing bool loader contract did not recover")
	}
	releasePreviewImage()
	after, err = os.ReadFile(valid)
	if err != nil || !bytes.Equal(after, buf.Bytes()) {
		t.Fatalf("valid preview changed source: %v", err)
	}
}
