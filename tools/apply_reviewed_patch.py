"""Apply exact locally reviewed edits on GitHub, never on the test device."""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path


def prepare(root, spec):
    if spec.get('schema') != 1 or not spec.get('edits') or not spec.get('message'):
        raise ValueError('Incomplete reviewed patch')
    output = {}
    for row in spec['edits']:
        name = row['path']
        path = Path(name)
        if path.is_absolute() or '..' in path.parts or '\\' in name or name.startswith('.git') or name in output:
            raise ValueError('Unsafe or duplicate path')
        p = root / path
        if p.is_symlink():
            raise ValueError('Symlink target')
        raw = p.read_bytes() if p.exists() else None
        expected = row['before_sha256']
        if (hashlib.sha256(raw).hexdigest() if raw is not None else None) != expected:
            raise ValueError('Before identity mismatch: '+name)
        if raw is None:
            text = row['new_content']
        else:
            text = raw.decode('utf-8')
            for old, new in row['replacements']:
                if not old or text.count(old) != 1:
                    raise ValueError('Nonunique replacement: '+name)
                text = text.replace(old, new, 1)
        value = text.encode('utf-8')
        if hashlib.sha256(value).hexdigest() != row['after_sha256']:
            raise ValueError('After identity mismatch: '+name)
        output[name] = value
    return output


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--stage',action='store_true')
    args=parser.parse_args()
    root=Path(__file__).resolve().parents[1]
    spec=json.loads((root/'qualification/reviewed-patch.json').read_text())
    if args.stage:
        for row in spec['edits']:
            assert hashlib.sha256((root/row['path']).read_bytes()).hexdigest()==row['after_sha256']
        subprocess.run(['git','add','--']+[r['path'] for r in spec['edits']],cwd=root,check=True)
        actual=subprocess.check_output(['git','diff','--cached','--name-only'],cwd=root,text=True).splitlines()
        assert set(actual)=={r['path'] for r in spec['edits']},actual
        subprocess.run(['git','commit','-m',spec['message']],cwd=root,check=True)
    else:
        output=prepare(root,spec)  # all identities validated before writing anything
        for name, raw in output.items():
            p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(raw)
        print('Applied '+str(len(output))+' exact reviewed files')

if __name__=='__main__':
    main()
