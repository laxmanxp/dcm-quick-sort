"""Record bounded PDF error evidence; preserve failed UI repeats and all numbered gates."""
from pathlib import Path
import copy, hashlib, json, re
ROOT=Path(__file__).resolve().parents[1]
TESTED='39af1659c01cbfa6bdb38106370197001027e946'
REPORT='qualification/2026-11-12-sumatra-errors.md'
EVIDENCE='qualification/evidence/sumatra-errors-native-summary.json'
EXPECTED={'todo.json':'f798fb6064fa8c9294c7c5aa37794763d8af2296','qualification/completion-50.json':'25ad3151413adc9284c9adb49750bcece7d1ca51'}
SUMMARY='39af1659: errors workflow12 records PASS once; all10 captures reviewed (6host,1modal,3content). Damaged-file false readiness and password-prompt startup timeout reproduced and fixed. Missing-runtime restoration, wrong/correct password,17s prompt hold,cancel/text recovery and file safety pass. Baseline369/0/1. Later navigation/lifecycle/error-repeat attempts failed visibility capture guards, not accepted. Long error paths clip. No numbered closure/default switch/release.'
def read(n):
 b=(ROOT/n).read_bytes();assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==EXPECTED[n],(n,'input changed');return b.decode()
def setline(s,k,v,i):
 p=re.compile(r'^'+(' '*i)+re.escape(json.dumps(k))+r':.*$',re.M);m=list(p.finditer(s));assert len(m)==1,(k,len(m))
 return p.sub(lambda _: ' '*i+json.dumps(k)+': '+json.dumps(v,ensure_ascii=False)+(',' if m[0].group().endswith(',') else ''),s)
def itemline(s,id,v):
 lines=[l for l in s.splitlines() if l.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(lines)==1,(id,len(lines))
 return s.replace(lines[0],'    '+json.dumps(v,ensure_ascii=False)+(',' if lines[0].endswith(',') else ''))
todo=read('todo.json');before=json.loads(todo)
todo=setline(todo,'checkpoint',REPORT,4);todo=setline(todo,'last_native_attempt',SUMMARY,4);todo=setline(todo,'candidate_code_commit',TESTED,4)
oldbase=before['checkpoints']['app_baseline_checkpoint'];line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
newbase=dict(oldbase,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261112-141022-88e2d46f',count=370,passed=369)
assert 'app_baseline_before_errors_checkpoint' not in before['checkpoints']
todo=todo.replace(line,'    "app_baseline_before_errors_checkpoint": '+json.dumps(oldbase)+',\n    "app_baseline_checkpoint": '+json.dumps(newbase)+',')
checkpoint={'commit':TESTED,'selector':'ui-sumatra-errors','passed_runs':1,'cases':12,'reviewed_captures':10,'host_windows':6,'password_dialogs':1,'content_crops':3,'exit_code':0,'password_wait_ms':17000,'password_prompt_survived':True,'new_status_parent':'TestSumatraDocumentStatusDoesNotClaimUnloadedPages','table_entries':9,'repeat_and_regression_status':'not_accepted; visibility_capture_guard_failures','review_evidence':EVIDENCE}
needle='    "final_package_qualified": false,';assert todo.count(needle)==1
todo=todo.replace(needle,'    "sumatra_errors_checkpoint": '+json.dumps(checkpoint)+',\n'+needle)
item=copy.deepcopy(next(x for x in before['items'] if x['id']=='DCM-SUMATRA-01'))
assert 'undo_native_test' not in item
item['undo_native_test']=item['native_test'];item['native_test']=SUMMARY
item['ui_test']='errors_pass_once_with_image_review; UI_repeat_and_regressions_visibility_blocked; broader_gates_open'
item['evidence_refs']+=[REPORT,EVIDENCE];todo=itemline(todo,item['id'],item)
issues=[
 {'id':'DCM-SUMATRA-DOCUMENT-STATE-01','feature':'Do not advertise successful PDF preview merely because a viewer attached','status':'fixed','severity':'low','ui_test':'native_damaged_file_RED_and_one_reviewed_GREEN; repeat_incomplete','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-29'],'acceptance':'A header-valid unloadable PDF reaches the decoder and shows a truthful no-page host state rather than ready navigation/zoom. Preserve native error, file bytes and text recovery. Pinned page-field observation is not a general PDF parser.','native_test':SUMMARY,'evidence_refs':[REPORT,EVIDENCE]},
 {'id':'DCM-SUMATRA-PASSWORD-01','feature':'Keep an owned password prompt alive while waiting for user input','status':'fixed','severity':'medium','ui_test':'native17s_timeout_RED_to_surviving_prompt_GREEN; known_fixture_cancel_wrong_correct_password_pass','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-29'],'acceptance':'Do not kill a visible owned password prompt on the15s attachment deadline; show password-required state, retain bounded startup after prompt closes, allow cancel/text recovery and known correct password rendering. Host must not read password text. One encryption fixture and17s wait are bounded evidence, not all encryption/modal variants.','native_test':SUMMARY,'evidence_refs':[REPORT,EVIDENCE]},
 {'id':'DCM-SUMATRA-ERROR-COPY-01','feature':'Make missing-runtime guidance and long error paths readable','status':'planned','severity':'low','finding_kind':'reviewed_UI_observation','ui_test':'long_path_clipping_observed_in_missing_runtime_and_native_decoder_banner','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-46'],'acceptance':'Provide concise truthful missing-runtime guidance and readable path/detail presentation at supported geometry. Retain actual failure cause and recovery behavior; do not confuse functional error detection with complete visual approval.','evidence_refs':[REPORT,EVIDENCE]}
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
 step=copy.deepcopy(next(x for x in pb['steps'] if x['id']==id));assert 'undo_native_test' not in step
 step['undo_native_test']=step['native_test'];step['native_test']=SUMMARY
 step['ui_test']='optional_errors_pass_once; repeat_and_regression_UI_capture_failures; remaining_acceptance_pending'
 step['evidence']+=[REPORT,EVIDENCE];plan=itemline(plan,id,step)
pa=json.loads(plan);assert len(pa['steps'])==50
for a,b in zip(pb['steps'],pa['steps']):
 assert a['completion']==b['completion'] and a['user_test']==b['user_test']
 if a['id'] not in ['DCM-28','DCM-29']:assert a==b
assert sum(x['completion']=='complete' for x in pa['steps'])==26
for n,s in [('todo.json',todo),('qualification/completion-50.json',plan)]:
 (ROOT/n).write_bytes(s.encode());print(n,hashlib.sha256(s.encode()).hexdigest())
