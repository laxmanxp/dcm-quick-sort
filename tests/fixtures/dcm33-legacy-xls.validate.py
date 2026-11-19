#!/usr/bin/env python3
"""Validate the pinned synthetic BIFF8 XLS independently of DCM and LibreOffice.

Requires xlrd 2.0.1. The intentionally narrow BIFF reader proves this fixture's
Workbook/worksheet BOFs and one unshared SUM-area formula with numeric cache.
It is a fixture validator, not a general spreadsheet parser or renderer.
"""
import argparse
from datetime import datetime
import hashlib
import io
import json
from pathlib import Path
import re
import struct
import sys
import xml.etree.ElementTree as ET
import zipfile

import xlrd
from xlrd.compdoc import CompDoc


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def coordinates(address):
    match = re.fullmatch(r"([A-Z]+)([1-9][0-9]*)", address)
    require(match is not None, "Invalid cell address: " + address)
    column = 0
    for char in match[1]:
        column = column * 26 + ord(char) - ord("A") + 1
    return int(match[2]) - 1, column - 1


def address(row, column, row_relative=True, column_relative=True):
    require(0 <= column < 26, "Fixture formula expects a single-letter column")
    return ("" if column_relative else "$") + chr(65 + column) + ("" if row_relative else "$") + str(row + 1)


def biff_records(stream):
    records = []
    offset = 0
    while offset < len(stream):
        require(offset + 4 <= len(stream), "Truncated BIFF record header")
        code, length = struct.unpack_from("<HH", stream, offset)
        require(offset + 4 + length <= len(stream), "Truncated BIFF record payload")
        records.append((offset, code, stream[offset + 4:offset + 4 + length]))
        offset += 4 + length
    return records


def inspect_biff(raw, sheet_name):
    require(raw[:8] == bytes.fromhex("d0cf11e0a1b11ae1"), "Not CFBF: renamed XLSX/CSV is not legacy XLS")
    require(len(raw) >= 512, "Truncated CFBF header")
    require(struct.unpack_from("<H", raw, 28)[0] == 0xFFFE, "Unexpected CFBF byte order")
    compound = CompDoc(raw, logfile=io.StringIO())
    streams = [{"name": node.name, "bytes": node.tot_size} for node in compound.dirlist if node.etype == 2]
    stream = compound.get_named_stream("Workbook")
    require(stream is not None, "CFBF does not contain a Workbook stream")
    records = biff_records(stream)
    bofs = []
    formulas = []
    boundsheets = []
    current_type = None
    for offset, code, data in records:
        if code == 0x0809:
            require(len(data) == 16, "BIFF8 BOF payload must contain 16 bytes")
            version, current_type = struct.unpack_from("<HH", data)
            require(version == 0x0600, "Workbook is not BIFF8 (BOF version 0x0600)")
            bofs.append({"offset": offset, "version": "0x0600", "type": f"0x{current_type:04X}"})
        if code == 0x002F:
            raise ValueError("Encrypted workbook is outside this fixture contract")
        if code == 0x0085:
            require(current_type == 5 and len(data) >= 8, "Malformed BoundSheet8")
            worksheet_offset = struct.unpack_from("<I", data)[0]
            visibility, kind, char_count, flags = data[4:8]
            require(visibility == 0 and kind == 0, "Fixture requires one visible worksheet")
            require(flags in (0, 1), "Unexpected BoundSheet8 string flags")
            encoded = data[8:8 + char_count * (2 if flags else 1)]
            name = encoded.decode("utf-16le" if flags else "latin1")
            boundsheets.append({"name": name, "offset": worksheet_offset, "visible": True})
        if code == 0x0006:
            require(current_type == 0x0010 and len(data) >= 22, "Formula is not in the worksheet")
            row, column, xf = struct.unpack_from("<HHH", data)
            require(data[12:14] != b"\xff\xff", "Formula cached result is not numeric")
            cache = struct.unpack_from("<d", data, 6)[0]
            flags = struct.unpack_from("<H", data, 14)[0]
            require(not flags & 0x0008, "Shared formula is outside this fixture contract")
            token_count = struct.unpack_from("<H", data, 20)[0]
            tokens = data[22:22 + token_count]
            require(len(data) == 22 + token_count == 35, "Expected the 13-byte SUM-area token sequence")
            require(tokens[0] == 0x25 and tokens[9:11] == b"\x19\x10", "Formula must contain PtgArea then PtgAttrSum")
            first_row, last_row, first_column, last_column = struct.unpack_from("<HHHH", tokens, 1)
            first = address(first_row, first_column & 0x3FFF, bool(first_column & 0x8000), bool(first_column & 0x4000))
            last = address(last_row, last_column & 0x3FFF, bool(last_column & 0x8000), bool(last_column & 0x4000))
            formulas.append({
                "address": address(row, column), "formula": f"=SUM({first}:{last})",
                "cached_value": cache, "xf_index": xf, "record_offset": offset,
                "record_id": "0x0006", "token_bytes": token_count, "tokens_hex": tokens.hex(),
                "token_types": ["PtgArea", "PtgAttrSum"],
            })
    require([entry["type"] for entry in bofs] == ["0x0005", "0x0010"], "Expected only workbook and worksheet substreams")
    require(bofs[0]["offset"] == 0, "Workbook stream must start with its BOF")
    require(sum(code == 0x000A for _, code, _ in records) == 2, "Expected two EOF records")
    require(boundsheets == [{"name": sheet_name, "offset": bofs[1]["offset"], "visible": True}], "BoundSheet8 does not identify the sole worksheet")
    require(len(formulas) == 1, "Expected exactly one real formula record")
    return {
        "cfbf_magic": raw[:8].hex(), "cfbf_major_version": struct.unpack_from("<H", raw, 26)[0],
        "sector_bytes": 1 << struct.unpack_from("<H", raw, 30)[0], "streams": streams,
        "workbook_stream_bytes": len(stream), "bof_records": bofs, "boundsheets": boundsheets,
        "encrypted": False, "formula_records": formulas,
    }


