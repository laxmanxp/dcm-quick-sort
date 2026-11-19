from pathlib import Path
import copy
import hashlib
import json
import re

ROOT = Path('.')
EXPECTED = {
    'todo.json': 'a23e8cb02bac30be0693417f88e4c4de65951690',
    'qualification/completion-50.json': '58f7c02c8134978c420b05d5067b38e1eed17f8c',
}
COMMIT = '996f67208c574a0821d1dda598ffb7e0f9c775b3'
CHECKPOINT = 'qualification/2026-11-11-dcm23.md'
EVIDENCE = 'qualification/evidence/dcm-23-native-summary.json'
DECODER = json.JSONDecoder()

def encode(value):
    return json.dumps(value, ensure_ascii=False)

def locate(text, key, old):
    matches = []
    for match in re.finditer(r'"' + re.escape(key) + r'"\s*:\s*', text):
        value, end = DECODER.raw_decode(text, match.end())
        if value == old:
            matches.append((match.end(), end))
    assert len(matches) == 1, (key, 'expected unique original value', len(matches))
    return matches[0]

def replace(text, key, old, new):
    start, end = locate(text, key, old)
    return text[:start] + encode(new) + text[end:]

def append_field(text, anchor, key, value, indent=4):
    assert text.count(anchor) == 1, ('ambiguous anchor', anchor)
    return text.replace(anchor, ' ' * indent + encode(key) + ': ' + encode(value) + ',\n' + anchor, 1)

def load_verified(path):
    raw = (ROOT / path).read_bytes()
    blob = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
    assert blob == EXPECTED[path], (path, blob, 'checkpoint base changed; refusing')
    return raw.decode('utf-8'), json.loads(raw)

