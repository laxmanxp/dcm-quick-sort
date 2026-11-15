"""One-shot, exact-source documentation closeout. Never changes application code.

Developed/tested in OpenAI; the GitHub runner applies it to the pinned checkout.
Raw Lenovo evidence is immutable; all acceptance annotations are separate.
"""
from __future__ import annotations
import copy
import hashlib
import json
from pathlib import Path

SOURCE = 'bfbb731dc131e2ba691c432555aff2641f44af8a'
RED = '12655696e591f5bc0ea0b9ca4ff2544855c7145f'
REPORT_COMMIT = 'f0caef5efb0f4f3ac370b412577d38ab94aa3fdb'
REPORT_PREFIX = 'apps/dcm-quicksort/dcm34-20261114-232225/docx-fallback-bfbb731'
SUMMARY = 'qualification/evidence/dcm-34-docx-fallback-native-summary.json'
GATE_ID = 'FILEFLOW-SUITE-READY-01'
MARKER = '## 15 November 2026 - DOCX closeout and suite release decision'
FILES = ('CONTEXT.md', 'todo.json', 'qualification/completion-50.json', 'qualification/dcm34-preview-failures.md')
EXPECTED_SHA256 = {
    'CONTEXT.md': 'dbc60d027c9f4cca66cb6e899954604e9cad99d95424ce3943beb850dde5131d',
    'todo.json': '89439f26492f58c5e51c7ca3ec3bfd4d937d654557d8c004bc7c333cbcee0ac2',
    'qualification/completion-50.json': '16c32fb9cf7421a179b459bca8ba4b6f6ea31b25415f1b578a97e41d75216955',
}
EXPECTED_BLOB = '00d08ae01663cc4ae2977d7bd10e8d7c292d5577'
GATE = {
    'id': GATE_ID, 'decision_date': '2026-11-15',
    'suite': 'FileFlow Suite',
    'members': ['DCM QuickSort', 'Flow Search', 'DeepSearch', 'FileFlow'],
    'requires_ready': ['Flow Search', 'DeepSearch'],
    'readiness': 'not_verified_in_this_DCM_step',
    'final_packaging_and_installation': 'deferred_until_required_suite_components_complete_and_accepted',
    'blocks_component_development_or_focused_tests': False,
    'rule': 'Keep independent applications/windows. Continue QuickSort source-bound builds and focused tests. Final suite packaging, installer acceptance and publication wait for Flow Search and DeepSearch readiness. No installer topology or codebase merger is implied; publication still requires explicit approval.'
}
NOTE = ('DCM-34-DOCX-FALLBACK-01 is fixed at bfbb731 within the built-in fallback contract: '
        'native RED preserved; 5 focused and 23 existing DOCX/Open XML tests pass; baseline 396/0/0; '
        '16 headed component states and 7 normal-app DOCX records pass with all 21 captures separately reviewed. '
        'The missing/failed-handler inputs are owned component boundary inputs, not separate absent/failing Word environments. '
        'Raw visual/acceptance flags remain unchanged. Existing PDF milestone remains separately attributed. '
        'Full DCM-34 is PARTIAL, 32 complete / 18 remaining; protected PDF, in-flight interruption, PDF error-control readiness and broader Office failure coverage remain. '
        'User acceptance and release remain open.')


def one(items: list, ident: str) -> dict:
    found = [x for x in items if x.get('id') == ident]
    if len(found) != 1:
        raise ValueError(f'Expected exactly one {ident}, found {len(found)}')
    return found[0]


def dump(data: object) -> str:
    return json.dumps(data, ensure_ascii=False, indent=2) + '\n'


def splice_id(text: str, ident: str, replacement: dict) -> str:
    """Replace only one JSON object; preserve every byte outside its span."""
    decoder = json.JSONDecoder()
    candidates = []
    for i, char in enumerate(text):
        if char != '{':
            continue
        try:
            obj, length = decoder.raw_decode(text[i:])
        except json.JSONDecodeError:
            continue
        if isinstance(obj, dict) and obj.get('id') == ident:
            candidates.append((i, i + length))
    if len(candidates) != 1:
        raise ValueError(f'Cannot uniquely splice {ident}')
    start, end = candidates[0]
    return text[:start] + json.dumps(replacement, ensure_ascii=False) + text[end:]


