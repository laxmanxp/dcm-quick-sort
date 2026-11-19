"""Record the reviewed DCM-26 boundary without changing other acceptance states."""
from pathlib import Path
import copy, hashlib, json, re
ROOT = Path(__file__).resolve().parents[1]
EXPECTED = {'todo.json':'58be4c0c01f161484cc72830bb54dfe4d0041408',
            'qualification/completion-50.json':'b7b471296f77df6f1e79c6567f99b5c3342b2705'}
REPORT='qualification/dcm26-complete.md'
EVIDENCE='qualification/evidence/dcm-26-native-summary.json'
def set_line(text,key,value,spaces):
    p=re.compile(r'^'+' '*spaces+re.escape(json.dumps(key))+r': .*?,?$',re.M)
    matches=list(p.finditer(text)); assert len(matches)==1,(key,len(matches))
    comma=',' if matches[0].group().endswith(',') else ''
    return p.sub(lambda _: ' '*spaces+json.dumps(key)+': '+json.dumps(value,ensure_ascii=False)+comma,text)
def transform(plan,todo,e):
    assert e['feature_accepted'] and e['native_ui']['passed']
    assert e['reviewed_images']==len(e['images'])==17
    assert all(x['reviewed'] for x in e['images'])
    pb,tb=json.loads(plan),json.loads(todo)
    assert pb['next_step']=='DCM-26'
    assert next(x for x in pb['steps'] if x['id']=='DCM-25')['completion']=='complete'
    for k,v in [('checkpoint',REPORT),('last_tested_source',e['tested_commit']),('next_step','DCM-27'),('next_runnable_step','DCM-27')]:
        plan=set_line(plan,k,v,2)
    step=copy.deepcopy(next(x for x in pb['steps'] if x['id']=='DCM-26'))
    assert step['completion']=='pending' and step['user_test']=='not_recorded'
    step.update(status='fixed',completion='complete',ui_test='passed_with_separate_image_review',evidence=[REPORT,EVIDENCE],native_test=e['qualification_summary'],scope_note=e['scope_note'])
    lines=[x for x in plan.splitlines() if '"id": "DCM-26"' in x];assert len(lines)==1
    plan=plan.replace(lines[0],'    '+json.dumps(step,ensure_ascii=False)+',')
    pa=json.loads(plan);assert len(pa['steps'])==50
    for before,after in zip(pb['steps'],pa['steps']):
        if before['id']!='DCM-26': assert before==after
    assert next(x for x in pa['steps'] if x['id']=='DCM-19')['completion']=='pending'
    products=tb['completion_plan']['completed_product_steps'];assert 'DCM-25' in products and 'DCM-26' not in products
    for k,v in [('completed_product_steps',products+['DCM-26']),('checkpoint',REPORT),('next_step','DCM-27'),('next_runnable_step','DCM-27'),('candidate_code_commit',e['tested_commit']),('last_native_attempt',e['qualification_summary'])]:
        todo=set_line(todo,k,v,4)
    old=next(x for x in todo.splitlines() if x.startswith('    "app_baseline_checkpoint":'))
    todo=todo.replace(old,'    "app_baseline_dcm25_checkpoint": '+json.dumps(tb['checkpoints']['app_baseline_checkpoint'])+',\n    "app_baseline_checkpoint": '+json.dumps(e['baseline'])+',')
    needle='    "final_package_qualified": false,';assert todo.count(needle)==1
    todo=todo.replace(needle,'    "ui_text_checkpoint": '+json.dumps(e['native_ui'])+',\n'+needle)
    needle='    "dcm25_completion": "qualification/dcm25-complete.md"';assert todo.count(needle)==1
    todo=todo.replace(needle,needle+',\n    "dcm26_completion": '+json.dumps(REPORT))
    issue={'id':'DCM-26-TEXT-01','feature':'Decode bounded text without silent corruption and retain truthful preview status','status':'fixed','severity':'medium','finding_kind':'native_encoding_limit_and_error_state_failures','ui_test':'passed_with_separate_image_review','user_test':'not_recorded','release_gate':True,'parent_steps':['DCM-26'],'acceptance':'Supported UTF-8 and BOM-marked UTF-16 display correctly; preview cutoff omits partial characters with a visible limit notice; malformed/unsupported encodings and read failures report their reason; manual text mode updates status; recovery preserves source bytes.','native_test':e['qualification_summary'],'evidence_refs':['dcm26_completion']}
    assert issue['id'] not in {x['id'] for x in tb['items']}
    assert todo.count('\n  ]\n}')==1
    todo=todo.replace('\n  ]\n}',',\n    '+json.dumps(issue)+'\n  ]\n}')
    ta=json.loads(todo);assert ta['items'][:-1]==tb['items']
    assert ta['completion_plan']['blocked_steps']==tb['completion_plan']['blocked_steps']
    assert ta['checkpoints']['user_acceptance']=='not_recorded' and ta['checkpoints']['final_package_qualified'] is False
    return plan,todo
def main():
    data={}
    for path,want in EXPECTED.items():
        raw=(ROOT/path).read_bytes();blob=hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest()
        assert blob==want,(path,blob,'base changed');data[path]=raw.decode()
    e=json.loads((ROOT/EVIDENCE).read_text())
    plan,todo=transform(data['qualification/completion-50.json'],data['todo.json'],e)
    for name,text in [('qualification/completion-50.json',plan),('todo.json',todo)]:
        (ROOT/name).write_text(text,encoding='utf-8');print(name,hashlib.sha256(text.encode()).hexdigest())
    out=ROOT/'checkpoint-exports';out.mkdir(exist_ok=True)
    lines=['# DCM QuickSort — 50-step checklist','','DCM-26 accepted. DCM-19 remains blocked. Five completed steps are setup/diagnostic work, not product features.','','| ID | Feature | Completion | UI test | User test |','|---|---|---|---|---|']
    for step in json.loads(plan)['steps']:
        lines.append('| '+' | '.join(str(step.get(k,'')).replace('|','/') for k in ('id','feature','completion','ui_test','user_test'))+' |')
    (out/'completion-50.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
if __name__=='__main__':main()
