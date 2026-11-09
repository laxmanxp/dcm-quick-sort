package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	docxMainPartReadLimit  int64 = 8 * 1024 * 1024
	docxStylesReadLimit    int64 = 2 * 1024 * 1024
	docxPreviewOutputLimit       = 240 * 1024
)

type DOCXPreviewStats struct {
	Paragraphs int
	Headings   int
	Tables     int
	Images     int
	Truncated  bool
}

type docxParagraph struct {
	Text         string
	StyleID      string
	StyleName    string
	HeadingLevel int
	Images       int
}

type docxTable struct {
	Rows [][]string
}

type docxBlock struct {
	Paragraph *docxParagraph
	Table     *docxTable
}

func xmlAttrLocal(start xml.StartElement, local string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == local {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

func parseDOCXStyles(data []byte) (map[string]string, error) {
	styles := map[string]string{}
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return styles, nil
			}
			return nil, fmt.Errorf("parse DOCX styles: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "style" {
			continue
		}
		styleType := xmlAttrLocal(start, "type")
		styleID := xmlAttrLocal(start, "styleId")
		var raw struct {
			Name struct {
				Val string `xml:"val,attr"`
			} `xml:"name"`
		}
		if err := dec.DecodeElement(&raw, &start); err != nil {
			return nil, fmt.Errorf("parse DOCX style %q: %w", styleID, err)
		}
		if styleType == "paragraph" && styleID != "" {
			name := strings.TrimSpace(raw.Name.Val)
			if name == "" {
				name = styleID
			}
			styles[styleID] = name
		}
	}
}

func docxHeadingLevel(styleID, styleName string) int {
	normalized := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		r := strings.NewReplacer(" ", "", "-", "", "_", "")
		return r.Replace(s)
	}
	for _, candidate := range []string{styleName, styleID} {
		c := normalized(candidate)
		if c == "title" {
			return 1
		}
		if strings.HasPrefix(c, "heading") {
			n, err := strconv.Atoi(strings.TrimPrefix(c, "heading"))
			if err == nil && n >= 1 && n <= 9 {
				return n
			}
		}
	}
	return 0
}

func docxRelationshipMap(rels []OpenXMLRelationship) map[string]OpenXMLRelationship {
	out := make(map[string]OpenXMLRelationship, len(rels))
	for _, rel := range rels {
		if rel.ID != "" {
			out[rel.ID] = rel
		}
	}
	return out
}

func docxStylesForPackage(packagePath, mainPart string, rels []OpenXMLRelationship) map[string]string {
	candidates := []string{}
	for _, rel := range rels {
		if rel.External {
			continue
		}
		if strings.HasSuffix(strings.ToLower(rel.Type), "/styles") && rel.ResolvedTarget != "" {
			candidates = append(candidates, rel.ResolvedTarget)
		}
	}
	if len(candidates) == 0 {
		candidates = append(candidates, "word/styles.xml")
	}
	for _, part := range candidates {
		data, err := readOpenXMLPart(packagePath, part, docxStylesReadLimit)
		if err != nil {
			continue
		}
		styles, err := parseDOCXStyles(data)
		if err == nil {
			return styles
		}
	}
	return map[string]string{}
}

func imagePlaceholder(desc, relationshipID string, rels map[string]OpenXMLRelationship) string {
	desc = strings.TrimSpace(desc)
	file := ""
	if rel, ok := rels[relationshipID]; ok && !rel.External && rel.ResolvedTarget != "" {
		file = filepath.Base(filepath.FromSlash(rel.ResolvedTarget))
	}
	switch {
	case desc != "" && file != "":
		return "[Image: " + desc + " — " + file + "]"
	case desc != "":
		return "[Image: " + desc + "]"
	case file != "":
		return "[Image: " + file + "]"
	default:
		return "[Image]"
	}
}

