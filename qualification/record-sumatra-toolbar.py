"""Record source-bound qualification only. Fail rather than overwrite changed trackers."""
from pathlib import Path
import copy
import hashlib
import json
import re

ROOT = Path(__file__).resolve().parents[1]
TESTED = 'c53f85263963886b1038dbd8933dd3883481aabd'
REPORT = 'qualification/2026-11-12-sumatra-toolbar.md'
EVIDENCE = 'qualification/evidence/sumatra-toolbar-native-summary.json'
BASE = {'todo.json': 'cacc60f795bb74093072a8071c0e704d1faf50eb', 'qualification/completion-50.json': '2108ef67d72eb302f461bbed032058c6cb31fa4f'}

def load(name):
    raw = (ROOT / name).read_bytes()
    blob = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
    assert blob == BASE[name], (name, 'input changed: reconcile first', blob)
    return raw.decode('utf-8')

def line(text, key, value, indent):
    p = re.compile(r'^' + ' ' * indent + re.escape(json.dumps(key)) + r':.*$', re.M)
    matches = list(p.finditer(text))
    assert len(matches) == 1, (key, len(matches))
    tail = ',' if matches[0].group().endswith(',') else ''
    return p.sub(lambda _: ' ' * indent + json.dumps(key) + ': ' + json.dumps(value, ensure_ascii=False) + tail, text)

def replace_item(text, item):
    lines = [s for s in text.splitlines() if '"id": ' + json.dumps(item['id']) in s]
    assert len(lines) == 1, item['id']
    old = lines[0]
    return text.replace(old, '    ' + json.dumps(item, ensure_ascii=False) + (',' if old.endswith(',') else ''))

t = load('todo.json'); original = json.loads(t)
p = load('qualification/completion-50.json'); original_plan = json.loads(p)
e = json.loads((ROOT / EVIDENCE).read_text())
assert e['tested_commit'] == TESTED and len(e['runs']) == 2
assert all(r['passed'] and r['cases'] == 19 and r['exit_code'] == 0 for r in e['runs'])
assert e['baseline']['passed'] == 366 and e['baseline']['failed'] == 0
assert e['numbered_steps_closed'] == []
summary = 'c53f8526: two fresh Sumatra runs pass 19 checks each; 17 first-run full images reviewed. Original page clipping corrected (Y=-13 to Y=8); host sizing controls hide/restore. Baseline 366/0/1; text17/image23 regressions pass. New narrow-toolbar and host repaint findings remain open; no numbered closure or default switch.'
t = line(t, 'checkpoint', REPORT, 4)
t = line(t, 'last_native_attempt', summary, 4)
t = line(t, 'candidate_code_commit', TESTED, 4)
baseline = dict(original['checkpoints']['app_baseline_checkpoint'], commit=TESTED, report=e['baseline']['report'], count=367, passed=366)
old = next(s for s in t.splitlines() if s.startswith('    "app_baseline_checkpoint":'))
assert 'app_baseline_sumatra_stage1_checkpoint' not in original['checkpoints']
t = t.replace(old, '    "app_baseline_sumatra_stage1_checkpoint": ' + json.dumps(original['checkpoints']['app_baseline_checkpoint']) + ',\n    "app_baseline_checkpoint": ' + json.dumps(baseline) + ',')
checkpoint = {'commit': TESTED, 'runs': e['runs'], 'full_images_reviewed': 17, 'original_top_clipping': 'fixed_at_tested_sizes', 'review': EVIDENCE, 'open_findings': ['DCM-SUMATRA-UI-02', 'DCM-UI-RESIZE-01'], 'default_changed': False, 'numbered_steps_closed': []}
needle = '    "final_package_qualified": false,'
assert t.count(needle) == 1
t = t.replace(needle, '    "sumatra_toolbar_checkpoint": ' + json.dumps(checkpoint) + ',\n' + needle)
for issue in original['items']:
    if issue['id'] not in ('DCM-SUMATRA-01', 'DCM-SUMATRA-UI-01'):
        continue
    new = copy.deepcopy(issue)
    new['evidence_refs'] += [REPORT, EVIDENCE]
    if issue['id'] == 'DCM-SUMATRA-01':
        new['ui_test'] = '19_checks_pass_twice; top_clipping_fixed; resize_layout_and_lifecycle_open'
        new['stage1_native_test'] = issue['native_test']
        new['native_test'] = summary
    else:
        new['status'] = 'fixed'
        new['ui_test'] = 'page_field_and_host_sizing_controls_verified_at_two_sizes; full_ui_not_approved'
        new['native_test'] = 'Geometry RED 8666752a retained. At c53f8526 page field moves from -13 to 8 within 43px toolbar, page entry and accessible navigation/zoom pass. New far-right overflow and host repaint artifacts have separate open IDs. Runtime/policy hashes unchanged; no print enablement.'
    t = replace_item(t, new)
