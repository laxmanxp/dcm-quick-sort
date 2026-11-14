# Owned native DCM qualification

These checks extend the assertions used on `qa/flowuivision-v1069` at
`2ece9b1e044545c666b6c6296052feb6b985b750`. They reuse FlowUIVision for readiness,
ordinary text/combobox operations, and state assertions. DCM owns the narrow native
adapter needed for asynchronous button activation, exact native ListBox selection
and readback, confirmation ownership, and reliable screen capture.

## Invocation

From the isolated DCM FlowDev profile, first run `update dcm-quicksort`, then
`test dcm-quicksort ui-tooling`. This parses the committed scripts, compiles the C#
adapter, and creates disposable fixtures; it does not launch DCM or certify it.
Run `test dcm-quicksort build` or `package` on the same commit. Set the exact receipt
path emitted by that check, rather than selecting an artifact by newest filename:

```powershell
$env:DCM_BUILD_RECEIPT = 'C:\FlowverseReports\dcm-quicksort-build-<exact-run>\receipt.json'
$env:DCM_FLOWUIVISION_EXE = 'C:\FlowverseReports\dcm-flowuivision-20261103\FlowUIVision.exe'
& $flowdev test dcm-quicksort ui-filters
& $flowdev test dcm-quicksort ui-rename
& $flowdev test dcm-quicksort ui-move
& $flowdev test dcm-quicksort ui-batch
& $flowdev test dcm-quicksort ui-pdf
& $flowdev test dcm-quicksort ui-previews
```

The runner requires the receipt's commit and per-file source hashes to match the
clean checkout. It verifies the app hash before copying it into a unique owned lab,
then launches only that copy with isolated configuration and temporary directories.
Every action checks the PID, executable path, process start time, and window owner.
Each lab retains its exact fixtures, app, source binding, raw FVS outputs, journal,
filesystem evidence, screenshots and cleanup outcome. No installed DCM files change.

## Behavioral gates

- `legacy-preview-unit`: focused DCM-33 legacy routing/body coverage and native
  unavailable-status transitions. Hidden owned controls exercise blank, prior
  TXT-success and prior native-success footers with missing/failed handlers;
  the tests do not instantiate Office or modify registry associations.
- `ui-legacy-doc`: the first DCM-33 headed baby step. It uses the pinned genuine
  Word 97 `tests/fixtures/dcm33-legacy.doc` and manifest, a matching clean build,
  a harmless unsupported-extension control, TXT recovery, native DOC or an
  honest unavailable state, exclusive release, reopening and normal close.
  Missing-provider simulation by changing live Office registrations is forbidden.
  Every captured full image and same-frame physical crop requires a separate
  semantic review. Raw feature/review flags stay false. This DOC-only gate does
  not close DCM-33's legacy XLS/PPT scope or qualify headed DOC without Office.
- `ui-legacy-xls`: the next DCM-33 headed baby step uses pinned binary
  `tests/fixtures/dcm33-legacy-xls.xls` and its manifest with an exact clean
  build. Its six records cover TXT, first XLS preview, TXT recovery, exclusive
  workbook release, reopening and normal close. The installed provider or
  honest unavailable path is selected naturally; registry registrations are
  read before and after and must remain unchanged. Shared unsupported-control
  coverage stays in `ui-legacy-doc` and is not repeated here. Every native
  capture checks the direct `EXCEL9`/`EXCEL7` hierarchy and both viewport sizes
  and origins against the owned pane in PMv2 coordinates, using the unchanged
  four-pixel tolerance from `ui-xlsx`. Every attempt, its geometry and its
  full-window image plus crop from that same saved frame are retained; bounds
  failures are reported after recovery and normal close. Initial placement
  occurs during TXT only, with no post-attachment Restore or Excel input.
  Separate image review must confirm the manifest's sheet, labels, complete
  date, numeric amounts and formula result. Nonblank pixels, accessible text
  and provider captions do not certify cell readability. Raw feature/format
  acceptance and capture review flags remain false. This representative XLS
  gate does not close the pending PPT scope. Installed-handler results do not
  qualify a missing-handler environment, other providers, damaged/protected
  files or arbitrary workbook fidelity.
- `pptx-unit`: focused DCM-32 PPTX/Open XML parser qualification with the committed
  synthetic `tests/fixtures/dcm32-slides.pptx` and its manifest (31,109 bytes,
  SHA-256 `af42fff55297109195b95039d2ac39c5d53b1fae23b59e7b40a7184d679b93f1`).
  Slide one contains `Bridge Inspection`, `September 2026` and an embedded bridge
  image; slide two contains `Payments` and the `Scheme`/`Amount` table with
  `Bridge`/`125000` and `Road`/`86000`. These are synthetic test values.
  `tools/test-pptx-unit.ps1` sets `DCM_REAL_PPTX` only for the native runner's
  process tree, restores the prior value, and verifies actual run/pass events for
  the unchanged `TestRealGeneratedPPTXIntegration`. Targeted skips fail. The
  fixture, its manifest and the exact source snapshot must agree. The original
  raw receipt is preserved; the additional `pptx-fixture-verification.json`
  records the fixture/event check. Tests consume the committed PPTX rather than
  regenerate it; `tests/fixtures/generate-pptx-fixture.py` records reproducibility.
