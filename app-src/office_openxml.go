package main

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type OpenXMLKind string

const (
	OpenXMLNone       OpenXMLKind = ""
	OpenXMLWord       OpenXMLKind = "word"
	OpenXMLExcel      OpenXMLKind = "excel"
	OpenXMLPowerPoint OpenXMLKind = "powerpoint"
)

const (
	contentTypeWordDocument           = "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"
	contentTypeExcelWorkbook          = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"
	contentTypePowerPointPresentation = "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"

	openXMLMetadataReadLimit int64 = 1024 * 1024
	openXMLMaxEntries              = 10000
)

type OpenXMLCoreProperties struct {
	Title          string
	Subject        string
	Creator        string
	LastModifiedBy string
	Modified       string
}

type OpenXMLExtendedProperties struct {
	Application string
	Pages       int
	Words       int
	Slides      int
}

type OpenXMLRelationship struct {
	ID             string
	Type           string
	Target         string
	External       bool
	ResolvedTarget string
}

func openXMLRelationshipPartName(part string) string {
	part = strings.TrimPrefix(strings.ReplaceAll(part, "\\", "/"), "/")
	if part == "" {
		return "_rels/.rels"
	}
	dir := path.Dir(part)
	base := path.Base(part)
	if dir == "." {
		return "_rels/" + base + ".rels"
	}
	return dir + "/_rels/" + base + ".rels"
}

func resolveOpenXMLRelationshipTarget(basePart, target string) (string, error) {
	target = strings.TrimSpace(strings.ReplaceAll(target, "\\", "/"))
	if target == "" {
		return "", errors.New("Open XML relationship target is empty")
	}
	var resolved string
	if strings.HasPrefix(target, "/") {
		resolved = path.Clean(strings.TrimPrefix(target, "/"))
	} else {
		basePart = strings.TrimPrefix(strings.ReplaceAll(basePart, "\\", "/"), "/")
		dir := path.Dir(basePart)
		if basePart == "" || dir == "." {
			dir = ""
		}
		resolved = path.Clean(path.Join(dir, target))
	}
	if resolved == "." || resolved == "" || resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("Open XML relationship target %q escapes package root", target)
	}
	return strings.TrimPrefix(resolved, "/"), nil
}

func readOpenXMLPart(packagePath, part string, limit int64) ([]byte, error) {
	zr, err := zip.OpenReader(packagePath)
	if err != nil {
		return nil, fmt.Errorf("invalid Open XML package: %w", err)
	}
	defer zr.Close()
	if len(zr.File) > openXMLMaxEntries {
		return nil, fmt.Errorf("Open XML package has too many parts (%d > %d)", len(zr.File), openXMLMaxEntries)
	}
	return readZipEntryLimited(zr.File, part, limit)
}

