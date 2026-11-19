"""Record verified individual Undo/failure coverage; keep broader gates pending."""
from pathlib import Path
import copy, hashlib, json, re
ROOT=Path(__file__).resolve().parents[1]
TESTED='cc04ae7cd8932cffc4e13d7e181ae51eb78095fc'
PRODUCT='0c9cdf87084bd9f18c0e062bee9f40ce8f6f0ace'
REPORT='qualification/2026-11-12-sumatra-undo.md'
EVIDENCE='qualification/evidence/sumatra-undo-native-summary.json'
EXPECTED={'todo.json':'794b2f4770ee1468425551018ea58d436737000e','qualification/completion-50.json':'0895a7e968f8170bc54bcc5646caddaa356dac81'}
SUMMARY='cc04ae7c: two consecutive nine-record Undo/viewer-exit runs pass; eight first-run full screenshots reviewed. Destination stale-selection RED at55e2b877 fixed by0c9cdf87. Windows baseline368/0/1 includes six release-refusal cases; Refresh23 and lifecycle11 regressions pass. Test readiness changed to posted selection/bounded polling; initial startup responsiveness remains open. No numbered closure, default switch or release.'
def read(n):
 b=(ROOT/n).read_bytes();assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==EXPECTED[n],(n,'input changed');return b.decode()
def setline(s,k,v,i):
 p=re.compile(r'^'+(' '*i)+re.escape(json.dumps(k))+r':.*$',re.M);m=list(p.finditer(s));assert len(m)==1,(k,len(m))
 return p.sub(lambda _: ' '*i+json.dumps(k)+': '+json.dumps(v,ensure_ascii=False)+(',' if m[0].group().endswith(',') else ''),s)
