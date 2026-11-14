package main

import (
	"strings"
	"testing"
)

// Rejected DOCX contents must remain unavailable throughout the local fallback
// chain. Legitimate blank documents and subsequent valid previews still succeed.
// The installed Windows provider and presenter are qualified separately.
type dcm34DOCXProbeCase struct {
	name            string
	documentXML     string
	missingMainPart bool
	valid           bool
}

func dcm34DOCXProbeCases() []dcm34DOCXProbeCase {
	const ns = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	return []dcm34DOCXProbeCase{
		{name: "empty_main_part", documentXML: ""},
		{name: "missing_document_element", documentXML: `<w:body ` + ns + `/>`},
		{name: "missing_body_element", documentXML: `<w:document ` + ns + `/>`},
		{name: "malformed_document_xml", documentXML: `<w:document ` + ns + `><w:body></w:document>`},
		{name: "malformed_after_readable_paragraph", documentXML: `<w:document ` + ns + `><w:body><w:p><w:r><w:t>Partial content must not survive</w:t></w:r></w:p></w:document>`},
		{name: "main_part_read_limit", documentXML: `<w:document ` + ns + `><w:body><w:p><w:r><w:t>` + strings.Repeat("A", int(docxMainPartReadLimit)) + `</w:t></w:r></w:p></w:body></w:document>`},
		{name: "missing_main_part_guard", missingMainPart: true},
		{name: "valid_blank_document", documentXML: `<w:document ` + ns + `><w:body/></w:document>`, valid: true},
	}
}

func writeDCM34DOCXProbeFixture(t *testing.T, tc dcm34DOCXProbeCase) string {
	t.Helper()
	parts := map[string]string{}
	if !tc.missingMainPart {
		parts["word/document.xml"] = tc.documentXML
	}
	return writeOpenXMLFixture(t, ".docx", map[string]string{
		"/word/document.xml": contentTypeWordDocument,
	}, parts)
}

func TestDCM34DOCXDamageStates(t *testing.T) {
	for _, tc := range dcm34DOCXProbeCases() {
		t.Run(tc.name, func(t *testing.T) {
			path := writeDCM34DOCXProbeFixture(t, tc)
			info, inspectErr := inspectOpenXMLPackage(path)
			t.Logf("container: kind=%q mainPart=%q err=%v", info.Kind, info.MainPart, inspectErr)
			if tc.missingMainPart {
				if inspectErr == nil {
					t.Fatal("fixture guard: missing main part unexpectedly passed package inspection")
				}
			} else if inspectErr != nil {
				t.Fatalf("fixture setup: valid ZIP/content types/main-part presence failed: %v", inspectErr)
			}

			t.Run("renderDOCXPreview", func(t *testing.T) {
				text, stats, err := renderDOCXPreview(path)
				t.Logf("result: err=%v text_bytes=%d stats=%+v", err, len(text), stats)
				if tc.valid {
					if err != nil || !strings.Contains(text, "DCM DOCX Preview") || stats != (DOCXPreviewStats{}) {
						t.Errorf("valid blank DOCX must succeed with zero body statistics: err=%v text=%q stats=%+v", err, text, stats)
					}
				} else if err == nil || text != "" {
					t.Errorf("damaged DOCX must not render as successful blank: err=%v text_bytes=%d stats=%+v", err, len(text), stats)
				}
			})

			t.Run("docxPreviewText", func(t *testing.T) {
				text, stats, ok := docxPreviewText(path)
				t.Logf("result: ok=%t text_bytes=%d stats=%+v", ok, len(text), stats)
				if tc.valid {
					if !ok || !strings.Contains(text, "DCM DOCX Preview") || stats != (DOCXPreviewStats{}) {
						t.Errorf("valid blank DOCX wrapper must remain successful: ok=%t text=%q stats=%+v", ok, text, stats)
					}
				} else if ok || text != "" {
					t.Errorf("damaged DOCX wrapper must not report success: ok=%t text_bytes=%d stats=%+v", ok, len(text), stats)
				}
			})

			t.Run("officeFoundationPreviewText", func(t *testing.T) {
				text, ok := officeFoundationPreviewText(path)
				validated := strings.Contains(text, "Open XML package validated.")
				t.Logf("result: ok=%t text_bytes=%d validated_label=%t", ok, len(text), validated)
				if tc.valid {
					if !ok || !validated {
						t.Errorf("valid blank DOCX foundation must remain successful: ok=%t text=%q", ok, text)
					}
				} else if ok || text != "" {
					t.Errorf("damaged DOCX must not regain success through metadata fallback: ok=%t text_bytes=%d validated_label=%t", ok, len(text), validated)
				}
			})
		})
	}
}

