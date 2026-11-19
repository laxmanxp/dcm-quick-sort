"""Apply only reviewed DCM-24 tracker changes; reject a changed source version."""
from pathlib import Path
import copy, hashlib, json, re

ROOT = Path(__file__).resolve().parents[1]
TESTED = 'aa6dceffa1485a1867e3748c9dbe8d1ec9ade359'
REPORT = 'qualification/dcm24-complete.md'
EVIDENCE = 'qualification/evidence/dcm-24-native-summary.json'
EXPECTED = {'todo.json': '0bd978c11f5c6818c1d7e248f125479e8bf1125c',
            'qualification/completion-50.json': '5cedc542eb1b2619e920572d9a2ee1564237d2eb'}

def read(name):
    raw = (ROOT/name).read_bytes()
    actual = hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest()
    assert actual == EXPECTED[name], (name, actual, 'base changed')
    return raw.decode('utf-8')

def set_line(text, key, value, spaces):
    pattern = re.compile(r'^'+(' '*spaces)+re.escape(json.dumps(key))+r': .*?,?$', re.M)
    found = list(pattern.finditer(text)); assert len(found) == 1, (key, len(found))
    old = found[0].group()
    return pattern.sub(lambda _: ' '*spaces+json.dumps(key)+': '+json.dumps(value, ensure_ascii=False)+(',' if old.endswith(',') else ''), text)

plan_text = read('qualification/completion-50.json')
plan_before = json.loads(plan_text)
for key, value in [('checkpoint', REPORT), ('last_tested_source', TESTED), ('next_step', 'DCM-25'), ('next_runnable_step', 'DCM-25')]:
    plan_text = set_line(plan_text, key, value, 2)
step = copy.deepcopy(next(x for x in plan_before['steps'] if x['id'] == 'DCM-24'))
step.update(status='fixed', completion='complete', ui_test='passed_with_separate_image_review',
            evidence=[REPORT, EVIDENCE],
            native_test='Native Ctrl-deselection RED at f208e3ad: Charlie remained selected while Alpha was the draft/preview. Fix aa6dceff: 16 recorded ui-selection checks passed (13 behavior assertions and 3 FlowUIVision setup/filter results), seven screenshots reviewed, exact two-file Move and Cancel verified, six fixture contents preserved, normal exit 0. Full native baseline 358 passed / 0 failed / 1 external-PPTX skip. Navigation, sorting, launch and filter commands pass at the same revision.',
            scope_note='Ctrl/Shift selection uses guarded SendInput mouse/modifier events, not direct LB_SETSEL. Actual user acceptance and full Batch Move/Undo remain separate; no claim that every file operation or physical human-input route was tested.')
old_lines = [line for line in plan_text.splitlines() if '"id": "DCM-24"' in line]
assert len(old_lines) == 1
plan_text = plan_text.replace(old_lines[0], '    '+json.dumps(step, ensure_ascii=False)+',')
plan_after = json.loads(plan_text)
assert len(plan_after['steps']) == 50
for before, after in zip(plan_before['steps'], plan_after['steps']):
    if before['id'] != 'DCM-24': assert before == after
assert next(x for x in plan_after['steps'] if x['id'] == 'DCM-19')['completion'] == 'pending'
assert step['user_test'] == 'not_recorded'

todo_text = read('todo.json'); todo_before = json.loads(todo_text)
products = todo_before['completion_plan']['completed_product_steps'] + ['DCM-24']
for key, value in [('completed_product_steps', products), ('next_step', 'DCM-25'), ('next_runnable_step', 'DCM-25'), ('checkpoint', REPORT), ('candidate_code_commit', TESTED), ('last_native_attempt', 'aa6dceff: ui-selection PASS, 16 recorded checks and seven reviewed screenshots; Ctrl deselection preview mismatch fixed, Cancel preserves the set, exactly alpha/delta move. Full baseline 358/0/1; navigation/sort/launch/filter commands PASS. One sort capture guard failure is retained; unchanged retry passed. Only DCM-24 newly closed; DCM-19 stays blocked.')]:
    todo_text = set_line(todo_text, key, value, 4)
old_baseline = todo_before['checkpoints']['app_baseline_checkpoint']
old_line = next(line for line in todo_text.splitlines() if line.startswith('    "app_baseline_checkpoint":'))
new_baseline = dict(old_baseline, commit=TESTED, report='20261111-124126-52bc061f', count=359, passed=358)
todo_text = todo_text.replace(old_line, '    "app_baseline_dcm23_checkpoint": '+json.dumps(old_baseline)+',\n    "app_baseline_checkpoint": '+json.dumps(new_baseline)+',')
selection = {'commit': TESTED, 'report': r'C:\FlowverseReports\dcm-quicksort-ui-selection-20261111-124301-63c57d1b', 'passed': True, 'case_count': 16, 'behavior_cases': 13, 'vision_setup_filter_cases': 3, 'reviewed_images': 7, 'app_exit_code': 0, 'source_unchanged': True, 'children_drained': True, 'moved': ['01-alpha.txt','04-delta.txt'], 'unchanged_contents_count': 6, 'review_evidence': EVIDENCE}
needle = '    "final_package_qualified": false,'
assert todo_text.count(needle) == 1
todo_text = todo_text.replace(needle, '    "ui_selection_checkpoint": '+json.dumps(selection)+',\n'+needle)
needle = '    "dcm23_checkpoint": "qualification/2026-11-11-dcm23.md"'
assert todo_text.count(needle) == 1
todo_text = todo_text.replace(needle, needle+',\n    "dcm24_completion": '+json.dumps(REPORT))
issue = {'id': 'DCM-24-SELECTION-01', 'feature': 'Keep Ctrl-deselection preview and operation targets aligned with the selected files', 'status': 'fixed', 'severity': 'high', 'finding_kind': 'native_selection_preview_mismatch', 'ui_test': 'passed_with_separate_image_review', 'user_test': 'not_recorded', 'release_gate': True, 'parent_steps': ['DCM-24'], 'acceptance': 'Actual selected indices, not an unselected keyboard caret, determine the current preview/draft and Move set. Preserve Ctrl/Shift anchors, allow zero selected files, and verify exact operation targets with disposable fixtures.', 'native_test': 'RED f208e3ad screenshot and index assertion showed Charlie selected with Alpha draft/preview. aa6dceff passes the unchanged SendInput-based suite, zero/one/range cases, Cancel and exact two-file Move. Original wrong-file movement was not attempted or claimed reproduced. Complete native baseline 358/0/1. Wider Move/Undo and final-package gates remain open.', 'evidence_refs': ['dcm24_completion']}
assert '\n  ]\n}' in todo_text
assert issue['id'] not in {x['id'] for x in todo_before['items']}
todo_text = todo_text.replace('\n  ]\n}', ',\n    '+json.dumps(issue)+'\n  ]\n}')
todo_after = json.loads(todo_text)
assert todo_after['items'][:-1] == todo_before['items']
assert todo_after['completion_plan']['blocked_steps'] == todo_before['completion_plan']['blocked_steps']
assert todo_after['checkpoints']['final_package_qualified'] is False
assert todo_after['checkpoints']['user_acceptance'] == 'not_recorded'
for name, text in [('todo.json', todo_text), ('qualification/completion-50.json', plan_text)]:
    (ROOT/name).write_text(text, encoding='utf-8')
    print(name, hashlib.sha256(text.encode()).hexdigest())