def itemline(s,id,v):
 lines=[l for l in s.splitlines() if l.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(lines)==1
 return s.replace(lines[0],'    '+json.dumps(v,ensure_ascii=False)+(',' if lines[0].endswith(',') else ''))
todo=read('todo.json');before=json.loads(todo)
todo=setline(todo,'checkpoint',REPORT,4);todo=setline(todo,'last_native_attempt',SUMMARY,4);todo=setline(todo,'candidate_code_commit',PRODUCT,4)
oldbase=before['checkpoints']['app_baseline_checkpoint'];line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
newbase=dict(oldbase,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261112-131448-2a7ddc9c',count=369,passed=368)
todo=todo.replace(line,'    "app_baseline_before_undo_checkpoint": '+json.dumps(oldbase)+',\n    "app_baseline_checkpoint": '+json.dumps(newbase)+',')
checkpoint={'commit':TESTED,'product_fix':PRODUCT,'selector':'ui-sumatra-undo','passed_runs':2,'cases_per_run':9,'reviewed_full_images':8,'content_crops_separately_reviewed':False,'normal_exit_codes':[0,0],'release_guard_parent':'TestSumatraReleaseFailurePreservesOperations','release_guard_nested_cases':6,'review_evidence':EVIDENCE,'startup_responsiveness':'open; test polling is not product performance repair'}
needle='    "final_package_qualified": false,';assert todo.count(needle)==1
todo=todo.replace(needle,'    "sumatra_undo_checkpoint": '+json.dumps(checkpoint)+',\n'+needle)
item=copy.deepcopy(next(x for x in before['items'] if x['id']=='DCM-SUMATRA-01'))
assert 'lifecycle_native_test' not in item
item['lifecycle_native_test']=item['native_test'];item['native_test']=SUMMARY
item['ui_test']='individual_Undo_and_controlled_exit_pass; injected_release_guards_pass; broader_provider_gates_open'
item['evidence_refs']+=[REPORT,EVIDENCE];todo=itemline(todo,item['id'],item)
issues=[
 {'id':'DCM-UNDO-REFRESH-01','feature':'Clear stale destination view after individual Undo moves its selected PDF back','status':'fixed','severity':'medium','ui_test':'native_RED_to_GREEN; two_final_runs; eight_full_images_reviewed','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-44'],'acceptance':'After Undo changes either source or destination directory, refresh the active list and clear a removed target; preserve original bytes and journal identity. Batch Undo/path-alias matrices remain separate.','native_test':SUMMARY,'evidence_refs':[REPORT,EVIDENCE]},
 {'id':'DCM-SUMATRA-UNDO-01','feature':'Qualify individual active-PDF Undo and controlled viewer-exit recovery','status':'tested','ui_test':'nine_records_pass_twice; eight_full_images_reviewed','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-29','DCM-37','DCM-44'],'acceptance':'Release active/reopened PDF for individual Undo Rename/Move, verify exact restoration and original journal IDs, show controlled owned-viewer exit truthfully, and recover on Reload. Windows injected wait/termination failures must not admit Rename, Move or Undo. This is not full live-permission-denial or Batch Undo qualification.','native_test':SUMMARY,'evidence_refs':[REPORT,EVIDENCE]},
 {'id':'DCM-SUMATRA-STARTUP-01','feature':'Investigate initial PDF selection responsiveness and native-message timeouts','status':'planned','severity':'low','finding_kind':'startup_readiness_observation','ui_test':'initial_selection_timeouts_preserved; readiness_polling_unblocks_acceptance_not_product_latency','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-29'],'acceptance':'Measure startup-stage costs and real host responsiveness under cold/warm runtime conditions. Do not claim asynchronous test notification or longer readiness polling repairs product responsiveness. Preserve native-message and capture failures.','native_test':'Initial synchronous selection/query deadlines occurred before Undo. Final test-only posting and 50ms queries within the existing20s readiness window retained exact target/content checks. Initial readiness in passing runs1437/1610ms; underlying startup cost and every earlier timeout cause remain unestablished.','evidence_refs':[REPORT,EVIDENCE]}
]
assert all(x['id'] not in {i['id'] for i in before['items']} for x in issues)
needle='\n  ]\n}';assert todo.count(needle)==1
todo=todo.replace(needle,',\n'+',\n'.join('    '+json.dumps(x,ensure_ascii=False) for x in issues)+needle)
after=json.loads(todo)
for a,b in zip(before['items'],after['items']):
 if a['id']!='DCM-SUMATRA-01':assert a==b
assert after['items'][-3:]==issues
assert after['completion_plan']['completed_product_steps']==before['completion_plan']['completed_product_steps']
assert after['completion_plan']['blocked_steps']==before['completion_plan']['blocked_steps']
assert after['checkpoints']['final_package_qualified'] is False
assert after['checkpoints']['user_acceptance']=='not_recorded'
plan=read('qualification/completion-50.json');pb=json.loads(plan)
plan=setline(plan,'checkpoint',REPORT,2);plan=setline(plan,'last_tested_source',TESTED,2)
for id in ['DCM-28','DCM-29']:
 step=copy.deepcopy(next(x for x in pb['steps'] if x['id']==id));assert 'lifecycle_native_test' not in step
 step['lifecycle_native_test']=step['native_test'];step['native_test']=SUMMARY
 step['ui_test']='optional_provider_individual_Undo_and_controlled_exit_pass; remaining_acceptance_pending'
 step['evidence']+=[REPORT,EVIDENCE];plan=itemline(plan,id,step)
pa=json.loads(plan);assert len(pa['steps'])==50
for a,b in zip(pb['steps'],pa['steps']):
 assert a['completion']==b['completion'] and a['user_test']==b['user_test']
 if a['id'] not in ['DCM-28','DCM-29']:assert a==b
assert sum(x['completion']=='complete' for x in pa['steps'])==26
for n,s in [('todo.json',todo),('qualification/completion-50.json',plan)]:
 (ROOT/n).write_bytes(s.encode());print(n,hashlib.sha256(s.encode()).hexdigest())