func parseDOCXParagraph(dec *xml.Decoder, start xml.StartElement, styles map[string]string, rels map[string]OpenXMLRelationship) (docxParagraph, error) {
	var b strings.Builder
	p := docxParagraph{}
	depth := 1
	pendingImageDescription := ""
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return p, fmt.Errorf("parse DOCX paragraph: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			switch t.Name.Local {
			case "del":
				// Deleted revision text should not appear in a normal preview.
				if err := dec.Skip(); err != nil {
					return p, fmt.Errorf("skip deleted DOCX content: %w", err)
				}
				depth--
			case "pStyle":
				p.StyleID = xmlAttrLocal(t, "val")
			case "t":
				var value string
				if err := dec.DecodeElement(&value, &t); err != nil {
					return p, fmt.Errorf("parse DOCX text run: %w", err)
				}
				depth--
				b.WriteString(value)
			case "tab":
				b.WriteByte('\t')
			case "br", "cr":
				b.WriteByte('\n')
			case "noBreakHyphen", "softHyphen":
				b.WriteByte('-')
			case "docPr":
				pendingImageDescription = xmlAttrLocal(t, "descr")
				if pendingImageDescription == "" {
					pendingImageDescription = xmlAttrLocal(t, "title")
				}
				if pendingImageDescription == "" {
					name := xmlAttrLocal(t, "name")
					if name != "" && !strings.HasPrefix(strings.ToLower(name), "picture ") {
						pendingImageDescription = name
					}
				}
			case "blip":
				rid := xmlAttrLocal(t, "embed")
				if rid == "" {
					rid = xmlAttrLocal(t, "link")
				}
				if rid != "" {
					if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
						b.WriteByte('\n')
					}
					b.WriteString(imagePlaceholder(pendingImageDescription, rid, rels))
					b.WriteByte('\n')
					p.Images++
					pendingImageDescription = ""
				}
			case "imagedata":
				rid := xmlAttrLocal(t, "id")
				desc := xmlAttrLocal(t, "title")
				if rid != "" {
					if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
						b.WriteByte('\n')
					}
					b.WriteString(imagePlaceholder(desc, rid, rels))
					b.WriteByte('\n')
					p.Images++
				}
			}
		case xml.EndElement:
			depth--
		}
	}
	p.StyleName = styles[p.StyleID]
	p.HeadingLevel = docxHeadingLevel(p.StyleID, p.StyleName)
	p.Text = strings.TrimSpace(strings.ReplaceAll(b.String(), "\r\n", "\n"))
	return p, nil
}

func tableCellText(p docxParagraph) string {
	text := strings.TrimSpace(p.Text)
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "\n", " / ")
	text = strings.ReplaceAll(text, "|", "¦")
	return text
}

func parseDOCXTable(dec *xml.Decoder, start xml.StartElement, styles map[string]string, rels map[string]OpenXMLRelationship) (docxTable, DOCXPreviewStats, error) {
	table := docxTable{}
	stats := DOCXPreviewStats{Tables: 1}
	depth := 1
	inCell := false
	inRow := false
	currentRow := []string(nil)
	currentCell := []string(nil)

	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return table, stats, fmt.Errorf("parse DOCX table: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			switch t.Name.Local {
			case "tr":
				if !inRow {
					inRow = true
					currentRow = []string{}
				}
			case "tc":
				if inRow && !inCell {
					inCell = true
					currentCell = []string{}
				}
			case "p":
				if inCell {
					p, err := parseDOCXParagraph(dec, t, styles, rels)
					if err != nil {
						return table, stats, err
					}
					depth-- // parseDOCXParagraph consumed </p>
					stats.Paragraphs++
					stats.Images += p.Images
					if p.HeadingLevel > 0 {
						stats.Headings++
					}
					if text := tableCellText(p); text != "" {
						currentCell = append(currentCell, text)
					}
				}
			case "tbl":
				if inCell {
					nested, nestedStats, err := parseDOCXTable(dec, t, styles, rels)
					if err != nil {
						return table, stats, err
					}
					depth-- // nested parser consumed </tbl>
					stats.Paragraphs += nestedStats.Paragraphs
					stats.Headings += nestedStats.Headings
					stats.Tables += nestedStats.Tables
					stats.Images += nestedStats.Images
					if text := strings.TrimSpace(renderDOCXTableRows(nested.Rows)); text != "" {
						currentCell = append(currentCell, text)
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "tc":
				if inCell {
					currentRow = append(currentRow, strings.Join(currentCell, " / "))
					currentCell = nil
					inCell = false
				}
			case "tr":
				if inRow {
					table.Rows = append(table.Rows, currentRow)
					currentRow = nil
					inRow = false
				}
			}
			depth--
		}
	}
	return table, stats, nil
}

func parseDOCXDocument(data []byte, styles map[string]string, rels map[string]OpenXMLRelationship) ([]docxBlock, DOCXPreviewStats, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	blocks := []docxBlock{}
	stats := DOCXPreviewStats{}
	inBody := false
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return blocks, stats, nil
			}
			return nil, stats, fmt.Errorf("parse DOCX document: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "body":
				inBody = true
			case "p":
				if inBody {
					p, err := parseDOCXParagraph(dec, t, styles, rels)
					if err != nil {
						return nil, stats, err
					}
					stats.Paragraphs++
					stats.Images += p.Images
					if p.HeadingLevel > 0 {
						stats.Headings++
					}
					blocks = append(blocks, docxBlock{Paragraph: &p})
				}
			case "tbl":
				if inBody {
					table, tableStats, err := parseDOCXTable(dec, t, styles, rels)
					if err != nil {
						return nil, stats, err
					}
					stats.Paragraphs += tableStats.Paragraphs
					stats.Headings += tableStats.Headings
					stats.Tables += tableStats.Tables
					stats.Images += tableStats.Images
					blocks = append(blocks, docxBlock{Table: &table})
				}
			}
		case xml.EndElement:
			if t.Name.Local == "body" {
				inBody = false
			}
		}
	}
}

