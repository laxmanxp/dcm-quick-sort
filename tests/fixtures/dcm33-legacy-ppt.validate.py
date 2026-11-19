#!/usr/bin/env python3
"""Inspect the bounded, single-edit DCM-33 PPT fixture independently of Office.

Requires xlrd 2.0.1 only for CFBF stream access. This is a fixture validator,
not an application PPT parser. Follows Microsoft [MS-PPT] 2.1.2, 2.3.1-2.3.5,
2.4.14.3, 2.4.14.5, 2.9.42-2.9.43 and the OfficeArtClientTextbox contract:
https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/

No LibreOffice, Microsoft Office or DCM application code extracts the evidence.
Default validation checks the pinned binary identity and semantic content.
--semantic-only is explicit regeneration/negative-probe mode, never native
acceptance of an unpinned replacement fixture.
"""

import argparse
import hashlib
import io
import json
from pathlib import Path
import struct

import xlrd
from xlrd.compdoc import CompDoc


def require(condition, message):
    if not condition:
        raise ValueError(message)


def u32(data, offset):
    require(0 <= offset <= len(data) - 4, "32-bit field outside stream")
    return struct.unpack_from("<I", data, offset)[0]


def record(data, offset, limit=None):
    limit = len(data) if limit is None else limit
    require(0 <= offset <= limit - 8 <= len(data) - 8, "Truncated PowerPoint record header")
    version_instance, kind, length = struct.unpack_from("<HHI", data, offset)
    end = offset + 8 + length
    require(end <= limit, "PowerPoint record exceeds its container")
    return {
        "offset": offset, "start": offset + 8, "end": end, "length": length,
        "version": version_instance & 15, "instance": version_instance >> 4, "kind": kind,
    }


def children(data, parent):
    offset = parent["start"]
    while offset < parent["end"]:
        child = record(data, offset, parent["end"])
        yield child
        offset = child["end"]


def expect(item, kind, version=0, instance=0, length=None):
    require(item["kind"] == kind and item["version"] == version and item["instance"] == instance,
            "Unexpected PowerPoint record identity")
    if length is not None:
        require(item["length"] == length, "Unexpected PowerPoint record length")


def text_blocks(data, parent, depth=0):
    require(depth <= 16, "Fixture record nesting exceeds the bound")
    blocks = []
    for child in children(data, parent):
        kind = child["kind"]
        body = data[child["start"]:child["end"]]
        if kind in (0x0FA0, 0x0FA8):  # TextCharsAtom / TextBytesAtom.
            expect(child, kind)
            require(len(body) <= 4096, "Fixture text exceeds the bound")
            require(kind != 0x0FA0 or len(body) % 2 == 0, "Odd UTF-16 text size")
            text = body.decode("utf-16-le" if kind == 0x0FA0 else "latin1")
            require(all(c in "\r\n\t" or 32 <= ord(c) < 127 for c in text), "Unexpected fixture text")
            blocks.append(text)
        elif child["version"] == 15 or kind == 0xF00D:
            # OfficeArtClientTextbox (F00D) stores PPT child records with recVer 0.
            blocks.extend(text_blocks(data, child, depth + 1))
    return blocks


def embedded_png(pictures):
    signature = b"\x89PNG\r\n\x1a\n"
    offset = pictures.find(signature)
    require(offset >= 0 and pictures.find(signature, offset + 1) == -1, "Expected one embedded PNG")
    cursor = offset + len(signature)
    while cursor < len(pictures):
        require(cursor + 12 <= len(pictures), "Truncated PNG chunk")
        length = struct.unpack_from(">I", pictures, cursor)[0]
        kind = pictures[cursor + 4:cursor + 8]
        cursor += length + 12
        require(cursor <= len(pictures), "PNG chunk outside Pictures stream")
        if kind == b"IEND":
            require(length == 0, "Invalid IEND chunk")
            return pictures[offset:cursor]
    raise ValueError("Embedded PNG has no IEND")


