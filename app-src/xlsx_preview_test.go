package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func writeXLSXFixture(t *testing.T, workbook, rels string, extra map[string]string) string {
	t.Helper()
	parts := map[string]string{
		"xl/workbook.xml":            workbook,
		"xl/_rels/workbook.xml.rels": rels,
	}
	for k, v := range extra {
		parts[k] = v
	}
	return writeOpenXMLFixture(t, ".xlsx", map[string]string{
		"/xl/workbook.xml": contentTypeExcelWorkbook,
	}, parts)
}

func basicWorkbook(date1904 bool) string {
	prop := ""
	if date1904 {
		prop = `<workbookPr date1904="1"/>`
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` + prop + `
 <sheets>
  <sheet name="Summary" sheetId="1" r:id="rId1"/>
  <sheet name="Data &amp; Bills" sheetId="2" r:id="rId2"/>
 </sheets>
</workbook>`
}

func basicWorkbookRels() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
 <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
 <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
</Relationships>`
}

func TestXLSXWorkbookDiscoversSheetsThroughRelationships(t *testing.T) {
	p := writeXLSXFixture(t, basicWorkbook(false), basicWorkbookRels(), map[string]string{
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
	})
	wb, err := parseXLSXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(wb.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %#v", wb.Sheets)
	}
	if wb.Sheets[0].Name != "Summary" || wb.Sheets[0].Part != "xl/worksheets/sheet1.xml" {
		t.Fatalf("bad first sheet: %#v", wb.Sheets[0])
	}
	if wb.Sheets[1].Name != "Data & Bills" || wb.Sheets[1].Part != "xl/worksheets/sheet2.xml" {
		t.Fatalf("bad second sheet: %#v", wb.Sheets[1])
	}
}

func TestXLSXSharedInlineBooleanFormulaAndSparseCells(t *testing.T) {
	p := writeXLSXFixture(t, basicWorkbook(false), basicWorkbookRels(), map[string]string{
		"xl/sharedStrings.xml": `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>Scheme</t></si><si><r><t>Bridge </t></r><r><t>Work</t></r></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
          <row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1" t="inlineStr"><is><t>Status</t></is></c></row>
          <row r="3"><c r="A3" t="s"><v>1</v></c><c r="B3"><v>125000.5</v></c><c r="C3" t="b"><v>1</v></c><c r="D3"><f>B3*2</f><v>250001</v></c></row>
        </sheetData></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
	})
	wb, err := parseXLSXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	s := wb.Sheets[0]
	wants := map[string]string{"A1": "Scheme", "C1": "Status", "A3": "Bridge Work", "B3": "125000.5", "C3": "TRUE", "D3": "250001"}
	for ref, want := range wants {
		if got := s.Cells[ref]; got != want {
			t.Fatalf("cell %s = %q; want %q", ref, got, want)
		}
	}
	if s.MaxRow != 3 || s.MaxCol != 4 {
		t.Fatalf("bad sparse bounds rows=%d cols=%d", s.MaxRow, s.MaxCol)
	}
}

func TestXLSXDateStyles1900System(t *testing.T) {
	p := writeXLSXFixture(t, basicWorkbook(false), basicWorkbookRels(), map[string]string{
		"xl/styles.xml":            `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cellXfs count="2"><xf numFmtId="0"/><xf numFmtId="14"/></cellXfs></styleSheet>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" s="1"><v>45292</v></c></row></sheetData></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
	})
	wb, err := parseXLSXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := wb.Sheets[0].Cells["A1"]; got != "2024-01-01" {
		t.Fatalf("date = %q; want 2024-01-01", got)
	}
}

func TestXLSXDateStyles1904SystemWithTime(t *testing.T) {
	p := writeXLSXFixture(t, basicWorkbook(true), basicWorkbookRels(), map[string]string{
		"xl/styles.xml":            `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="yyyy-mm-dd hh:mm"/></numFmts><cellXfs count="2"><xf numFmtId="0"/><xf numFmtId="164"/></cellXfs></styleSheet>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" s="1"><v>43831.5</v></c></row></sheetData></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
	})
	wb, err := parseXLSXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := wb.Sheets[0].Cells["A1"]; got != "2024-01-02 12:00" {
		t.Fatalf("1904 date = %q; want 2024-01-02 12:00", got)
	}
}

func TestXLSXMergedRangesCollected(t *testing.T) {
	p := writeXLSXFixture(t, basicWorkbook(false), basicWorkbookRels(), map[string]string{
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/><mergeCells count="2"><mergeCell ref="A1:C1"/><mergeCell ref="B4:D5"/></mergeCells></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
	})
	wb, err := parseXLSXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(wb.Sheets[0].MergedRanges, ",")
	if got != "A1:C1,B4:D5" {
		t.Fatalf("merged ranges = %q", got)
	}
}

func TestRenderXLSXSheetGridShowsCoordinatesAndBlanks(t *testing.T) {
	s := XLSXSheetPreview{Name: "Summary", Cells: map[string]string{"A1": "Scheme", "C1": "Status", "A3": "Bridge Work", "B3": "125000"}, MaxRow: 3, MaxCol: 3, MergedRanges: []string{"A1:B1"}}
	text := renderXLSXSheetGrid(s)
	for _, want := range []string{"Sheet: Summary", "Merged: A1:B1", "A", "B", "C", "1", "Scheme", "Status", "3", "Bridge Work", "125000"} {
		if !strings.Contains(text, want) {
			t.Fatalf("grid missing %q:\n%s", want, text)
		}
	}
}

func TestXLSXPreviewBoundsRowsColumnsAndCellText(t *testing.T) {
	var cells strings.Builder
	cells.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for row := 1; row <= xlsxPreviewMaxRows+20; row++ {
		cells.WriteString(`<row r="` + itoa(row) + `"><c r="A` + itoa(row) + `" t="inlineStr"><is><t>` + strings.Repeat("X", xlsxPreviewMaxCellChars+50) + `</t></is></c><c r="ZZ` + itoa(row) + `"><v>9</v></c></row>`)
	}
	cells.WriteString(`</sheetData></worksheet>`)
	p := writeXLSXFixture(t, basicWorkbook(false), basicWorkbookRels(), map[string]string{
		"xl/worksheets/sheet1.xml": cells.String(),
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData/></worksheet>`,
	})
	wb, err := parseXLSXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	s := wb.Sheets[0]
	if !s.Truncated {
		t.Fatal("large sheet should report truncation")
	}
	if s.MaxRow > xlsxPreviewMaxRows || s.MaxCol > xlsxPreviewMaxCols {
		t.Fatalf("bounds exceeded rows=%d cols=%d", s.MaxRow, s.MaxCol)
	}
	if utf8.RuneCountInString(s.Cells["A1"]) > xlsxPreviewMaxCellChars+1 {
		t.Fatalf("cell not bounded: %d runes", utf8.RuneCountInString(s.Cells["A1"]))
	}
	if _, ok := s.Cells["ZZ1"]; ok {
		t.Fatal("column outside preview bound should not be stored")
	}
}

func TestXLSXRejectsMissingWorksheetRelationship(t *testing.T) {
	p := writeXLSXFixture(t, basicWorkbook(false), `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`, map[string]string{})
	if _, err := parseXLSXPreview(p); err == nil || !strings.Contains(strings.ToLower(err.Error()), "relationship") {
		t.Fatalf("expected relationship error, got %v", err)
	}
}

func itoa(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = digits[n%10]
		n /= 10
	}
	return string(b[i:])
}
