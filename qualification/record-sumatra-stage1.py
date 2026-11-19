from pathlib import Path
import json,hashlib,copy
root=Path(__file__).resolve().parents[1]
tested='b05024332f018b544f7d1e704c5e017fac66f364'
report='qualification/2026-11-12-sumatra-stage1.md'
evidence='qualification/evidence/sumatra-stage1-native-summary.json'
raw={n:(root/n).read_bytes() for n in ['todo.json','qualification/completion-50.json']}
expected={'todo.json':'95fb79bd27baaee7e3aab309ecc09209a646d14f','qualification/completion-50.json':'407dab2a5a0342dea69133007521ad4a312e7f44'}
for n,b in raw.items():
 assert hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()==expected[n], (n,'base changed')
# Preserve original formatting wherever possible; use line replacement for compact entries.
txt={n:b.decode() for n,b in raw.items()}
def setline(s,key,value,spaces):
 import re
 pattern=re.compile(r'^'+ ' '*spaces + re.escape(json.dumps(key))+r':.*$',re.M)
 matches=list(pattern.finditer(s)); assert len(matches)==1,(key,len(matches))
 comma=',' if matches[0].group().endswith(',') else ''
 return pattern.sub(lambda _: ' '*spaces+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+comma,s)
t=json.loads(txt['todo.json']); p=json.loads(txt['qualification/completion-50.json'])
txt['todo.json']=setline(txt['todo.json'],'updated_on','2026-11-12',2)
for key,value in [('checkpoint',report),('last_native_attempt','b0502433: opt-in Sumatra stage-1 UI passes eight behavior checks; six full screenshots and five pixel crops inspected. Baseline 365 passed / 0 failed / 1 retained external-PPTX skip. Default-provider text/image regressions pass. Toolbar page field clipped; DCM-28/29 remain pending; no new numbered completion or default switch.'),('candidate_code_commit',tested)]:
 txt['todo.json']=setline(txt['todo.json'],key,value,4)
oldbase=t['checkpoints']['app_baseline_checkpoint']
newbase=dict(oldbase,commit=tested,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261111-234138-747a8dff',count=366,passed=365)
needle=next(l for l in txt['todo.json'].splitlines() if l.startswith('    "app_baseline_checkpoint":'))
assert 'app_baseline_before_sumatra_checkpoint' not in t['checkpoints']
txt['todo.json']=txt['todo.json'].replace(needle,'    "app_baseline_before_sumatra_checkpoint": '+json.dumps(oldbase)+',\n    "app_baseline_checkpoint": '+json.dumps(newbase)+',')
entry={'commit':tested,'report':r'D:\FlowverseReportsRepo\dcm-quicksort-ui-sumatra-20261111-233859-93834514','passed':True,'behavior_cases':8,'reviewed_full_images':6,'reviewed_pixel_crops':5,'app_exit_code':0,'default_changed':False,'packaged':False,'numbered_gate_closed':False,'open_visual_issue':'DCM-SUMATRA-UI-01','review_evidence':evidence}
needle='    "final_package_qualified": false,'; assert txt['todo.json'].count(needle)==1
txt['todo.json']=txt['todo.json'].replace(needle,'    "sumatra_stage1_checkpoint": '+json.dumps(entry)+',\n'+needle)
# Append issues without rewriting prior items, using final array terminator.
issues=[
 {'id':'DCM-SUMATRA-01','feature':'Optional embedded Sumatra PDF provider approved by user','status':'implemented','ui_test':'stage1_native_pass_with_open_toolbar_issue','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-29'],'acceptance':'Retain default Windows.Data.Pdf while qualifying a pinned separate Sumatra process with deny-write/deny-print policy, own-process release before operations, actual page/zoom/resize checks, failure/cancel/recovery, Undo and source preservation. Only then consider default selection and packaging.','native_test':'b0502433: 8 behavior checks pass. Both known PDF pages and last-page boundary visually verified, internal Open blocked, text switch releases PDF, representative Rename/Move retain bytes, owned processes gone after normal exit. 6 full images and 5 crops reviewed. No claim of complete provider or numbered-gate acceptance.','evidence_refs':[report,evidence]},
 {'id':'DCM-SUMATRA-UI-01','feature':'Fix clipped embedded Sumatra page-number field and resolve redundant disabled host PDF controls','status':'planned','ui_test':'observed_in_stage1_screenshots','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-28','DCM-29'],'acceptance':'Keep PrinterAccess=0 and DiskAccess=0; make page controls completely readable/clickable and qualify page/zoom controls at tested DPI/window sizes. Do not enable printing merely to restore toolbar geometry.','evidence_refs':[report,evidence]},
 {'id':'DCM-SUMATRA-DISTRIBUTION-01','feature':'Qualify managed Sumatra runtime packaging and applicable distribution terms','status':'planned','ui_test':'not_applicable','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-47','DCM-48'],'acceptance':'Before bundling or publishing, verify exact runtime source/licence notices and applicable source distribution obligations, installer payload identity, private policy installation and upgrade behavior. Stage 1 copies an existing test runtime only; no executable is bundled or released.','evidence_refs':[report]}
]
assert '\n  ]\n}' in txt['todo.json']
assert all(i['id'] not in [x['id'] for x in t['items']] for i in issues)
txt['todo.json']=txt['todo.json'].replace('\n  ]\n}',',\n    '+',\n    '.join(json.dumps(i,ensure_ascii=False) for i in issues)+'\n  ]\n}')
for k,v in [('updated_on','2026-11-12'),('checkpoint',report),('last_tested_source',tested)]:
 txt['qualification/completion-50.json']=setline(txt['qualification/completion-50.json'],k,v,2)
step=copy.deepcopy(next(x for x in p['steps'] if x['id']=='DCM-28'))
step.update(status='implemented',completion='pending',ui_test='partial_optional_provider_stage1',evidence=[report,evidence],native_test='User-approved Sumatra provider stage 1: 8 checks pass at b0502433, 11 captures reviewed. First-page boundary, broader page UI/zoom/resize/lifecycle acceptance and page-field clipping remain open. DCM-28 is not complete.')
line=next(l for l in txt['qualification/completion-50.json'].splitlines() if '"id": "DCM-28"' in l)
txt['qualification/completion-50.json']=txt['qualification/completion-50.json'].replace(line,'    '+json.dumps(step,ensure_ascii=False)+',')
for name in txt:
 o=json.loads(txt[name])
 if name=='todo.json':
  assert o['items'][:-len(issues)]==t['items']
  assert o['completion_plan']['completed_product_steps']==t['completion_plan']['completed_product_steps']
  assert o['checkpoints']['final_package_qualified'] is False
 else:
  assert len(o['steps'])==50
  for a,b in zip(p['steps'],o['steps']):
   if a['id']!='DCM-28': assert a==b
  assert sum(s['completion']=='complete' for s in o['steps'])==26
 output={'todo.json':'ab31c4cd925a76c2ebdb5428f1082ff677149d35825b40889924b7c4b8635207','qualification/completion-50.json':'5391440831e07c9e18bb6b7cf7f3db5ec0fa251f12a364b8cef25f6acdb9993b'}
 assert hashlib.sha256(txt[name].encode()).hexdigest()==output[name], (name,'output differs')
 (root/name).write_text(txt[name],encoding='utf-8')
 print('OUTPUT',name,hashlib.sha256(txt[name].encode()).hexdigest())