def main():
    text, original = load_verified('todo.json')
    plan = original['completion_plan']
    assert plan['next_step'] == 'DCM-23' and 'DCM-19' in plan['blocked_steps']
    assert 'DCM-23' not in plan['completed_product_steps']
    text = replace(text, 'completed_product_steps', plan['completed_product_steps'], plan['completed_product_steps'] + ['DCM-23'])
    text = replace(text, 'next_step', 'DCM-23', 'DCM-24')
    text = replace(text, 'next_runnable_step', 'DCM-23', 'DCM-24')
    text = replace(text, 'checkpoint', plan['checkpoint'], CHECKPOINT)
    text = replace(text, 'candidate_code_commit', original['checkpoints']['candidate_code_commit'], COMMIT)
    result = ('996f6720: 22 native navigation cases passed with eight independently reviewed images after reproduced Next selection mismatch. '
              'Both app instances closed normally with exit 0; fixture names/bytes and source unchanged. Full baseline 357 pass / 0 fail / '
              '1 retained external-PPTX skip. ui-sort and ui-launch commands also passed; their new images were not separately reviewed. '
              'Only DCM-23 newly closed; DCM-19 remains blocked.')
    text = replace(text, 'last_native_attempt', original['checkpoints']['last_native_attempt'], result)
    assert 'app_baseline_dcm22_checkpoint' not in original['checkpoints']
    assert text.count('"app_baseline_checkpoint":') == 1
    text = text.replace('"app_baseline_checkpoint":', '"app_baseline_dcm22_checkpoint":', 1)
    baseline = copy.deepcopy(original['checkpoints']['app_baseline_checkpoint'])
    baseline.update(commit=COMMIT, report='20261111-044006-c8527279')
    text = append_field(text, '    "final_package_qualified": false,', 'app_baseline_checkpoint', baseline)
    native = {
        'commit': COMMIT, 'report': r'C:\FlowverseReports\dcm-quicksort-ui-navigation-20261111-044215-b415f633',
        'passed': True, 'case_count': 22, 'reviewed_images': 8, 'exit_codes': [0, 0],
        'source_unchanged': True, 'fixtures_unchanged': True, 'operation_journal_empty': True,
        'keyboard_input': 'Owned queued native messages, not physical keyboard certification',
        'review_evidence': EVIDENCE,
        'app_sha256': '016da629bd77f57988eac7d2f162384d4c77583e052f4036e3e1581fe76e0fb0',
    }
    assert 'ui_navigation_checkpoint' not in original['checkpoints']
    text = append_field(text, '    "final_package_qualified": false,', 'ui_navigation_checkpoint', native)
    anchor = '    "dcm22_checkpoint":'
    assert text.count(anchor) == 1
    text = text.replace(anchor, '    "dcm23_checkpoint": ' + encode(CHECKPOINT) + ',\n' + anchor, 1)
    bug = {
        'id': 'DCM-23-SELECTION-01', 'feature': 'Keep explicit Next selection aligned with current file and preview',
        'status': 'fixed', 'severity': 'medium', 'finding_kind': 'native_UI_assertion_failure',
        'ui_test': 'passed_with_separate_image_review', 'user_test': 'not_recorded', 'release_gate': True,
        'parent_steps': ['DCM-23'],
        'acceptance': 'Next selects the new visible file, or clears selection at end; highlight, draft and preview agree. Preserve shared multi-selection handling and final-candidate regression.',
        'native_test': 'RED at b2a4e5ad: second-file draft/preview with first row selected. Three executable-line fix at 996f6720; 22 native cases and eight reviewed images pass. Full baseline 357/0/1; sorting/launch regressions pass. Existing Up/Down/Home/End tested with owned queued messages.',
        'evidence_refs': ['dcm23_checkpoint'],
    }
    assert not any(item['id'] == bug['id'] for item in original['items'])
    start, end = locate(text, 'items', original['items'])
    array_text = text[start:end]
    text = text[:start] + array_text[:-1].rstrip() + ',\n    ' + encode(bug) + '\n  ]' + text[end:]
    updated = json.loads(text)
    assert updated['items'][:-1] == original['items'], 'historical items changed'
    assert updated['completion_plan']['blocked_steps'] == original['completion_plan']['blocked_steps']
    assert updated['checkpoints']['user_acceptance'] == 'not_recorded'
    assert updated['checkpoints']['final_package_qualified'] is False
    for key, value in original['checkpoints'].items():
        if key not in {'candidate_code_commit', 'last_native_attempt', 'app_baseline_checkpoint'}:
            assert updated['checkpoints'][key] == value, ('historical checkpoint changed', key)
    assert updated['checkpoints']['app_baseline_dcm22_checkpoint'] == original['checkpoints']['app_baseline_checkpoint']
    for key, value in original['evidence'].items():
        assert updated['evidence'][key] == value

    checklist, old_list = load_verified('qualification/completion-50.json')
    assert len(old_list['steps']) == 50
    checklist = replace(checklist, 'checkpoint', old_list['checkpoint'], CHECKPOINT)
    checklist = replace(checklist, 'last_tested_source', old_list['last_tested_source'], COMMIT)
    checklist = replace(checklist, 'next_step', 'DCM-23', 'DCM-24')
    checklist = replace(checklist, 'next_runnable_step', 'DCM-23', 'DCM-24')
    step = next(item for item in old_list['steps'] if item['id'] == 'DCM-23')
    assert step['completion'] == 'pending' and step['user_test'] == 'not_recorded'
    new_step = copy.deepcopy(step)
    new_step.update(status='fixed', completion='complete', ui_test='passed_with_separate_image_review',
                    evidence=[CHECKPOINT, EVIDENCE], native_test=result,
                    scope_note='Existing Next and Up/Down/Home/End routes; no Previous button added. Queued owned native key messages are not physical keyboard certification. Multi-selection remains DCM-24.')
    matches = []
    for match in re.finditer(r'\{\s*"id"\s*:\s*"DCM-23"', checklist):
        value, end = DECODER.raw_decode(checklist, match.start())
        if value == step:
            matches.append((match.start(), end))
    assert len(matches) == 1
    start, end = matches[0]
    checklist = checklist[:start] + encode(new_step) + checklist[end:]
    new_list = json.loads(checklist)
    assert len(new_list['steps']) == 50 and len({i['id'] for i in new_list['steps']}) == 50
    for old, new in zip(old_list['steps'], new_list['steps']):
        if old['id'] != 'DCM-23':
            assert old == new, ('unrelated step changed', old['id'])
    assert next(i for i in new_list['steps'] if i['id'] == 'DCM-19')['completion'] == 'pending'
    assert sum(i['completion'] == 'complete' for i in new_list['steps']) == 22
    assert new_list['blocked_steps'] == old_list['blocked_steps']
    pending = {'todo.json': text, 'qualification/completion-50.json': checklist}
    for path, data in pending.items():
        (ROOT / path).write_text(data, encoding='utf-8', newline='')
        print(path, hashlib.sha256(data.encode()).hexdigest())

if __name__ == '__main__':
    main()
