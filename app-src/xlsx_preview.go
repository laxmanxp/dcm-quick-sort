package main

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	xlsxWorkbookReadLimit      int64 = 2 * 1024 * 1024
	xlsxWorksheetReadLimit     int64 = 16 * 1024 * 1024
	xlsxSharedStringsReadLimit int64 = 8 * 1024 * 1024
	xlsxStylesReadLimit        int64 = 4 * 1024 * 1024
	xlsxPreviewMaxRows               = 500
	xlsxPreviewMaxCols               = 64
	xlsxPreviewMaxCellChars          = 512
	xlsxGridMaxDisplayChars          = 24
)

type XLSXSheetPreview struct {
	Name         string
	Part         string
	Cells        map[string]string
	MaxRow       int
	MaxCol       int
	MergedRanges []string
	Truncated    bool
}

type XLSXWorkbookPreview struct {
	Sheets   []XLSXSheetPreview
	Date1904 bool
}

type xlsxWorkbookSheet struct {
	Name  string `xml:"name,attr"`
	RelID string `xml:"id,attr"`
}

type xlsxWorkbookXML struct {
	WorkbookPr struct {
		Date1904 string `xml:"date1904,attr"`
	} `xml:"workbookPr"`
	Sheets struct {
		Sheet []xlsxWorkbookSheet `xml:"sheet"`
	} `xml:"sheets"`
}

type xlsxRichRun struct {
	Text string `xml:"t"`
}

type xlsxStringItem struct {
	Text string        `xml:"t"`
	Runs []xlsxRichRun `xml:"r"`
}

type xlsxSharedStringsXML struct {
	Items []xlsxStringItem `xml:"si"`
}

type xlsxStyleSheetXML struct {
	NumFmts struct {
		Formats []struct {
			ID   int    `xml:"numFmtId,attr"`
			Code string `xml:"formatCode,attr"`
		} `xml:"numFmt"`
	} `xml:"numFmts"`
	CellXfs struct {
		XFs []struct {
			NumFmtID int `xml:"numFmtId,attr"`
		} `xml:"xf"`
	} `xml:"cellXfs"`
}

type xlsxCellXML struct {
	Ref     string         `xml:"r,attr"`
	Type    string         `xml:"t,attr"`
	Style   int            `xml:"s,attr"`
	Value   string         `xml:"v"`
	Formula string         `xml:"f"`
	Inline  xlsxStringItem `xml:"is"`
}

type xlsxRowXML struct {
	Number int           `xml:"r,attr"`
	Cells  []xlsxCellXML `xml:"c"`
}

type xlsxWorksheetXML struct {
	SheetData struct {
		Rows []xlsxRowXML `xml:"row"`
	} `xml:"sheetData"`
	MergeCells struct {
		Cells []struct {
			Ref string `xml:"ref,attr"`
		} `xml:"mergeCell"`
	} `xml:"mergeCells"`
}

type xlsxStyleInfo struct {
	NumFmtID int
	IsDate   bool
}

func xlsxStringValue(item xlsxStringItem) string {
	if item.Text != "" {
		return item.Text
	}
	var b strings.Builder
	for _, run := range item.Runs {
		b.WriteString(run.Text)
	}
	return b.String()
}

func parseXLSXSharedStrings(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var doc xlsxSharedStringsXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse XLSX shared strings: %w", err)
	}
	out := make([]string, 0, len(doc.Items))
	for _, item := range doc.Items {
		out = append(out, boundXLSXCellText(xlsxStringValue(item)))
	}
	return out, nil
}

func xlsxBuiltInDateFormat(id int) bool {
	if id >= 14 && id <= 22 {
		return true
	}
	if id >= 27 && id <= 36 {
		return true
	}
	if id >= 45 && id <= 47 {
		return true
	}
	if id >= 50 && id <= 58 {
		return true
	}
	return false
}

func xlsxFormatLooksLikeDate(code string) bool {
	code = strings.ToLower(code)
	var b strings.Builder
	inQuote := false
	escaped := false
	for i := 0; i < len(code); i++ {
		ch := code[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		if ch == '_' || ch == '*' {
			if i+1 < len(code) {
				i++
			}
			continue
		}
		if ch == '[' {
			end := strings.IndexByte(code[i:], ']')
			if end >= 0 {
				token := code[i+1 : i+end]
				if strings.ContainsAny(token, "hms") {
					b.WriteString(token)
				}
				i += end
				continue
			}
		}
		b.WriteByte(ch)
	}
	s := b.String()
	if strings.ContainsAny(s, "ydhs") {
		return true
	}
	return strings.Contains(s, "m/") || strings.Contains(s, "/m") || strings.Contains(s, "m-") || strings.Contains(s, "-m")
}

func parseXLSXStyles(data []byte) ([]xlsxStyleInfo, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var doc xlsxStyleSheetXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse XLSX styles: %w", err)
	}
	custom := map[int]string{}
	for _, nf := range doc.NumFmts.Formats {
		custom[nf.ID] = nf.Code
	}
	out := make([]xlsxStyleInfo, len(doc.CellXfs.XFs))
	for i, xf := range doc.CellXfs.XFs {
		date := xlsxBuiltInDateFormat(xf.NumFmtID)
		if code, ok := custom[xf.NumFmtID]; ok && xlsxFormatLooksLikeDate(code) {
			date = true
		}
		out[i] = xlsxStyleInfo{NumFmtID: xf.NumFmtID, IsDate: date}
	}
	return out, nil
}

