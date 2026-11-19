"""Record only the reviewed DCM-25 boundary, preserving other steps and history."""
from pathlib import Path
import copy
import hashlib
import json
import re

ROOT = Path(__file__).resolve().parents[1]
EXPECTED = {
    'todo.json': 'f2fb38f0b99ea7d81759367f772944b227c5449a',
    'qualification/completion-50.json': 'fe94451a46e9f5246bc06e8e31d7060e0d599bea',
}
REPORT = 'qualification/dcm25-complete.md'
EVIDENCE = 'qualification/evidence/dcm-25-native-summary.json'


def set_line(text, key, value, spaces):
    pattern = re.compile(r'^' + ' ' * spaces + re.escape(json.dumps(key)) + r': .*?,?$', re.M)
    found = list(pattern.finditer(text))
    assert len(found) == 1, (key, len(found))
    comma = ',' if found[0].group().endswith(',') else ''
    return pattern.sub(lambda _: ' ' * spaces + json.dumps(key) + ': ' + json.dumps(value, ensure_ascii=False) + comma, text)


def transform(plan_text, todo_text, evidence):
    assert evidence['feature_accepted'] is True
    assert evidence['native_ui']['passed'] is True
    assert evidence['reviewed_images'] == len(evidence['images']) > 0
    tested = evidence['tested_commit']
    before_plan, before_todo = json.loads(plan_text), json.loads(todo_text)
    assert before_plan['next_step'] == 'DCM-25'
    assert next(x for x in before_plan['steps'] if x['id'] == 'DCM-24')['completion'] == 'complete'
    note = evidence['qualification_summary']
    for key, value in [('checkpoint', REPORT), ('last_tested_source', tested), ('next_step', 'DCM-26'), ('next_runnable_step', 'DCM-26')]:
        plan_text = set_line(plan_text, key, value, 2)
    step = copy.deepcopy(next(x for x in before_plan['steps'] if x['id'] == 'DCM-25'))
    assert step['completion'] == 'pending' and step['user_test'] == 'not_recorded'
    step.update(status='fixed', completion='complete', ui_test='passed_with_separate_image_review', evidence=[REPORT, EVIDENCE], native_test=note)
    old = [line for line in plan_text.splitlines() if '"id": "DCM-25"' in line]
    assert len(old) == 1
    plan_text = plan_text.replace(old[0], '    ' + json.dumps(step, ensure_ascii=False) + ',')
    after_plan = json.loads(plan_text)
    assert len(after_plan['steps']) == 50
    for a, b in zip(before_plan['steps'], after_plan['steps']):
        if a['id'] != 'DCM-25':
            assert a == b
    assert next(x for x in after_plan['steps'] if x['id'] == 'DCM-19')['completion'] == 'pending'
    products = before_todo['completion_plan']['completed_product_steps']
    assert 'DCM-24' in products and 'DCM-25' not in products
    for key, value in [('completed_product_steps', products + ['DCM-25']), ('next_step', 'DCM-26'), ('next_runnable_step', 'DCM-26'), ('checkpoint', REPORT), ('candidate_code_commit', tested), ('last_native_attempt', note)]:
        todo_text = set_line(todo_text, key, value, 4)
    baseline = before_todo['checkpoints']['app_baseline_checkpoint']
    line = next(x for x in todo_text.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
    todo_text = todo_text.replace(line, '    "app_baseline_dcm24_checkpoint": ' + json.dumps(baseline) + ',\n    "app_baseline_checkpoint": ' + json.dumps(evidence['baseline']) + ',')
    needle = '    "final_package_qualified": false,'
    assert todo_text.count(needle) == 1
    todo_text = todo_text.replace(needle, '    "ui_refresh_checkpoint": ' + json.dumps(evidence['native_ui']) + ',\n' + needle)
    needle = '    "dcm24_completion": "qualification/dcm24-complete.md"'
    assert todo_text.count(needle) == 1
    todo_text = todo_text.replace(needle, needle + ',\n    "dcm25_completion": ' + json.dumps(REPORT))
    issue = {'id': 'DCM-25-REFRESH-01', 'feature': 'Invalidate stale source targets when Refresh cannot read the source folder', 'status': 'fixed', 'severity': 'medium', 'finding_kind': 'native_unavailable_source_stale_view', 'ui_test': 'passed_with_separate_image_review', 'user_test': 'not_recorded', 'release_gate': True, 'parent_steps': ['DCM-25'], 'acceptance': 'A failed source read clears the cached file set, selection, draft and old preview; shows a truthful unavailable state; stale Rename/Move actions are harmless; restoring the source and Reload recovers. Preserve normal source switching and exact operation targets.', 'native_test': note, 'evidence_refs': ['dcm25_completion']}
    assert issue['id'] not in {x['id'] for x in before_todo['items']}
    assert todo_text.count('\n  ]\n}') == 1
    todo_text = todo_text.replace('\n  ]\n}', ',\n    ' + json.dumps(issue) + '\n  ]\n}')
    after_todo = json.loads(todo_text)
    assert after_todo['items'][:-1] == before_todo['items']
    assert after_todo['completion_plan']['blocked_steps'] == before_todo['completion_plan']['blocked_steps']
    assert after_todo['checkpoints']['final_package_qualified'] is False
    assert after_todo['checkpoints']['user_acceptance'] == 'not_recorded'
    return plan_text, todo_text


def main():
    data = {}
    for name, wanted in EXPECTED.items():
        raw = (ROOT / name).read_bytes()
        actual = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
        assert actual == wanted, (name, actual, 'base changed; refusing')
        data[name] = raw.decode('utf-8')
    evidence = json.loads((ROOT / EVIDENCE).read_text())
    plan, todo = transform(data['qualification/completion-50.json'], data['todo.json'], evidence)
    for name, text in [('qualification/completion-50.json', plan), ('todo.json', todo)]:
        (ROOT / name).write_text(text, encoding='utf-8')
        print(name, hashlib.sha256(text.encode()).hexdigest())
    output = ROOT / 'checkpoint-exports'
    output.mkdir(exist_ok=True)
    lines = ['# DCM QuickSort — 50-step checklist', '', 'DCM-25 accepted. DCM-19 remains blocked. Five completed steps are setup/diagnostic work, not product features.', '', '| ID | Feature | Completion | UI test | User test |', '|---|---|---|---|---|']
    for step in json.loads(plan)['steps']:
        lines.append('| ' + ' | '.join(str(step.get(k, '')).replace('|', '/') for k in ('id', 'feature', 'completion', 'ui_test', 'user_test')) + ' |')
    (output / 'completion-50.md').write_text('\n'.join(lines) + '\n', encoding='utf-8')


if __name__ == '__main__':
    main()
