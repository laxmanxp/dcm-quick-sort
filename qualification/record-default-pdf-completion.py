"""Close only the demonstrated PDF navigation/zoom gates; preserve failed repeats."""
from pathlib import Path
import copy, hashlib, json, re
ROOT = Path(__file__).resolve().parents[1]
TESTED = '9a2ed85a38e25c3ad9698ee41ec993379dff9686'
REPORT = 'qualification/2026-11-13-default-pdf-completion.md'
EVIDENCE = 'qualification/evidence/default-pdf-completion-summary.json'
SUMMARY = '9a2ed85a: default Windows.Data.Pdf enhanced workflow19 action/readback records passes once; ten full images reviewed. Actual pages1/2, disabled boundary controls,100/110/return100,Fit Page/Fit Width and readable native pan verified; files/journal/close intact. Two unchanged repeats failed capture visibility guards and remain incomplete. Baseline379/0/1 and optional Sumatra23 pass. App byte-identical to2cafe818; historical blank cause unknown, no product fix claimed. Close only DCM28/29 within demonstrated two-page scope; human/package/default states unchanged.'

def blob(b):
    return hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()

def line(s, key, value, indent):
    pattern = re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$', re.M)
    found = list(pattern.finditer(s))
    assert len(found) == 1, (key, len(found))
    return pattern.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value, ensure_ascii=False)+(',' if found[0].group().endswith(',') else ''), s)

def itemline(s, id, value):
    pattern = re.compile(r'^    \{(?:\n      |)"id": '+re.escape(json.dumps(id))+',', re.M)
    found = list(pattern.finditer(s))
    assert len(found) == 1, (id, len(found))
    start = found[0].start()+4
    old, length = json.JSONDecoder().raw_decode(s[start:])
    assert old['id'] == id
    return s[:start]+json.dumps(value, ensure_ascii=False)+s[start+length:]