def inspect_ppt(path):
    raw = path.read_bytes()
    require(512 <= len(raw) <= 2 * 1024 * 1024, "Unexpected fixture size")
    require(raw[:8] == bytes.fromhex("d0cf11e0a1b11ae1"), "Not a CFBF binary PPT")
    compound = CompDoc(raw, logfile=io.StringIO())
    data = compound.get_named_stream("PowerPoint Document")
    current = compound.get_named_stream("Current User")
    pictures = compound.get_named_stream("Pictures")
    require(data is not None and current is not None and pictures is not None,
            "Required PowerPoint Document, Current User or Pictures stream absent")
    current_record = record(current, 0)
    expect(current_record, 0x0FF6)
    require(current_record["end"] == len(current) and current_record["length"] >= 20,
            "Unexpected Current User stream size")
    require(u32(current, 8) == 20 and u32(current, 12) == 0xE391C05F,
            "Encrypted or unexpected Current User header")
    user = record(data, u32(current, 16))
    expect(user, 0x0FF5, length=28)
    require(u32(data, user["start"] + 8) == 0, "Incremental-edit files are outside this fixture gate")
    directory_offset = u32(data, user["start"] + 12)
    document_id = u32(data, user["start"] + 16)
    require(0 < directory_offset < user["offset"] and document_id == 1, "Invalid live-document reference")
    directory = record(data, directory_offset)
    expect(directory, 0x1772)
    objects = {}
    cursor = directory["start"]
    while cursor < directory["end"]:
        packed = u32(data, cursor)
        first, count = packed & 0xFFFFF, packed >> 20
        cursor += 4
        require(1 <= count <= 256 and first <= 0xFFFFE and cursor + 4 * count <= directory["end"],
                "Invalid persist directory entry")
        for index in range(count):
            identity = first + index
            offset = u32(data, cursor + index * 4)
            require(identity not in objects and offset < directory_offset, "Duplicate/invalid persist object")
            objects[identity] = offset
        cursor += 4 * count
    require(document_id in objects, "Live document missing from persist directory")
    document = record(data, objects[document_id])
    expect(document, 0x03E8, version=15)
    lists = [r for r in children(data, document) if r["kind"] == 0x0FF0 and r["instance"] == 0]
    require(len(lists) == 1, "Expected one presentation slide list")
    expect(lists[0], 0x0FF0, version=15)
    slides = []
    for reference in children(data, lists[0]):
        expect(reference, 0x03F3, length=20)
        identity = u32(data, reference["start"])
        require(identity in objects, "Slide persist object missing")
        slide = record(data, objects[identity])
        expect(slide, 0x03EE, version=15)
        slides.append({
            "number": len(slides) + 1, "persist_id": identity,
            "slide_id": u32(data, reference["start"] + 12),
            "offset": slide["offset"], "text_blocks": text_blocks(data, slide),
        })
    require(1 <= len(slides) <= 8, "Unexpected fixture slide count")
    png = embedded_png(pictures)
    return {
        "bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest(),
        "cfbf_magic": raw[:8].hex(), "cfbf_reader": "xlrd " + xlrd.__version__,
        "streams": [{"name": n.name, "bytes": n.tot_size} for n in compound.dirlist if n.etype == 2],
        "encrypted": False, "user_edit_count": 1, "persist_object_count": len(objects),
        "slide_count": len(slides), "slides": slides,
        "embedded_png_bytes": len(png), "embedded_png_sha256": hashlib.sha256(png).hexdigest(),
    }


def validate(path, manifest, semantic_only=False):
    result = inspect_ppt(path)
    require(result["slide_count"] == manifest["slide_count"], "Slide count differs from fixture")
    require(len(manifest["slide_oracles"]) == result["slide_count"], "Incomplete slide oracles")
    for actual, oracle in zip(result["slides"], manifest["slide_oracles"]):
        require(actual["number"] == oracle["number"], "Slide oracle order differs")
        require(actual["text_blocks"] == oracle["text_blocks"], "Ordered slide text differs from fixture")
        require(all(t in actual["text_blocks"] for t in oracle["required_text"]), "Required slide text absent")
    require(result["embedded_png_sha256"] == manifest["embedded_png_sha256"], "Embedded picture differs")
    if not semantic_only:
        require(result["bytes"] == manifest["bytes"], "Fixture size differs from manifest")
        require(result["sha256"] == manifest["sha256"], "Fixture SHA-256 differs from manifest")
    result["pinned_identity_checked"] = not semantic_only
    result["passed"] = True
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--file", type=Path)
    parser.add_argument("--manifest", type=Path, default=Path(__file__).with_name("dcm33-legacy-ppt.manifest.json"))
    parser.add_argument("--semantic-only", action="store_true")
    args = parser.parse_args()
    definition = json.loads(args.manifest.read_text(encoding="utf-8"))
    target = args.file or args.manifest.parent / definition["file"]
    print(json.dumps(validate(target, definition, args.semantic_only), indent=2))
