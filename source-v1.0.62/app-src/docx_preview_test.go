package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDOCXPreviewFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sample.docx")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	files := map[string]string{
		"[Content_Types].xml":          `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="` + contentTypeWordDocument + `"/></Types>`,
		"docProps/core.xml":            `<?xml version="1.0"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Bridge Report</dc:title><dc:creator>Laxman</dc:creator></cp:coreProperties>`,
		"word/styles.xml":              `<?xml version="1.0"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="Heading 2"/></w:style></w:styles>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rIdImg" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/site-photo.png"/></Relationships>`,
		"word/document.xml": `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><w:body>
          <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Bridge Inspection</w:t></w:r></w:p>
          <w:p><w:r><w:t xml:space="preserve">Site </w:t></w:r><w:r><w:t>verification</w:t><w:tab/><w:t>2026</w:t><w:br/><w:t>Second line</w:t></w:r></w:p>
          <w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr><w:r><w:t>Measurements</w:t></w:r></w:p>
          <w:tbl><w:tr><w:tc><w:p><w:r><w:t>Item</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Value</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>Length</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>24.5 m</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
          <w:p><w:r><w:drawing><wp:inline><wp:docPr id="1" name="Picture 1" descr="Bridge site photograph"/><a:graphic><a:graphicData><a:blip r:embed="rIdImg"/></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>
        </w:body></w:document>`,
		"word/media/site-photo.png": "fakepng",
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseDOCXStylesMapsParagraphStyleNames(t *testing.T) {
	got, err := parseDOCXStyles([]byte(`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/></w:style><w:style w:type="character" w:styleId="Emphasis"><w:name w:val="Emphasis"/></w:style></w:styles>`))
	if err != nil {
		t.Fatal(err)
	}
	if got["Heading1"] != "heading 1" {
		t.Fatalf("missing paragraph style: %#v", got)
	}
	if _, ok := got["Emphasis"]; ok {
		t.Fatalf("character style should not be included: %#v", got)
	}
}

func TestDOCXHeadingLevelUsesStyleNameAndID(t *testing.T) {
	cases := []struct {
		id, name string
		want     int
	}{
		{"Heading1", "heading 1", 1},
		{"Heading2", "Heading 2", 2},
		{"Heading3", "", 3},
		{"Title", "Title", 1},
		{"Normal", "Normal", 0},
	}
	for _, tc := range cases {
		if got := docxHeadingLevel(tc.id, tc.name); got != tc.want {
			t.Fatalf("docxHeadingLevel(%q,%q)=%d; want %d", tc.id, tc.name, got, tc.want)
		}
	}
}

func TestRenderDOCXPreviewIncludesBodyHeadingsTableAndImage(t *testing.T) {
	p := writeDOCXPreviewFixture(t)
	text, stats, err := renderDOCXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"DCM DOCX Preview",
		"Bridge Report",
		"Bridge Inspection",
		"=================",
		"Site verification\t2026",
		"Second line",
		"Measurements",
		"TABLE 1",
		"| Item | Value |",
		"| Length | 24.5 m |",
		"[Image: Bridge site photograph — site-photo.png]",
		"Rendered locally from DOCX Open XML",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("preview missing %q:\n%s", want, text)
		}
	}
	if stats.Paragraphs < 7 || stats.Headings != 2 || stats.Tables != 1 || stats.Images != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestDOCXPreviewPreservesBodyOrderAroundTable(t *testing.T) {
	p := writeDOCXPreviewFixture(t)
	text, _, err := renderDOCXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Index(text, "Measurements")
	table := strings.Index(text, "TABLE 1")
	image := strings.Index(text, "[Image:")
	if !(h >= 0 && table > h && image > table) {
		t.Fatalf("body order incorrect: heading=%d table=%d image=%d\n%s", h, table, image, text)
	}
}

func TestDOCXPreviewRejectsNonWordOpenXML(t *testing.T) {
	p := writeOpenXMLFixture(t, ".xlsx", map[string]string{"/xl/workbook.xml": contentTypeExcelWorkbook}, map[string]string{"xl/workbook.xml": `<workbook/>`})
	if _, _, err := renderDOCXPreview(p); err == nil {
		t.Fatal("XLSX must not be rendered by DOCX renderer")
	}
}

func TestDOCXPreviewOutputIsBounded(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big.docx")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	parts := map[string]string{
		"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="` + contentTypeWordDocument + `"/></Types>`,
		"word/document.xml":   `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + strings.Repeat("A", docxPreviewOutputLimit*2) + `</w:t></w:r></w:p></w:body></w:document>`,
	}
	for name, body := range parts {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	_ = f.Close()
	text, stats, err := renderDOCXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(text) > docxPreviewOutputLimit+2048 {
		t.Fatalf("preview not bounded: %d", len(text))
	}
	if !stats.Truncated || !strings.Contains(text, "Preview truncated") {
		t.Fatalf("expected truncation: stats=%#v len=%d", stats, len(text))
	}
}

func TestTextPreviewFontPixelHeightTracksZoom(t *testing.T) {
	cases := []struct{ zoom, want int }{{25, 8}, {100, 16}, {125, 20}, {200, 32}, {400, 64}}
	for _, tc := range cases {
		if got := textPreviewFontPixelHeight(tc.zoom); got != tc.want {
			t.Fatalf("textPreviewFontPixelHeight(%d)=%d; want %d", tc.zoom, got, tc.want)
		}
	}
}
