package main

import (
	"strings"
	"testing"
)

func basicPresentation() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
 <p:sldIdLst>
  <p:sldId id="256" r:id="rId1"/>
  <p:sldId id="257" r:id="rId2"/>
 </p:sldIdLst>
 <p:sldSz cx="12192000" cy="6858000"/>
</p:presentation>`
}

func basicPresentationRels() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
 <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
 <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide2.xml"/>
</Relationships>`
}

func writePPTXFixture(t *testing.T, presentation, rels string, extra map[string]string) string {
	t.Helper()
	parts := map[string]string{
		"ppt/presentation.xml":            presentation,
		"ppt/_rels/presentation.xml.rels": rels,
	}
	for k, v := range extra {
		parts[k] = v
	}
	return writeOpenXMLFixture(t, ".pptx", map[string]string{
		"/ppt/presentation.xml": contentTypePowerPointPresentation,
	}, parts)
}

func TestPPTXDiscoversSlidesInPresentationOrder(t *testing.T) {
	p := writePPTXFixture(t, basicPresentation(), basicPresentationRels(), map[string]string{
		"ppt/slides/slide1.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree/></p:cSld></p:sld>`,
		"ppt/slides/slide2.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree/></p:cSld></p:sld>`,
	})
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if deck.Width != 12192000 || deck.Height != 6858000 {
		t.Fatalf("bad slide size %dx%d", deck.Width, deck.Height)
	}
	if len(deck.Slides) != 2 {
		t.Fatalf("expected 2 slides, got %#v", deck.Slides)
	}
	if deck.Slides[0].Part != "ppt/slides/slide1.xml" || deck.Slides[1].Part != "ppt/slides/slide2.xml" {
		t.Fatalf("presentation order not preserved: %#v", deck.Slides)
	}
}

func TestPPTXParsesTextBreaksAndShapeGeometry(t *testing.T) {
	slide := `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
 <p:cSld><p:spTree>
  <p:sp>
   <p:nvSpPr><p:cNvPr id="2" name="Title 1" descr="Main title"/></p:nvSpPr>
   <p:spPr><a:xfrm><a:off x="100" y="200"/><a:ext cx="3000" cy="400"/></a:xfrm></p:spPr>
   <p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:t>Bridge</a:t></a:r><a:br/><a:r><a:t>Inspection</a:t></a:r></a:p><a:p><a:r><a:t>September 2026</a:t></a:r></a:p></p:txBody>
  </p:sp>
 </p:spTree></p:cSld></p:sld>`
	p := writePPTXFixture(t, basicPresentation(), basicPresentationRels(), map[string]string{
		"ppt/slides/slide1.xml": slide,
		"ppt/slides/slide2.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree/></p:cSld></p:sld>`,
	})
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(deck.Slides[0].Shapes) != 1 {
		t.Fatalf("shapes=%#v", deck.Slides[0].Shapes)
	}
	s := deck.Slides[0].Shapes[0]
	if s.Kind != PPTXShapeText || s.X != 100 || s.Y != 200 || s.CX != 3000 || s.CY != 400 {
		t.Fatalf("bad text shape %#v", s)
	}
	if s.Text != normalizePreviewNewlines("Bridge\nInspection\nSeptember 2026") {
		t.Fatalf("text=%q", s.Text)
	}
	if s.AltText != "Main title" || s.Name != "Title 1" {
		t.Fatalf("metadata=%#v", s)
	}
}