def add_root(text: str, name: str, value: object) -> str:
    original = json.loads(text)
    if name in original or not text.lstrip().startswith('{'):
        raise ValueError(f'Duplicate or invalid root field {name}')
    pos = text.index('{') + 1
    out = text[:pos] + '\n  ' + json.dumps(name) + ': ' + json.dumps(value, ensure_ascii=False) + ',' + text[pos:]
    check = json.loads(out)
    del check[name]
    if check != original:
        raise ValueError('Root insertion changed existing data')
    return out


def make_summary() -> dict:
    stages = [
        ('01-empty-pane', 'Empty preview; no stale document; sizing controls disabled.'),
        ('02-malformed-after-empty', 'Explicit unavailable body and correct missing/failed-handler footer; controls disabled.'),
        ('03-valid-blank', 'Legitimate zero-paragraph DOCX accepted; zero counts and ready footer; generic sizing controls enabled, Width disabled.'),
        ('04-valid-body', 'Recovery body and Bridge count 7 readable; one-paragraph footer.'),
        ('05-malformed-after-success', 'Explicit unavailable state replaces the prior success body and footer.'),
        ('06-valid-recovery', 'Correct body and one-paragraph ready footer restored.'),
        ('07-missing-body-after-success', 'Missing-body DOCX unavailable; no stale successful content.'),
        ('08-valid-second-recovery', 'Correct body and ready footer restored again.'),
    ]
    images = []
    for route in ('missing-handler-boundary', 'failed-handler-boundary'):
        for name, observation in stages:
            images.append({'path': f'native/regressions-20261114-232504-ac6f1c6e/docx-component/{route}/{name}.png',
                           'observation': observation, 'review': 'pass_bounded_scope'})
    native = [
        ('01-initial-text', 'All three TXT fixture lines readable; no DOCX remnants.'),
        ('02-native-letter', 'Gurundia Block letter, Road Repair, amount 25,000 with rupee symbol and signature readable.'),
        ('04-native-structured', 'Field-report heading, 31 units, mixed formatting, Cafe with accent, table Bolts 12/Panels 3, green image and end marker readable.'),
        ('08-text-recovery', 'Original three TXT lines restored; no document overlay.'),
        ('10-native-letter-reopened', 'Same correct letter content readable after reopening.'),
    ]
    images += [{'path': f'native/ui-docx-20261114-232542-a73d445b/{name}.png',
                'observation': obs, 'review': 'pass_bounded_scope'} for name, obs in native]
    return {
        'schema': 1, 'feature': 'DCM-34-DOCX-FALLBACK-01', 'status': 'fixed_with_bounded_acceptance',
        'recorded_on': '2026-11-15', 'tested_source': SOURCE,
        'parent_step_status': 'PARTIAL', 'numbered_plan': {'complete': 32, 'remaining': 18},
        'new_native_run_claimed': False, 'user_test': 'not_recorded',
        'red': {'source': RED, 'top_level': 4, 'passed_top_level': 2, 'failed_top_level': 2,
                'failed_tests': ['TestDCM34DOCXDamageStates', 'TestDCM34DOCXDocumentStructure'], 'skipped': 0, 'exit_code': 1},
        'native_gates': {
            'update': {'passed': True, 'received_commit': SOURCE},
            'docx-failure-unit': {'passed_top_level': 5, 'failed': 0, 'skipped': 0},
            'docx-unit': {'passed_top_level': 23, 'failed': 0, 'skipped': 0},
            'ui-tooling': {'passed': True},
            'baseline': {'passed': 396, 'failed': 0, 'skipped': 0, 'real_pptx_one_run_one_pass': True},
            'build': {'passed': True, 'source_files': 478, 'app_size': 7372288,
                      'app_sha256': 'e25bb96009872a8cd0a09f6bbac886f814d9ac68a4a5207895cbb4c21a69a084'},
            'docx-fallback-component': {'passed_top_level': 1, 'headed_states': 16, 'captures': 16,
                'scope': 'Owned Windows component invokes production loadOfficeFallbackPreview and previewWindowProc with explicit missing/failed boundary inputs; no Office activation or registry changes.'},
            'ui-docx': {'passed_records': 7, 'captures': 5, 'scope': 'Representative normal-app installed Word success, TXT recovery and reopening regression.'},
        },
        'normal_app_missing_or_failed_word_environment_qualified': False,
        'separate_visual_review': {'reviewed_on': '2026-11-15', 'images_reviewed': 21,
            'method': 'Each of the 21 original PNGs read and visually inspected separately through Desktop Commander.',
            'raw_flags_modified': False, 'images': images},
        'preservation': {'source_unchanged': True, 'fixtures_preserved': True, 'normal_app_journal_empty': True,
                         'component_exclusive_release_passed': True, 'normal_app_release_and_close_passed': True},
        'publication': {'repository': 'laxmanxp/flowverse-reports', 'commit': REPORT_COMMIT, 'prefix': REPORT_PREFIX,
            'raw_files': 213, 'publication_files': 217,
            'archive_sha256': '1cae8ad7f5662be2e26f1754c2aebd0ecc0520705738e67599557364bc5da31a',
            'identity_checks': '213 archive entries, source originals and extracted files matched the existing SHA-256 manifest; all 217 staged report blobs matched exact unfiltered bytes.',
            'limit': 'The earlier PDF documentation-sync subset has no execution.json; its available raw files are preserved. No recovered prior 305-check audit is claimed.'},
        'remaining': ['Default PDF protected-file handling', 'Observed in-flight interruption and late-result rejection',
                      'Generic sizing readiness on native PDF error', 'Broader Office failure/protection scope',
                      'DCM-35 lifecycle', 'DCM-46 final workflow', 'Suite packaging/installer readiness', 'Actual user acceptance'],
    }


