"""Record bounded native resize evidence without closing broader PDF gates."""
from pathlib import Path
import json, hashlib, copy, re
R=Path(__file__).resolve().parents[1]
TESTED='24ff2f243db4c5d8d29b8d64ce81b07c11112efe'
REPORT='qualification/2026-11-12-sumatra-resize.md'
EVIDENCE='qualification/evidence/sumatra-resize-native-summary.json'
EXPECTED={'todo.json':'b60fec161e289ffba83fafe7a45b481711415fe9','qualification/completion-50.json':'335479f336f137b8b133e7cc09b1048e7167e559'}
texts={}
for name,expected in EXPECTED.items():
 raw=(R/name).read_bytes()
 assert hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest()==expected,(name,'changed input; reconcile')
 texts[name]=raw.decode()
original={name:json.loads(s) for name,s in texts.items()}
def line(s,key,value,indent):
 p=re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$',re.M)
 matches=list(p.finditer(s));assert len(matches)==1,(key,len(matches))
 comma=',' if matches[0].group().endswith(',') else ''
 return p.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+comma,s)
summary='24ff2f24: two Sumatra runs pass 23 checks each; all 20 first-run unassisted full images reviewed. Repaint RED (2426/2171/2426 changed samples) becomes zero, and Aa is visibly contained at three sizes. Baseline 367/0/1; text17/image23/launch12 regressions pass. Narrow filter/Processed-label crowding and broader lifecycle remain open. No numbered closure, default switch or release.'
s=texts['todo.json'];before=original['todo.json']
for key,value,indent in [('checkpoint',REPORT,4),('candidate_code_commit',TESTED,4),('last_native_attempt',summary,4)]:s=line(s,key,value,indent)
base=before['checkpoints']['app_baseline_checkpoint']
assert 'app_baseline_sumatra_toolbar_checkpoint' not in before['checkpoints']
old=next(x for x in s.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
new=dict(base,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261112-082052-d55bc2d2',count=368,passed=367)
s=s.replace(old,'    "app_baseline_sumatra_toolbar_checkpoint": '+json.dumps(base)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
checkpoint={'commit':TESTED,'report':r'D:\FlowverseReportsRepo\dcm-quicksort-ui-sumatra-20261112-081912-e85ac86f','repeat_report':r'D:\FlowverseReportsRepo\dcm-quicksort-ui-sumatra-20261112-082224-ce406ad4','passed':True,'checks_per_run':23,'reviewed_unassisted_full_images':20,'source_unchanged':True,'fixtures_preserved':True,'host_exit_code':0,'defects_fixed_at_tested_sizes':['DCM-SUMATRA-UI-02','DCM-UI-RESIZE-01'],'raw_repaint_differences':[0,0,0,0],'numbered_steps_closed':[],'default_changed':False,'review_evidence':EVIDENCE}
needle='    "final_package_qualified": false,';assert s.count(needle)==1
s=s.replace(needle,'    "sumatra_resize_checkpoint": '+json.dumps(checkpoint)+',\n'+needle)
needle='  "evidence": {\n';assert s.count(needle)==1
s=s.replace(needle,needle+'    "sumatra_resize_checkpoint": '+json.dumps(REPORT)+',\n')
changed_ids={'DCM-SUMATRA-01','DCM-SUMATRA-UI-02','DCM-UI-RESIZE-01','DCM-UI-LABEL-01'}
# Existing item JSON occupies either one compact line or an indented block.
for item in before['items']:
 if item['id'] not in changed_ids:continue
 revised=copy.deepcopy(item)
 revised.setdefault('evidence_refs',[]).extend([REPORT,EVIDENCE])
 if item['id']=='DCM-SUMATRA-01':
  revised['toolbar_native_test']=revised['native_test']
  revised.update(ui_test='23_checks_pass_twice; resize_defects_fixed_at_three_sizes; broader_lifecycle_and_general_labels_open',native_test=summary)
 elif item['id']=='DCM-SUMATRA-UI-02':
  revised['prior_ui_test']=revised['ui_test']
  revised.update(status='fixed',ui_test='Aa_visibly_contained_at_1280_1216_1152_host_widths; no_universal_layout_claim',native_test='24ff2f24 reserves preview width via bounded side panels. All 20 first-run full captures reviewed; Aa and other rightmost icons visible. MSAA geometry underreported prior clipping and is supplementary only. Narrow Processed header is more compressed; DCM-UI-LABEL-01 remains open.')
 elif item['id']=='DCM-UI-RESIZE-01':
  revised['prior_ui_test']=revised['ui_test']
  revised.update(status='fixed',ui_test='native_repaint_comparison_RED_to_GREEN; unassisted_full_captures_reviewed',native_test='1b5cadca RED had 2426/2171/2426 changed samples against explicit redraw reference after restore/repeated resize; 24ff2f24 has zero across 107276 sampled host points, repeated twice. App queues invalidation after final control layout. Diagnostic references occur after raw captures, not before acceptance. Tested matrix only; fixture contents unchanged.')
 else:
  revised['resize_observation']='At narrowed Sumatra host sizes, Size/Sort controls overlap and borrowing side-panel width further compresses the Processed heading. These remain open under DCM-46; toolbar/repaint corrections do not approve all labels.'
 # Find exact item's structural span without guessing brace nesting.
 start=s.index('"items": ['); decoder=json.JSONDecoder();pos=s.index('[',start)+1
 while True:
  while s[pos].isspace() or s[pos]==',':pos+=1
  if s[pos]==']':raise AssertionError('item missing')
  value,end=decoder.raw_decode(s,pos)
  if value['id']==item['id']:
   s=s[:pos]+json.dumps(revised,ensure_ascii=False)+s[end:];break
  pos=end
texts['todo.json']=s
p=texts['qualification/completion-50.json']
for key,value in [('checkpoint',REPORT),('last_tested_source',TESTED)]:p=line(p,key,value,2)
for step in original['qualification/completion-50.json']['steps']:
 if step['id'] not in ['DCM-28','DCM-29']:continue
 revised=copy.deepcopy(step)
 revised['toolbar_native_test']=revised['native_test']
 revised.update(ui_test='optional_Sumatra_resize_navigation_zoom_pass; full_provider_acceptance_pending',native_test=summary)
 revised['evidence'].extend([REPORT,EVIDENCE])
 old=next(x for x in p.splitlines() if '"id": "'+step['id']+'"' in x)
 p=p.replace(old,'    '+json.dumps(revised,ensure_ascii=False)+',')
texts['qualification/completion-50.json']=p
for name,text in texts.items():
 after=json.loads(text);before=original[name]
 if name=='todo.json':
  for a,b in zip(before['items'],after['items']):
   if a['id'] not in changed_ids:assert a==b
  assert len(before['items'])==len(after['items'])
  assert after['completion_plan']['completed_product_steps']==before['completion_plan']['completed_product_steps']
  assert after['completion_plan']['blocked_steps']==['DCM-19']
  assert after['checkpoints']['final_package_qualified'] is False
  assert after['checkpoints']['user_acceptance']=='not_recorded'
 else:
  assert len(after['steps'])==50
  for a,b in zip(before['steps'],after['steps']):
   assert a['completion']==b['completion'] and a['user_test']==b['user_test']
   if a['id'] not in ['DCM-28','DCM-29']:assert a==b
  assert sum(x['completion']=='complete' for x in after['steps'])==26
 (R/name).write_text(text,encoding='utf-8')
 print(name,hashlib.sha256(text.encode()).hexdigest())
