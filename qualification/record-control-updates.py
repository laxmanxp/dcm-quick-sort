"""Record the bounded live-control optimization; preserve all numbered/user gates."""
from pathlib import Path
import copy, hashlib, json, re
ROOT = Path(__file__).resolve().parents[1]
TESTED = 'cc6a1a37308ba57be37c33c81e743b668bb89a1c'
REPORT = 'qualification/2026-11-13-control-update-optimization.md'
EVIDENCE = 'qualification/evidence/control-update-optimization-summary.json'
SUMMARY = 'cc6a1a37: live-state dropdown/zoom guards preserve full pane layout. Two four-opening runs queue16.8-156.3ms, all below unchanged250ms target; layout9.6-52.8ms. Eight redundant combo moves and five enable calls skipped per attachment; required hides retained. Baseline377/0/1; pending-close3,lifecycle11,Undo9,errors12,scroll14,resize23+12bursts,text17,image23,sort9 and filters74 action/readback records pass. Nine full images reviewed. Wider environment/manual acceptance and numbered gates remain open.'
def blob(b):
    return hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()
def line(s, key, value, indent):
    p = re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$', re.M)
    found = list(p.finditer(s)); assert len(found) == 1, (key, len(found))
    return p.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+(',' if found[0].group().endswith(',') else ''), s)
def itemline(s, id, value):
    found = [x for x in s.splitlines() if x.lstrip().startswith('{"id": '+json.dumps(id)+',')]
    assert len(found) == 1, (id, len(found))
    old = found[0]
    return s.replace(old,'    '+json.dumps(value,ensure_ascii=False)+(',' if old.endswith(',') else ''))
def transform(todo, plan):
    t, p = json.loads(todo), json.loads(plan)
    todo = line(todo,'checkpoint',REPORT,4)
    todo = line(todo,'last_native_attempt',SUMMARY,4)
    todo = line(todo,'candidate_code_commit',TESTED,4)
    old = t['checkpoints']['app_baseline_checkpoint']
    new = dict(old,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261113-044443-a6a2f74c',count=378,passed=377)
    current = next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
    assert 'app_baseline_before_control_updates' not in t['checkpoints']
    todo = todo.replace(current,'    "app_baseline_before_control_updates": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
    record = {'commit':TESTED,'implemented':True,'full_layout_and_preview_sizing_retained':True,'geometry_cache':False,'runs':2,'opens_per_run':4,'all_eight_within_provisional_250ms':True,'max_queue_ms_range':[16.8132,156.3303],'layout_ms_range':[9.584,52.769],'attachment_ms_range':[49.78,166.016],'redundant_combo_skips_per_attachment':8,'redundant_enable_skips_per_attachment':5,'required_visibility_changes_retained':True,'full_images_reviewed':9,'physical_display_scaling_tested':False,'broader_environment_acceptance':False,'evidence':EVIDENCE,'numbered_steps_closed':[]}
    anchor = '    "final_package_qualified": false,'; assert todo.count(anchor) == 1
    todo = todo.replace(anchor,'    "control_update_optimization_checkpoint": '+json.dumps(record)+',\n'+anchor)
    x = copy.deepcopy(next(x for x in t['items'] if x['id']=='DCM-SUMATRA-STARTUP-01'))
    x['pre_control_update_native_test'] = x['native_test']
    x['native_test'] = SUMMARY
    x['status'] = 'fixed'
    x['progress'] = 'bounded_async_and_redundant_control_corrections_verified; wider_environment_acceptance_separate'
    x['ui_test'] = 'two_startup_runs_within_target; affected_regressions_pass; nine_images_reviewed'
    x['tested_matrix_response_accepted'] = True
    x['performance_accepted'] = False
    x['performance_scope_note'] = 'The tested eight-opening matrix passes; cold-boot, physical display scaling and universal/shared-load guarantees are not certified.'
    x['control_update_optimization'] = record
    x['evidence_refs'] += [REPORT,EVIDENCE]
    todo = itemline(todo,x['id'],x)
    a = json.loads(todo)
    assert len(a['items']) == len(t['items'])
    for old,new in zip(t['items'],a['items']):
        assert old['user_test'] == new['user_test']
        if old['id'] != 'DCM-SUMATRA-STARTUP-01': assert old == new
    assert a['completion_plan']['completed_product_steps'] == t['completion_plan']['completed_product_steps']
    assert a['completion_plan']['blocked_steps'] == t['completion_plan']['blocked_steps']
    assert a['checkpoints']['final_package_qualified'] is False
    assert a['checkpoints']['user_acceptance'] == t['checkpoints']['user_acceptance']
    plan = line(plan,'checkpoint',REPORT,2); plan = line(plan,'last_tested_source',TESTED,2)
    for id in ['DCM-28','DCM-29']:
        x = copy.deepcopy(next(x for x in p['steps'] if x['id']==id))
        x['control_update_optimization'] = record
        x['evidence'] += [REPORT,EVIDENCE]
        plan = itemline(plan,id,x)
    a = json.loads(plan); assert len(a['steps']) == 50
    for old,new in zip(p['steps'],a['steps']):
        assert old['completion'] == new['completion'] and old['user_test'] == new['user_test']
        if old['id'] not in ['DCM-28','DCM-29']: assert old == new
    assert sum(x['completion']=='complete' for x in a['steps']) == 26
    return todo,plan
if __name__ == '__main__':
    expected = {'todo.json':'fb3a8044b0d19ab8346285336841000e577e986a','qualification/completion-50.json':'ab95789b232de6647ca5024bd1c0628699243b4d'}
    raw = {name:(ROOT/name).read_bytes() for name in expected}
    for name,b in raw.items(): assert blob(b) == expected[name], (name,'input changed; reconcile')
    for name,s in zip(expected,transform(raw['todo.json'].decode(),raw['qualification/completion-50.json'].decode())):
        (ROOT/name).write_bytes(s.encode()); print(name,hashlib.sha256(s.encode()).hexdigest())
