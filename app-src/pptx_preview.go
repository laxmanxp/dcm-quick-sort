package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	pptxPresentationReadLimit    int64 = 2 * 1024 * 1024
	pptxSlideReadLimit           int64 = 16 * 1024 * 1024
	pptxImageReadLimit           int64 = 16 * 1024 * 1024
	pptxPreviewMaxSlides               = 200
	pptxPreviewMaxShapesPerSlide       = 256
	pptxPreviewMaxShapeTextChars       = 16 * 1024
)

type PPTXShapeKind string

const (
	PPTXShapeText  PPTXShapeKind = "text"
	PPTXShapeImage PPTXShapeKind = "image"
	PPTXShapeTable PPTXShapeKind = "table"
)

type PPTXShapePreview struct {
	Kind      PPTXShapeKind
	Name      string
	AltText   string
	Text      string
	X, Y      int64
	CX, CY    int64
	ImagePart string
}

type PPTXSlidePreview struct {
	Number    int
	Part      string
	Hidden    bool
	Title     string
	Shapes    []PPTXShapePreview
	Truncated bool
	Error     string
}

type PPTXPresentationPreview struct {
	Width, Height int64
	Slides        []PPTXSlidePreview
	Truncated     bool
	PackagePath   string
}

type pptxTransformXML struct {
	Off struct {
		X int64 `xml:"x,attr"`
		Y int64 `xml:"y,attr"`
	} `xml:"off"`
	Ext struct {
		CX int64 `xml:"cx,attr"`
		CY int64 `xml:"cy,attr"`
	} `xml:"ext"`
}

type pptxTextShapeXML struct {
	Nv struct {
		CNvPr struct {
			Name  string `xml:"name,attr"`
			Descr string `xml:"descr,attr"`
			Title string `xml:"title,attr"`
		} `xml:"cNvPr"`
	} `xml:"nvSpPr"`
	SpPr struct {
		Xfrm pptxTransformXML `xml:"xfrm"`
	} `xml:"spPr"`
	TxBody struct {
		Inner string `xml:",innerxml"`
	} `xml:"txBody"`
}

type pptxPictureXML struct {
	Nv struct {
		CNvPr struct {
			Name  string `xml:"name,attr"`
			Descr string `xml:"descr,attr"`
			Title string `xml:"title,attr"`
		} `xml:"cNvPr"`
	} `xml:"nvPicPr"`
	BlipFill struct {
		Blip struct {
			Embed string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships embed,attr"`
			Link  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships link,attr"`
		} `xml:"blip"`
	} `xml:"blipFill"`
	SpPr struct {
		Xfrm pptxTransformXML `xml:"xfrm"`
	} `xml:"spPr"`
}

type pptxGraphicFrameXML struct {
	Nv struct {
		CNvPr struct {
			Name  string `xml:"name,attr"`
			Descr string `xml:"descr,attr"`
			Title string `xml:"title,attr"`
		} `xml:"cNvPr"`
	} `xml:"nvGraphicFramePr"`
	Xfrm    pptxTransformXML `xml:"xfrm"`
	Graphic struct {
		GraphicData struct {
			Inner string `xml:",innerxml"`
		} `xml:"graphicData"`
	} `xml:"graphic"`
}

func pptxBoundText(s string) string {
	s = normalizePreviewNewlines(strings.TrimSpace(s))
	if utf8.RuneCountInString(s) <= pptxPreviewMaxShapeTextChars {
		return s
	}
	r := []rune(s)
	if len(r) > pptxPreviewMaxShapeTextChars {
		r = r[:pptxPreviewMaxShapeTextChars]
	}
	return string(r) + "…"
}

func parsePPTXTextXML(inner string) string {
	dec := xml.NewDecoder(strings.NewReader("<root>" + inner + "</root>"))
	var out strings.Builder
	paragraphOpen := false
	paragraphHasText := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch v := tok.(type) {
		case xml.StartElement:
			switch v.Name.Local {
			case "p":
				if paragraphOpen && paragraphHasText {
					out.WriteByte('\n')
				}
				paragraphOpen = true
				paragraphHasText = false
			case "t":
				var text string
				if dec.DecodeElement(&text, &v) == nil {
					out.WriteString(text)
					if text != "" {
						paragraphHasText = true
					}
				}
			case "br":
				out.WriteByte('\n')
				paragraphHasText = true
			case "tab":
				out.WriteByte('\t')
				paragraphHasText = true
			}
		case xml.EndElement:
			if v.Name.Local == "p" {
				if paragraphOpen && paragraphHasText {
					out.WriteByte('\n')
				}
				paragraphOpen = false
				paragraphHasText = false
			}
		}
	}
	return pptxBoundText(strings.TrimRight(out.String(), "\r\n"))
}