def transform(todo, plan):
    t, p = json.loads(todo), json.loads(plan)
    evidence = json.loads((ROOT/EVIDENCE).read_text())
    assert evidence['tested_commit'] == TESTED and evidence['first_run']['passed']
    assert evidence['first_run']['records'] == 19 and evidence['visual_review']['full_images_individually_reviewed'] == 10
    assert evidence['baseline']['passed'] == 379 and evidence['baseline']['failed'] == 0
    assert len(evidence['repeat_attempts']) == 2 and all(not x['passed'] for x in evidence['repeat_attempts'])
    assert evidence['application_code_changed'] is False and evidence['user_acceptance'] == 'not_recorded'
    assert evidence['numbered_steps_closed'] == ['DCM-28','DCM-29']
    for key, value in [('checkpoint', REPORT), ('last_native_attempt', SUMMARY), ('candidate_code_commit', TESTED), ('next_step', 'DCM-30'), ('next_runnable_step', 'DCM-30')]:
        todo = line(todo, key, value, 4)
    completed = t['completion_plan']['completed_product_steps']
    assert all(x not in completed for x in ['DCM-28','DCM-29'])
    todo = line(todo, 'completed_product_steps', completed+['DCM-28','DCM-29'], 4)
    old = t['checkpoints']['app_baseline_checkpoint']
    new = dict(old, commit=TESTED, report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261113-064704-4847f825', count=380, passed=379)
    current = next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
    assert 'app_baseline_before_default_pdf' not in t['checkpoints']
    todo = todo.replace(current, '    "app_baseline_before_default_pdf": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
    record = {'commit': TESTED, 'provider': 'default Windows.Data.Pdf', 'complete_enhanced_runs': 1, 'records': 19, 'full_images_reviewed': 10, 'incomplete_visibility_repeats': 2, 'repeatability_qualification_complete': False, 'historical_blank_reproduced': False, 'historical_blank_cause_known': False, 'application_changed': False, 'app_sha256': evidence['build']['app_sha256'], 'evidence': EVIDENCE, 'numbered_steps_closed': ['DCM-28','DCM-29'], 'scope': 'Known two-page navigation, disabled boundaries,100/110/return100,Fit/FitWidth and native pan; no human/package/physical-input or arbitrary-document certification.'}
    anchor = '    "final_package_qualified": false,'
    assert todo.count(anchor) == 1
    todo = todo.replace(anchor, '    "default_pdf_completion_checkpoint": '+json.dumps(record)+',\n'+anchor)
    anchor = '  "evidence": {\n'
    assert todo.count(anchor) == 1 and 'default_pdf_completion' not in t['evidence']
    todo = todo.replace(anchor, anchor+'    "default_pdf_completion": '+json.dumps(REPORT)+',\n')
    x = copy.deepcopy(next(x for x in t['items'] if x['id']=='DCM-PDF-01'))
    x['previous_ui_test'] = x['ui_test']
    x['previous_blocked_by'] = x['blocked_by']
    x['blocked_by'] = []
    x['status'] = 'tested'
    x['ui_test'] = 'current_default_provider_visual_qualification_passed_once; two_repeat_capture_failures_preserved'
    x['native_test'] = SUMMARY
    x['disposition'] = 'Current build meets the bounded page/zoom visual acceptance. Old v1.0.69 blank cause is unresolved historically, not a newly fixed product defect. Final-candidate/manual gates remain separate.'
    x['evidence_refs'] += ['default_pdf_completion', EVIDENCE]
    todo = itemline(todo, x['id'], x)
    after = json.loads(todo)
    assert len(after['items']) == len(t['items'])
    for before, after_item in zip(t['items'], after['items']):
        assert before['user_test'] == after_item['user_test']
        if before['id'] != 'DCM-PDF-01':
            assert before == after_item
    assert after['completion_plan']['blocked_steps'] == t['completion_plan']['blocked_steps']
    assert after['completion_plan']['active_blockers'] == t['completion_plan']['active_blockers']
    assert after['checkpoints']['final_package_qualified'] is False
    assert after['checkpoints']['user_acceptance'] == t['checkpoints']['user_acceptance']
    for key, value in [('checkpoint', REPORT), ('last_tested_source', TESTED), ('next_step', 'DCM-30'), ('next_runnable_step', 'DCM-30')]:
        plan = line(plan, key, value, 2)
    for id in ['DCM-28','DCM-29']:
        x = copy.deepcopy(next(x for x in p['steps'] if x['id']==id))
        assert x['completion'] == 'pending'
        x['pre_default_provider_native_test'] = x.get('native_test', '')
        x['status'] = 'tested'
        x['completion'] = 'complete'
        x['ui_test'] = 'default_workflow_passed_with_ten_image_review; repeat_visibility_failures_retained'
        x['native_test'] = SUMMARY
        x['scope_note'] = record['scope']
        x['evidence'] += [REPORT, EVIDENCE]
        x['default_provider_completion'] = record
        plan = itemline(plan, id, x)
    after = json.loads(plan)
    assert len(after['steps']) == 50
    changed = []
    for before, after_step in zip(p['steps'], after['steps']):
        assert before['user_test'] == after_step['user_test']
        if before['completion'] != after_step['completion']:
            changed.append(before['id'])
        if before['id'] not in ['DCM-28','DCM-29']:
            assert before == after_step
    assert changed == ['DCM-28','DCM-29']
    assert sum(x['completion']=='complete' for x in after['steps']) == 28
    return todo, plan

if __name__ == '__main__':
    expected = {'todo.json':'3beac6c0d07fa396f88eac2a0cc1b5f5ea45359d', 'qualification/completion-50.json':'8f34457007039ab32e6af11792f11cf721fdf109'}
    raw = {name:(ROOT/name).read_bytes() for name in expected}
    for name, b in raw.items():
        assert blob(b) == expected[name], (name, 'input changed; reconcile')
    for name, s in zip(expected, transform(raw['todo.json'].decode(), raw['qualification/completion-50.json'].decode())):
        (ROOT/name).write_bytes(s.encode())
        print(name, hashlib.sha256(s.encode()).hexdigest())
