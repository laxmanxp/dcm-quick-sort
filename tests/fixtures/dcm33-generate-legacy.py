#!/usr/bin/env python3
"""Generate synthetic DCM-33 source and export a genuine legacy Word .doc.

Requires python-docx and a local LibreOffice soffice. Output goes to an explicit
directory; generation does not update the committed fixture manifest. The DOCX
source has fixed metadata and ZIP timestamps. LibreOffice's binary export is
pinned separately; byte-identical regeneration is not assumed.
"""

import argparse
from datetime import datetime, timezone
import hashlib
import io
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import zipfile

import docx
from docx import Document
from docx.oxml.ns import qn
from docx.shared import Inches, Pt, RGBColor


REQUIRED_TEXT = [
    "DCM-33 LEGACY DOC",
    "Inspection date: 14 November 2026",
    "Bridge count: 7",
    "Status: Ready for review",
]
FIXED_DATE = datetime(2026, 11, 14, tzinfo=timezone.utc)


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def generate(output_dir, soffice):
    output_dir.mkdir(parents=True, exist_ok=True)
    source = output_dir / "dcm33-legacy.source.docx"
    document = Document()
    section = document.sections[0]
    section.page_width, section.page_height = Inches(8.5), Inches(11)
    section.top_margin = section.bottom_margin = Inches(0.65)
    section.left_margin = section.right_margin = Inches(0.65)
    for name, size in [("Normal", 16), ("Title", 24)]:
        style = document.styles[name]
        style.font.name = "Arial"
        style.font.size = Pt(size)
        style.font.color.rgb = RGBColor(0, 0, 0)
        style.paragraph_format.space_after = Pt(12)
        for border in style.element.xpath(".//w:pBdr"):
            border.getparent().remove(border)
        for fonts in style.element.xpath("./w:rPr/w:rFonts"):
            for attribute in ("asciiTheme", "hAnsiTheme", "eastAsiaTheme", "cstheme"):
                fonts.attrib.pop(qn("w:" + attribute), None)
    document.styles["Title"].font.bold = True
    document.add_paragraph(REQUIRED_TEXT[0], style="Title")
    for text in REQUIRED_TEXT[1:]:
        document.add_paragraph(text)
    props = document.core_properties
    props.title = "DCM-33 LEGACY DOC"
    props.subject = "Synthetic legacy DOC preview qualification fixture"
    props.author = "DCM QuickSort qualification"
    props.last_modified_by = "DCM QuickSort qualification"
    props.comments = "Entirely synthetic content; no personal or sensitive data."
    props.created = props.modified = FIXED_DATE
    props.revision = 1
    memory = io.BytesIO()
    document.save(memory)
    with zipfile.ZipFile(memory) as incoming, zipfile.ZipFile(source, "w") as outgoing:
        for name in sorted(incoming.namelist()):
            info = zipfile.ZipInfo(name, (2026, 11, 14, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.create_system = 0
            outgoing.writestr(info, incoming.read(name))
    version = subprocess.run(
        [soffice, "--version"], check=True, capture_output=True, text=True, timeout=15
    ).stdout.strip()
    with tempfile.TemporaryDirectory(prefix="dcm33-lo-profile-") as profile:
        command = [
            soffice, "--headless", "--nologo", "--nodefault", "--nofirststartwizard",
            "-env:UserInstallation=" + Path(profile).as_uri(),
            "--convert-to", "doc:MS Word 97", "--outdir", str(output_dir), str(source),
        ]
        result = subprocess.run(
            command, check=True, capture_output=True, text=True, timeout=45
        )
    exported = output_dir / "dcm33-legacy.source.doc"
    if not exported.is_file():
        raise RuntimeError("LibreOffice did not create the Word 97 export: " + result.stdout + result.stderr)
    fixture = output_dir / "dcm33-legacy.doc"
    exported.replace(fixture)
    if fixture.read_bytes()[:8] != bytes.fromhex("d0cf11e0a1b11ae1"):
        raise ValueError("Export is not a Compound File Binary Format document")
    print(json.dumps({
        "file": fixture.name, "bytes": fixture.stat().st_size, "sha256": sha256(fixture),
        "source": source.name, "source_sha256": sha256(source),
        "required_text": REQUIRED_TEXT, "python_docx_version": docx.__version__,
        "libreoffice_version": version, "export_filter": "MS Word 97",
        "conversion_stdout": result.stdout.strip(), "conversion_stderr": result.stderr.strip(),
    }, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--soffice", default=shutil.which("soffice"))
    args = parser.parse_args()
    if not args.soffice:
        parser.error("No local soffice found; pass --soffice")
    generate(args.output_dir.resolve(), args.soffice)
