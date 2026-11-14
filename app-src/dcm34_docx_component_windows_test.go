//go:build windows

package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// This is a component test of the real fallback function, edit/status controls
// and preview painter. It neither selects a provider nor starts an Office host.
// Only the named FlowDev component selector requests visible screen captures.
func TestDCM34DOCXFallbackComponent(t *testing.T) {
	evidence := os.Getenv("DCM_DOCX_COMPONENT_EVIDENCE_ROOT")
	headed := evidence != ""
	if headed && !filepath.IsAbs(evidence) {
		t.Fatal("component evidence root must be absolute")
	}
	if !headed {
		evidence = t.TempDir()
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(evidence, "receipt.json")
	if _, err := os.Stat(receiptPath); err == nil {
		t.Fatal("refusing to overwrite existing component evidence")
	}
	records := []map[string]any{}
	receipt := map[string]any{
		"schema": 1, "feature": "DCM-34-DOCX-FALLBACK-01", "headed": headed,
		"scope":  "owned Windows component invoking production loadOfficeFallbackPreview and previewWindowProc; not normal-app provider selection or installed Word behavior",
		"passed": false, "feature_accepted": false, "visual_accepted": false,
		"visual_review_required": headed, "user_test": "not_recorded", "registry_mutation": false,
		"office_activation": false, "started_utc": time.Now().UTC().Format(time.RFC3339Nano),
	}
	defer func() {
		receipt["passed"] = !t.Failed()
		receipt["cases"] = records
		receipt["case_count"] = len(records)
		receipt["finished_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
		dcm34ComponentJSON(t, receiptPath, receipt)
		t.Logf("DOCX_COMPONENT_RECEIPT=%s", receiptPath)
	}()
	if exe, err := os.Executable(); err == nil {
		receipt["test_executable"] = exe
		receipt["test_executable_sha256"] = dcm34ComponentHash(t, exe)
	} else {
		t.Fatal(err)
	}
	const ns = ` xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	fixtures := map[string]string{}
	for _, item := range []struct{ name, body string }{
		{"valid-body.docx", `<w:document` + ns + `><w:body><w:p><w:r><w:t>DCM-34 recovery body: Bridge count 7.</w:t></w:r></w:p></w:body></w:document>`},
		{"valid-blank.docx", `<w:document` + ns + `><w:body/></w:document>`},
		{"malformed-xml.docx", `<w:document` + ns + `><w:body><w:p><w:r>`},
		{"missing-body.docx", `<w:document` + ns + `/>`},
	} {
		path := dcm34ComponentFixture(t, evidence, item.name, item.body)
		fixtures[item.name] = dcm34ComponentHash(t, path)
	}
	receipt["fixture_sha256"] = fixtures
	for _, outcome := range []struct {
		name       string
		registered bool
	}{
		{"missing-handler-boundary", false},
		{"failed-handler-boundary", true},
	} {
		t.Run(outcome.name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			oldDPI, _, _ := spSetThreadDPIContext.Call(^uintptr(3)) // per-monitor v2
			if oldDPI == 0 {
				t.Fatal("could not establish native component DPI context")
			}
			defer spSetThreadDPIContext.Call(oldDPI)
			oldApp := app
			app = appState{}
			defer func() { app = oldApp }()
			destroy := dcm34ComponentWindow(t, headed)
			defer destroy()
			groupDir := filepath.Join(evidence, outcome.name)
			if err := os.MkdirAll(groupDir, 0700); err != nil {
				t.Fatal(err)
			}
			var originalSuccess string
			for _, stage := range []struct{ name, file, want string }{
				{"01-empty-pane", "", "empty"},
				{"02-malformed-after-empty", "malformed-xml.docx", "error"},
				{"03-valid-blank", "valid-blank.docx", "blank"},
				{"04-valid-body", "valid-body.docx", "success"},
				{"05-malformed-after-success", "malformed-xml.docx", "error"},
				{"06-valid-recovery", "valid-body.docx", "recovery"},
				{"07-missing-body-after-success", "missing-body.docx", "error"},
				{"08-valid-second-recovery", "valid-body.docx", "recovery"},
			} {
				setText(app.hwnd, "DCM-34 DOCX fallback component | "+outcome.name+" | "+stage.name)
				if stage.file != "" {
					// These are loadPreview's existing state-reset preconditions.
					// Rendering/routing below uses the exact production fallback tail.
					hideTextPreview()
					hideXLSXPreview()
					hidePPTXPreview()
					resetPreviewZoomState()
					loadOfficeFallbackPreview(filepath.Join(evidence, stage.file), outcome.registered)
				}
				passed := true
				check := func(ok bool, format string, args ...any) {
					if !ok {
						passed = false
						t.Errorf("%s: %s", stage.name, fmt.Sprintf(format, args...))
					}
				}
				body := getText(app.textPreviewEdit)
				status := getText(app.statusLabel)
				style, _, _ := spGetWindowStyle.Call(app.textPreviewEdit, ^uintptr(15))
				editShown := style&WS_VISIBLE != 0
				check(app.preview == nil && !app.previewActive && !app.xlsxPreviewActive && !app.pptxPreviewActive, "foreign or wrong-format preview state remained")
				switch stage.want {
				case "empty":
					check(!app.textPreviewActive && !editShown && body == "" && app.fallback == "" && status == "", "initial pane was not blank")
				case "error":
					wantStatus := "Preview unavailable: DOCX (no Windows preview handler installed)"
					wantBody := "No preview available\n\nNo Windows preview handler is installed for DOCX files."
					if outcome.registered {
						wantStatus = "Preview unavailable: DOCX (Windows handler failed)"
						wantBody = "Preview unavailable\n\nWindows has a preview handler for DOCX files, but it could not preview this file."
					}
					check(status == wantStatus, "rejected DOCX claimed success or wrong error: %q", status)
					check(!app.textPreviewActive && !editShown && body == "" && app.textPreviewKind == "", "old/successful text preview was retained")
					check(strings.HasPrefix(app.fallback, wantBody) && strings.Contains(app.fallback, "Name: "+stage.file), "wrong/unidentified error body: %q", app.fallback)
					check(!strings.Contains(app.fallback, "Open XML package validated") && !strings.Contains(app.fallback, "DCM-34 recovery body"), "false success or stale document remained in fallback")
				case "blank", "success", "recovery":
					paragraphs := 1
					if stage.want == "blank" {
						paragraphs = 0
					}
					wantStatus := fmt.Sprintf("DCM DOCX Preview: %d paragraphs · 0 tables · 0 images", paragraphs)
					check(status == wantStatus && app.textPreviewActive && editShown && app.textPreviewKind == "DOCX" && app.fallback == "", "valid DOCX did not reach the real structured-text surface: status=%q kind=%q", status, app.textPreviewKind)
					check(strings.HasPrefix(body, "DCM DOCX Preview") && strings.Contains(body, "Title: DCM-34 component fixture") && strings.Contains(body, "Rendered locally from DOCX Open XML."), "valid DOCX body missing expected framing")
					check(strings.Contains(body, "DCM-34 recovery body: Bridge count 7.") == (paragraphs == 1), "blank/content contract differs")
					check(!strings.Contains(body, "Open XML package validated"), "metadata foundation replaced structured-text renderer")
					if stage.want == "success" {
						originalSuccess = body
					} else if stage.want == "recovery" {
						check(originalSuccess != "" && body == originalSuccess, "recovery differs from original success body")
					}
				}
				buttons := map[string]bool{}
				for name, handle := range map[string]uintptr{"zoom_out": app.zoomOutBtn, "actual": app.zoom100Btn, "zoom_in": app.zoomInBtn, "fit": app.zoomFitBtn, "width": app.zoomFitWidthBtn} {
					enabled, _, _ := user32.NewProc("IsWindowEnabled").Call(handle)
					buttons[name] = enabled != 0
					want := app.textPreviewActive && name != "width"
					check(buttons[name] == want, "control %s enabled=%v; want %v", name, buttons[name], want)
				}
				fixturesPreserved, fixturesReleased := true, true
				for name, hash := range fixtures {
					path := filepath.Join(evidence, name)
					preserved, released := dcm34ComponentHash(t, path) == hash, dcm34ComponentExclusive(path)
					fixturesPreserved = fixturesPreserved && preserved
					fixturesReleased = fixturesReleased && released
					check(preserved, "fixture changed: %s", name)
					check(released, "fixture is still locked: %s", name)
				}
				record := map[string]any{"name": outcome.name + "/" + stage.name, "passed": passed, "file": stage.file, "expected_state": stage.want, "handler_registered_boundary_input": outcome.registered, "text_active": app.textPreviewActive, "edit_style_visible": editShown, "text_kind": app.textPreviewKind, "status": status, "displayed_text": body, "fallback_text": app.fallback, "controls_enabled": buttons, "fixture_bytes_unchanged": fixturesPreserved, "fixtures_exclusively_released": fixturesReleased, "visual_reviewed": false}
				if headed {
					capturePath := filepath.Join(groupDir, stage.name+".png")
					if err := dcm34ComponentCapture(app.hwnd, capturePath); err != nil {
						t.Errorf("%s screen capture: %v", stage.name, err)
						record["passed"] = false
						record["capture_error"] = err.Error()
					} else {
						record["capture"] = capturePath
						record["capture_sha256"] = dcm34ComponentHash(t, capturePath)
					}
				}
				records = append(records, record)
			}
		})
	}
}

func dcm34ComponentFixture(t *testing.T, root, name, document string) string {
	t.Helper()
	path := filepath.Join(root, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z := zip.NewWriter(f)
	for _, part := range []struct{ name, text string }{
		{"[Content_Types].xml", `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Override PartName="/word/document.xml" ContentType="` + contentTypeWordDocument + `"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/></Types>`},
		{"_rels/.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rIdDocument" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rIdCore" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/></Relationships>`},
		{"docProps/core.xml", `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>DCM-34 component fixture</dc:title></cp:coreProperties>`},
		{"word/document.xml", document},
	} {
		h := &zip.FileHeader{Name: part.name, Method: zip.Deflate, Modified: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)}
		w, err := z.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(part.text)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func dcm34ComponentHash(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func dcm34ComponentJSON(t *testing.T, path string, value any) {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err == nil {
		err = os.WriteFile(path, append(b, '\n'), 0600)
	}
	if err != nil {
		t.Errorf("write component evidence: %v", err)
	}
}

func dcm34ComponentExclusive(path string) bool {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return false
	}
	return syscall.CloseHandle(h) == nil
}

func dcm34ComponentWindow(t *testing.T, headed bool) func() {
	t.Helper()
	hInst, _, _ := procGetModuleHandleW.Call(0)
	var atoms []uintptr
	var parent, font uintptr
	ready := false
	cleanup := func() {
		if parent != 0 {
			hideTextPreview()
			if ok, _, _ := user32.NewProc("DestroyWindow").Call(parent); ok == 0 {
				t.Error("owned component host destruction failed")
			}
		}
		if font != 0 {
			procDeleteObject.Call(font)
		}
		for i := len(atoms) - 1; i >= 0; i-- {
			if ok, _, _ := user32.NewProc("UnregisterClassW").Call(atoms[i], hInst); ok == 0 {
				t.Error("owned component class cleanup failed")
			}
		}
	}
	// Fatal setup errors must also release partial resources on this locked
	// Windows thread, before the caller restores globals and unlocks it.
	defer func() {
		if !ready {
			cleanup()
		}
	}()
	for _, item := range []struct {
		name string
		proc uintptr
	}{{"DCM34DOCXComponentHost", procDefWindowProcW.Addr()}, {"DCM34DOCXComponentPreview", previewWndProc}} {
		wc := WNDCLASSEXW{CbSize: uint32(unsafe.Sizeof(WNDCLASSEXW{})), LpfnWndProc: item.proc, HInstance: hInst, HbrBackground: COLOR_WINDOW + 1, LpszClassName: wstr(item.name)}
		atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			t.Fatalf("owned component class: %v", err)
		}
		atoms = append(atoms, atom)
	}
	style := uint32((WS_OVERLAPPEDWINDOW &^ (WS_THICKFRAME | WS_MAXIMIZEBOX)) | WS_CLIPCHILDREN)
	parent = createControl("DCM34DOCXComponentHost", "DCM-34 DOCX fallback component", style, 24, 24, 940, 650, 0, 0)
	if parent == 0 {
		t.Fatal("owned component host creation failed")
	}
	app.hwnd = parent
	font = createUIFont(-16, FW_NORMAL)
	if font == 0 {
		t.Fatal("owned component font creation failed")
	}
	app.uiFont = font
	header := createControl("STATIC", "DCM-34 DOCX fallback component | production renderer and native controls", WS_CHILD|WS_VISIBLE, 20, 15, 870, 26, parent, 960)
	app.previewHwnd = createControl("DCM34DOCXComponentPreview", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_CLIPCHILDREN, 20, 55, 890, 430, parent, 961)
	app.textPreviewEdit = createControl("EDIT", "", WS_CHILD|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 0, 0, 890, 430, app.previewHwnd, 962)
	app.statusLabel = createControl("STATIC", "", WS_CHILD|WS_VISIBLE, 20, 500, 890, 32, parent, 963)
	for i, dest := range []*uintptr{&app.zoomOutBtn, &app.zoom100Btn, &app.zoomInBtn, &app.zoomFitBtn, &app.zoomFitWidthBtn} {
		*dest = createControl("BUTTON", []string{"-", "100%", "+", "Fit", "Width"}[i], WS_CHILD|WS_VISIBLE, int32(20+i*90), 545, 80, 28, parent, 964+i)
		if *dest == 0 {
			t.Fatal("owned component sizing control creation failed")
		}
		setControlFont(*dest, app.uiFont)
	}
	for _, h := range []uintptr{parent, header, app.previewHwnd, app.textPreviewEdit, app.statusLabel} {
		if h == 0 {
			t.Fatal("owned component child creation failed")
		}
		var pid uint32
		user32.NewProc("GetWindowThreadProcessId").Call(h, uintptr(unsafe.Pointer(&pid)))
		if pid != uint32(os.Getpid()) {
			t.Fatal("component window ownership mismatch")
		}
		setControlFont(h, app.uiFont)
	}
	resetPreviewZoomState()
	if headed {
		procShowWindow.Call(parent, SW_SHOW)
		procSetForegroundWindow.Call(parent)
	}
	ready = true
	return cleanup
}