func parsePPTXTableXML(inner string) string {
	dec := xml.NewDecoder(strings.NewReader("<root>" + inner + "</root>"))
	var rows [][]string
	var currentRow []string
	inCell := false
	var cell strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch v := tok.(type) {
		case xml.StartElement:
			switch v.Name.Local {
			case "tr":
				currentRow = nil
			case "tc":
				inCell = true
				cell.Reset()
			case "t":
				if inCell {
					var text string
					if dec.DecodeElement(&text, &v) == nil {
						cell.WriteString(text)
					}
				}
			case "br":
				if inCell {
					cell.WriteString(" / ")
				}
			}
		case xml.EndElement:
			switch v.Name.Local {
			case "tc":
				currentRow = append(currentRow, strings.TrimSpace(cell.String()))
				inCell = false
			case "tr":
				if len(currentRow) > 0 {
					rows = append(rows, currentRow)
				}
			}
		}
	}
	var out strings.Builder
	for i, row := range rows {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(strings.Join(row, "\t"))
	}
	return pptxBoundText(out.String())
}

func pptxAltText(descr, title string) string {
	if s := strings.TrimSpace(descr); s != "" {
		return s
	}
	return strings.TrimSpace(title)
}

func parsePPTXSlideBytes(data []byte, slidePart string, rels []OpenXMLRelationship, number int) (PPTXSlidePreview, error) {
	slide := PPTXSlidePreview{Number: number, Part: slidePart}
	relMap := make(map[string]OpenXMLRelationship, len(rels))
	for _, rel := range rels {
		if rel.ID != "" {
			relMap[rel.ID] = rel
		}
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	rootSeen := false
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return slide, fmt.Errorf("parse PPTX slide %d: %w", number, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !rootSeen && start.Name.Local == "sld" {
			rootSeen = true
			for _, a := range start.Attr {
				if a.Name.Local == "show" {
					v := strings.ToLower(strings.TrimSpace(a.Value))
					slide.Hidden = v == "0" || v == "false" || v == "off" || v == "no"
				}
			}
			continue
		}
		if len(slide.Shapes) >= pptxPreviewMaxShapesPerSlide {
			slide.Truncated = true
			// Continue parsing only enough to validate the XML, but don't keep more shapes.
			continue
		}
		switch start.Name.Local {
		case "sp":
			var raw pptxTextShapeXML
			if err := dec.DecodeElement(&raw, &start); err != nil {
				return slide, fmt.Errorf("parse PPTX text shape on slide %d: %w", number, err)
			}
			text := parsePPTXTextXML(raw.TxBody.Inner)
			if text == "" && strings.TrimSpace(raw.Nv.CNvPr.Name) == "" {
				continue
			}
			sh := PPTXShapePreview{
				Kind:    PPTXShapeText,
				Name:    strings.TrimSpace(raw.Nv.CNvPr.Name),
				AltText: pptxAltText(raw.Nv.CNvPr.Descr, raw.Nv.CNvPr.Title),
				Text:    text,
				X:       raw.SpPr.Xfrm.Off.X, Y: raw.SpPr.Xfrm.Off.Y,
				CX: raw.SpPr.Xfrm.Ext.CX, CY: raw.SpPr.Xfrm.Ext.CY,
			}
			slide.Shapes = append(slide.Shapes, sh)
			if slide.Title == "" && text != "" {
				first := text
				if i := strings.IndexByte(first, '\n'); i >= 0 {
					first = first[:i]
				}
				first = strings.TrimSpace(first)
				if len([]rune(first)) > 80 {
					r := []rune(first)
					first = string(r[:80]) + "…"
				}
				slide.Title = first
			}
		case "pic":
			var raw pptxPictureXML
			if err := dec.DecodeElement(&raw, &start); err != nil {
				return slide, fmt.Errorf("parse PPTX picture on slide %d: %w", number, err)
			}
			relID := strings.TrimSpace(raw.BlipFill.Blip.Embed)
			if relID == "" {
				relID = strings.TrimSpace(raw.BlipFill.Blip.Link)
			}
			imagePart := ""
			if rel, ok := relMap[relID]; ok && !rel.External {
				imagePart = rel.ResolvedTarget
			}
			slide.Shapes = append(slide.Shapes, PPTXShapePreview{
				Kind:    PPTXShapeImage,
				Name:    strings.TrimSpace(raw.Nv.CNvPr.Name),
				AltText: pptxAltText(raw.Nv.CNvPr.Descr, raw.Nv.CNvPr.Title),
				X:       raw.SpPr.Xfrm.Off.X, Y: raw.SpPr.Xfrm.Off.Y,
				CX: raw.SpPr.Xfrm.Ext.CX, CY: raw.SpPr.Xfrm.Ext.CY,
				ImagePart: imagePart,
			})
		case "graphicFrame":
			var raw pptxGraphicFrameXML
			if err := dec.DecodeElement(&raw, &start); err != nil {
				return slide, fmt.Errorf("parse PPTX graphic frame on slide %d: %w", number, err)
			}
			text := parsePPTXTableXML(raw.Graphic.GraphicData.Inner)
			if text == "" {
				continue
			}
			slide.Shapes = append(slide.Shapes, PPTXShapePreview{
				Kind:    PPTXShapeTable,
				Name:    strings.TrimSpace(raw.Nv.CNvPr.Name),
				AltText: pptxAltText(raw.Nv.CNvPr.Descr, raw.Nv.CNvPr.Title),
				Text:    text,
				X:       raw.Xfrm.Off.X, Y: raw.Xfrm.Off.Y,
				CX: raw.Xfrm.Ext.CX, CY: raw.Xfrm.Ext.CY,
			})
		}
	}
	if !rootSeen {
		return slide, fmt.Errorf("PPTX slide %d has no slide root element", number)
	}
	return slide, nil
}

func parsePPTXPresentationXML(data []byte) (width, height int64, relIDs []string, err error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, e := dec.Token()
		if e != nil {
			if e == io.EOF {
				break
			}
			return 0, 0, nil, fmt.Errorf("parse PPTX presentation: %w", e)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "sldId":
			relID := ""
			for _, a := range start.Attr {
				if a.Name.Local == "id" && strings.Contains(strings.ToLower(a.Name.Space), "relationships") {
					relID = strings.TrimSpace(a.Value)
					break
				}
			}
			if relID == "" {
				// Some parsers/writers expose namespace-less attributes after normalization.
				for _, a := range start.Attr {
					if a.Name.Local == "id" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.Value)), "rid") {
						relID = strings.TrimSpace(a.Value)
						break
					}
				}
			}
			if relID != "" {
				relIDs = append(relIDs, relID)
			}
		case "sldSz":
			for _, a := range start.Attr {
				switch a.Name.Local {
				case "cx":
					fmt.Sscan(strings.TrimSpace(a.Value), &width)
				case "cy":
					fmt.Sscan(strings.TrimSpace(a.Value), &height)
				}
			}
		}
	}
	return width, height, relIDs, nil
}

