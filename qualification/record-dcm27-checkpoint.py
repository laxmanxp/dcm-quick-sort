"""Guarded metadata-only DCM-27 closeout; never edits product/tests or raw receipts."""
from pathlib import Path
import copy, hashlib, json, re

ROOT = Path(__file__).resolve().parents[1]
TESTED = 'b92c9abdac87e7f9c937e51ca8e7023eae5e3417'
PRODUCT = '4043f3e564d5c22dbac8a318f4c7fc07de970a6f'
REPORT = 'qualification/dcm27-complete.md'
EVIDENCE = 'qualification/evidence/dcm-27-native-summary.json'
BASE = {'todo.json': '0cfa81b5d07da008ec5ee4995ae106dbe0b34b6e',
        'qualification/completion-50.json': '7f379c7b4b0e8ffbf90066947a8f69aea2a661df'}

def read(name):
    raw = (ROOT / name).read_bytes()
    sha = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
    assert sha == BASE[name], (name, sha, 'input changed; refusing')
    return raw.decode('utf-8')

def set_line(text, key, value, spaces):
    pattern = re.compile(r'^' + ' ' * spaces + re.escape(json.dumps(key)) + r': .*?,?$', re.M)
    found = list(pattern.finditer(text))
    assert len(found) == 1, (key, len(found))
    comma = ',' if found[0].group().endswith(',') else ''
    return pattern.sub(lambda _: ' '*spaces + json.dumps(key) + ': ' + json.dumps(value, ensure_ascii=False) + comma, text)

summary = json.loads((ROOT/EVIDENCE).read_text())
assert summary['tested_commit'] == TESTED
assert summary['ui']['passed'] and summary['ui']['case_count'] == 23
assert len(summary['visual_review']['reviewed_images']) == 22
assert all(x['reviewed'] for x in summary['visual_review']['reviewed_images'])
assert summary['baseline']['passed'] == 362 and summary['baseline']['failed'] == 0

plan_text = read('qualification/completion-50.json')
plan_before = json.loads(plan_text)
for key, val in [('checkpoint', REPORT), ('last_tested_source', TESTED), ('next_step', 'DCM-28'), ('next_runnable_step', 'DCM-28')]:
    plan_text = set_line(plan_text, key, val, 2)
step = copy.deepcopy(next(x for x in plan_before['steps'] if x['id'] == 'DCM-27'))
assert step['completion'] == 'pending'
step.update(status='fixed', completion='complete', ui_test='passed_with_separate_image_review', evidence=[REPORT, EVIDENCE],
    native_test='b92c9abd: fresh image workflow passed 23 checks on 12 fixtures; all 22 screenshots inspected. Nineteen render checks compare actual pixels/geometry; empty/corrupt/locked errors, recovery and source-handle release verified. Full native baseline 362 passed / 0 failed / 1 external-PPTX skip; text/navigation commands pass. Existing fix 4043f3e5 was recovered and verified, not reapplied. No further product/test changes in retry.',
    scope_note='Static GIF and single-page TIFF; no animation/multipage TIFF, all EXIF/profile/DPI variants or physical-input claim. Pan uses owned native messages. Human acceptance and final packaging remain separate.')
old = [l for l in plan_text.splitlines() if '"id": "DCM-27"' in l]
assert len(old) == 1
plan_text = plan_text.replace(old[0], '    '+json.dumps(step, ensure_ascii=False)+',')
plan_after = json.loads(plan_text)
assert len(plan_after['steps']) == 50
assert sum(x['completion']=='complete' for x in plan_after['steps']) == 26
assert step['user_test'] == 'not_recorded'
for a,b in zip(plan_before['steps'], plan_after['steps']):
    if a['id'] != 'DCM-27': assert a == b
assert next(x for x in plan_after['steps'] if x['id']=='DCM-19')['completion']=='pending'

todo_text = read('todo.json')
todo_before = json.loads(todo_text)
products = todo_before['completion_plan']['completed_product_steps'] + ['DCM-27']
for key,val in [('completed_product_steps', products), ('next_step','DCM-28'), ('next_runnable_step','DCM-28'), ('checkpoint',REPORT), ('candidate_code_commit',PRODUCT), ('last_native_attempt','b92c9abd: image acceptance 23 checks PASS, 22 individually reviewed screenshots; all captures first-attempt. Full baseline 362/0/1, text/navigation commands PASS. Canonical D: workspace resolves old C: junction identity failure. Existing image-error fix 4043f3e5 verified without further product/test changes. Only DCM-27 newly closed; DCM-19 stays blocked.')]:
    todo_text = set_line(todo_text, key, val, 4)
old_baseline = todo_before['checkpoints']['app_baseline_checkpoint']
new_baseline = dict(old_baseline, commit=TESTED, report=summary['baseline']['report'], count=363, passed=362)
old_line = next(l for l in todo_text.splitlines() if l.startswith('    "app_baseline_checkpoint":'))
assert 'app_baseline_dcm26_checkpoint' not in todo_before['checkpoints']
todo_text = todo_text.replace(old_line, '    "app_baseline_dcm26_checkpoint": '+json.dumps(old_baseline)+',\n    "app_baseline_checkpoint": '+json.dumps(new_baseline)+',')
selection = dict(summary['ui'], commit=TESTED, reviewed_images=22, review_evidence=EVIDENCE, build_receipt=summary['build']['receipt'], app_sha256=summary['build']['app_sha256'])
needle = '    "final_package_qualified": false,'
assert todo_text.count(needle)==1
todo_text = todo_text.replace(needle, '    "ui_image_checkpoint": '+json.dumps(selection)+',\n'+needle)
needle = '    "dcm26_completion": "qualification/dcm26-complete.md"'
assert todo_text.count(needle)==1
todo_text = todo_text.replace(needle, needle+',\n    "dcm27_completion": '+json.dumps(REPORT))
issue = {'id':'DCM-27-IMAGE-ERROR-01', 'feature':'Report image read/decode failures without misleading handler or stale success states', 'status':'fixed', 'severity':'medium', 'finding_kind':'native_image_error_state', 'ui_test':'passed_with_separate_image_review', 'user_test':'not_recorded', 'release_gate':True, 'parent_steps':['DCM-27'], 'acceptance':'Empty, corrupt and locked supported images retain specific errors, clear old content and disable sizing; valid reselection recovers without changing source bytes or holding the source file open.', 'native_test':'Retained RED at 2dc099d5 and inspected screenshot show empty PNG described as missing handler with stale fit footer. Product fix 4043f3e5; fresh full image and baseline GREEN at b92c9abd. This retry changes checkpoint metadata only. Wider format, physical-input and package gates remain separate.', 'evidence_refs':['dcm27_completion']}
assert issue['id'] not in {x['id'] for x in todo_before['items']}
assert todo_text.count('\n  ]\n}')==1
todo_text = todo_text.replace('\n  ]\n}', ',\n    '+json.dumps(issue)+'\n  ]\n}')
todo_after = json.loads(todo_text)
assert todo_after['items'][:-1] == todo_before['items']
assert todo_after['completion_plan']['blocked_steps'] == todo_before['completion_plan']['blocked_steps']
assert todo_after['checkpoints']['user_acceptance']=='not_recorded'
assert todo_after['checkpoints']['final_package_qualified'] is False
for name,text in [('todo.json',todo_text),('qualification/completion-50.json',plan_text)]:
    (ROOT/name).write_text(text,encoding='utf-8')
    print(name,hashlib.sha256(text.encode()).hexdigest())
