"""Record bounded async-startup results without closing latency/release gates."""
from pathlib import Path
import copy,hashlib,json,re
R=Path(__file__).resolve().parents[1]
TESTED='4c1c32d668955a2073ac496d0f3d31e5fe1cdf43'
RESTORED='4e07b99b313027794c05a6bf280208513d869c34'
REPORT='qualification/2026-11-13-sumatra-async-startup.md'
EVIDENCE='qualification/evidence/sumatra-async-startup-summary.json'
SUMMARY='Async validation/process preparation implemented. Full native matrix at4c1c32d: baseline372/0/1, startup2x4, pending-close3, lifecycle11, Undo9, errors12, scroll14, resize23+12bursts pass. Later queue samples46-105ms; earlier async repeats275/321ms retained, broader250ms gate open. Speculative transition layout failed visible containment and was reverted at4e07b99b; rebuilt binary exactly matches tested app. No numbered closure, default switch or release.'
EXPECTED={'todo.json':'a2b994753b5a87f572fb14e14fdeb3bc8f79fa41','qualification/completion-50.json':'840ec8bcd0352d84693e8347eab2e759be2bf059'}
def setline(s,key,value,indent):
 pat=re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$',re.M);m=list(pat.finditer(s));assert len(m)==1,(key,len(m))
 return pat.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+(',' if m[0].group().endswith(',') else ''),s)
def itemline(s,id,value):
 m=[x for x in s.splitlines() if x.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(m)==1,id
 return s.replace(m[0],'    '+json.dumps(value,ensure_ascii=False)+(',' if m[0].endswith(',') else ''))
raw={n:(R/n).read_bytes() for n in EXPECTED}
for n,b in raw.items():assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==EXPECTED[n],(n,'input changed; reconcile')
todo=raw['todo.json'].decode();before=json.loads(todo)
for k,v in [('checkpoint',REPORT),('last_native_attempt',SUMMARY),('candidate_code_commit',RESTORED)]:todo=setline(todo,k,v,4)
old=before['checkpoints']['app_baseline_checkpoint']
line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
new=dict(old,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261113-015918-9335041c',count=373,passed=372)
todo=todo.replace(line,'    "app_baseline_before_async_checkpoint": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
checkpoint={'ui_tested_commit':TESTED,'restored_build_commit':RESTORED,'binary_matches_tested_app':True,'app_sha256':'6f09034154c40fca5737797cd5e341f6817d301b3bfc6e3375fe0bea0930916a','full_ui_rerun_after_rollback':False,'latest_two_runs_under_provisional_budget':True,'broader_latency_gate_closed':False,'first_queue_ms':[45.7167,59.1361],'earlier_async_repeat_over_budget_ms':[275.3351,320.752],'evidence':EVIDENCE}
anchor='    "final_package_qualified": false,';assert todo.count(anchor)==1
todo=todo.replace(anchor,'    "sumatra_async_startup_checkpoint": '+json.dumps(checkpoint)+',\n'+anchor)
it=copy.deepcopy(next(x for x in before['items'] if x['id']=='DCM-SUMATRA-STARTUP-01'))
it['pre_async_native_test']=it['native_test'];it['native_test']=SUMMARY;it['progress']='async_creation_implemented_and_safety_tested; broader_latency_gate_open; speculative_layout_optimization_reverted';it['performance_accepted']=False;it['ui_test']='later_two_response_runs_under_budget; safety_matrix_pass; earlier_over_budget_samples_retained';it['async_creation_implemented']=True;it['evidence_refs'] += [REPORT,EVIDENCE]
todo=itemline(todo,it['id'],it)
after=json.loads(todo)
assert len(after['items'])==len(before['items'])
for a,b in zip(before['items'],after['items']):
 if a['id']!='DCM-SUMATRA-STARTUP-01':assert a==b
for k in ['completed_product_steps','completed_setup_steps','blocked_steps']:assert after['completion_plan'][k]==before['completion_plan'][k]
assert after['checkpoints']['final_package_qualified'] is False
assert after['checkpoints']['user_acceptance']==before['checkpoints']['user_acceptance']
plan=raw['qualification/completion-50.json'].decode();p_before=json.loads(plan)
plan=setline(plan,'checkpoint',REPORT,2);plan=setline(plan,'last_tested_source',TESTED,2)
for id in ['DCM-28','DCM-29']:
 s=copy.deepcopy(next(x for x in p_before['steps'] if x['id']==id));s['async_startup_checkpoint']=checkpoint;s['evidence'] += [REPORT,EVIDENCE];plan=itemline(plan,id,s)
p_after=json.loads(plan);assert len(p_after['steps'])==50
for a,b in zip(p_before['steps'],p_after['steps']):
 assert a['completion']==b['completion'] and a['user_test']==b['user_test']
 if a['id'] not in ['DCM-28','DCM-29']:assert a==b
assert sum(s['completion']=='complete' for s in p_after['steps'])==26
for n,s in [('todo.json',todo),('qualification/completion-50.json',plan)]:
 (R/n).write_bytes(s.encode());print(n,hashlib.sha256(s.encode()).hexdigest())