func parsePPTXPreview(packagePath string) (PPTXPresentationPreview, error) {
	info, err := inspectOpenXMLPackage(packagePath)
	if err != nil {
		return PPTXPresentationPreview{}, err
	}
	if info.Kind != OpenXMLPowerPoint {
		return PPTXPresentationPreview{}, fmt.Errorf("not a PPTX presentation")
	}
	data, err := readOpenXMLPart(packagePath, info.MainPart, pptxPresentationReadLimit)
	if err != nil {
		return PPTXPresentationPreview{}, err
	}
	width, height, relIDs, err := parsePPTXPresentationXML(data)
	if err != nil {
		return PPTXPresentationPreview{}, err
	}
	if width <= 0 {
		width = 12192000
	}
	if height <= 0 {
		height = 6858000
	}
	rels, err := readOpenXMLRelationships(packagePath, info.MainPart)
	if err != nil {
		return PPTXPresentationPreview{}, fmt.Errorf("PPTX presentation relationships: %w", err)
	}
	relMap := make(map[string]OpenXMLRelationship, len(rels))
	for _, rel := range rels {
		if rel.ID != "" {
			relMap[rel.ID] = rel
		}
	}
	if len(relIDs) == 0 {
		return PPTXPresentationPreview{}, fmt.Errorf("PPTX presentation has no slides")
	}

	deck := PPTXPresentationPreview{Width: width, Height: height, PackagePath: packagePath}
	if len(relIDs) > pptxPreviewMaxSlides {
		relIDs = relIDs[:pptxPreviewMaxSlides]
		deck.Truncated = true
	}
	for i, relID := range relIDs {
		rel, ok := relMap[relID]
		if !ok || rel.External || rel.ResolvedTarget == "" || !strings.HasSuffix(strings.ToLower(rel.Type), "/slide") {
			return PPTXPresentationPreview{}, fmt.Errorf("PPTX slide %d relationship %q is missing or invalid", i+1, relID)
		}
		slide := PPTXSlidePreview{Number: i + 1, Part: rel.ResolvedTarget}
		slideBytes, readErr := readOpenXMLPart(packagePath, rel.ResolvedTarget, pptxSlideReadLimit)
		if readErr != nil {
			slide.Error = readErr.Error()
			deck.Slides = append(deck.Slides, slide)
			continue
		}
		slideRels, relErr := readOpenXMLRelationships(packagePath, rel.ResolvedTarget)
		if relErr != nil {
			// Relationship parts are optional when a slide has no related resources.
			slideRels = nil
		}
		parsed, parseErr := parsePPTXSlideBytes(slideBytes, rel.ResolvedTarget, slideRels, i+1)
		if parseErr != nil {
			slide.Error = parseErr.Error()
			deck.Slides = append(deck.Slides, slide)
			continue
		}
		deck.Slides = append(deck.Slides, parsed)
	}
	return deck, nil
}