func readOpenXMLRelationships(packagePath, basePart string) ([]OpenXMLRelationship, error) {
	relsPart := openXMLRelationshipPartName(basePart)
	data, err := readOpenXMLPart(packagePath, relsPart, openXMLMetadataReadLimit)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Relationships []struct {
			ID         string `xml:"Id,attr"`
			Type       string `xml:"Type,attr"`
			Target     string `xml:"Target,attr"`
			TargetMode string `xml:"TargetMode,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid Open XML relationship part %q: %w", relsPart, err)
	}
	out := make([]OpenXMLRelationship, 0, len(doc.Relationships))
	for _, raw := range doc.Relationships {
		rel := OpenXMLRelationship{
			ID:       strings.TrimSpace(raw.ID),
			Type:     strings.TrimSpace(raw.Type),
			Target:   strings.TrimSpace(raw.Target),
			External: strings.EqualFold(strings.TrimSpace(raw.TargetMode), "External"),
		}
		if !rel.External {
			resolved, err := resolveOpenXMLRelationshipTarget(basePart, rel.Target)
			if err != nil {
				return nil, err
			}
			rel.ResolvedTarget = resolved
		}
		out = append(out, rel)
	}
	return out, nil
}

type OpenXMLPackageInfo struct {
	Kind       OpenXMLKind
	MainPart   string
	EntryCount int
	Core       OpenXMLCoreProperties
	Extended   OpenXMLExtendedProperties
}

func openXMLKindForPath(path string) (OpenXMLKind, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx":
		return OpenXMLWord, true
	case ".xlsx":
		return OpenXMLExcel, true
	case ".pptx":
		return OpenXMLPowerPoint, true
	default:
		return OpenXMLNone, false
	}
}

func openXMLKindLabel(kind OpenXMLKind) string {
	switch kind {
	case OpenXMLWord:
		return "Word document (DOCX)"
	case OpenXMLExcel:
		return "Excel workbook (XLSX)"
	case OpenXMLPowerPoint:
		return "PowerPoint presentation (PPTX)"
	default:
		return "Office Open XML document"
	}
}

func openXMLContentTypeKind(contentType string) OpenXMLKind {
	switch strings.TrimSpace(contentType) {
	case contentTypeWordDocument:
		return OpenXMLWord
	case contentTypeExcelWorkbook:
		return OpenXMLExcel
	case contentTypePowerPointPresentation:
		return OpenXMLPowerPoint
	default:
		return OpenXMLNone
	}
}

type openXMLTypesDocument struct {
	Defaults []struct {
		Extension   string `xml:"Extension,attr"`
		ContentType string `xml:"ContentType,attr"`
	} `xml:"Default"`
	Overrides []struct {
		PartName    string `xml:"PartName,attr"`
		ContentType string `xml:"ContentType,attr"`
	} `xml:"Override"`
}

func inspectOpenXMLPackage(path string) (OpenXMLPackageInfo, error) {
	expected, ok := openXMLKindForPath(path)
	if !ok {
		return OpenXMLPackageInfo{}, fmt.Errorf("%s is not a supported Office Open XML preview type", filepath.Ext(path))
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return OpenXMLPackageInfo{}, fmt.Errorf("invalid Open XML package: %w", err)
	}
	defer zr.Close()

	if len(zr.File) == 0 {
		return OpenXMLPackageInfo{}, errors.New("invalid Open XML package: archive is empty")
	}
	if len(zr.File) > openXMLMaxEntries {
		return OpenXMLPackageInfo{}, fmt.Errorf("Open XML package has too many parts (%d > %d)", len(zr.File), openXMLMaxEntries)
	}

	typeBytes, err := readZipEntryLimited(zr.File, "[Content_Types].xml", openXMLMetadataReadLimit)
	if err != nil {
		return OpenXMLPackageInfo{}, fmt.Errorf("invalid Open XML package: %w", err)
	}
	var types openXMLTypesDocument
	if err := xml.Unmarshal(typeBytes, &types); err != nil {
		return OpenXMLPackageInfo{}, fmt.Errorf("invalid [Content_Types].xml: %w", err)
	}

	detected := OpenXMLNone
	mainPart := ""
	for _, ov := range types.Overrides {
		kind := openXMLContentTypeKind(ov.ContentType)
		if kind == OpenXMLNone {
			continue
		}
		if detected != OpenXMLNone && detected != kind {
			return OpenXMLPackageInfo{}, errors.New("Open XML package declares more than one supported main document type")
		}
		detected = kind
		mainPart = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(ov.PartName)), "/")
	}
	if detected == OpenXMLNone || mainPart == "" {
		// Some producers legally identify the main Office part through the package
		// root officeDocument relationship while assigning its content type via a
		// Default entry rather than an Override. Accept that form too.
		rootRelsData, relErr := readZipEntryLimited(zr.File, "_rels/.rels", openXMLMetadataReadLimit)
		if relErr == nil {
			var rootRels struct {
				Relationships []struct {
					Type       string `xml:"Type,attr"`
					Target     string `xml:"Target,attr"`
					TargetMode string `xml:"TargetMode,attr"`
				} `xml:"Relationship"`
			}
			if xml.Unmarshal(rootRelsData, &rootRels) == nil {
				for _, rel := range rootRels.Relationships {
					if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(rel.Type)), "/officedocument") || strings.EqualFold(strings.TrimSpace(rel.TargetMode), "External") {
						continue
					}
					resolved, resolveErr := resolveOpenXMLRelationshipTarget("", rel.Target)
					if resolveErr != nil {
						continue
					}
					ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(resolved)), ".")
					for _, def := range types.Defaults {
						if !strings.EqualFold(strings.TrimSpace(def.Extension), ext) {
							continue
						}
						kind := openXMLContentTypeKind(def.ContentType)
						if kind != OpenXMLNone {
							detected = kind
							mainPart = resolved
							break
						}
					}
					if detected != OpenXMLNone && mainPart != "" {
						break
					}
				}
			}
		}
	}
	if detected == OpenXMLNone || mainPart == "" {
		return OpenXMLPackageInfo{}, errors.New("Open XML package does not declare a supported DOCX, XLSX, or PPTX main part")
	}
	if detected != expected {
		return OpenXMLPackageInfo{}, fmt.Errorf("file extension %s does not match package type %s", strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")), openXMLKindLabel(detected))
	}
	if findZipEntry(zr.File, mainPart) == nil {
		return OpenXMLPackageInfo{}, fmt.Errorf("Open XML main part %q is missing", mainPart)
	}

	info := OpenXMLPackageInfo{
		Kind:       detected,
		MainPart:   mainPart,
		EntryCount: len(zr.File),
	}
	if data, err := readZipEntryLimited(zr.File, "docProps/core.xml", openXMLMetadataReadLimit); err == nil {
		info.Core = parseOpenXMLCoreProperties(data)
	}
	if data, err := readZipEntryLimited(zr.File, "docProps/app.xml", openXMLMetadataReadLimit); err == nil {
		info.Extended = parseOpenXMLExtendedProperties(data)
	}
	return info, nil
}

func findZipEntry(files []*zip.File, name string) *zip.File {
	wanted := strings.TrimPrefix(filepath.ToSlash(name), "/")
	for _, f := range files {
		if strings.TrimPrefix(filepath.ToSlash(f.Name), "/") == wanted {
			return f
		}
	}
	return nil
}

func readZipEntryLimited(files []*zip.File, name string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("read limit must be positive")
	}
	f := findZipEntry(files, name)
	if f == nil {
		return nil, fmt.Errorf("package part %q is missing", name)
	}
	if f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("package part %q exceeds read limit of %d bytes", name, limit)
	}
	r, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open package part %q: %w", name, err)
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read package part %q: %w", name, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("package part %q exceeds read limit of %d bytes", name, limit)
	}
	return data, nil
}

func parseOpenXMLCoreProperties(data []byte) OpenXMLCoreProperties {
	values := map[string]string{}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "title", "subject", "creator", "lastModifiedBy", "modified":
			var v string
			if err := dec.DecodeElement(&v, &start); err == nil {
				values[start.Name.Local] = strings.TrimSpace(v)
			}
		}
	}
	return OpenXMLCoreProperties{
		Title:          values["title"],
		Subject:        values["subject"],
		Creator:        values["creator"],
		LastModifiedBy: values["lastModifiedBy"],
		Modified:       values["modified"],
	}
}

func parseOpenXMLExtendedProperties(data []byte) OpenXMLExtendedProperties {
	values := map[string]string{}
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "Application", "Pages", "Words", "Slides":
			var v string
			if err := dec.DecodeElement(&v, &start); err == nil {
				values[start.Name.Local] = strings.TrimSpace(v)
			}
		}
	}
	atoi := func(s string) int {
		n, _ := strconv.Atoi(strings.TrimSpace(s))
		return n
	}
	return OpenXMLExtendedProperties{
		Application: values["Application"],
		Pages:       atoi(values["Pages"]),
		Words:       atoi(values["Words"]),
		Slides:      atoi(values["Slides"]),
	}
}

func officeFoundationPreviewText(path string) (string, bool) {
	if _, ok := openXMLKindForPath(path); !ok {
		return "", false
	}
	info, err := inspectOpenXMLPackage(path)
	if err != nil {
		return "", false
	}

	lines := []string{
		"DCM Office Preview",
		"",
		openXMLKindLabel(info.Kind),
		"Open XML package validated.",
	}
	if info.Core.Title != "" {
		lines = append(lines, "", "Title: "+info.Core.Title)
	}
	if info.Core.Subject != "" {
		lines = append(lines, "Subject: "+info.Core.Subject)
	}
	if info.Core.Creator != "" {
		lines = append(lines, "Author: "+info.Core.Creator)
	}
	if info.Core.LastModifiedBy != "" {
		lines = append(lines, "Last modified by: "+info.Core.LastModifiedBy)
	}
	if info.Core.Modified != "" {
		lines = append(lines, "Modified: "+info.Core.Modified)
	}
	if info.Extended.Pages > 0 {
		lines = append(lines, fmt.Sprintf("Pages: %d", info.Extended.Pages))
	}
	if info.Extended.Words > 0 {
		lines = append(lines, fmt.Sprintf("Words: %d", info.Extended.Words))
	}
	if info.Extended.Slides > 0 && info.Kind == OpenXMLPowerPoint {
		lines = append(lines, fmt.Sprintf("Slides: %d", info.Extended.Slides))
	}
	lines = append(lines,
		"",
		fmt.Sprintf("Package parts: %d", info.EntryCount),
		"",
		"Full document rendering is not part of this foundation build yet.",
		"DCM can now validate this Office file and hand its package to the built-in DOCX/XLSX/PPTX renderers added in the next preview stages.",
	)
	return normalizePreviewNewlines(strings.Join(lines, "\n")), true
}
