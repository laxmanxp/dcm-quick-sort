"""Record bounded readable-scroll acceptance while preserving all earlier gates."""
from pathlib import Path
import copy,hashlib,json,re
ROOT=Path(__file__).resolve().parents[1]
REPORT='qualification/2026-11-13-sumatra-scroll.md'
EVIDENCE='qualification/evidence/sumatra-scroll-native-summary.json'
EXPECTED={'todo.json':'73b33a5f4a8959116da9266bfcdb2dedadd2dedb','qualification/completion-50.json':'e70437746e855fa8537cc95f592b1cca6d251eb1'}
e=json.loads((ROOT/EVIDENCE).read_text());tested=e['tested_commit'];summary=e['summary']
assert tested=='401b44521ed36c835c04f39ffcd79ef6f3ee79af'
assert not e['product_changed'] and not e['numbered_steps_closed']
assert e['review']['individually_inspected']==12 and all(x['reviewed'] for x in e['review']['names_and_hashes'])
for k in ['scroll_first','scroll_repeat']:assert e['runs'][k]['passed'] and e['runs'][k]['case_count']==14
assert e['runs']['resize_regression']['passed'] and e['runs']['resize_regression']['all_sample_differences']==0

def read(n):
 b=(ROOT/n).read_bytes();assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==EXPECTED[n],(n,'base changed; reconcile');return b.decode()
def setline(s,k,v,i):
 p=re.compile(r'^'+(' '*i)+re.escape(json.dumps(k))+r':.*$',re.M);m=list(p.finditer(s));assert len(m)==1,(k,len(m))
 return p.sub(lambda _: ' '*i+json.dumps(k)+': '+json.dumps(v,ensure_ascii=False)+(',' if m[0].group().endswith(',') else ''),s)
def itemline(s,id,v):
 lines=[l for l in s.splitlines() if l.lstrip().startswith('{"id": '+json.dumps(id)+',')];assert len(lines)==1,(id,len(lines))
 return s.replace(lines[0],'    '+json.dumps(v,ensure_ascii=False)+(',' if lines[0].endswith(',') else ''))

todo=read('todo.json');before=json.loads(todo)
todo=setline(todo,'updated_on','2026-11-13',2)
for k,v in [('checkpoint',REPORT),('last_native_attempt',summary)]:todo=setline(todo,k,v,4)
old=before['checkpoints']['app_baseline_checkpoint'];b=e['baseline']
line=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
assert 'app_baseline_before_scroll_checkpoint' not in before['checkpoints']
new=dict(old,commit=tested,report=e['report_root']+'\\'+b['report'],count=b['count'],passed=b['passed'])
todo=todo.replace(line,'    "app_baseline_before_scroll_checkpoint": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
cp={'commit':tested,'selector':'ui-sumatra-scroll','passed_runs':2,'records_per_run':14,'reviewed_full_images':12,'product_changed':False,'fixture_sha256':e['fixture']['sha256'],'resize_regression_records':23,'resize_regression_bursts':12,'resize_sample_differences':0,'review_evidence':EVIDENCE,'default_changed':False}
needle='    "final_package_qualified": false,';assert todo.count(needle)==1
todo=todo.replace(needle,'    "sumatra_scroll_checkpoint": '+json.dumps(cp)+',\n'+needle)
item=copy.deepcopy(next(x for x in before['items'] if x['id']=='DCM-SUMATRA-01'))
item['before_readable_scroll']={k:item[k] for k in ['status','ui_test','native_test'] if k in item}
item['native_test']=summary;item['ui_test']='readable_100_percent_and_FitWidth_scroll_pass_twice; startup_presentation_and_release_gates_open'
item['evidence_refs'] += [REPORT,EVIDENCE];todo=itemline(todo,item['id'],item)
newitem={'id':'DCM-SUMATRA-SCROLL-01','feature':'Verify readable content after scrolling at 100 percent and Fit Width','status':'tested','ui_test':'14_records_pass_twice; all12_first_run_full_images_reviewed','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-29'],'acceptance':'Using the exact optional provider, expose readable top/middle/bottom and reverse-scroll content on page1, plus distinct page2 top/bottom; validate scroll/page identity and source bytes, then review actual headings and digits. Native scroll messages and one fixture are not continuous-mode or physical-input acceptance.','native_test':summary,'evidence_refs':[REPORT,EVIDENCE]}
assert not any(x['id']==newitem['id'] for x in before['items'])
needle='\n  ]\n}';assert todo.count(needle)==1
todo=todo.replace(needle,',\n    '+json.dumps(newitem)+needle)
after=json.loads(todo)
assert len(after['items'])==len(before['items'])+1
for a,b in zip(before['items'],after['items']):
 assert a['id']==b['id'] and a['user_test']==b['user_test']
 if a['id']!='DCM-SUMATRA-01':assert a==b
assert after['completion_plan']['completed_product_steps']==before['completion_plan']['completed_product_steps']
assert after['completion_plan']['blocked_steps']==before['completion_plan']['blocked_steps']
assert after['checkpoints']['candidate_code_commit']==before['checkpoints']['candidate_code_commit']
assert after['checkpoints']['final_package_qualified'] is False and after['checkpoints']['user_acceptance']=='not_recorded'
plan=read('qualification/completion-50.json');p_before=json.loads(plan)
for k,v in [('updated_on','2026-11-13'),('checkpoint',REPORT),('last_tested_source',tested)]:plan=setline(plan,k,v,2)
step=copy.deepcopy(next(x for x in p_before['steps'] if x['id']=='DCM-29'))
step['resize_timing_native_test']=step['native_test'];step['native_test']=summary
step['ui_test']='optional_100_percent_FitWidth_readable_scroll_verified; broader_provider_acceptance_pending'
step['evidence'] += [REPORT,EVIDENCE];plan=itemline(plan,'DCM-29',step)
p_after=json.loads(plan)
assert len(p_after['steps'])==50
for a,b in zip(p_before['steps'],p_after['steps']):
 assert a['completion']==b['completion'] and a['user_test']==b['user_test']
 if a['id']!='DCM-29':assert a==b
assert sum(x['completion']=='complete' for x in p_after['steps'])==26
for n,s in [('todo.json',todo),('qualification/completion-50.json',plan)]:
 (ROOT/n).write_bytes(s.encode());print(n,hashlib.sha256(s.encode()).hexdigest())