- `ui-pptx`: focused DCM-32 visible presentation qualification using that same
  pinned deck and an exact clean build receipt. The installed Windows preview
  handler or built-in fallback is selected naturally; associations are never
  altered. Actual text/image readability must be reviewed in the captures.
  The harness also checks text recovery, reopening, unchanged source/fixture
  bytes, an empty operation journal, file release and normal host close.
  The native branch binds the observed unique PowerPoint `childClass` directly
  beneath the owned preview pane and revalidates its HWND, PID, thread, executable
  path and start time before each posted Page Down/Up. It checks slide 1 → 2 → 1
  using stable changed/restored pixels and retains every bounded capture attempt.
  Native crops derive from the same full-window image using physical coordinates.
  Native navigation passed at `b3be3a0` with separate 18-image review; see
  `qualification/evidence/dcm-32-native-summary.json`. The earlier first-slide-only
  run remains partial historical evidence. The fallback
  uses owned combo 1025 for the same sequence. A provider caption or slide label alone
  cannot prove the visible slide content. Protected/damaged presentations remain
  DCM-34, and broad repeated-switching lifecycle remains DCM-35.
- Built-in PPTX fidelity is bounded: it renders text, tables as tab-separated
  rows, and embedded images with simplified fonts and geometry. It does not
  reproduce slide-master/layout inheritance, grouped transforms, original rich
  text/table styling, image crop/effects, charts/SmartArt, animation or media.
  It limits previews to 200 slides, 256 shapes per slide and 16K characters per
  shape. Parser and image-relationship coverage does not prove Windows image
  decoding or native PowerPoint fidelity. Record which provider was observed;
  do not count an installed-handler run as headed fallback qualification.
- `ui-xlsx`: focused DCM-31 text → workbook → text → workbook qualification,
  using the exact historical `06-orders.xlsx` fixture under
  `D:\FlowverseReportsRepo\dcm-flowuivision-20261103\inbox` (SHA-256
  `5059729f4390ecd2e169bb2d1eb850c768532ebbb24b9733e87b82b012d6d24f`).
  It lets the existing provider choose naturally and never changes associations.
  Native Excel may run in-process or out-of-process; its provider footer and
  visible subtree are observed, but actual cell readability requires separate
  screenshot review. No foreign/native Excel sheet input is assumed. The built-in
  fallback additionally verifies `Orders`/`Read me` names, actual grid cells and
  sheet selection through its owned combo (1024), notifying the preview parent.
  Both routes check text recovery, source bytes, exclusive file access after
  switching, normal host close and exact build/source identity. Raw
  `feature_accepted` remains false until independent image review is recorded.
  For the installed Excel route, `-RequireNativeBounds` checks both observed
  `EXCEL9` and `EXCEL7` viewport rectangles against the owned pane in a PMv2
  observer context (four-pixel border/rounding tolerance). The workflow preserves
  captures, text recovery and normal close before reporting a bounds failure.
  This geometry assertion does not establish actual cell readability. Other
  native providers require explicit adapter qualification; fallback cell checks
  remain separate.
  The related `xlsx-unit` selector runs focused XLSX/Open XML units. Damaged,
  protected and interrupted files remain DCM-34; broad lifecycle is DCM-35.
- `ui-xlsx-layout`: diagnostic version of the same bounded workflow. It retains
  the ordinary Restore/300 ms capture, adds a capture before restore and natural
  adjacent captures after 250/1,000/4,000 ms of accumulated explicit waits, and
  records paired DPI-context/window geometry and host layout state. Observation
  adds execution time; timestamps are authoritative. Natural captures issue no
  resize, repaint, SetRect, fence or native-provider input. The existing bounded
  host layout trace is enabled in the disposable process. A later clean frame
  is diagnostic evidence, not a production fix or automatic visual acceptance.
- `ui-filters`: text/extension/name/age/size filters, no-match state, deterministic
  Name and File Type order, and Next, with native ListBox readback. `ui-rename`
  checks Clean spacing, Rename, durable Processed history after restart, and Undo
  Rename, using original/restored byte hashes and exact journal operation identity.
  `ui-basics` combines both. Extra or missing fixture files fail the filesystem oracle.
- `ui-move`: select one exact fixture, enter the existing logical destination
  `Accepted`, move it, check source absence/destination hash/durable completed
  journal, restart the app, select the recovered Processed record and Undo. Verify
  the original bytes and durable undone state; unselected files remain unchanged.
- `ui-batch`: select exactly two filenames through the native extended-selection
  ListBox and read back the actual indexes. Verify the two-file UI label, cancel
  once and prove no file changed, then approve the explicitly scoped dialog.
  Check both files and every journal event's BatchID, restart, and Undo Batch.
- `ui-pdf`: render an owned two-page PDF, capture Fit Width, 100%, 110%, return to
  100%, page two, and return to page one. Window restoration, visibility, PID,
  geometry and nine occlusion probes guard each screen capture. Separate document
  crops must contain the fixture's teal/orange pixels, so a footer alone cannot pass.
