#!/usr/bin/env node
/**
 * Author the synthetic XLSX with the bundled @oai/artifact-tool and export BIFF8
 * with local LibreOffice. Writes only to an explicit fresh output directory.
 * Run from an OS temp directory whose node_modules symlink points to
 * CODEX_PRIMARY_RUNTIME_NODE_MODULES, using CODEX_PRIMARY_RUNTIME_NODE.
 * The committed fixture/manifest are pinned separately; regeneration never
 * updates them. No XLSX or XLS post-export normalization is performed.
 */
import fs from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createHash } from "node:crypto";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { Workbook, SpreadsheetFile } from "@oai/artifact-tool";

const run = promisify(execFile);
const args = process.argv.slice(2);
const option = (name, fallback) => {
  const index = args.indexOf(name);
  if (index < 0) return fallback;
  if (!args[index + 1] || args[index + 1].startsWith("--")) {
    throw new Error(`${name} needs a value`);
  }
  return args[index + 1];
};
const outputDir = option("--output-dir");
if (!outputDir || !path.isAbsolute(outputDir)) {
  throw new Error("Use --output-dir <absolute fresh workspace path>/outputs/<run-id>");
}
if (!process.env.CODEX_PRIMARY_RUNTIME_NODE ||
    await fs.realpath(process.execPath) !== await fs.realpath(process.env.CODEX_PRIMARY_RUNTIME_NODE)) {
  throw new Error("Run with CODEX_PRIMARY_RUNTIME_NODE from the bundled runtime");
}
await fs.mkdir(outputDir, { recursive: true });
const stem = "dcm33-legacy-xls";
const sourcePath = path.join(outputDir, `${stem}.source.xlsx`);
try {
  await fs.access(sourcePath);
  throw new Error("Output already exists; choose a fresh directory");
} catch (error) {
  if (error.code !== "ENOENT") throw error;
}

const workbook = Workbook.create();
const sheet = workbook.worksheets.add("Inspection");
sheet.showGridLines = false;
sheet.getRange("A1:B9").format.font = { name: "Arial", size: 12, color: "#17212E" };
sheet.getRange("A1:B9").format.verticalAlignment = "center";
sheet.getRange("A1:B9").format.rowHeightPx = 30;
sheet.getRange("A1:A9").format.columnWidthPx = 330;
sheet.getRange("B1:B9").format.columnWidthPx = 245;
sheet.getRange("A1:B1").format.rowHeightPx = 8;
sheet.getRange("A2:B2").format.rowHeightPx = 38;
sheet.getRange("A4:B4").format.rowHeightPx = 18;
sheet.getRange("A9:B9").format.rowHeightPx = 8;
sheet.getRange("A2").values = [["DCM-33 LEGACY XLS"]];
sheet.getRange("A2").format.font = { name: "Arial", size: 16, bold: true, color: "#17212E" };
sheet.getRange("A2:B2").format.borders = { bottom: { style: "thin", color: "#A6B1BD" } };
sheet.getRange("A3:B3").values = [["Inspection date", new Date("2026-11-14T00:00:00.000Z")]];
sheet.getRange("B3").setNumberFormat("dd mmmm yyyy");
sheet.getRange("A5:B8").values = [
  ["Description", "Amount"],
  ["Bridge inspection", 125000],
  ["Road repair", 86000],
  ["Total", null],
];
sheet.getRange("B8").formulas = [["=SUM(B6:B7)"]];
sheet.getRange("B6:B8").setNumberFormat("#,##0");
sheet.getRange("A5:B5").format.fill = "#334559";
sheet.getRange("A5:B5").format.font = { name: "Arial", size: 12, bold: true, color: "#FFFFFF" };
sheet.getRange("A5:B5").format.horizontalAlignment = "center";
sheet.getRange("A6:A8").format.horizontalAlignment = "left";
sheet.getRange("B3").format.horizontalAlignment = "right";
sheet.getRange("B6:B8").format.horizontalAlignment = "right";
sheet.getRange("A8:B8").format.font = { name: "Arial", size: 12, bold: true, color: "#17212E" };
sheet.getRange("A8:B8").format.fill = "#EDF1F5";
sheet.getRange("A8:B8").format.borders = { top: { style: "thin", color: "#A6B1BD" } };

