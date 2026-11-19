# v1.0.70 source import

The app-src and installer-src baseline is copied byte-for-byte from the user-supplied DCM-QuickSort-v1.0.70-Source-and-Installer(1).zip. ARCHIVE_MANIFEST.json records every supplied file and its SHA-256. The original archive SHA-256 is 64e1cb1666acc9a70aeeb9656b4f30ca26fc30ad142ea32e154cb7a63dc7b5aa.

The supplied app and embedded installer payload have the same SHA-256, 458fbc737047fb9b783dde06724a514215f6a71db53f4d6cc24da3c7acc52ad7. The supplied installer SHA-256 is 4c021a5871bc70e58efb6492a0cbb12313300ba7faca6e7aa7404089740511fc. These are historical supplied binaries, not builds newly qualified by this import.

Compiled executables are excluded from the source checkout. The release runner must build the app and stage those exact bytes into installer-src/payload/DCM-QuickSort.exe before building/testing the Windows installer. The archive's qualification and transfer documents are preserved verbatim as supplier history; they do not close the later D1–D6 findings or replace current Windows acceptance.

The working release branch is release/v1.0.71. Changes after this baseline must retain source identity, failing regression evidence, native validation results and exact app/installer hashes. Existing v1.0.63 publication remains separate until a newer release is qualified and published.