def transform(docs: dict[str, str]) -> dict[str, str]:
    if set(docs) != set(FILES) or any(not value for value in docs.values()):
        raise ValueError('Expected all four nonempty canonical documents')
    if MARKER in docs['CONTEXT.md']:
        raise ValueError('Already applied; do not rewrite acceptance history')
    todo = json.loads(docs['todo.json'])
    plan = json.loads(docs['qualification/completion-50.json'])
    if len(plan['steps']) != 50 or sum(s['completion'] == 'complete' for s in plan['steps']) != 32:
        raise ValueError('Numbered plan moved; reconcile instead of overwriting')
    item = copy.deepcopy(one(todo['items'], 'DCM-34-DOCX-FALLBACK-01'))
    if item['status'] != 'planned' or one(plan['steps'], 'DCM-34')['completion'] != 'in_progress':
        raise ValueError('Unexpected current acceptance state')
    item.update(status='fixed', ui_test='bounded_owned_component_and_normal_DOCX_regression_pass_with_separate_21_image_review',
                user_test='not_recorded', tested_source=SOURCE, current_evidence=NOTE, evidence_summary=SUMMARY,
                normal_app_missing_or_failed_word_environment_qualified=False)
    release = copy.deepcopy(one(todo['items'], 'DCM-RELEASE-01'))
    release['blocked_by'] = list(dict.fromkeys(release.get('blocked_by', []) + [GATE_ID]))
    out = dict(docs)
    out['todo.json'] = splice_id(out['todo.json'], item['id'], item)
    out['todo.json'] = splice_id(out['todo.json'], release['id'], release)
    out['todo.json'] = add_root(out['todo.json'], 'suite_release_gate', GATE)
    out['todo.json'] = add_root(out['todo.json'], 'dcm34_docx_closeout', {'summary': SUMMARY, 'tested_source': SOURCE, 'status': NOTE})
    for step in plan['steps']:
        if step['id'] == 'DCM-34':
            updated = copy.deepcopy(step)
            updated['prior_pdf_milestone_note'] = updated.get('qualification_note', '')
            updated['qualification_note'] = NOTE
            updated['docx_milestone'] = {'source': SOURCE, 'status': 'fixed_bounded_fallback_contract', 'evidence': SUMMARY}
            updated['evidence'] = list(dict.fromkeys(updated.get('evidence', []) + [SUMMARY]))
        elif int(step['id'][-2:]) >= 47:
            updated = copy.deepcopy(step)
            updated['suite_release_gate'] = GATE_ID
            updated['blocked_by'] = list(dict.fromkeys(updated.get('blocked_by', []) + [GATE_ID]))
        else:
            continue
        out['qualification/completion-50.json'] = splice_id(out['qualification/completion-50.json'], step['id'], updated)
    out['qualification/completion-50.json'] = add_root(out['qualification/completion-50.json'], 'suite_release_gate', GATE)
    report_url = f'https://github.com/laxmanxp/flowverse-reports/tree/{REPORT_COMMIT}/{REPORT_PREFIX}'
    text = (f'\n{MARKER}\n\n{NOTE}\n\n'
            f'Original RED/GREEN evidence: [{REPORT_COMMIT}]({report_url}). '
            f'The separate acceptance and 21-image review are in [{SUMMARY}](https://github.com/laxmanxp/dcm-quick-sort/blob/release/v1.0.71/{SUMMARY}). '
            'This records existing tests; it does not claim a new application/UI run. '
            'Earlier DOCX open/unfixed statements below describe the pre-fix PDF/probe checkpoint and are superseded here.\n\n'
            '**Suite decision:** FileFlow Suite includes DCM QuickSort, Flow Search, DeepSearch and FileFlow as separate applications/windows. '
            'Final suite packaging and installation are deferred until Flow Search and DeepSearch are complete and accepted for the intended scope. '
            'Their readiness was not checked in this DCM step. QuickSort development, source-bound builds and focused tests continue independently. '
            'No UI/codebase merger or installer topology is assumed. Publication still requires explicit approval.\n\n'
            '**Next QuickSort work:** continue the remaining DCM-34 preview cases; do not recreate the already-pushed DOCX fix or restart completed source recovery.\n')
    for name in ('CONTEXT.md', 'qualification/dcm34-preview-failures.md'):
        first, rest = docs[name].split('\n', 1)
        out[name] = first + '\n' + text + rest
    out['qualification/dcm34-preview-failures.md'] = out['qualification/dcm34-preview-failures.md'].replace(
        '## Separate confirmed DOCX fallback defect', '## Historical DOCX fallback probe - superseded by accepted fix above')
    out[SUMMARY] = dump(make_summary())
    # Scope invariants: no existing records other than the stated target objects change.
    new_todo = json.loads(out['todo.json'])
    for before, after in zip(todo['items'], new_todo['items'], strict=True):
        if before['id'] not in (item['id'], release['id']) and before != after:
            raise ValueError('Unrelated todo item changed')
    for key in todo:
        if key != 'items' and todo[key] != new_todo[key]:
            raise ValueError(f'Historical todo field changed: {key}')
    new_plan = json.loads(out['qualification/completion-50.json'])
    if [s['completion'] for s in new_plan['steps']] != [s['completion'] for s in plan['steps']]:
        raise ValueError('Numbered completion changed')
    return out


def apply(root: Path) -> None:
    docs = {}
    for name in FILES:
        raw = (root / name).read_bytes()
        if name in EXPECTED_SHA256:
            valid = hashlib.sha256(raw).hexdigest() == EXPECTED_SHA256[name]
        else:
            blob = b'blob ' + str(len(raw)).encode() + b'\0' + raw
            valid = hashlib.sha1(blob).hexdigest() == EXPECTED_BLOB
        if not valid:
            raise ValueError(f'Exact input identity mismatch: {name}; no writes performed')
        docs[name] = raw.decode('utf-8')
    if (root / SUMMARY).exists():
        raise ValueError('Summary already exists; no writes performed')
    output = transform(docs)  # validate every transformation before the first write
    for name, text in output.items():
        path = root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(text.encode('utf-8'))
    print(f'PASS: {len(output)} documentation files; 32/50 unchanged; DCM-34 partial; no app/installer changes')


if __name__ == '__main__':
    apply(Path(__file__).resolve().parents[1])
