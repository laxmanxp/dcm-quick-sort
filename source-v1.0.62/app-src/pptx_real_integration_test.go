package main

import (
	"os"
	"strings"
	"testing"
)

func TestRealGeneratedPPTXIntegration(t *testing.T) {
	p := os.Getenv("DCM_REAL_PPTX")
	if p == "" {
		t.Skip("DCM_REAL_PPTX not set")
	}
	deck, err := parsePPTXPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(deck.Slides) != 2 {
		t.Fatalf("slides=%d", len(deck.Slides))
	}
	text1 := renderPPTXSlideText(deck.Slides[0])
	if !strings.Contains(text1, "Bridge Inspection") || !strings.Contains(text1, "September 2026") {
		t.Fatalf("slide 1 missing text:\n%s", text1)
	}
	foundImage := false
	for _, sh := range deck.Slides[0].Shapes {
		if sh.Kind == PPTXShapeImage && sh.ImagePart != "" {
			foundImage = true
		}
	}
	if !foundImage {
		t.Fatal("real PPTX image relationship not discovered")
	}
	text2 := renderPPTXSlideText(deck.Slides[1])
	for _, want := range []string{"Payments", "Scheme", "125000", "Road", "86000"} {
		if !strings.Contains(text2, want) {
			t.Fatalf("slide 2 missing %q:\n%s", want, text2)
		}
	}
}
