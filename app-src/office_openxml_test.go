package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeOpenXMLFixture(t *testing.T, ext string, overrides map[string]string, extra map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fixture"+ext)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	var types strings.Builder
	types.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	types.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	for part, contentType := range overrides {
		types.WriteString(`<Override PartName="` + part + `" ContentType="` + contentType + `"/>`)
	}
	types.WriteString(`</Types>`)
	files := map[string]string{
		"[Content_Types].xml": types.String(),
		"docProps/core.xml": `<?xml version="1.0" encoding="UTF-8"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
 xmlns:dc="http://purl.org/dc/elements/1.1/"
 xmlns:dcterms="http://purl.org/dc/terms/"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
 <dc:title>Bridge Inspection Report</dc:title>
 <dc:subject>Site verification</dc:subject>
 <dc:creator>Laxman</dc:creator>
 <cp:lastModifiedBy>DCM Test</cp:lastModifiedBy>
 <dcterms:modified xsi:type="dcterms:W3CDTF">2026-09-21T10:15:00Z</dcterms:modified>
</cp:coreProperties>`,
		"docProps/app.xml": `<?xml version="1.0" encoding="UTF-8"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties">
 <Application>Microsoft Office</Application><Pages>4</Pages><Words>1234</Words><Slides>7</Slides>
</Properties>`,
	}
	for k, v := range extra {
		files[k] = v
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

func TestOpenXMLCandidateExtensions(t *testing.T) {
	cases := []struct {
		name string
		kind OpenXMLKind
		ok   bool
	}{
		{"report.docx", OpenXMLWord, true},
		{"BOOK.XLSX", OpenXMLExcel, true},
		{"slides.PptX", OpenXMLPowerPoint, true},
		{"old.doc", OpenXMLNone, false},
		{"macro.xlsm", OpenXMLNone, false},
		{"archive.zip", OpenXMLNone, false},
	}
	for _, tc := range cases {
		got, ok := openXMLKindForPath(tc.name)
		if got != tc.kind || ok != tc.ok {
			t.Fatalf("openXMLKindForPath(%q) = (%q,%v); want (%q,%v)", tc.name, got, ok, tc.kind, tc.ok)
		}
	}
}

func TestInspectOpenXMLPackageRecognizesWord(t *testing.T) {
	p := writeOpenXMLFixture(t, ".docx", map[string]string{
		"/word/document.xml": contentTypeWordDocument,
	}, map[string]string{"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`})
	info, err := inspectOpenXMLPackage(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != OpenXMLWord || info.MainPart != "word/document.xml" {
		t.Fatalf("unexpected Word package: %#v", info)
	}
	if info.Core.Title != "Bridge Inspection Report" || info.Core.Creator != "Laxman" {
		t.Fatalf("core properties missing: %#v", info.Core)
	}
	if info.Extended.Application != "Microsoft Office" || info.Extended.Pages != 4 || info.Extended.Words != 1234 {
		t.Fatalf("extended properties missing: %#v", info.Extended)
	}
	if info.EntryCount < 4 {
		t.Fatalf("entry count too small: %d", info.EntryCount)
	}
}

func TestInspectOpenXMLPackageRecognizesExcel(t *testing.T) {
	p := writeOpenXMLFixture(t, ".xlsx", map[string]string{
		"/xl/workbook.xml": contentTypeExcelWorkbook,
	}, map[string]string{"xl/workbook.xml": `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"/>`})
	info, err := inspectOpenXMLPackage(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != OpenXMLExcel || info.MainPart != "xl/workbook.xml" {
		t.Fatalf("unexpected Excel package: %#v", info)
	}
}

func TestInspectOpenXMLPackageRecognizesPowerPoint(t *testing.T) {
	p := writeOpenXMLFixture(t, ".pptx", map[string]string{
		"/ppt/presentation.xml": contentTypePowerPointPresentation,
	}, map[string]string{"ppt/presentation.xml": `<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"/>`})
	info, err := inspectOpenXMLPackage(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != OpenXMLPowerPoint || info.MainPart != "ppt/presentation.xml" {
		t.Fatalf("unexpected PowerPoint package: %#v", info)
	}
	if info.Extended.Slides != 7 {
		t.Fatalf("expected slide metadata, got %#v", info.Extended)
	}
}

func TestInspectOpenXMLPackageRejectsRenamedZip(t *testing.T) {
	p := writeOpenXMLFixture(t, ".docx", map[string]string{
		"/xl/workbook.xml": contentTypeExcelWorkbook,
	}, map[string]string{"xl/workbook.xml": `<workbook/>`})
	_, err := inspectOpenXMLPackage(p)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "does not match") {
		t.Fatalf("expected type mismatch error, got %v", err)
	}
}

func TestInspectOpenXMLPackageRejectsOrdinaryZipRenamedDocx(t *testing.T) {
	p := filepath.Join(t.TempDir(), "not-office.docx")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("hello.txt")
	_, _ = w.Write([]byte("hello"))
	_ = zw.Close()
	_ = f.Close()
	_, err = inspectOpenXMLPackage(p)
	if err == nil {
		t.Fatal("ordinary ZIP renamed to DOCX must not be accepted")
	}
}

func TestReadZipEntryLimitedRejectsOversizedXML(t *testing.T) {
	p := writeOpenXMLFixture(t, ".docx", map[string]string{
		"/word/document.xml": contentTypeWordDocument,
	}, map[string]string{"word/document.xml": strings.Repeat("A", 1024)})
	zr, err := zip.OpenReader(p)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	_, err = readZipEntryLimited(zr.File, "word/document.xml", 32)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "limit") {
		t.Fatalf("expected read limit error, got %v", err)
	}
}

func TestOfficeFoundationPreviewSummary(t *testing.T) {
	p := writeOpenXMLFixture(t, ".docx", map[string]string{
		"/word/document.xml": contentTypeWordDocument,
	}, map[string]string{"word/document.xml": `<w:document/>`})
	text, ok := officeFoundationPreviewText(p)
	if !ok {
		t.Fatal("expected built-in Office foundation preview")
	}
	for _, want := range []string{
		"DCM Office Preview",
		"Word document (DOCX)",
		"Bridge Inspection Report",
		"Laxman",
		"Open XML package validated",
		"Full document rendering",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("preview summary missing %q:\n%s", want, text)
		}
	}
}

func TestOfficeFoundationPreviewRefusesCorruptPackage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "broken.xlsx")
	if err := os.WriteFile(p, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	text, ok := officeFoundationPreviewText(p)
	if ok || text != "" {
		t.Fatalf("corrupt package should fall back normally, got ok=%v text=%q", ok, text)
	}
}

func TestReadZipEntryLimitedReturnsExactBytes(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("a.xml")
	_, _ = w.Write([]byte("<a>ok</a>"))
	_ = zw.Close()
	path := filepath.Join(t.TempDir(), "x.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	got, err := readZipEntryLimited(zr.File, "a.xml", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "<a>ok</a>" {
		t.Fatalf("unexpected content %q", got)
	}
}

func TestOpenXMLRelationshipPartName(t *testing.T) {
	cases := []struct{ part, want string }{
		{"word/document.xml", "word/_rels/document.xml.rels"},
		{"xl/workbook.xml", "xl/_rels/workbook.xml.rels"},
		{"ppt/presentation.xml", "ppt/_rels/presentation.xml.rels"},
		{"", "_rels/.rels"},
	}
	for _, tc := range cases {
		if got := openXMLRelationshipPartName(tc.part); got != tc.want {
			t.Fatalf("openXMLRelationshipPartName(%q) = %q; want %q", tc.part, got, tc.want)
		}
	}
}

func TestResolveOpenXMLRelationshipTarget(t *testing.T) {
	cases := []struct{ base, target, want string }{
		{"word/document.xml", "media/image1.png", "word/media/image1.png"},
		{"xl/workbook.xml", "worksheets/sheet1.xml", "xl/worksheets/sheet1.xml"},
		{"ppt/presentation.xml", "/ppt/slides/slide1.xml", "ppt/slides/slide1.xml"},
		{"word/document.xml", "../customXml/item1.xml", "customXml/item1.xml"},
	}
	for _, tc := range cases {
		got, err := resolveOpenXMLRelationshipTarget(tc.base, tc.target)
		if err != nil {
			t.Fatalf("resolve %q + %q: %v", tc.base, tc.target, err)
		}
		if got != tc.want {
			t.Fatalf("resolve %q + %q = %q; want %q", tc.base, tc.target, got, tc.want)
		}
	}
}

func TestResolveOpenXMLRelationshipTargetRejectsTraversal(t *testing.T) {
	if _, err := resolveOpenXMLRelationshipTarget("word/document.xml", "../../../../evil.bin"); err == nil {
		t.Fatal("relationship target escaping package root must be rejected")
	}
}

func TestReadOpenXMLRelationships(t *testing.T) {
	p := writeOpenXMLFixture(t, ".docx", map[string]string{
		"/word/document.xml": contentTypeWordDocument,
	}, map[string]string{
		"word/document.xml": `<w:document/>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
 <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image1.png"/>
 <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com" TargetMode="External"/>
</Relationships>`,
		"word/media/image1.png": "fake",
	})
	rels, err := readOpenXMLRelationships(p, "word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) != 2 {
		t.Fatalf("expected 2 relationships, got %#v", rels)
	}
	if rels[0].ID != "rId1" || rels[0].ResolvedTarget != "word/media/image1.png" || rels[0].External {
		t.Fatalf("bad internal relationship: %#v", rels[0])
	}
	if rels[1].ID != "rId2" || !rels[1].External || rels[1].ResolvedTarget != "" {
		t.Fatalf("bad external relationship: %#v", rels[1])
	}
}

func TestReadOpenXMLPart(t *testing.T) {
	p := writeOpenXMLFixture(t, ".xlsx", map[string]string{
		"/xl/workbook.xml": contentTypeExcelWorkbook,
	}, map[string]string{"xl/workbook.xml": `<workbook><sheets/></workbook>`})
	got, err := readOpenXMLPart(p, "xl/workbook.xml", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "<sheets/>") {
		t.Fatalf("unexpected part content %q", got)
	}
}

func TestInspectOpenXMLPackageAcceptsDefaultContentTypeWithRootOfficeRelationship(t *testing.T) {
	p := filepath.Join(t.TempDir(), "default-main.xlsx")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":     `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheets/></workbook>`,
	}
	for name, body := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	_ = f.Close()
	info, err := inspectOpenXMLPackage(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Kind != OpenXMLExcel || info.MainPart != "xl/workbook.xml" {
		t.Fatalf("unexpected info: %#v", info)
	}
}