func TestPPTXParsesImageRelationshipAndAltText(t *testing.T) {
	slide := `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
 <p:cSld><p:spTree>
  <p:pic>
   <p:nvPicPr><p:cNvPr id="4" name="Picture 3" descr="Bridge site photo"/></p:nvPicPr>
   <p:blipFill><a:blip r:embed="rId5"/></p:blipFill>
   <p:spPr><a:xfrm><a:off x="500" y="700"/><a:ext cx="4200" cy="2600"/></a:xfrm></p:spPr>
  </p:pic>
 </p:spTree></p:cSld></p:sld>`
	p := writePPTXFixture(t, basicPresentation(), basicPresentationRels(), map[string]string{
		"ppt/slides/slide1.xml":            slide,
		"ppt/slides/_rels/slide1.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId5" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/></Relationships>`,
		"ppt/media/image1.png":             "fake image bytes",
		"ppt/slides/slide2.xml":            `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree/></p:cSld></p:sld>`,
	})
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	s := deck.Slides[0].Shapes[0]
	if s.Kind != PPTXShapeImage || s.ImagePart != "ppt/media/image1.png" {
		t.Fatalf("bad image shape %#v", s)
	}
	if s.AltText != "Bridge site photo" || s.Name != "Picture 3" {
		t.Fatalf("bad image metadata %#v", s)
	}
}

func TestPPTXParsesTableText(t *testing.T) {
	slide := `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
 <p:cSld><p:spTree><p:graphicFrame>
  <p:nvGraphicFramePr><p:cNvPr id="5" name="Table 1"/></p:nvGraphicFramePr>
  <p:xfrm><a:off x="100" y="100"/><a:ext cx="8000" cy="3000"/></p:xfrm>
  <a:graphic><a:graphicData><a:tbl>
   <a:tr><a:tc><a:txBody><a:p><a:r><a:t>Scheme</a:t></a:r></a:p></a:txBody></a:tc><a:tc><a:txBody><a:p><a:r><a:t>Amount</a:t></a:r></a:p></a:txBody></a:tc></a:tr>
   <a:tr><a:tc><a:txBody><a:p><a:r><a:t>Bridge</a:t></a:r></a:p></a:txBody></a:tc><a:tc><a:txBody><a:p><a:r><a:t>125000</a:t></a:r></a:p></a:txBody></a:tc></a:tr>
  </a:tbl></a:graphicData></a:graphic>
 </p:graphicFrame></p:spTree></p:cSld></p:sld>`
	p := writePPTXFixture(t, basicPresentation(), basicPresentationRels(), map[string]string{
		"ppt/slides/slide1.xml": slide,
		"ppt/slides/slide2.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree/></p:cSld></p:sld>`,
	})
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	s := deck.Slides[0].Shapes[0]
	if s.Kind != PPTXShapeTable || !strings.Contains(s.Text, "Scheme\tAmount") || !strings.Contains(s.Text, "Bridge\t125000") {
		t.Fatalf("bad table %#v", s)
	}
}

func TestPPTXMarksHiddenSlide(t *testing.T) {
	p := writePPTXFixture(t, basicPresentation(), basicPresentationRels(), map[string]string{
		"ppt/slides/slide1.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" show="0"><p:cSld><p:spTree/></p:cSld></p:sld>`,
		"ppt/slides/slide2.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree/></p:cSld></p:sld>`,
	})
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if !deck.Slides[0].Hidden || deck.Slides[1].Hidden {
		t.Fatalf("hidden flags wrong %#v", deck.Slides)
	}
}

func TestPPTXBrokenSlideDoesNotKillDeck(t *testing.T) {
	p := writePPTXFixture(t, basicPresentation(), basicPresentationRels(), map[string]string{
		// slide1 intentionally missing
		"ppt/slides/slide2.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree><p:sp><p:nvSpPr><p:cNvPr id="2" name="Text"/></p:nvSpPr><p:txBody><a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:r><a:t>Still visible</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`,
	})
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(deck.Slides) != 2 {
		t.Fatalf("slides=%d", len(deck.Slides))
	}
	if deck.Slides[0].Error == "" {
		t.Fatal("missing slide should be represented as per-slide error")
	}
	if deck.Slides[1].Error != "" || len(deck.Slides[1].Shapes) == 0 {
		t.Fatalf("second slide lost: %#v", deck.Slides[1])
	}
}

func TestPPTXRejectsMissingSlideRelationship(t *testing.T) {
	p := writePPTXFixture(t, basicPresentation(), `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`, map[string]string{})
	if _, err := parsePPTXPreview(p); err == nil || !strings.Contains(strings.ToLower(err.Error()), "relationship") {
		t.Fatalf("expected relationship error, got %v", err)
	}
}