def displayed(cell, kind, number_format, datemode):
    if kind == "text":
        return cell.value
    if kind == "date":
        require(number_format.replace("\\ ", " ") == "dd mmmm yyyy", "Unexpected date format")
        date = xlrd.xldate_as_datetime(cell.value, datemode)
        months = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"]
        return f"{date.day:02d} {months[date.month - 1]} {date.year:04d}"
    require(number_format == "#,##0" and cell.value == int(cell.value), "Unexpected numeric display contract")
    return f"{int(cell.value):,}"


def inspect_source(path, manifest):
    raw = path.read_bytes()
    require(digest(raw) == manifest["generator"]["source_sha256"], "Source XLSX SHA-256 mismatch")
    require(len(raw) == manifest["generator"]["source_bytes"], "Source XLSX size mismatch")
    namespace = {"x": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}
    with zipfile.ZipFile(io.BytesIO(raw)) as archive:
        workbook = ET.fromstring(archive.read("xl/workbook.xml"))
        names = [item.attrib["name"] for item in workbook.findall("x:sheets/x:sheet", namespace)]
        require(names == [manifest["sheet_name"]], "Source XLSX sheet mismatch")
        worksheet = ET.fromstring(archive.read("xl/worksheets/sheet1.xml"))
        styles = ET.fromstring(archive.read("xl/styles.xml"))
    formats = {0: "General"}
    formats.update({int(item.attrib["numFmtId"]): item.attrib["formatCode"] for item in styles.findall("x:numFmts/x:numFmt", namespace)})
    xfs = styles.findall("x:cellXfs/x:xf", namespace)
    cells = {item.attrib["r"]: item for item in worksheet.findall("x:sheetData/x:row/x:c", namespace)}
    populated = {key for key, item in cells.items() if item.find("x:v", namespace) is not None}
    require(populated == {item["address"] for item in manifest["cell_oracles"]}, "Source XLSX populated-cell contract mismatch")
    for oracle in manifest["cell_oracles"]:
        cell = cells[oracle["address"]]
        value = cell.findtext("x:v", namespaces=namespace)
        formula = cell.findtext("x:f", namespaces=namespace)
        fmt = formats[int(xfs[int(cell.attrib["s"])].attrib["numFmtId"])]
        require(fmt == oracle["number_format"], oracle["address"] + " source number format mismatch")
        if oracle["type"] == "date":
            require(cell.attrib.get("t") == "d", "Source B3 must be a typed date")
            require(datetime.fromisoformat(value.replace("Z", "+00:00")).date().isoformat() == oracle["iso_date"], "Source date mismatch")
        elif oracle["type"] == "text":
            require(cell.attrib.get("t") == "str" and value == oracle["value"], oracle["address"] + " source text mismatch")
        else:
            require(cell.attrib.get("t") == "n" and float(value) == oracle["value"], oracle["address"] + " source numeric/cache mismatch")
        require(formula == (oracle.get("formula", "")[1:] or None), oracle["address"] + " source formula mismatch")
    return {"file": path.name, "bytes": len(raw), "sha256": digest(raw), "cell_contract_matches": True}


