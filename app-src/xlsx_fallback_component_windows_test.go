//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Bounded actual XLSX fallback presenter, controls and worksheet rendering.
// Missing/faulting Office handlers are controlled boundary inputs, not changes
// to Windows registrations or evidence of actual Excel provider failure.
func TestXLSXFallbackComponentRecovery(t *testing.T) {
	evidence := os.Getenv("DCM_XLSX_COMPONENT_EVIDENCE_ROOT")
	headed := evidence != ""
	if headed && !filepath.IsAbs(evidence) {
		t.Fatal("XLSX evidence root must be absolute")
	}
	if !headed {
		evidence = t.TempDir()
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(evidence, "receipt.json")
	if _, err := os.Stat(receiptPath); err == nil {
		t.Fatal("refusing to replace existing XLSX component receipt")
	}
	var records []map[string]any
	receipt := map[string]any{
		"schema": 1, "feature": "DCM-34-XLSX-FALLBACK-01",
		"scope": "Owned Windows component calling production loadOfficeFallbackPreview with simulated missing/failed Excel handler input; not actual COM provider failure or registry mutation",
		"passed": false, "headed": headed, "feature_accepted": false,
		"visual_review_required": headed, "user_test": "not_recorded",
		"registry_mutation": false, "excel_activation": false,
		"started_utc": time.Now().UTC().Format(time.RFC3339Nano),
	}
	defer func() {
		receipt["passed"] = !t.Failed()
		receipt["case_count"] = len(records)
		receipt["cases"] = records
		receipt["finished_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
		dcm34ComponentJSON(t, receiptPath, receipt)
		t.Logf("XLSX_COMPONENT_RECEIPT=%s", receiptPath)
	}()

	// A deterministic two-sheet Open XML workbook. No Excel automation,
	// external workbook modification, macro execution or installed handler.
	originalFixture := writeXLSXFixture(t, basicWorkbook(false), basicWorkbookRels(), map[string]string{
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Bridge Work</t></is></c><c r="B1"><v>125000</v></c><c r="C1" t="b"><v>1</v></c></row><row r="3"><c r="A3" t="inlineStr"><is><t>Coast Road</t></is></c><c r="B3"><v>86000</v></c></row></sheetData></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Stage</t></is></c><c r="B1" t="inlineStr"><is><t>Approved</t></is></c></row><row r="2"><c r="A2" t="inlineStr"><is><t>Road</t></is></c><c r="B2"><v>86000</v></c></row></sheetData></worksheet>`,
	})
	workbookBytes, err := os.ReadFile(originalFixture)
	if err != nil {
		t.Fatal(err)
	}
	receipt["valid_fixture_sha256"] = dcm34ComponentHash(t, originalFixture)
	for _, tc := range []struct {
		name       string
		registered bool
	}{
		{"missing-handler-boundary", false},
		{"failed-handler-boundary", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			valid := filepath.Join(root, "valid.xlsx")
			invalid := filepath.Join(root, "corrupt.xlsx")
			invalidBytes := []byte("DCM-34 invalid XLSX: synthetic bytes, not an Open XML workbook")
			if err := os.WriteFile(valid, workbookBytes, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(invalid, invalidBytes, 0600); err != nil {
				t.Fatal(err)
			}
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			oldApp := app
			app = appState{}
			defer func() { app = oldApp }()
			destroy := dcm34ComponentWindow(t, headed)
			defer func() { hideXLSXPreview(); destroy() }()
			if headed {
				setText(app.hwnd, "DCM-34 XLSX fallback | "+tc.name)
				header, _, _ := user32.NewProc("GetDlgItem").Call(app.hwnd, 960)
				if header == 0 {
					t.Fatal("owned XLSX component header is missing")
				}
				setText(header, "XLSX fallback | "+tc.name+" | invalid - valid sheets - invalid - recovery")
			}
			app.xlsxSheetCombo = createControl("COMBOBOX", "", WS_CHILD|WS_TABSTOP|CBS_DROPDOWNLIST|WS_VSCROLL,
				6, 4, 850, 280, app.previewHwnd, ID_XLSX_SHEET)
			app.xlsxGridEdit = createControl("EDIT", "", WS_CHILD|WS_VSCROLL|WS_HSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_AUTOHSCROLL|ES_READONLY,
				0, 34, 890, 390, app.previewHwnd, 0)
			if app.xlsxSheetCombo == 0 || app.xlsxGridEdit == 0 {
				t.Fatal("owned XLSX grid or sheet selector could not be created")
			}
			setControlFont(app.xlsxSheetCombo, app.uiFont)
			setControlFont(app.xlsxGridEdit, app.uiFont)

			capture := func(stage, expected string) {
				t.Helper()
				status := getText(app.statusLabel)
				grid := getText(app.xlsxGridEdit)
				count, _, _ := procSendMessageW.Call(app.xlsxSheetCombo, CB_GETCOUNT, 0, 0)
				selected, _, _ := procSendMessageW.Call(app.xlsxSheetCombo, CB_GETCURSEL, 0, 0)
				controls := map[string]bool{}
				for name, hwnd := range map[string]uintptr{
					"zoom_out": app.zoomOutBtn, "actual": app.zoom100Btn,
					"zoom_in": app.zoomInBtn, "fit": app.zoomFitBtn,
					"width": app.zoomFitWidthBtn,
				} {
					enabled, _, _ := user32.NewProc("IsWindowEnabled").Call(hwnd)
					controls[name] = enabled != 0
					want := app.xlsxPreviewActive && name != "width"
					if controls[name] != want {
						t.Errorf("%s control %s enabled=%v, want %v", stage, name, controls[name], want)
					}
				}
				preserved := true
				released := true
				for _, f := range []struct {
					path  string
					bytes []byte
				}{{valid, workbookBytes}, {invalid, invalidBytes}} {
					b, err := os.ReadFile(f.path)
					ok := err == nil && bytes.Equal(b, f.bytes)
					exclusive := dcm34ComponentExclusive(f.path)
					if !ok {
						t.Errorf("%s fixture bytes changed: %s: %v", stage, f.path, err)
					}
					if !exclusive {
						t.Errorf("%s source fixture still locked: %s", stage, f.path)
					}
					preserved = preserved && ok
					released = released && exclusive
				}
				rec := map[string]any{
					"name": tc.name + "/" + stage, "expected_state": expected,
					"handler_registered_boundary_input": tc.registered,
					"xlsx_active": app.xlsxPreviewActive, "sheet_index": app.xlsxSheetIndex,
					"sheet_count": len(app.xlsxWorkbook.Sheets), "combo_count": int(count),
					"combo_selection": int(selected), "status": status, "grid_text": grid,
					"fallback_text": app.fallback, "controls_enabled": controls,
					"fixture_bytes_unchanged": preserved, "fixtures_exclusively_released": released,
					"office_activated": false, "visual_reviewed": false,
				}
				if headed {
					group := filepath.Join(evidence, tc.name)
					if err := os.MkdirAll(group, 0700); err != nil {
						t.Error(err)
					} else {
						filename := filepath.Join(group, stage+".png")
						if err := dcm34ComponentCapture(app.hwnd, filename); err != nil {
							rec["capture_error"] = err.Error()
							t.Errorf("%s headed screenshot: %v", stage, err)
						} else {
							rec["capture"] = filename
							rec["capture_sha256"] = dcm34ComponentHash(t, filename)
						}
					}
				}
				records = append(records, rec)
			}
			load := func(path string) {
				t.Helper()
				// These are the production loadPreview reset preconditions;
				// routing and native control presentation below are real.
				hideTextPreview()
				hideXLSXPreview()
				hidePPTXPreview()
				resetPreviewZoomState()
				loadOfficeFallbackPreview(path, tc.registered)
			}
			assertBad := func(stage string) {
				t.Helper()
				load(invalid)
				wantStatus := "Preview unavailable: XLSX (no Windows preview handler installed)"
				wantBody := "No preview available\n\nNo Windows preview handler is installed for XLSX files."
				if tc.registered {
					wantStatus = "Preview unavailable: XLSX (Windows handler failed)"
					wantBody = "Preview unavailable\n\nWindows has a preview handler for XLSX files, but it could not preview this file."
				}
				if got := getText(app.statusLabel); got != wantStatus {
					t.Errorf("%s status %q, want %q", stage, got, wantStatus)
				}
				if app.xlsxPreviewActive || app.textPreviewActive || app.pptxPreviewActive ||
					app.previewActive || app.preview != nil || len(app.xlsxWorkbook.Sheets) != 0 ||
					app.xlsxSheetIndex != -1 || app.xlsxGridFont != 0 {
					t.Errorf("%s stale workbook or other preview after corrupt XLSX", stage)
				}
				count, _, _ := procSendMessageW.Call(app.xlsxSheetCombo, CB_GETCOUNT, 0, 0)
				if count != 0 || getText(app.xlsxGridEdit) != "" {
					t.Errorf("%s stale XLSX sheet selector/grid after error, count=%d", stage, count)
				}
				if !strings.HasPrefix(app.fallback, wantBody) ||
					!strings.Contains(app.fallback, "Name: corrupt.xlsx") ||
					strings.Contains(app.fallback, "Bridge Work") {
					t.Errorf("%s false success or misleading unavailable body: %q", stage, app.fallback)
				}
				capture(stage, "error")
			}
			assertGood := func(stage string) {
				t.Helper()
				load(valid)
				if app.previewActive || app.preview != nil || app.pptxPreviewActive ||
					app.textPreviewActive || !app.xlsxPreviewActive || app.fallback != "" ||
					len(app.xlsxWorkbook.Sheets) != 2 || app.xlsxSheetIndex != 0 {
					t.Errorf("%s failed to present structured XLSX workbook", stage)
					return
				}
				if s := app.xlsxWorkbook.Sheets[0]; s.Name != "Summary" || s.MaxRow != 3 ||
					s.MaxCol != 3 || s.Cells["A1"] != "Bridge Work" ||
					s.Cells["B1"] != "125000" || s.Cells["C1"] != "TRUE" ||
					s.Cells["A3"] != "Coast Road" || s.Cells["B3"] != "86000" {
					t.Errorf("%s sheet 1 parsed content/shape not preserved: %+v", stage, s)
				}
				if want := "DCM XLSX Preview: Summary · 3 rows × 3 columns"; getText(app.statusLabel) != want {
					t.Errorf("%s first sheet status %q", stage, getText(app.statusLabel))
				}
				if grid := getText(app.xlsxGridEdit); !strings.Contains(grid, "Bridge Work") || !strings.Contains(grid, "125000") || !strings.Contains(grid, "Coast Road") {
					t.Errorf("%s displayed first grid lost workbook values: %q", stage, grid)
				}
				count, _, _ := procSendMessageW.Call(app.xlsxSheetCombo, CB_GETCOUNT, 0, 0)
				if count != 2 {
					t.Errorf("%s sheet selector count %d, want 2", stage, count)
				}
				capture(stage+"-sheet1", "first-sheet")
				showXLSXSheet(1)
				if app.xlsxSheetIndex != 1 || getText(app.statusLabel) != "DCM XLSX Preview: Data & Bills · 2 rows × 2 columns" {
					t.Errorf("%s sheet 2 not selected/displayed", stage)
				}
				if grid := getText(app.xlsxGridEdit); !strings.Contains(grid, "Approved") || !strings.Contains(grid, "86000") || strings.Contains(grid, "Bridge Work") {
					t.Errorf("%s displayed second sheet stale/wrong: %q", stage, grid)
				}
				capture(stage+"-sheet2", "second-sheet")
				showXLSXSheet(0)
				if app.xlsxSheetIndex != 0 || getText(app.statusLabel) != "DCM XLSX Preview: Summary · 3 rows × 3 columns" {
					t.Errorf("%s first-sheet return failed", stage)
				}
				capture(stage+"-sheet1-return", "first-sheet-return")
			}
			assertBad("initial-error")
			assertGood("first-recovery")
			assertBad("error-after-success")
			assertGood("second-recovery")
			hideXLSXPreview()
			if app.xlsxPreviewActive || len(app.xlsxWorkbook.Sheets) != 0 ||
				app.xlsxSheetIndex != -1 || app.xlsxGridFont != 0 ||
				getText(app.xlsxGridEdit) != "" {
				t.Error("XLSX preview survived unload or leaked grid/font resources")
			}
			if got, _, _ := procSendMessageW.Call(app.xlsxSheetCombo, CB_GETCOUNT, 0, 0); got != 0 {
				t.Errorf("stale sheet options after final XLSX unload: %d", got)
			}
			for _, file := range []string{valid, invalid} {
				if !dcm34ComponentExclusive(file) {
					t.Errorf("XLSX fixture remained locked after unload: %s", file)
				}
			}
		})
	}
}