const inputChangeChecks = [];
for (const [input, expected] of [[1, 86001], [0, 86000], [125000, 211000]]) {
  sheet.getRange("B6").values = [[input]];
  const actual = sheet.getRange("B8").values[0][0];
  if (actual !== expected) throw new Error(`SUM recalculation: ${input} yielded ${actual}, expected ${expected}`);
  inputChangeChecks.push({ B6: input, B7: 86000, B8: actual });
}
workbook.recalculate();
const inspection = await workbook.inspect({
  kind: "table", range: "Inspection!A1:B9", include: "values,formulas",
  tableMaxRows: 9, tableMaxCols: 2, maxChars: 5000,
});
const errorScan = await workbook.inspect({
  kind: "match", searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A|#NUM!|#NULL!|#SPILL!|#CALC!",
  options: { useRegex: true, maxResults: 20 }, summary: "final formula error scan", maxChars: 2000,
});
await fs.writeFile(path.join(outputDir, "artifact-inspection.ndjson"), inspection.ndjson);
await fs.writeFile(path.join(outputDir, "artifact-formula-error-scan.ndjson"), errorScan.ndjson);
const preview = await workbook.render({ sheetName: "Inspection", range: "A1:B9", scale: 1, format: "png" });
await fs.writeFile(path.join(outputDir, "artifact-preview.png"), new Uint8Array(await preview.arrayBuffer()));
const xlsx = await SpreadsheetFile.exportXlsx(workbook);
await xlsx.save(sourcePath);

const soffice = option("--soffice", "soffice");
const execOptions = { encoding: "utf8", timeout: 45000, maxBuffer: 1024 * 1024,
  env: { ...process.env, TZ: "UTC", LC_ALL: "C.UTF-8" } };
const version = (await run(soffice, ["--version"], execOptions)).stdout.trim();
const conversionLog = [];
async function convert(input, filter, directory) {
  await fs.mkdir(directory, { recursive: true });
  const profile = await fs.mkdtemp(path.join(os.tmpdir(), "dcm33-xls-lo-"));
  try {
    const result = await run(soffice, ["--headless", "--nologo", "--nodefault", "--nofirststartwizard",
      `-env:UserInstallation=${pathToFileURL(profile).href}`, "--convert-to", filter,
      "--outdir", directory, input], execOptions);
    conversionLog.push({ input: path.basename(input), filter, stdout: result.stdout.trim(), stderr: result.stderr.trim() });
  } finally {
    await fs.rm(profile, { recursive: true, force: true });
  }
}
await convert(sourcePath, "xls:MS Excel 97", outputDir);
const fixturePath = path.join(outputDir, `${stem}.xls`);
await fs.rename(path.join(outputDir, `${stem}.source.xls`), fixturePath);
const fixtureBytes = await fs.readFile(fixturePath);
if (fixtureBytes.subarray(0, 8).toString("hex") !== "d0cf11e0a1b11ae1") {
  throw new Error("Export lacks the CFBF signature");
}
await convert(sourcePath, "pdf:calc_pdf_Export", path.join(outputDir, "source-render"));
await convert(fixturePath, "pdf:calc_pdf_Export", path.join(outputDir, "binary-render"));
const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");
const sourceBytes = await fs.readFile(sourcePath);
const packageInfo = JSON.parse(await fs.readFile(path.join(process.env.CODEX_PRIMARY_RUNTIME_NODE_MODULES,
  "@oai/artifact-tool/package.json"), "utf8"));
const summary = {
  file: path.basename(fixturePath), bytes: fixtureBytes.length, sha256: sha256(fixtureBytes),
  source_file: path.basename(sourcePath), source_bytes: sourceBytes.length, source_sha256: sha256(sourceBytes),
  generator_file: path.basename(fileURLToPath(import.meta.url)),
  node_version: process.version, artifact_tool_version: packageInfo.version,
  libreoffice_version: version, export_filter: "MS Excel 97", post_export_normalization: false,
  sheet_name: "Inspection", date_system: "1900", input_change_checks: inputChangeChecks,
  final_total: sheet.getRange("B8").values[0][0], formula: sheet.getRange("B8").formulas[0][0],
  conversion_log: conversionLog,
  acceptance_scope: "Local fixture preparation only. Native DCM/Windows acceptance not run by this generator.",
};
await fs.writeFile(path.join(outputDir, "generation.json"), JSON.stringify(summary, null, 2) + "\n");
console.log(JSON.stringify(summary, null, 2));