def validate(fixture, manifest_path):
    require(xlrd.__version__ == "2.0.1", "Use the pinned xlrd 2.0.1 validator dependency")
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    require(manifest["schema_version"] == 1, "Unsupported fixture manifest schema")
    raw = fixture.read_bytes()
    structure = inspect_biff(raw, manifest["sheet_name"])
    book = xlrd.open_workbook(file_contents=raw, formatting_info=True, logfile=io.StringIO())
    require(book.biff_version == 80 and book.datemode == manifest["datemode"] == 0, "Expected BIFF8 with the 1900 date system")
    require(book.sheet_names() == [manifest["sheet_name"]], "Expected exactly the Inspection worksheet")
    sheet = book.sheet_by_index(0)
    require(sheet.visibility == 0, "Fixture sheet is hidden")
    expected_types = {"text": xlrd.XL_CELL_TEXT, "number": xlrd.XL_CELL_NUMBER, "date": xlrd.XL_CELL_DATE, "formula": xlrd.XL_CELL_NUMBER}
    actual_cells = []
    for oracle in manifest["cell_oracles"]:
        row, column = coordinates(oracle["address"])
        cell = sheet.cell(row, column)
        fmt = book.format_map[book.xf_list[cell.xf_index].format_key].format_str
        require(cell.ctype == expected_types[oracle["type"]], oracle["address"] + " cell type mismatch")
        require(cell.value == oracle["value"], oracle["address"] + " value mismatch")
        require(fmt == oracle.get("xls_number_format", oracle["number_format"]), oracle["address"] + " number format mismatch")
        visible = displayed(cell, oracle["type"], fmt, book.datemode)
        require(visible == oracle["display_text"], oracle["address"] + " display text mismatch")
        if oracle["type"] == "date":
            require(xlrd.xldate_as_datetime(cell.value, book.datemode).isoformat() == oracle["iso_date"] + "T00:00:00", "Date semantic mismatch")
        if oracle["type"] == "formula":
            record = structure["formula_records"][0]
            require(record["address"] == oracle["address"] and record["formula"] == oracle["formula"], "BIFF formula expression mismatch")
            require(record["cached_value"] == oracle["cached_value"] == cell.value, "BIFF formula cache mismatch")
        actual_cells.append({"address": oracle["address"], "xlrd_type": cell.ctype, "value": cell.value, "number_format": fmt, "display_text": visible})
    populated = {address(r, c) for r in range(sheet.nrows) for c in range(sheet.ncols) if sheet.cell(r, c).ctype not in (xlrd.XL_CELL_EMPTY, xlrd.XL_CELL_BLANK)}
    require(populated == {item["address"] for item in manifest["cell_oracles"]}, "Unexpected populated cells")
    require([item["display_text"] for item in actual_cells] == manifest["required_text"], "required_text differs from the cell contract")
    require(digest(raw) == manifest["sha256"], "XLS SHA-256 mismatch")
    require(len(raw) == manifest["bytes"], "XLS size mismatch")
    source = inspect_source(manifest_path.parent / manifest["generator"]["source_file"], manifest)
    return {"passed": True, "file": fixture.name, "bytes": len(raw), "sha256": digest(raw),
            "validator_dependency": "xlrd " + xlrd.__version__, "biff_version": book.biff_version,
            "datemode": book.datemode, "sheet_name": sheet.name, "cell_oracles": actual_cells,
            "structure": structure, "source_validation": source,
            "display_check_scope": "Invariant display strings computed from typed values and exact formats. Actual local rendering and native UI review are separate evidence.",
            "native_office_acceptance": "not_run"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    base = Path(__file__).resolve().parent
    parser.add_argument("--file", type=Path, default=base / "dcm33-legacy-xls.xls")
    parser.add_argument("--manifest", type=Path, default=base / "dcm33-legacy-xls.manifest.json")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        result = validate(args.file, args.manifest)
    except Exception as error:
        result = {"passed": False, "file": args.file.name, "error": str(error)}
    text = json.dumps(result, indent=2, ensure_ascii=False) + "\n"
    if args.output:
        args.output.write_text(text, encoding="utf-8")
    print(text, end="")
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