// Capture actual on-screen pixels of the owned visible component window.
// PrintWindow/WM_PRINT and synthetic renderings are deliberately not used.
func dcm34ComponentCapture(hwnd uintptr, path string) error {
	procSetForegroundWindow.Call(hwnd)
	procInvalidateRect.Call(hwnd, 0, 1)
	user32.NewProc("RedrawWindow").Call(hwnd, 0, 0, 0x0185) // invalidate, erase, all children, update now
	until := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(until) {
		var msg MSG
		for {
			more, _, _ := user32.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
			if more == 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if fg, _, _ := user32.NewProc("GetForegroundWindow").Call(); fg != hwnd {
		return fmt.Errorf("owned component is not the foreground window")
	}
	if visible, _, _ := procIsWindowVisible.Call(hwnd); visible == 0 {
		return fmt.Errorf("owned component is not visible")
	}
	var rect RECT
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ok == 0 {
		return fmt.Errorf("component rectangle unavailable")
	}
	w, h := rect.Right-rect.Left, rect.Bottom-rect.Top
	sx, _, _ := user32.NewProc("GetSystemMetrics").Call(76)
	sy, _, _ := user32.NewProc("GetSystemMetrics").Call(77)
	sw, _, _ := user32.NewProc("GetSystemMetrics").Call(78)
	sh, _, _ := user32.NewProc("GetSystemMetrics").Call(79)
	if w <= 0 || h <= 0 || w > 2000 || h > 1500 || rect.Left < int32(sx) || rect.Top < int32(sy) || rect.Right > int32(sx)+int32(sw) || rect.Bottom > int32(sy)+int32(sh) {
		return fmt.Errorf("component rectangle is not fully on screen: %+v", rect)
	}
	screen, _, _ := user32.NewProc("GetDC").Call(0)
	if screen == 0 {
		return fmt.Errorf("screen DC unavailable")
	}
	defer user32.NewProc("ReleaseDC").Call(0, screen)
	memory, _, _ := gdi32.NewProc("CreateCompatibleDC").Call(screen)
	if memory == 0 {
		return fmt.Errorf("capture DC unavailable")
	}
	defer gdi32.NewProc("DeleteDC").Call(memory)
	type bitmapInfo struct {
		Size          uint32
		Width, Height int32
		Planes, Bits  uint16
		Compression   uint32
		ImageSize     uint32
		XPels, YPels  int32
		Colors, Used  uint32
	}
	info := bitmapInfo{Size: 40, Width: w, Height: -h, Planes: 1, Bits: 32, ImageSize: uint32(w * h * 4)}
	var pixels uintptr
	bitmap, _, _ := gdi32.NewProc("CreateDIBSection").Call(screen, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 || pixels == 0 {
		return fmt.Errorf("capture bitmap unavailable")
	}
	defer procDeleteObject.Call(bitmap)
	prior, _, _ := gdi32.NewProc("SelectObject").Call(memory, bitmap)
	if prior == 0 || prior == ^uintptr(0) {
		return fmt.Errorf("capture bitmap selection failed")
	}
	defer gdi32.NewProc("SelectObject").Call(memory, prior)
	if ok, _, _ := gdi32.NewProc("BitBlt").Call(memory, 0, 0, uintptr(w), uintptr(h), screen, uintptr(rect.Left), uintptr(rect.Top), 0x00CC0020); ok == 0 {
		return fmt.Errorf("screen capture failed")
	}
	gdi32.NewProc("GdiFlush").Call()
	bgra := unsafe.Slice((*byte)(unsafe.Pointer(pixels)), int(w*h*4))
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	for i := 0; i < len(bgra); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = bgra[i+2], bgra[i+1], bgra[i], 255
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if err = png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