func parseBoolish(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseCellReference(ref string) (col, row int, ok bool) {
	ref = strings.TrimSpace(strings.ToUpper(ref))
	if ref == "" {
		return 0, 0, false
	}
	i := 0
	for i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z' {
		i++
	}
	if i == 0 || i == len(ref) {
		return 0, 0, false
	}
	col = 0
	for _, ch := range ref[:i] {
		col = col*26 + int(ch-'A'+1)
	}
	row64, err := strconv.ParseInt(ref[i:], 10, 32)
	if err != nil || row64 <= 0 {
		return 0, 0, false
	}
	return col, int(row64), true
}

func xlsxColumnName(col int) string {
	if col <= 0 {
		return ""
	}
	var buf [16]byte
	i := len(buf)
	for col > 0 {
		col--
		i--
		buf[i] = byte('A' + col%26)
		col /= 26
	}
	return string(buf[i:])
}

func excelSerialTime(serial float64, date1904 bool) (time.Time, bool) {
	if math.IsNaN(serial) || math.IsInf(serial, 0) || serial < 0 {
		return time.Time{}, false
	}
	days := math.Floor(serial)
	frac := serial - days
	var base time.Time
	if date1904 {
		base = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		base = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)
		// Excel deliberately preserves Lotus 1-2-3's fictitious 1900-02-29.
		if days >= 60 {
			days--
		}
	}
	seconds := math.Round(frac * 86400)
	return base.AddDate(0, 0, int(days)).Add(time.Duration(seconds) * time.Second), true
}

func formatExcelSerial(serial string, date1904 bool) string {
	n, err := strconv.ParseFloat(strings.TrimSpace(serial), 64)
	if err != nil {
		return strings.TrimSpace(serial)
	}
	// Keep Excel's historical leap-day display for the one impossible date.
	if !date1904 && math.Floor(n) == 60 {
		frac := n - math.Floor(n)
		if math.Abs(frac) < 1e-9 {
			return "1900-02-29"
		}
		seconds := int(math.Round(frac * 86400))
		return fmt.Sprintf("1900-02-29 %02d:%02d", (seconds/3600)%24, (seconds/60)%60)
	}
	tm, ok := excelSerialTime(n, date1904)
	if !ok {
		return strings.TrimSpace(serial)
	}
	if math.Abs(n-math.Floor(n)) < 1e-9 {
		return tm.Format("2006-01-02")
	}
	return tm.Format("2006-01-02 15:04")
}

func boundXLSXCellText(s string) string {
	s = normalizePreviewNewlines(s)
	if utf8.RuneCountInString(s) <= xlsxPreviewMaxCellChars {
		return s
	}
	r := []rune(s)
	if len(r) > xlsxPreviewMaxCellChars {
		r = r[:xlsxPreviewMaxCellChars]
	}
	return string(r) + "…"
}

func formatXLSXCell(cell xlsxCellXML, shared []string, styles []xlsxStyleInfo, date1904 bool) string {
	typ := strings.TrimSpace(cell.Type)
	raw := strings.TrimSpace(cell.Value)
	switch typ {
	case "s":
		idx, err := strconv.Atoi(raw)
		if err == nil && idx >= 0 && idx < len(shared) {
			return boundXLSXCellText(shared[idx])
		}
		return raw
	case "inlineStr":
		return boundXLSXCellText(xlsxStringValue(cell.Inline))
	case "b":
		if raw == "1" || strings.EqualFold(raw, "true") {
			return "TRUE"
		}
		return "FALSE"
	case "str":
		return boundXLSXCellText(raw)
	case "e":
		return boundXLSXCellText(raw)
	case "d":
		return boundXLSXCellText(raw)
	}
	if cell.Style >= 0 && cell.Style < len(styles) && styles[cell.Style].IsDate && raw != "" {
		return formatExcelSerial(raw, date1904)
	}
	if raw != "" {
		return boundXLSXCellText(raw)
	}
	if strings.TrimSpace(cell.Formula) != "" {
		return boundXLSXCellText("=" + strings.TrimSpace(cell.Formula))
	}
	return ""
}

