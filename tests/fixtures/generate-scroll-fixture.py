"""Generate deterministic ASCII-only two-page PDF for native scroll acceptance."""
from pathlib import Path
import json
import hashlib

ROOT = Path(__file__).resolve().parent
COLORS = [(0,140,160),(40,80,210),(0,140,60),(230,100,0),(150,40,180),(200,30,40)]
WORDS = ['ALPHA','BRAVO','CHARLIE','DELTA','ECHO','FOXTROT']
streams=[]
regions=[]
for page in range(1,3):
    parts=['0 0 0 rg BT /F1 20 Tf 60 965 Td (DCM READABLE SCROLL - PAGE %d OF 2) Tj ET'%page]
    for index,(section,y) in enumerate([('TOP',900),('MIDDLE',500),('BOTTOM',100)]):
        i=(page-1)*3+index
        r,g,b=COLORS[i]
        text=f'PAGE {page} - {section} - {WORDS[i]}'
        parts += [f'{r/255:.6f} {g/255:.6f} {b/255:.6f} rg 60 {y-20} 16 54 re f',
                  f'0 0 0 rg BT /F1 18 Tf 90 {y+14} Td ({text}) Tj ET',
                  f'0 0 0 rg BT /F1 14 Tf 90 {y-10} Td (Readable letters and numbers: 0123456789) Tj ET']
        regions.append(dict(page=page,section=section,token=WORDS[i],text=text,rgb=COLORS[i],marker=[60,y-20,16,54]))
    for n in range(1,25):
        y=880-31*n
        if abs(y-500)<65:continue
        parts.append(f'0 0 0 rg BT /F1 12 Tf 90 {y} Td (Page {page}, line {n:02d}: Keep this text readable while scrolling.) Tj ET')
    parts.append(f'0 0 0 rg BT /F1 12 Tf 60 35 Td (END OF PAGE {page} - disposable fixture, not a user document.) Tj ET')
    streams.append('\n'.join(parts))
objects=['<< /Type /Catalog /Pages 2 0 R >>','<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>',
         '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 1000] /Resources << /Font << /F1 5 0 R >> >> /Contents 6 0 R >>',
         '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 1000] /Resources << /Font << /F1 5 0 R >> >> /Contents 7 0 R >>',
         '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>']
objects += [f'<< /Length {len(s.encode("ascii"))} >>\nstream\n{s}\nendstream' for s in streams]
pdf='%PDF-1.4\n% DCM byte-exact scroll fixture\n'; offsets=[]
for n,obj in enumerate(objects,1):
    offsets.append(len(pdf.encode('ascii')));pdf+=f'{n} 0 obj\n{obj}\nendobj\n'
xref=len(pdf.encode('ascii'));pdf+='xref\n0 8\n0000000000 65535 f \n'
pdf+=''.join(f'{p:010d} 00000 n \n' for p in offsets)
pdf+=f'trailer\n<< /Size 8 /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n'
data=pdf.encode('ascii');(ROOT/'sumatra-scroll.pdf').write_bytes(data)
meta=dict(schema=1,filename='sumatra-scroll.pdf',sha256=hashlib.sha256(data).hexdigest(),pages=2,media_box=[0,0,612,1000],regions=regions)
(ROOT/'sumatra-scroll.json').write_text(json.dumps(meta,indent=2)+'\n')
print(meta['sha256'],len(data))
