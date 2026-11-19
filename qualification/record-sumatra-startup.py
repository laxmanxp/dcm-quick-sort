"""Save measured startup failure; preserve every numbered/user completion state."""
from pathlib import Path
import json,hashlib,copy,re
ROOT=Path(__file__).resolve().parents[1]
TESTED='497dbb29c66092db690c3ba501501a09f9acb64e'
REPORT='qualification/2026-11-13-sumatra-startup.md'
EVIDENCE='qualification/evidence/sumatra-startup-summary.json'
SUMMARY='497dbb29: two diagnostic runs complete four PDF starts each with files/cleanup intact. First ready status2389/1874ms; posted queue acknowledgement1758/1497ms. CreateProcessW interval1695/1433ms dominates; repeat creation14-19ms. Performance NOT fixed; first-open responsiveness exceeds provisional250ms target. Baseline370/0/1, scroll14 and resize23+12bursts pass. Four first-run full images reviewed. No numbered closure/default switch/release.'
def setline(s,key,value,indent):
 pattern=re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$',re.M);matches=list(pattern.finditer(s));assert len(matches)==1,(key,len(matches))
 return pattern.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+(',' if matches[0].group().endswith(',') else ''),s)
def itemline(s,id,item):
 matches=[line for line in s.splitlines() if line.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(matches)==1,(id,len(matches))
 line=matches[0];return s.replace(line,'    '+json.dumps(item,ensure_ascii=False)+(',' if line.endswith(',') else ''))
def transform(todo,plan):
 before=json.loads(todo);p_before=json.loads(plan)
 todo=setline(todo,'checkpoint',REPORT,4);todo=setline(todo,'last_native_attempt',SUMMARY,4)
 old=before['checkpoints']['app_baseline_checkpoint'];new=dict(old,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261113-010443-73f34848',count=371,passed=370)
 line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
 assert 'app_baseline_before_startup_checkpoint' not in before['checkpoints']
 todo=todo.replace(line,'    "app_baseline_before_startup_checkpoint": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
 record={'commit':TESTED,'diagnostic_runs':2,'starts_per_run':4,'measurement_complete':True,'performance_fixed':False,'responsiveness_accepted':False,'first_ready_ms':[2389.3218,1874.4617],'first_max_queue_ack_ms':[1757.7565,1497.2459],'first_create_process_ms':[1695.003,1432.646],'provisional_queue_budget_ms':250,'reviewed_full_images':4,'evidence':EVIDENCE,'default_changed':False,'numbered_gate_closed':False}
 anchor='    "final_package_qualified": false,';assert todo.count(anchor)==1
 todo=todo.replace(anchor,'    "sumatra_startup_checkpoint": '+json.dumps(record)+',\n'+anchor)
 item=copy.deepcopy(next(x for x in before['items'] if x['id']=='DCM-SUMATRA-STARTUP-01'))
 item['previous_observation']=item['native_test'];item['native_test']=SUMMARY;item['status']='tested';item['progress']='measured_responsiveness_failure; fix_pending';item['performance_accepted']=False;item['ui_test']='diagnostics_complete_twice; first-open_queue_delay_exceeds_provisional_budget'
 item['evidence_refs'] += [REPORT,EVIDENCE];todo=itemline(todo,item['id'],item)
 after=json.loads(todo)
 for a,b in zip(before['items'],after['items']):
  if a['id']!='DCM-SUMATRA-STARTUP-01': assert a==b
 assert len(after['items'])==len(before['items'])
 assert after['completion_plan']['completed_product_steps']==before['completion_plan']['completed_product_steps']
 assert after['checkpoints']['final_package_qualified'] is False
 assert after['checkpoints']['user_acceptance']==before['checkpoints']['user_acceptance']
 assert after['completion_plan']['blocked_steps']==before['completion_plan']['blocked_steps']
 plan=setline(plan,'checkpoint',REPORT,2);plan=setline(plan,'last_tested_source',TESTED,2)
 for id in ['DCM-28','DCM-29']:
  step=copy.deepcopy(next(x for x in p_before['steps'] if x['id']==id));step['startup_measurement']=record;step['evidence'] += [REPORT,EVIDENCE];plan=itemline(plan,id,step)
 p_after=json.loads(plan);assert len(p_after['steps'])==50
 for a,b in zip(p_before['steps'],p_after['steps']):
  assert a['completion']==b['completion'] and a['user_test']==b['user_test']
  if a['id'] not in ['DCM-28','DCM-29']:assert a==b
 assert sum(x['completion']=='complete' for x in p_after['steps'])==26
 return todo,plan
if __name__=='__main__':
 expected={'todo.json':'9662c99077587faa821176fa43d47df3c562bb6c','qualification/completion-50.json':'5759d8e7285d0bba80ce92e3ffb1b3e3c254d552'}
 raw={name:(ROOT/name).read_bytes() for name in expected}
 for name,b in raw.items():assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==expected[name],(name,'input changed; reconcile')
 outputs=transform(raw['todo.json'].decode(),raw['qualification/completion-50.json'].decode())
 for name,s in zip(expected,outputs):
  (ROOT/name).write_bytes(s.encode());print(name,hashlib.sha256(s.encode()).hexdigest())