new_issues = [
    {'id': 'DCM-SUMATRA-UI-02', 'feature': 'Avoid far-right Sumatra toolbar control clipping at narrower host width', 'status': 'planned', 'ui_test': 'observed_in_reviewed_smaller_window_capture', 'user_test': 'not_recorded', 'release_gate': True, 'parent_steps': ['DCM-28', 'DCM-29', 'DCM-46'], 'acceptance': 'At supported narrower sizes, make toolbar controls available without clipping or provide a verified overflow route. Extend beyond the Edit/Static containment assertion. Retain strict runtime policy.', 'evidence_refs': [REPORT, EVIDENCE]},
    {'id': 'DCM-UI-RESIZE-01', 'feature': 'Remove stale host button and footer fragments after resize restoration', 'status': 'planned', 'ui_test': 'observed_in_reviewed_resize_restored_and_later_captures', 'user_test': 'not_recorded', 'release_gate': True, 'parent_steps': ['DCM-46'], 'acceptance': 'Reproduce resize repaint trails with a focused assertion; restore clean host background and controls after shrinking/enlarging without masking raw capture evidence. Exact cause is not yet established.', 'evidence_refs': [REPORT, EVIDENCE]}
]
assert not any(x['id'] in [i['id'] for i in original['items']] for x in new_issues)
needle = '\n  ]\n}'
assert t.count(needle) == 1
t = t.replace(needle, ',\n' + ',\n'.join('    ' + json.dumps(x) for x in new_issues) + needle)
for key, value in [('checkpoint', REPORT), ('last_tested_source', TESTED)]:
    p = line(p, key, value, 2)
for step in original_plan['steps']:
    if step['id'] not in ('DCM-28', 'DCM-29'):
        continue
    new = copy.deepcopy(step)
    new.update(status='implemented', completion='pending', ui_test='optional_Sumatra_navigation_zoom_pass; resize_layout_and_provider_acceptance_open')
    new['evidence'] += [REPORT, EVIDENCE]
    if 'native_test' in step:
        new['stage1_native_test'] = step['native_test']
    new['native_test'] = summary
    p = replace_item(p, new)
after = json.loads(t); after_plan = json.loads(p)
assert after['completion_plan']['completed_product_steps'] == original['completion_plan']['completed_product_steps']
assert after['completion_plan']['blocked_steps'] == original['completion_plan']['blocked_steps']
for old, new in zip(original['items'], after['items']):
    if old['id'] not in ('DCM-SUMATRA-01', 'DCM-SUMATRA-UI-01'):
        assert old == new
assert len(after_plan['steps']) == 50
assert sum(s['completion'] == 'complete' for s in after_plan['steps']) == 26
for old, new in zip(original_plan['steps'], after_plan['steps']):
    if old['id'] not in ('DCM-28', 'DCM-29'):
        assert old == new
assert after['checkpoints']['final_package_qualified'] is False
assert after['checkpoints']['user_acceptance'] == 'not_recorded'
for name, text in [('todo.json', t), ('qualification/completion-50.json', p)]:
    (ROOT / name).write_text(text, encoding='utf-8')
    print('OUTPUT', name, hashlib.sha256(text.encode()).hexdigest())
