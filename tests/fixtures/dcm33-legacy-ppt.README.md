# DCM-33 legacy PowerPoint fixture

`dcm33-legacy-ppt.ppt` is a genuine PowerPoint 97–2003 binary presentation.
It was exported from the existing pinned `dcm32-slides.pptx` without editing
the source, content or layout. The `DCM-32` footers intentionally remain as
part of that source. All names and amounts are synthetic.

The two slides provide distinct content for native preview qualification:

1. **Bridge Inspection** and **September 2026**, with the embedded bridge image.
2. **Payments**, with Scheme/Amount columns, Bridge/125000 and Road/86000 rows.

The manifest pins the binary's SHA-256 and size, the ordered slide text,
the source PPTX identity, and the original PNG identity. The PowerPoint binary
contains the original PNG bytes unchanged. It has `PowerPoint Document`,
`Current User` and `Pictures` streams, one user edit and two live presentation
slides resolved through their persist-directory references.

## Local development commands

Generate into a new directory using the local LibreOffice runtime:

```sh
python tests/fixtures/dcm33-legacy-ppt.generate.py --output-dir <new-output-directory>
```

The generator uses the `MS PowerPoint 97` export filter and an isolated profile.
It refuses to overwrite an existing output fixture or generation record. The
committed source and fixture manifest are unchanged by generation.

Inspect the committed binary independently of the exporter and application:

```sh
python tests/fixtures/dcm33-legacy-ppt.validate.py
```

The validator requires `xlrd==2.0.1` for CFBF stream access. It follows the live
document and ordered slide references, then reads text atoms only within those
slides. It compares every text block and the embedded PNG hash. Its supported
scope is this small, unencrypted, single-edit fixture; it is not an application
PowerPoint parser. The default command also requires the exact pinned binary
hash and size.

To compare a fresh export's content without asserting its binary identity:

```sh
python tests/fixtures/dcm33-legacy-ppt.validate.py --file <new-output-directory>/dcm33-legacy-ppt.ppt --semantic-only
```

`--semantic-only` explicitly reports `pinned_identity_checked=false`. It is
for regeneration and negative probes, not for substituting an unpinned file
into the native acceptance run. Binary metadata may vary across export runs
or runtime versions; byte-identical regeneration is not a general guarantee.

The initial validation rejects three controlled negative inputs at the
intended content/structure guards: renamed PPTX bytes, a changed payment amount
with unchanged file size, and swapped slide references. Local binary-to-PDF
rendering and review cover both slides. Native Windows rendering, navigation,
unavailable-provider behavior, recovery and file release require their own
registered FlowDev qualification and separate image review.

## Format references

The validator uses Microsoft's primary format specification:

- [PowerPoint Document Stream and live object lookup](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/1fc22d56-28f9-4818-bd45-67c2bf721ccf)
- [RecordHeader](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/df201194-0cd0-4dfb-bf10-eea353d8eabc)
- [CurrentUserAtom](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/940d5700-e4d7-4fc0-ab48-fed5dbc48bc1)
- [UserEditAtom](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/3ffb3fab-95de-4873-98aa-d508fbbac981)
- [PersistDirectoryEntry](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/6214b5a6-7ca2-4a86-8a0e-5fd3d3eff1c9)
- [SlideListWithTextContainer](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/307e6d12-7304-47a8-acbd-3e7b8041ad3c)
- [TextCharsAtom](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/a3c5c8d5-e530-4167-a242-7743bc99aeac)
- [TextBytesAtom](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-ppt/80aae34b-2699-43fa-9e6a-c560ae790cd7)
