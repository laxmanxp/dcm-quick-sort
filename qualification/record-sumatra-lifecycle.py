"""Record bounded native lifecycle evidence without closing broader provider gates."""
from pathlib import Path
import copy
import hashlib
import json
import re

ROOT = Path(__file__).resolve().parents[1]
TESTED = 'deb8373dbb52c44d963f20fce49b9ea074f7d59e'
REPORT = 'qualification/2026-11-12-sumatra-lifecycle.md'
EVIDENCE = 'qualification/evidence/sumatra-lifecycle-native-summary.json'
EXPECTED = {'todo.json':'b0313d4321e4464a68644fbd417f2c836991fd0c', 'qualification/completion-50.json':'0f2e2c40fdf295620b2fd4251c3a167df8abb451'}
SUMMARY = 'deb8373d: two new lifecycle runs pass 11 records each; 40 selections/run, 10 observed loading cancellations/run, four attached A-B-text cycles, empty/header errors and recovery, normal active-PDF close with no private processes/profiles/locks. All 10 first-run captures inspected. Baseline 367/0/1 and existing 23-check Sumatra suite pass. Test-only additions; no app changes, default switch, numbered closure or release.'

def read(name):
    raw=(ROOT/name).read_bytes()
    assert hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest()==EXPECTED[name], (name,'base changed; reconcile')
    return raw.decode()

def setline(s,key,value,indent):
    pattern=re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$',re.M)
    matches=list(pattern.finditer(s));assert len(matches)==1,(key,len(matches))
    comma=',' if matches[0].group().endswith(',') else ''
    return pattern.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+comma,s)

def itemline(s,id,value):
    lines=[line for line in s.splitlines() if line.lstrip().startswith('{"id": '+json.dumps(id)+',')]
    assert len(lines)==1,(id,len(lines))
    old=lines[0];return s.replace(old,'    '+json.dumps(value,ensure_ascii=False)+(',' if old.endswith(',') else ''))

todo=read('todo.json');before=json.loads(todo)
todo=setline(todo,'checkpoint',REPORT,4)
todo=setline(todo,'last_native_attempt',SUMMARY,4)
oldbase=before['checkpoints']['app_baseline_checkpoint']
line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
newbase=dict(oldbase,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261112-093728-3abdf19f')
todo=todo.replace(line,'    "app_baseline_before_lifecycle_checkpoint": '+json.dumps(oldbase)+',\n    "app_baseline_checkpoint": '+json.dumps(newbase)+',')
checkpoint={'commit':TESTED,'new_selector':'ui-sumatra-lifecycle','passed_runs':2,'cases_per_run':11,'selections_per_run':40,'observed_loading_cancellations_per_run':10,'attached_cycles_per_run':4,'reviewed_images':10,'normal_exit_codes':[0,0],'close_ms':[393,374],'product_changed':False,'review_evidence':EVIDENCE}
needle='    "final_package_qualified": false,';assert todo.count(needle)==1
todo=todo.replace(needle,'    "sumatra_lifecycle_checkpoint": '+json.dumps(checkpoint)+',\n'+needle)
item=copy.deepcopy(next(x for x in before['items'] if x['id']=='DCM-SUMATRA-01'))
item['resize_native_test']=item['native_test'];item['native_test']=SUMMARY
item['ui_test']='navigation_resize_and_bounded_lifecycle_pass; Undo_failure_and_distribution_gates_open'
item['evidence_refs'] += [REPORT,EVIDENCE]
todo=itemline(todo,item['id'],item)
new={'id':'DCM-SUMATRA-LIFECYCLE-01','feature':'Verify bounded PDF switching, loading abandonment, basic error recovery and active-PDF close','status':'tested','ui_test':'11_records_pass_twice; all_10_first_run_captures_reviewed','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-29'],'acceptance':'Distinct PDF/text switching, selection-based cancellation observed during loading, truthful empty/header errors and recovery, normal active-PDF close; preserve source bytes and leave no owned viewer, private session profile or source lock. This bounded gate does not cover Undo, crashes or release refusal.','native_test':SUMMARY,'evidence_refs':[REPORT,EVIDENCE]}
assert not any(x['id']==new['id'] for x in before['items'])
needle='\n  ]\n}';assert todo.count(needle)==1
todo=todo.replace(needle,',\n    '+json.dumps(new)+needle)
after=json.loads(todo)
assert after['items'][-1]['id']==new['id']
for a,b in zip(before['items'],after['items']):
    if a['id']!='DCM-SUMATRA-01':assert a==b
assert after['completion_plan']['completed_product_steps']==before['completion_plan']['completed_product_steps']
assert after['checkpoints']['candidate_code_commit']==before['checkpoints']['candidate_code_commit']
assert after['checkpoints']['final_package_qualified'] is False
assert after['checkpoints']['user_acceptance']=='not_recorded'

plan=read('qualification/completion-50.json');p_before=json.loads(plan)
plan=setline(plan,'checkpoint',REPORT,2);plan=setline(plan,'last_tested_source',TESTED,2)
for id in ['DCM-28','DCM-29']:
    step=copy.deepcopy(next(x for x in p_before['steps'] if x['id']==id))
    step['resize_native_test']=step['native_test'];step['native_test']=SUMMARY
    step['ui_test']='optional_navigation_zoom_and_bounded_lifecycle_pass; remaining_provider_acceptance_pending'
    step['evidence'] += [REPORT,EVIDENCE]
    plan=itemline(plan,id,step)
p_after=json.loads(plan)
assert len(p_after['steps'])==50
for a,b in zip(p_before['steps'],p_after['steps']):
    assert a['completion']==b['completion'] and a['user_test']==b['user_test']
    if a['id'] not in ['DCM-28','DCM-29']:assert a==b
assert sum(x['completion']=='complete' for x in p_after['steps'])==26
for name,s in [('todo.json',todo),('qualification/completion-50.json',plan)]:
    (ROOT/name).write_bytes(s.encode())
    print(name,hashlib.sha256(s.encode()).hexdigest())