func parseXLSXWorksheet(data []byte, name, part string, shared []string, styles []xlsxStyleInfo, date1904 bool) (XLSXSheetPreview, error) {
	var doc xlsxWorksheetXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return XLSXSheetPreview{}, fmt.Errorf("parse XLSX worksheet %q: %w", name, err)
	}
	s := XLSXSheetPreview{Name: name, Part: part, Cells: map[string]string{}}
	for _, mr := range doc.MergeCells.Cells {
		ref := strings.TrimSpace(mr.Ref)
		if ref != "" && len(s.MergedRanges) < 128 {
			s.MergedRanges = append(s.MergedRanges, ref)
		} else if ref != "" {
			s.Truncated = true
		}
	}
	for rowIdx, row := range doc.SheetData.Rows {
		rowNum := row.Number
		if rowNum <= 0 {
			rowNum = rowIdx + 1
		}
		fallbackCol := 1
		if rowNum > xlsxPreviewMaxRows {
			s.Truncated = true
			continue
		}
		for _, cell := range row.Cells {
			col, r, ok := parseCellReference(cell.Ref)
			if !ok {
				col, r = fallbackCol, rowNum
			}
			fallbackCol = col + 1
			if r <= 0 {
				r = rowNum
			}
			if r > xlsxPreviewMaxRows || col > xlsxPreviewMaxCols {
				s.Truncated = true
				continue
			}
			if col <= 0 {
				continue
			}
			ref := xlsxColumnName(col) + strconv.Itoa(r)
			s.Cells[ref] = formatXLSXCell(cell, shared, styles, date1904)
			if r > s.MaxRow {
				s.MaxRow = r
			}
			if col > s.MaxCol {
				s.MaxCol = col
			}
		}
	}
	return s, nil
}

func parseXLSXPreview(packagePath string) (XLSXWorkbookPreview, error) {
	info, err := inspectOpenXMLPackage(packagePath)
	if err != nil {
		return XLSXWorkbookPreview{}, err
	}
	if info.Kind != OpenXMLExcel {
		return XLSXWorkbookPreview{}, errors.New("not an XLSX workbook")
	}

	zr, err := zip.OpenReader(packagePath)
	if err != nil {
		return XLSXWorkbookPreview{}, fmt.Errorf("invalid XLSX package: %w", err)
	}
	defer zr.Close()
	if len(zr.File) > openXMLMaxEntries {
		return XLSXWorkbookPreview{}, fmt.Errorf("XLSX package has too many parts")
	}

	wbData, err := readZipEntryLimited(zr.File, info.MainPart, xlsxWorkbookReadLimit)
	if err != nil {
		return XLSXWorkbookPreview{}, err
	}
	var wbXML xlsxWorkbookXML
	if err := xml.Unmarshal(wbData, &wbXML); err != nil {
		return XLSXWorkbookPreview{}, fmt.Errorf("parse XLSX workbook: %w", err)
	}
	wb := XLSXWorkbookPreview{Date1904: parseBoolish(wbXML.WorkbookPr.Date1904)}

	rels, err := readOpenXMLRelationships(packagePath, info.MainPart)
	if err != nil {
		return XLSXWorkbookPreview{}, fmt.Errorf("read XLSX workbook relationships: %w", err)
	}
	relByID := map[string]OpenXMLRelationship{}
	for _, rel := range rels {
		relByID[rel.ID] = rel
	}

	var shared []string
	if findZipEntry(zr.File, "xl/sharedStrings.xml") != nil {
		data, err := readZipEntryLimited(zr.File, "xl/sharedStrings.xml", xlsxSharedStringsReadLimit)
		if err != nil {
			return XLSXWorkbookPreview{}, err
		}
		shared, err = parseXLSXSharedStrings(data)
		if err != nil {
			return XLSXWorkbookPreview{}, err
		}
	}
	var styles []xlsxStyleInfo
	if findZipEntry(zr.File, "xl/styles.xml") != nil {
		data, err := readZipEntryLimited(zr.File, "xl/styles.xml", xlsxStylesReadLimit)
		if err != nil {
			return XLSXWorkbookPreview{}, err
		}
		styles, err = parseXLSXStyles(data)
		if err != nil {
			return XLSXWorkbookPreview{}, err
		}
	}

	for _, rawSheet := range wbXML.Sheets.Sheet {
		name := strings.TrimSpace(rawSheet.Name)
		if name == "" {
			name = "Sheet" + strconv.Itoa(len(wb.Sheets)+1)
		}
		rel, ok := relByID[strings.TrimSpace(rawSheet.RelID)]
		if !ok || rel.External || rel.ResolvedTarget == "" || !strings.HasSuffix(strings.ToLower(rel.Type), "/worksheet") {
			return XLSXWorkbookPreview{}, fmt.Errorf("worksheet %q relationship %q is missing or invalid", name, rawSheet.RelID)
		}
		if findZipEntry(zr.File, rel.ResolvedTarget) == nil {
			return XLSXWorkbookPreview{}, fmt.Errorf("worksheet %q part %q is missing", name, rel.ResolvedTarget)
		}
		data, err := readZipEntryLimited(zr.File, rel.ResolvedTarget, xlsxWorksheetReadLimit)
		if err != nil {
			return XLSXWorkbookPreview{}, err
		}
		sheet, err := parseXLSXWorksheet(data, name, rel.ResolvedTarget, shared, styles, wb.Date1904)
		if err != nil {
			return XLSXWorkbookPreview{}, err
		}
		wb.Sheets = append(wb.Sheets, sheet)
	}
	if len(wb.Sheets) == 0 {
		return XLSXWorkbookPreview{}, errors.New("XLSX workbook contains no worksheets")
	}
	return wb, nil
}

