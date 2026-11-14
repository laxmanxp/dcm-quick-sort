#!/usr/bin/env python3
"""Export the pinned DCM-32 slides as a genuine PowerPoint 97 binary fixture.

Uses the existing source unchanged. Requires a local LibreOffice soffice.
Creates an explicit output directory's PPT and generation.json, never changes
the committed source or manifest. Binary metadata may differ between exports;
use the validator's explicit --semantic-only mode when comparing regeneration.
"""

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile


SOURCE_SHA256 = "af42fff55297109195b95039d2ac39c5d53b1fae23b59e7b40a7184d679b93f1"


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def generate(output_dir, soffice):
    source = Path(__file__).with_name("dcm32-slides.pptx").resolve()
    if sha256(source) != SOURCE_SHA256:
        raise ValueError("Pinned source PPTX changed")
    output_dir.mkdir(parents=True, exist_ok=True)
    fixture = output_dir / "dcm33-legacy-ppt.ppt"
    report = output_dir / "generation.json"
    if fixture.exists() or report.exists():
        raise ValueError("Choose an output directory without an existing fixture or generation.json")
    version = subprocess.run(
        [soffice, "--version"], check=True, capture_output=True, text=True, timeout=15
    ).stdout.strip()
    with tempfile.TemporaryDirectory(prefix="dcm33-ppt-export-") as temporary:
        temporary = Path(temporary)
        profile = temporary / "profile"
        converted = temporary / "converted"
        converted.mkdir()
        result = subprocess.run([
            soffice, "--headless", "--nologo", "--nodefault", "--nofirststartwizard",
            "-env:UserInstallation=" + profile.as_uri(),
            "--convert-to", "ppt:MS PowerPoint 97", "--outdir", str(converted), str(source),
        ], check=True, capture_output=True, text=True, timeout=45)
        exported = converted / "dcm32-slides.ppt"
        if not exported.is_file():
            raise RuntimeError("PowerPoint 97 export absent: " + result.stdout + result.stderr)
        raw = exported.read_bytes()
        if raw[:8] != bytes.fromhex("d0cf11e0a1b11ae1"):
            raise ValueError("PowerPoint 97 export is not CFBF")
        with fixture.open("xb") as stream:
            stream.write(raw)
    if sha256(source) != SOURCE_SHA256:
        raise ValueError("Source changed during conversion")
    detail = {
        "file": fixture.name, "bytes": fixture.stat().st_size, "sha256": sha256(fixture),
        "source": source.name, "source_sha256": SOURCE_SHA256, "source_unchanged": True,
        "libreoffice_version": version, "export_filter": "MS PowerPoint 97",
        "conversion_stdout": result.stdout.strip(), "conversion_stderr": result.stderr.strip(),
        "byte_identical_regeneration_assumed": False,
    }
    with report.open("x", encoding="utf-8") as stream:
        json.dump(detail, stream, indent=2)
        stream.write("\n")
    print(json.dumps(detail, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--soffice", default=shutil.which("soffice"))
    args = parser.parse_args()
    if not args.soffice:
        parser.error("No local soffice found; pass --soffice")
    generate(args.output_dir.resolve(), args.soffice)
