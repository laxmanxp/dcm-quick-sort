"""Record bounded label acceptance without changing numbered or user gates."""
from pathlib import Path
import copy, hashlib, json, re
ROOT = Path(__file__).resolve().parents[1]
TESTED = '2cafe818b16ccb2c861ac1d5cd74731ab446c7db'
REPORT = 'qualification/2026-11-13-compact-labels.md'
EVIDENCE = 'qualification/evidence/compact-labels-summary.json'
SUMMARY = '2cafe818: compact filter wrapping, full Processed title/count row and wider form labels fix seven reproduced clipping findings. Two focused runs pass24 behavior records and20 label audits each at1264x761,1200x697,1136x697 logical sizes. Nine first-run full images plus RED reviewed. Baseline379/0/1; resize23+12bursts, startup2x4 within250ms, filters74 records,sort9,selection16 records,Undo9,scroll14,text17,image23,launch12 pass. Default Windows.Data.Pdf visible page/manual-zoom check remains the next DCM28/29 gap. No numbered/user completion or default change.'
def blob(b):
    return hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()
def line(s,key,value,indent):
    p=re.compile(r'^'+(' '*indent)+re.escape(json.dumps(key))+r':.*$',re.M)
    found=list(p.finditer(s)); assert len(found)==1,(key,len(found))
    return p.sub(lambda _: ' '*indent+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+(',' if found[0].group().endswith(',') else ''),s)
def itemline(s,id,value):
    found=[x for x in s.splitlines() if x.lstrip().startswith('{"id": '+json.dumps(id)+',')]
    assert len(found)==1,(id,len(found))
    return s.replace(found[0],'    '+json.dumps(value,ensure_ascii=False)+(',' if found[0].endswith(',') else ''))
def transform(todo,plan):
    t,p=json.loads(todo),json.loads(plan)
    todo=line(todo,'checkpoint',REPORT,4)
    todo=line(todo,'last_native_attempt',SUMMARY,4)
    todo=line(todo,'candidate_code_commit',TESTED,4)
    old=t['checkpoints']['app_baseline_checkpoint']
    new=dict(old,commit=TESTED,report=r'D:\FlowverseReportsRepo\dcm-quicksort-baseline-20261113-053940-7203938f',count=380,passed=379)
    current=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
    assert 'app_baseline_before_compact_labels' not in t['checkpoints']
    todo=todo.replace(current,'    "app_baseline_before_compact_labels": '+json.dumps(old)+',\n    "app_baseline_checkpoint": '+json.dumps(new)+',')
    record={'commit':TESTED,'status':'fixed_at_tested_geometry','focused_runs':2,'behavior_records_per_run':24,'label_audits_per_run':20,'native_logical_sizes':[[1264,761],[1200,697],[1136,697]],'reviewed_green_full_images':9,'reviewed_RED_images':1,'one_row_wide_layout':'unit_coverage_only','physical_display_scaling_tested':False,'final_candidate_recheck_required':True,'evidence':EVIDENCE,'numbered_steps_closed':[]}
    anchor='    "final_package_qualified": false,'; assert todo.count(anchor)==1
    todo=todo.replace(anchor,'    "compact_label_checkpoint": '+json.dumps(record)+',\n'+anchor)
    x=copy.deepcopy(next(x for x in t['items'] if x['id']=='DCM-UI-LABEL-01'))
    x['previous_ui_test']=x['ui_test']
    x['status']='fixed'
    x['ui_test']='two_24_record_runs_and_20_label_audits_each_pass; nine_green_images_reviewed'
    x['native_test']=SUMMARY
    x['qualification_scope']='Fixed for the tested source/font/three native geometries; one-row planning unit-tested. Recheck final release candidate under DCM-46. Physical DPI, alternate fonts and manual acceptance not certified.'
    x['evidence_refs'] += [REPORT,EVIDENCE]
    todo=itemline(todo,x['id'],x)
    a=json.loads(todo); assert len(a['items'])==len(t['items'])
    for old,new in zip(t['items'],a['items']):
        assert old['user_test']==new['user_test']
        if old['id']!='DCM-UI-LABEL-01': assert old==new
    assert a['completion_plan']['completed_product_steps']==t['completion_plan']['completed_product_steps']
    assert a['completion_plan']['blocked_steps']==t['completion_plan']['blocked_steps']
    assert a['checkpoints']['final_package_qualified'] is False
    assert a['checkpoints']['user_acceptance']==t['checkpoints']['user_acceptance']
    plan=line(plan,'checkpoint',REPORT,2); plan=line(plan,'last_tested_source',TESTED,2)
    audit={'source':TESTED,'optional_sumatra':'Source-bound page/zoom/lifecycle evidence retained; focused navigation/zoom/resize and scroll regressions pass.','default_provider':'Windows.Data.Pdf still default; current-source actual visible page/manual-zoom acceptance not run in this task.','next':'Qualify default-provider displayed page/zoom content, then disposition DCM-28/29 within demonstrated scope.','default_provider_changed':False}
    for id in ['DCM-28','DCM-29','DCM-46']:
        x=copy.deepcopy(next(x for x in p['steps'] if x['id']==id))
        x['compact_label_checkpoint']=record
        x['evidence'] += [REPORT,EVIDENCE]
        if id in ['DCM-28','DCM-29']: x['provider_acceptance_audit']=audit
        else: x['label_followup_disposition']='DCM-UI-LABEL-01 fixed at2cafe818 for tested geometry; retain final-candidate workflow/image recheck and unrecorded user acceptance.'
        plan=itemline(plan,id,x)
    a=json.loads(plan); assert len(a['steps'])==50
    for old,new in zip(p['steps'],a['steps']):
        assert old['completion']==new['completion'] and old['user_test']==new['user_test']
        if old['id'] not in ['DCM-28','DCM-29','DCM-46']: assert old==new
    assert sum(x['completion']=='complete' for x in a['steps'])==26
    return todo,plan
if __name__=='__main__':
    expected={'todo.json':'cdbe5ded11a122065c583146afbf000908b7998b','qualification/completion-50.json':'5aae3a58b9275e6cb3463dcfbeba66c2727b5362'}
    raw={name:(ROOT/name).read_bytes() for name in expected}
    for name,b in raw.items(): assert blob(b)==expected[name],(name,'input changed; reconcile')
    for name,s in zip(expected,transform(raw['todo.json'].decode(),raw['qualification/completion-50.json'].decode())):
        (ROOT/name).write_bytes(s.encode()); print(name,hashlib.sha256(s.encode()).hexdigest())