func cleanXLSXGridCell(s string) string {
	s = normalizePreviewNewlines(s)
	s = strings.ReplaceAll(s, "\n", " ↵ ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > xlsxGridMaxDisplayChars {
		return string(r[:xlsxGridMaxDisplayChars-1]) + "…"
	}
	return s
}

func padXLSXGridCell(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		r = r[:width]
	}
	if len(r) < width {
		return string(r) + strings.Repeat(" ", width-len(r))
	}
	return string(r)
}

func renderXLSXSheetGrid(s XLSXSheetPreview) string {
	maxRow, maxCol := s.MaxRow, s.MaxCol
	if maxRow < 1 {
		maxRow = 1
	}
	if maxCol < 1 {
		maxCol = 1
	}
	if maxRow > xlsxPreviewMaxRows {
		maxRow = xlsxPreviewMaxRows
	}
	if maxCol > xlsxPreviewMaxCols {
		maxCol = xlsxPreviewMaxCols
	}

	widths := make([]int, maxCol+1)
	for c := 1; c <= maxCol; c++ {
		widths[c] = len(xlsxColumnName(c))
		if widths[c] < 8 {
			widths[c] = 8
		}
	}
	for ref, v := range s.Cells {
		col, row, ok := parseCellReference(ref)
		if !ok || row > maxRow || col > maxCol {
			continue
		}
		n := utf8.RuneCountInString(cleanXLSXGridCell(v))
		if n > xlsxGridMaxDisplayChars {
			n = xlsxGridMaxDisplayChars
		}
		if n > widths[col] {
			widths[col] = n
		}
	}
	var b strings.Builder
	b.WriteString("Sheet: " + s.Name + "\n")
	if len(s.MergedRanges) > 0 {
		merges := append([]string(nil), s.MergedRanges...)
		sort.Strings(merges)
		b.WriteString("Merged: " + strings.Join(merges, ", ") + "\n")
	}
	if s.Truncated {
		b.WriteString("Preview limited to the first 500 rows, 64 columns, and bounded cell text.\n")
	}
	b.WriteString("\n")
	rowW := len(strconv.Itoa(maxRow))
	if rowW < 3 {
		rowW = 3
	}
	b.WriteString(strings.Repeat(" ", rowW) + " | ")
	for c := 1; c <= maxCol; c++ {
		b.WriteString(padXLSXGridCell(xlsxColumnName(c), widths[c]))
		if c < maxCol {
			b.WriteString(" | ")
		}
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", rowW) + "-+-")
	for c := 1; c <= maxCol; c++ {
		b.WriteString(strings.Repeat("-", widths[c]))
		if c < maxCol {
			b.WriteString("-+-")
		}
	}
	b.WriteString("\n")
	for r := 1; r <= maxRow; r++ {
		b.WriteString(fmt.Sprintf("%*d | ", rowW, r))
		for c := 1; c <= maxCol; c++ {
			ref := xlsxColumnName(c) + strconv.Itoa(r)
			b.WriteString(padXLSXGridCell(cleanXLSXGridCell(s.Cells[ref]), widths[c]))
			if c < maxCol {
				b.WriteString(" | ")
			}
		}
		if r < maxRow {
			b.WriteString("\n")
		}
	}
	return normalizePreviewNewlines(b.String())
}

func xlsxPreviewForPath(path string) (XLSXWorkbookPreview, bool) {
	if strings.ToLower(filepath.Ext(path)) != ".xlsx" {
		return XLSXWorkbookPreview{}, false
	}
	wb, err := parseXLSXPreview(path)
	return wb, err == nil
}
