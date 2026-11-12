"""Record bounded native resize timing correction; retain all old evidence/gates."""
from pathlib import Path
import copy,hashlib,json,re
ROOT=Path(__file__).resolve().parents[1]
REPORT='qualification/2026-11-12-sumatra-resize-timing.md'
EVIDENCE='qualification/evidence/sumatra-resize-timing-summary.json'
EXPECTED={'todo.json':'4621b23ab833c236a2f6908b48a088e42c2122ee','qualification/completion-50.json':'d992845604c8f5475afc5dd1ddd9cc4929291267'}
e=json.loads((ROOT/EVIDENCE).read_text());summary=e['summary'];tested=e['tested_commit']
assert tested=='02e6efb39f08c5cfa29da4787cd4f41bf71b642a'
assert e['runs']['unsynchronized_red']['passed'] is False
for k in ['fixed_timing_first','fixed_timing_repeat']:
 assert e['runs'][k]['passed'] and e['runs'][k]['bursts']==12 and e['runs'][k]['all_sample_differences']==0

def read(n):
 b=(ROOT/n).read_bytes();assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==EXPECTED[n],(n,'base changed; reconcile');return b.decode()
def setline(s,k,v,i):
 p=re.compile(r'^'+(' '*i)+re.escape(json.dumps(k))+r':.*$',re.M);m=list(p.finditer(s));assert len(m)==1,(k,len(m))
 return p.sub(lambda _: ' '*i+json.dumps(k)+': '+json.dumps(v,ensure_ascii=False)+(',' if m[0].group().endswith(',') else ''),s)
def itemline(s,id,v):
 lines=[l for l in s.splitlines() if l.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(lines)==1,(id,len(lines))
 return s.replace(lines[0],'    '+json.dumps(v,ensure_ascii=False)+(',' if lines[0].endswith(',') else ''))

todo=read('todo.json');before=json.loads(todo)
for k,v in [('checkpoint',REPORT),('last_native_attempt',summary),('candidate_code_commit',tested)]:todo=setline(todo,k,v,4)
old=before['checkpoints']['app_baseline_checkpoint'];b=e['baseline']
line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
assert 'app_baseline_before_resize_timing_checkpoint' not in before['checkpoints']
new=dict(old,commit=tested,report=e['report_root']+'\\'+b['report'],count=b['count'],passed=b['passed'])
todo=todo.replace(line,'    "app_baseline_before_resize_timing_checkpoint": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
cp={'commit':tested,'passed_timing_runs':2,'behavior_records_per_run':23,'timing_bursts_per_run':12,'requests_per_burst':6,'raw_natural_comparisons_per_run':24,'sample_differences':0,'reviewed_green_images':18,'reviewed_red_images':3,'resolution':e['resolution'],'review_evidence':EVIDENCE}
needle='    "final_package_qualified": false,';assert todo.count(needle)==1
todo=todo.replace(needle,'    "sumatra_resize_timing_checkpoint": '+json.dumps(cp)+',\n'+needle)
changed={'DCM-SUMATRA-01','DCM-UI-RESIZE-01'}
for id in changed:
 item=copy.deepcopy(next(x for x in before['items'] if x['id']==id))
 item['before_resize_timing']={k:item[k] for k in ['status','ui_test','native_test','resolution_state','acceptance'] if k in item}
 item['evidence_refs'] += [REPORT,EVIDENCE];item['native_test']=summary
 if id=='DCM-UI-RESIZE-01':
  item['status']='fixed';item['resolution_state']=e['resolution']
  item['ui_test']='fixed_wait_diagnostic_RED_to_GREEN_twice; 18_GREEN_and3_RED_images_reviewed'
  item['acceptance']='Diagnose repeated-resize partial frames without masking raw evidence; preserve completion/paint traces and verify clean controls under identical fixed-wait RED/GREEN sequencing. Scope closure to the reproduced queued-layout path and tested geometry; retain other historic cause uncertainty and general layout gates.'
 else:item['ui_test']='resize_timing12_bursts_twice_and_ordinary23_pass; provider_acceptance_remains_pending'
 todo=itemline(todo,id,item)
after=json.loads(todo)
assert len(after['items'])==len(before['items'])
for a,b in zip(before['items'],after['items']):
 assert a['id']==b['id'] and a['user_test']==b['user_test']
 if a['id'] not in changed:assert a==b
assert after['completion_plan']['completed_product_steps']==before['completion_plan']['completed_product_steps']
assert after['completion_plan']['blocked_steps']==before['completion_plan']['blocked_steps']
assert after['checkpoints']['final_package_qualified'] is False and after['checkpoints']['user_acceptance']=='not_recorded'
plan=read('qualification/completion-50.json');p_before=json.loads(plan)
plan=setline(plan,'checkpoint',REPORT,2);plan=setline(plan,'last_tested_source',tested,2)
for id in ['DCM-28','DCM-29']:
 step=copy.deepcopy(next(x for x in p_before['steps'] if x['id']==id))
 step['readable_retry_native_test']=step['native_test'];step['native_test']=summary
 step['ui_test']='queued_layout_resize_path_corrected_and_verified; broader_provider_acceptance_pending'
 step['evidence'] += [REPORT,EVIDENCE];plan=itemline(plan,id,step)
p_after=json.loads(plan)
assert len(p_after['steps'])==50
for a,b in zip(p_before['steps'],p_after['steps']):
 assert a['completion']==b['completion'] and a['user_test']==b['user_test']
 if a['id'] not in ['DCM-28','DCM-29']:assert a==b
assert sum(x['completion']=='complete' for x in p_after['steps'])==26
for n,s in [('todo.json',todo),('qualification/completion-50.json',plan)]:
 (ROOT/n).write_bytes(s.encode());print(n,hashlib.sha256(s.encode()).hexdigest())