- `ui-previews`: repeat PDF qualification and reuse the previous TXT/PNG/DOCX/XLSX/
  PPTX readiness assertions plus XLSX sheet and PPTX slide navigation. Four prior
  Office/image fixture files are copied read-only from the historical lab and
  hashed in the new receipt. Missing inputs fail explicitly. Office screenshot
  readability/fidelity still requires review and is not inferred from a visible
  control or filename.

Screenshots always begin with `visual_reviewed=false`. Pixel presence detects a
blank PDF pane; it is not an OCR/readability, font fidelity, or complete layout test.
The reviewer must inspect the actual images and record a separate decision. A
failed script stops before issuing further action. A forced app/renderer cleanup
fails lifecycle qualification and remains in the report.

## Installer qualification without touching the installed app

`installer-regressions` and `installer-unit` exercise installer logic against
disposable test directories. `installer-integration` stages the committed Windows
engine test into a copied installer module. `package` runs this integration stage
after both unit suites and also checks native builds and exact embedded payload.

The engine test uses `RegOpenCurrentUser` to obtain an independent real HKCU handle,
snapshots the five live DCM registration trees, creates one uniquely named private
qualification key, and checks
`RegOverridePredefKey(HKCU, privateRoot)` plus an ownership marker before calling any
installer action. Within this process only, it exercises fresh install, actual
ShellLink creation/readback, uninstall/App Paths/Explorer values, upgrade,
configuration backup/restore, and uninstall. Install paths, roaming configuration,
TEMP/TMP and `DCM_QUICKSORT_BACKUP_DIR` all point inside the owned evidence root.
It checks exact embedded/installed app bytes, preserves PDF/Vault preferences,
aliases, learning and naming templates, waits for the owned uninstall cleanup,
resets the override, removes its private key, and verifies live DCM registry trees
are byte-for-byte unchanged. It never launches the setup EXE under the override;
the override would not propagate to a child process.

The private root is nonvolatile because the unchanged installer creates normal
nonvolatile children, which Windows prohibits below a volatile parent. Its exact
unique path is logged before installation and removed during cleanup. Existing
keys are never reused. `RegOpenKeyEx(HKCU, NULL, ...)` is deliberately avoided
because that would return the predefined pseudo-handle rather than an independent
handle to the original root.

This engine gate is explicitly **in-process coverage**. `install()` copies the
running test executable into the uninstaller location, so it does not claim that
the packaged uninstaller executable was exercised. The package payload check is
separate. See Microsoft documentation for
[RegOverridePredefKey](https://learn.microsoft.com/en-us/windows/win32/api/winreg/nf-winreg-regoverridepredefkey)
and [COM initialization](https://learn.microsoft.com/en-us/windows/win32/api/combaseapi/nf-combaseapi-coinitializeex).

`installer-ui-cancel` launches the actual hash-bound packaged setup, checks its
version and isolated source textbox, captures its window, and activates **only
Cancel (control 1004)**. It requires a clean zero exit, absence of installed files
and shortcuts in the fixture, and unchanged host app/config hashes and DCM registry
snapshots. Set `DCM_PACKAGE_RECEIPT` to the exact successful package receipt before
invoking it through FlowDev. Its capture still needs visual review.

Full installer registration must run in a fresh Windows Sandbox, disposable VM,
or dedicated disposable Windows account with its own HKCU. Changing only APPDATA
and LOCALAPPDATA is insufficient: the installer writes fixed uninstall, App Paths
and Explorer integration keys under HKCU.

The isolated-environment sequence is:

1. Transfer only the package's exact app, setup executable, hashes and receipt.
   Rehash both executables before running anything. Record the disposable account,
   OS and initial absence of DCM install paths and HKCU integration keys.
2. Install the candidate normally or with the supported `/silent` option inside
   that environment. Verify the installed app hash exactly matches the qualified
   payload, launch it, inspect its version, and verify shortcut/App Paths/Explorer
   registrations resolve to that environment's installed location.
3. Seed a marked disposable configuration including `default_source`,
   `pdf_default_view`, `vault_root`, destinations, source aliases and naming
   templates. Save independent original hashes and contents. Exercise supported
   upgrade/reinstall and backup/restore paths; compare all expected configuration
   values and validate restored logical Vault destinations.
4. Exercise uninstall there. Verify expected executable/shortcut/integration key
   removal and the intended configuration-backup retention contract. Preserve
   before/after registry exports, file hashes, installer logs and screenshots.
5. Close the sandbox or discard the disposable account/VM. Keep the package and
   receipt unchanged. An unavailable disposable environment leaves this gate
   pending; host installation must not be substituted or described as tested.

Actual installer command-line spelling and restore switches should be checked
against the committed `installer-src/main_windows.go` for the package revision.
This final packaged-executable plan is separate from the available in-process
engine and setup-open/cancel gates. Record their results separately; never relabel
them as a packaged setup/uninstaller end-to-end installation test.
