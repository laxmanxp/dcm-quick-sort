"""Save measured attachment costs without closing performance or numbered gates."""
from pathlib import Path
import hashlib,json,re,copy
ROOT=Path(__file__).resolve().parents[1]
TESTED='5294d6cf32e84038aeed93750800bd7308bad7ad'
REPORT='qualification/2026-11-13-sumatra-attachment-profile.md'
EVIDENCE='qualification/evidence/sumatra-attachment-profile-summary.json'
SUMMARY='5294d6cf: opt-in phase profile preserves full layout. Two four-open measurements complete; run A queue297/327ms exceeds provisional250ms, unchanged B140-246ms. Layout114-246ms plus host zoom-state38-88ms account for86-89% of recorded attachment handler. No performance fix. Baseline373/0/1, pending-close3, lifecycle11, resize23+12bursts pass. Four first-run full images reviewed. No numbered closure/default switch/release.'
def blob(b):return hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()
def line(s,key,value,indent):
 p=re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$',re.M);found=list(p.finditer(s));assert len(found)==1,(key,len(found))
 return p.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+(',' if found[0].group().endswith(',') else ''),s)
def itemline(s,id,value):
 found=[x for x in s.splitlines() if x.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(found)==1,(id,len(found))
 old=found[0];return s.replace(old,'    '+json.dumps(value,ensure_ascii=False)+(',' if old.endswith(',') else ''))
def transform(todo,plan):
 t=json.loads(todo);p=json.loads(plan)
 todo=line(todo,'checkpoint',REPORT,4);todo=line(todo,'last_native_attempt',SUMMARY,4)
 old=t['checkpoints']['app_baseline_checkpoint'];new=dict(old,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261113-032229-c7441c1d',count=374,passed=373)
 current=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
 assert 'app_baseline_before_attachment_profile' not in t['checkpoints']
 todo=todo.replace(current,'    "app_baseline_before_attachment_profile": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
 record={'commit':TESTED,'diagnostic_runs':2,'opens_per_run':4,'all_geometry_calls_retained':True,'max_queue_ms_by_run':[[218.4492,297.0317,327.4957,203.6431],[200.7428,246.485,167.4977,139.725]],'performance_fixed':False,'performance_accepted':False,'layout_ms_range':[113.56,246.036],'host_zoom_update_ms_range':[38.038,87.91],'full_images_reviewed':4,'evidence':EVIDENCE,'numbered_steps_closed':[]}
 anchor='    "final_package_qualified": false,';assert todo.count(anchor)==1
 todo=todo.replace(anchor,'    "sumatra_attachment_profile_checkpoint": '+json.dumps(record)+',\n'+anchor)
 x=copy.deepcopy(next(x for x in t['items'] if x['id']=='DCM-SUMATRA-STARTUP-01'))
 x['pre_attachment_native_test']=x['native_test'];x['native_test']=SUMMARY
 x['progress']='async_creation_retained; remaining_attachment_control_costs_measured; broader_latency_gate_open'
 x['ui_test']='phase_diagnostic_complete_twice; two_over_budget_samples_retained; safety_regressions_pass'
 x['attachment_profile']=record;x['evidence_refs'] += [REPORT,EVIDENCE]
 assert x['performance_accepted'] is False
 todo=itemline(todo,x['id'],x)
 a=json.loads(todo)
 assert len(a['items'])==len(t['items'])
 for old,new in zip(t['items'],a['items']):
  assert old['user_test']==new['user_test']
  if old['id']!='DCM-SUMATRA-STARTUP-01': assert old==new
 assert a['completion_plan']['completed_product_steps']==t['completion_plan']['completed_product_steps']
 assert a['completion_plan']['blocked_steps']==t['completion_plan']['blocked_steps']
 assert a['checkpoints']['final_package_qualified'] is False
 assert a['checkpoints']['user_acceptance']==t['checkpoints']['user_acceptance']
 plan=line(plan,'checkpoint',REPORT,2);plan=line(plan,'last_tested_source',TESTED,2)
 for id in ['DCM-28','DCM-29']:
  x=copy.deepcopy(next(x for x in p['steps'] if x['id']==id));x['attachment_profile']=record;x['evidence'] += [REPORT,EVIDENCE];plan=itemline(plan,id,x)
 a=json.loads(plan);assert len(a['steps'])==50
 for old,new in zip(p['steps'],a['steps']):
  assert old['completion']==new['completion'] and old['user_test']==new['user_test']
  if old['id'] not in ['DCM-28','DCM-29']: assert old==new
 assert sum(x['completion']=='complete' for x in a['steps'])==26
 return todo,plan
if __name__=='__main__':
 expected={'todo.json': '9d7f8fefd10b20f01ee7643930c3d43fbd565a08', 'qualification/completion-50.json': '4cfbbb3dc980f2c2bd1bb762b63f2f88ec2e2a65'}
 raw={name:(ROOT/name).read_bytes() for name in expected}
 for name,b in raw.items():assert blob(b)==expected[name],(name,'source changed')
 for name,s in zip(expected,transform(raw['todo.json'].decode(),raw['qualification/completion-50.json'].decode())):
  assert hashlib.sha256(s.encode()).hexdigest()=={'todo.json': '3b2a93dd89d017a349547f6ac3246147f684df61d7e0ef4e55e88b89108d1072', 'qualification/completion-50.json': '2b12578db639ff4cad4326179373099442dc41251a6a04d65c4078839dd633ad'}[name],(name,'output differs')
  (ROOT/name).write_bytes(s.encode());print(name,hashlib.sha256(s.encode()).hexdigest())
