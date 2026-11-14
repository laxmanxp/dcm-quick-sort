#!/usr/bin/env python3
"""Independently inspect the small DCM-33 binary DOC fixture; not an app parser.

Requires xlrd 2.0.1 for CFBF stream access. The Word FIB/CLX piece-table walk
follows Microsoft [MS-DOC] 2.4.1 and accepts only this fixture's plain text.
No LibreOffice, Microsoft Office or DCM application code is used for extraction.
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


def u16(data, offset):
    return struct.unpack_from("<H", data, offset)[0]


def u32(data, offset):
    return struct.unpack_from("<I", data, offset)[0]


def inspect_doc(path):
    raw = path.read_bytes()
    require(512 <= len(raw) <= 2 * 1024 * 1024, "Unexpected fixture size")
    require(raw[:8] == bytes.fromhex("d0cf11e0a1b11ae1"), "Not a CFBF binary file")
    require(u16(raw, 28) == 0xFFFE, "Unexpected CFBF byte order")
    compound = CompDoc(raw, logfile=io.StringIO())
    word = compound.get_named_stream("WordDocument")
    require(word is not None and len(word) >= 154, "Missing/truncated WordDocument")
    require(u16(word, 0) == 0xA5EC, "Missing Word binary FIB signature")
    flags = u16(word, 10)
    require(not flags & 0x0100, "Encrypted fixture is outside this gate")
    table_name = "1Table" if flags & 0x0200 else "0Table"
    table = compound.get_named_stream(table_name)
    require(table is not None, "FIB-selected table stream is absent")

    # The FIB's counted arrays locate FibRgLw97 and FibRgFcLcb97.
    csw = u16(word, 32)
    cslw_offset = 34 + 2 * csw
    cslw = u16(word, cslw_offset)
    lw_start = cslw_offset + 2
    require(cslw >= 4, "FIB has no main-story character count")
    ccp_text = u32(word, lw_start + 12)
    require(0 < ccp_text <= 4096, "Unexpected main-story size")
    pair_count_offset = lw_start + 4 * cslw
    pair_count = u16(word, pair_count_offset)
    require(pair_count > 33, "FIB has no CLX field pair")
    fc_clx, lcb_clx = struct.unpack_from("<II", word, pair_count_offset + 2 + 33 * 8)
    require(0 < lcb_clx <= len(table) and fc_clx + lcb_clx <= len(table), "CLX outside table stream")
    clx = table[fc_clx:fc_clx + lcb_clx]
    offset = 0
    while offset < len(clx) and clx[offset] == 1:  # Optional Prc records.
        offset += 3 + u16(clx, offset + 1)
    require(offset + 5 <= len(clx) and clx[offset] == 2, "CLX has no Pcdt")
    size = u32(clx, offset + 1)
    plc = clx[offset + 5:offset + 5 + size]
    require(len(plc) == size and size >= 16 and (size - 4) % 12 == 0, "Invalid PlcPcd size")
    pieces = (size - 4) // 12
    require(1 <= pieces <= 64, "Unexpected piece count")
    positions = struct.unpack_from("<" + "I" * (pieces + 1), plc, 0)
    require(positions[0] == 0 and positions[-1] >= ccp_text, "Piece table omits main story")
    require(all(a < b for a, b in zip(positions, positions[1:])), "Unordered piece positions")
    text_parts = []
    for index in range(pieces):
        count = positions[index + 1] - positions[index]
        require(count <= 4096, "Unexpected piece length")
        fc_encoded = u32(plc, 4 * (pieces + 1) + 8 * index + 2)
        require(not fc_encoded & 0x80000000, "Invalid FcCompressed reserved bit")
        compressed = bool(fc_encoded & 0x40000000)
        start = fc_encoded & 0x3FFFFFFF
        if compressed:
            require(start % 2 == 0, "Invalid compressed text offset")
            start //= 2
        length = count if compressed else 2 * count
        require(start + length <= len(word), "Text piece exceeds WordDocument")
        text_parts.append(word[start:start + length].decode("ascii" if compressed else "utf-16-le"))
    text = "".join(text_parts)[:ccp_text]
    require(all(c in "\r\n\t" or 32 <= ord(c) < 127 for c in text), "Unexpected non-fixture text")
    return {
        "bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest(),
        "cfbf_magic": raw[:8].hex(), "cfbf_major_version": u16(raw, 26),
        "sector_bytes": 1 << u16(raw, 30),
        "streams": [{"name": n.name, "bytes": n.tot_size} for n in compound.dirlist if n.etype == 2],
        "fib_wIdent": "0x%04X" % u16(word, 0), "fib_nFib": "0x%04X" % u16(word, 2),
        "fib_nFibBack": "0x%04X" % u16(word, 12), "encrypted": False,
        "table_stream": table_name, "csw": csw, "cslw": cslw,
        "fc_lcb_pair_count": pair_count, "ccpText": ccp_text,
        "fcClx": fc_clx, "lcbClx": lcb_clx, "piece_count": pieces,
        "main_text": text.replace("\r", "\n"), "cfbf_reader": "xlrd " + xlrd.__version__,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--file", type=Path)
    parser.add_argument("--manifest", type=Path, default=Path(__file__).with_name("dcm33-legacy.manifest.json"))
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text())
    path = args.file or args.manifest.parent / manifest["file"]
    result = inspect_doc(path)
    require(result["bytes"] == manifest["bytes"], "Fixture size differs from manifest")
    require(result["sha256"] == manifest["sha256"], "Fixture SHA-256 differs from manifest")
    lines = result["main_text"].splitlines()
    require(lines == manifest["required_text"], "Main-story text differs from the exact fixture content")
    require(result["fib_nFib"] == manifest["structure"]["fib_nFib"], "FIB version differs from manifest")
    require(result["table_stream"] == manifest["structure"]["table_stream"], "Table stream differs from manifest")
    result["passed"] = True
    print(json.dumps(result, indent=2))