func TestDCM34DOCXRecoveryAfterDamagedInput(t *testing.T) {
	for _, tc := range dcm34DOCXProbeCases() {
		if tc.valid {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			damagedPath := writeDCM34DOCXProbeFixture(t, tc)
			_, _, _ = renderDOCXPreview(damagedPath)
			_, _, _ = docxPreviewText(damagedPath)
			_, _ = officeFoundationPreviewText(damagedPath)

			validPath := writeDOCXPreviewFixture(t)
			text, stats, err := renderDOCXPreview(validPath)
			if err != nil || !strings.Contains(text, "Bridge Inspection") || !strings.Contains(text, "| Length | 24.5 m |") || stats.Tables != 1 || stats.Images != 1 {
				t.Fatalf("valid DOCX after damaged input must render its own contents: err=%v text=%q stats=%+v", err, text, stats)
			}
			wrappedText, wrappedStats, ok := docxPreviewText(validPath)
			if !ok || wrappedText != text || wrappedStats != stats {
				t.Fatalf("valid DOCX wrapper after damaged input must preserve successful result: ok=%t stats=%+v", ok, wrappedStats)
			}
			t.Logf("recovery: text_bytes=%d stats=%+v required heading/table present", len(text), stats)
		})
	}
}

func TestDCM34DOCXDocumentStructure(t *testing.T) {
	const ns = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	const blank = `<w:document ` + ns + `><w:body/></w:document>`
	cases := []struct {
		name  string
		xml   string
		valid bool
	}{
		{name: "wrong_root", xml: `<w:other ` + ns + `><w:body/></w:other>`},
		{name: "nested_document", xml: `<wrapper>` + blank + `</wrapper>`},
		{name: "nested_body", xml: `<w:document ` + ns + `><wrapper><w:body/></wrapper></w:document>`},
		{name: "duplicate_body", xml: `<w:document ` + ns + `><w:body/><w:body/></w:document>`},
		{name: "second_document", xml: blank + blank},
		{name: "leading_text", xml: `not XML` + blank},
		{name: "trailing_text", xml: blank + `not XML`},
		{name: "wrong_document_namespace", xml: `<w:document xmlns:w="urn:not-word"><w:body/></w:document>`},
		{name: "wrong_body_namespace", xml: `<w:document ` + ns + `><body xmlns="urn:not-word"/></w:document>`},
		{name: "valid_blank", xml: blank, valid: true},
		{name: "valid_blank_with_utf8_bom", xml: "\uFEFF" + blank, valid: true},
		{name: "valid_blank_with_prolog_and_comments", xml: `<?xml version="1.0"?><!-- before -->` + blank + `<!-- after -->`, valid: true},
		{name: "valid_blank_strict_namespace", xml: `<w:document xmlns:w="http://purl.oclc.org/ooxml/wordprocessingml/main"><w:body/></w:document>`, valid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeDCM34DOCXProbeFixture(t, dcm34DOCXProbeCase{documentXML: tc.xml})
			text, stats, err := renderDOCXPreview(path)
			foundation, ok := officeFoundationPreviewText(path)
			if tc.valid {
				if err != nil || !strings.Contains(text, "DCM DOCX Preview") || stats != (DOCXPreviewStats{}) || !ok {
					t.Fatalf("valid blank structure rejected: err=%v stats=%+v foundation_ok=%t", err, stats, ok)
				}
			} else if err == nil || text != "" || ok || foundation != "" {
				t.Fatalf("invalid structure regained success: err=%v text_bytes=%d foundation_ok=%t foundation_bytes=%d", err, len(text), ok, len(foundation))
			}
		})
	}
}

func TestDCM34DOCXStrictContent(t *testing.T) {
	path := writeDCM34DOCXProbeFixture(t, dcm34DOCXProbeCase{documentXML: `<w:document xmlns:w="http://purl.oclc.org/ooxml/wordprocessingml/main" xmlns:r="http://purl.oclc.org/ooxml/officeDocument/relationships" xmlns:a="http://purl.oclc.org/ooxml/drawingml/main"><w:body>
<w:p><w:r><w:t>Strict document content</w:t></w:r></w:p>
<w:tbl><w:tr><w:tc><w:p><w:r><w:t>Cell</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
<w:p><w:r><w:drawing><a:blip r:embed="rIdPhoto"/></w:drawing></w:r></w:p>
</w:body></w:document>`})
	text, stats, err := renderDOCXPreview(path)
	if err != nil || !strings.Contains(text, "Strict document content") || !strings.Contains(text, "| Cell |") || !strings.Contains(text, "[Image]") || stats.Paragraphs != 3 || stats.Tables != 1 || stats.Images != 1 {
		t.Fatalf("valid Strict body lost content: err=%v stats=%+v text=%q", err, stats, text)
	}
	if wrapped, wrappedStats, ok := docxPreviewText(path); !ok || wrapped != text || wrappedStats != stats {
		t.Fatalf("valid Strict wrapper changed body: ok=%t stats=%+v", ok, wrappedStats)
	}
	if text, ok := officeFoundationPreviewText(path); !ok || !strings.Contains(text, "Open XML package validated.") {
		t.Fatalf("valid Strict foundation rejected: ok=%t text=%q", ok, text)
	}
}