func pptxPreviewForPath(packagePath string) (PPTXPresentationPreview, bool) {
	if !strings.EqualFold(filepath.Ext(packagePath), ".pptx") {
		return PPTXPresentationPreview{}, false
	}
	deck, err := parsePPTXPreview(packagePath)
	return deck, err == nil && len(deck.Slides) > 0
}

func renderPPTXSlideText(slide PPTXSlidePreview) string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("Slide %d", slide.Number))
	if slide.Hidden {
		out.WriteString(" · Hidden")
	}
	out.WriteByte('\n')
	if slide.Title != "" {
		out.WriteString(slide.Title)
		out.WriteByte('\n')
	}
	if slide.Error != "" {
		out.WriteString("\nCould not render this slide: ")
		out.WriteString(slide.Error)
		return normalizePreviewNewlines(out.String())
	}
	for _, shape := range slide.Shapes {
		out.WriteByte('\n')
		switch shape.Kind {
		case PPTXShapeText:
			if shape.Text != "" {
				out.WriteString(shape.Text)
				out.WriteByte('\n')
			}
		case PPTXShapeTable:
			if shape.Name != "" {
				out.WriteString("[Table: ")
				out.WriteString(shape.Name)
				out.WriteString("]\n")
			}
			out.WriteString(shape.Text)
			out.WriteByte('\n')
		case PPTXShapeImage:
			out.WriteString("[Image")
			if shape.AltText != "" {
				out.WriteString(": ")
				out.WriteString(shape.AltText)
			} else if shape.Name != "" {
				out.WriteString(": ")
				out.WriteString(shape.Name)
			}
			if shape.ImagePart != "" {
				out.WriteString(" — ")
				out.WriteString(filepath.Base(filepath.FromSlash(shape.ImagePart)))
			}
			out.WriteString("]\n")
		}
	}
	if slide.Truncated {
		out.WriteString("\n[Preview truncated: additional slide shapes were omitted.]\n")
	}
	return normalizePreviewNewlines(strings.TrimSpace(out.String()))
}

func pptxSlidePixelSize(slideW, slideH int64, availW, availH int32, zoom int, fit bool) (int32, int32) {
	return pptxSlidePixelSizeMode(slideW, slideH, availW, availH, zoom, fit, false)
}

func pptxSlidePixelSizeMode(slideW, slideH int64, availW, availH int32, zoom int, fitPage, fitWidth bool) (int32, int32) {
	if slideW <= 0 || slideH <= 0 || availW <= 0 || availH <= 0 {
		return 0, 0
	}
	scaleW := float64(availW) / float64(slideW)
	scaleH := float64(availH) / float64(slideH)
	scale := scaleW
	if !fitWidth && scaleH < scale {
		scale = scaleH
	}
	if !fitPage && !fitWidth {
		z := clampPreviewZoom(zoom)
		base := scaleW
		if scaleH < base {
			base = scaleH
		}
		scale = base
		scale *= float64(z) / 100.0
	}
	w := int32(float64(slideW)*scale + 0.5)
	h := int32(float64(slideH)*scale + 0.5)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func pptxShapePixelRect(shape PPTXShapePreview, slideW, slideH int64, slideX, slideY, slidePixelW, slidePixelH int32, fallbackIndex int) (int32, int32, int32, int32) {
	if slideW > 0 && slideH > 0 && shape.CX > 0 && shape.CY > 0 {
		x := slideX + int32(float64(shape.X)*float64(slidePixelW)/float64(slideW)+0.5)
		y := slideY + int32(float64(shape.Y)*float64(slidePixelH)/float64(slideH)+0.5)
		w := int32(float64(shape.CX)*float64(slidePixelW)/float64(slideW) + 0.5)
		h := int32(float64(shape.CY)*float64(slidePixelH)/float64(slideH) + 0.5)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		return x, y, w, h
	}
	margin := int32(18)
	h := int32(48)
	y := slideY + margin + int32(fallbackIndex)*56
	w := slidePixelW - margin*2
	if w < 1 {
		w = 1
	}
	return slideX + margin, y, w, h
}
