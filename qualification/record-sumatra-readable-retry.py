"""Record the verified retry without hiding the intermittent navigation failure."""
from pathlib import Path
import copy, hashlib, json, re
ROOT=Path(__file__).resolve().parents[1]
REPORT='qualification/2026-11-12-sumatra-readable-retry.md'
EVIDENCE='qualification/evidence/sumatra-readable-retry-summary.json'
EXPECTED={'todo.json':'50d317807daa792db4055ded0a9e75c356c0250a','qualification/completion-50.json':'eff3b0db2271d6de5e35a05cd9737024936feb49'}
e=json.loads((ROOT/EVIDENCE).read_text());summary=e['summary'];tested=e['tested_commit']
assert tested=='715cd61d7da00ee0125eb006246e77711411acc2'
assert e['runs']['navigation_resize_failure']['passed'] is False
assert e['runs']['navigation_resize_repeat']['passed'] is True

def read(n):
 b=(ROOT/n).read_bytes();assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==EXPECTED[n],(n,'input changed; reconcile');return b.decode()
def setline(s,k,v,i):
 p=re.compile(r'^'+(' '*i)+re.escape(json.dumps(k))+r':.*$',re.M);m=list(p.finditer(s));assert len(m)==1,(k,len(m))
 return p.sub(lambda _: ' '*i+json.dumps(k)+': '+json.dumps(v,ensure_ascii=False)+(',' if m[0].group().endswith(',') else ''),s)
def itemline(s,id,v):
 lines=[l for l in s.splitlines() if l.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(lines)==1,(id,len(lines))
 return s.replace(lines[0],'    '+json.dumps(v,ensure_ascii=False)+(',' if lines[0].endswith(',') else ''))

todo=read('todo.json');before=json.loads(todo)
todo=setline(todo,'checkpoint',REPORT,4);todo=setline(todo,'last_native_attempt',summary,4)
oldbase=before['checkpoints']['app_baseline_checkpoint']
assert 'app_baseline_before_readable_retry_checkpoint' not in before['checkpoints']
line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
b=e['baseline'];newbase=dict(oldbase,commit=tested,report=e['report_root']+'\\'+b['report'],count=b['count'],passed=b['passed'])
todo=todo.replace(line,'    "app_baseline_before_readable_retry_checkpoint": '+json.dumps(oldbase)+',\n    "app_baseline_checkpoint": '+json.dumps(newbase)+',')
checkpoint={'commit':tested,'errors_passed_runs':2,'error_cases_per_run':12,'reviewed_error_images':10,'lifecycle_passed':11,'navigation_first_run':'failed_repaint_comparison','navigation_unchanged_retry_passed':23,'resize_issue':'reopened_intermittent','source_changed_in_retry':False,'review_evidence':EVIDENCE}
needle='    "final_package_qualified": false,';assert todo.count(needle)==1
todo=todo.replace(needle,'    "sumatra_readable_retry_checkpoint": '+json.dumps(checkpoint)+',\n'+needle)
changed={'DCM-SUMATRA-01','DCM-SUMATRA-ERROR-COPY-01','DCM-UI-RESIZE-01'}
for id in changed:
 item=copy.deepcopy(next(x for x in before['items'] if x['id']==id))
 item['before_readable_retry']={k:item[k] for k in ['status','ui_test','native_test'] if k in item}
 item['evidence_refs'] += [REPORT,EVIDENCE]
 if id=='DCM-SUMATRA-01':
  item['native_test']=summary
  item['ui_test']='error12_twice_and_lifecycle11_pass; navigation_intermittent_repaint_failure_then23_pass; acceptance_open'
 elif id=='DCM-SUMATRA-ERROR-COPY-01':
  item['status']='implemented';item['resolution_state']='partial'
  item['ui_test']='missing_runtime_message_fully_readable_in_reviewed_capture; native_damaged_file_banner_path_still_clips'
  item['native_test']='At715cd61d two fresh12-record error runs pass; all10 first-run captures reviewed. Missing-runtime short message and recovery guidance are fully visible; original OS/path cause retained by Unwrap. Native damaged-file long-path banner remains clipped, so this combined issue is not closed.'
 else:
  item['status']='implemented';item['resolution_state']='reopened_intermittent'
  item['ui_test']='raw_repeated_resize_capture_failed; unchanged_retry_passed; root_cause_open'
  item['native_test']='At715cd61d report152144-1254fe66 fails with32603/107276 repaint differences and1746 background mismatches. Three raw/adjacent/reference images reviewed and actual incomplete host controls confirmed. Unchanged report152939-74d333d0 passes23 checks with zero paint differences. Retain both outcomes; later pass is not a correction. Earlier24ff2f24 evidence remains historical.'
 todo=itemline(todo,id,item)
after=json.loads(todo)
assert len(after['items'])==len(before['items'])
for a,b in zip(before['items'],after['items']):
 assert a['id']==b['id'] and a['user_test']==b['user_test']
 if a['id'] not in changed:assert a==b
assert after['completion_plan']['completed_product_steps']==before['completion_plan']['completed_product_steps']
assert after['completion_plan']['blocked_steps']==before['completion_plan']['blocked_steps']
assert after['checkpoints']['candidate_code_commit']==before['checkpoints']['candidate_code_commit']
assert after['checkpoints']['final_package_qualified'] is False
assert after['checkpoints']['user_acceptance']=='not_recorded'

plan=read('qualification/completion-50.json');p_before=json.loads(plan)
plan=setline(plan,'checkpoint',REPORT,2);plan=setline(plan,'last_tested_source',tested,2)
for id in ['DCM-28','DCM-29']:
 step=copy.deepcopy(next(x for x in p_before['steps'] if x['id']==id))
 step['errors_native_test']=step['native_test'];step['native_test']=summary
 step['ui_test']='errors_and_lifecycle_pass; navigation_intermittent_repaint_failure_then_pass; acceptance_pending'
 step['evidence'] += [REPORT,EVIDENCE]
 plan=itemline(plan,id,step)
p_after=json.loads(plan)
assert len(p_after['steps'])==50
for a,b in zip(p_before['steps'],p_after['steps']):
 assert a['completion']==b['completion'] and a['user_test']==b['user_test']
 if a['id'] not in ['DCM-28','DCM-29']:assert a==b
assert sum(x['completion']=='complete' for x in p_after['steps'])==26
for name,s in [('todo.json',todo),('qualification/completion-50.json',plan)]:
 (ROOT/name).write_bytes(s.encode());print(name,hashlib.sha256(s.encode()).hexdigest())