func TestPPTXBoundsSlideCount(t *testing.T) {
	var ids strings.Builder
	var rels strings.Builder
	ids.WriteString(`<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst>`)
	rels.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	extra := map[string]string{}
	for i := 1; i <= pptxPreviewMaxSlides+5; i++ {
		ids.WriteString(`<p:sldId id="` + itoa(255+i) + `" r:id="rId` + itoa(i) + `"/>`)
		rels.WriteString(`<Relationship Id="rId` + itoa(i) + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide` + itoa(i) + `.xml"/>`)
		extra["ppt/slides/slide"+itoa(i)+".xml"] = `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree/></p:cSld></p:sld>`
	}
	ids.WriteString(`</p:sldIdLst><p:sldSz cx="1000" cy="600"/></p:presentation>`)
	rels.WriteString(`</Relationships>`)
	p := writePPTXFixture(t, ids.String(), rels.String(), extra)
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(deck.Slides) != pptxPreviewMaxSlides || !deck.Truncated {
		t.Fatalf("slide bounds not enforced: %d truncated=%v", len(deck.Slides), deck.Truncated)
	}
}

func TestPPTXBoundsShapesAndText(t *testing.T) {
	presentation := `<?xml version="1.0"?><p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst><p:sldSz cx="1000" cy="600"/></p:presentation>`
	rels := `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/></Relationships>`
	var shapes strings.Builder
	shapes.WriteString(`<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree>`)
	for j := 0; j < pptxPreviewMaxShapesPerSlide+5; j++ {
		body := "x"
		if j == 0 {
			body = strings.Repeat("X", pptxPreviewMaxShapeTextChars+20)
		}
		shapes.WriteString(`<p:sp><p:nvSpPr><p:cNvPr id="` + itoa(j+2) + `" name="S"/></p:nvSpPr><p:txBody><a:p><a:r><a:t>` + body + `</a:t></a:r></a:p></p:txBody></p:sp>`)
	}
	shapes.WriteString(`</p:spTree></p:cSld></p:sld>`)
	p := writePPTXFixture(t, presentation, rels, map[string]string{"ppt/slides/slide1.xml": shapes.String()})
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(deck.Slides[0].Shapes) != pptxPreviewMaxShapesPerSlide || !deck.Slides[0].Truncated {
		t.Fatalf("shape bounds not enforced: %d truncated=%v", len(deck.Slides[0].Shapes), deck.Slides[0].Truncated)
	}
	if len([]rune(deck.Slides[0].Shapes[0].Text)) > pptxPreviewMaxShapeTextChars+1 {
		t.Fatalf("shape text not bounded: %d", len([]rune(deck.Slides[0].Shapes[0].Text)))
	}
}

func TestRenderPPTXSlideTextIsUsefulFallback(t *testing.T) {
	slide := PPTXSlidePreview{Number: 3, Hidden: true, Shapes: []PPTXShapePreview{
		{Kind: PPTXShapeText, Name: "Title", Text: "Bridge Inspection"},
		{Kind: PPTXShapeTable, Name: "Table 1", Text: "Scheme\tAmount\nBridge\t125000"},
		{Kind: PPTXShapeImage, Name: "Picture 1", AltText: "Site photo", ImagePart: "ppt/media/image1.png"},
	}}
	got := renderPPTXSlideText(slide)
	for _, want := range []string{"Slide 3", "Hidden", "Bridge Inspection", "Scheme", "125000", "Site photo", "image1.png"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestPPTXSlidePixelSizeFitAndZoom(t *testing.T) {
	w, h := pptxSlidePixelSize(12192000, 6858000, 700, 350, 100, true)
	if w <= 0 || h <= 0 || w > 700 || h > 350 {
		t.Fatalf("bad fit size %dx%d", w, h)
	}
	zw, zh := pptxSlidePixelSize(12192000, 6858000, 700, 350, 200, false)
	if zw != w*2 || zh != h*2 {
		t.Fatalf("200%% zoom = %dx%d; want %dx%d", zw, zh, w*2, h*2)
	}
}

func TestPPTXShapePixelRectUsesSlideCoordinates(t *testing.T) {
	sh := PPTXShapePreview{X: 1219200, Y: 685800, CX: 6096000, CY: 1371600}
	x, y, w, h := pptxShapePixelRect(sh, 12192000, 6858000, 10, 20, 1000, 500, 0)
	if x != 110 || y != 70 || w != 500 || h != 100 {
		t.Fatalf("shape rect = %d,%d %dx%d", x, y, w, h)
	}
}