func renderDOCXHeading(text string, level int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	runeCount := utf8.RuneCountInString(text)
	if runeCount < 3 {
		runeCount = 3
	}
	if runeCount > 80 {
		runeCount = 80
	}
	switch level {
	case 1:
		return text + "\n" + strings.Repeat("=", runeCount)
	case 2:
		return text + "\n" + strings.Repeat("-", runeCount)
	default:
		if level > 6 {
			level = 6
		}
		return strings.Repeat("#", level) + " " + text
	}
}

func renderDOCXTableRows(rows [][]string) string {
	var b strings.Builder
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		b.WriteString("| ")
		for i, cell := range row {
			if i > 0 {
				b.WriteString(" | ")
			}
			b.WriteString(strings.TrimSpace(cell))
		}
		b.WriteString(" |\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncateUTF8At(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(s) <= limit {
		return s
	}
	n := limit
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func renderDOCXPreview(packagePath string) (string, DOCXPreviewStats, error) {
	info, err := inspectOpenXMLPackage(packagePath)
	if err != nil {
		return "", DOCXPreviewStats{}, err
	}
	if info.Kind != OpenXMLWord {
		return "", DOCXPreviewStats{}, fmt.Errorf("%s is not a DOCX package", filepath.Base(packagePath))
	}

	rels, _ := readOpenXMLRelationships(packagePath, info.MainPart)
	relMap := docxRelationshipMap(rels)
	styles := docxStylesForPackage(packagePath, info.MainPart, rels)
	doc, err := readOpenXMLPart(packagePath, info.MainPart, docxMainPartReadLimit)
	if err != nil {
		return "", DOCXPreviewStats{}, err
	}
	blocks, stats, err := parseDOCXDocument(doc, styles, relMap)
	if err != nil {
		return "", stats, err
	}

	var b strings.Builder
	b.WriteString("DCM DOCX Preview\n")
	if info.Core.Title != "" {
		b.WriteString("Title: " + info.Core.Title + "\n")
	}
	if info.Core.Creator != "" {
		b.WriteString("Author: " + info.Core.Creator + "\n")
	}
	if info.Core.Modified != "" {
		b.WriteString("Modified: " + info.Core.Modified + "\n")
	}
	b.WriteString(strings.Repeat("-", 64) + "\n\n")

	tableNumber := 0
	blankPending := false
	for _, block := range blocks {
		if block.Paragraph != nil {
			p := *block.Paragraph
			if p.Text == "" {
				blankPending = true
				continue
			}
			if blankPending && b.Len() > 0 {
				b.WriteByte('\n')
			}
			blankPending = false
			if p.HeadingLevel > 0 {
				b.WriteString(renderDOCXHeading(p.Text, p.HeadingLevel))
			} else {
				b.WriteString(p.Text)
			}
			b.WriteString("\n\n")
		}
		if block.Table != nil {
			tableNumber++
			if blankPending && b.Len() > 0 {
				b.WriteByte('\n')
			}
			blankPending = false
			b.WriteString(fmt.Sprintf("TABLE %d\n", tableNumber))
			b.WriteString(renderDOCXTableRows(block.Table.Rows))
			b.WriteString("\n\n")
		}
	}

	b.WriteString(strings.Repeat("-", 64) + "\n")
	b.WriteString(fmt.Sprintf("%d paragraphs · %d headings · %d tables · %d images\n", stats.Paragraphs, stats.Headings, stats.Tables, stats.Images))
	b.WriteString("Rendered locally from DOCX Open XML. Windows Word preview is still preferred when available.")

	text := normalizePreviewNewlines(b.String())
	if len(text) > docxPreviewOutputLimit {
		const notice = "\r\n\r\n[Preview truncated for performance. Open the document to see the remaining content.]"
		text = truncateUTF8At(text, docxPreviewOutputLimit-len(notice)) + notice
		stats.Truncated = true
	}
	return text, stats, nil
}

func docxPreviewText(packagePath string) (string, DOCXPreviewStats, bool) {
	kind, ok := openXMLKindForPath(packagePath)
	if !ok || kind != OpenXMLWord {
		return "", DOCXPreviewStats{}, false
	}
	text, stats, err := renderDOCXPreview(packagePath)
	if err != nil {
		return "", stats, false
	}
	return text, stats, true
}

func textPreviewFontPixelHeight(zoom int) int {
	zoom = clampPreviewZoom(zoom)
	px := (16*zoom + 50) / 100
	if px < 8 {
		px = 8
	}
	return px
}
