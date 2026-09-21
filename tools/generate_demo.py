from PIL import Image, ImageDraw, ImageFont
from pathlib import Path
import subprocess
import shutil

W, H, FPS = 1280, 720, 30
ROOT = Path(__file__).resolve().parents[1]
BUILD = ROOT / ".demo-build"
MEDIA = ROOT / "media"
BUILD.mkdir(exist_ok=True)
MEDIA.mkdir(exist_ok=True)

FONT_REG = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
FONT_BOLD = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"

C = {
    "bg":"#F4F6F9","ink":"#172033","muted":"#687487","blue":"#1859B7",
    "blue2":"#0E3F88","line":"#D8DEE7","panel":"#FFFFFF","soft":"#EAF2FF",
    "green":"#177548","sel":"#E8F1FF","yellow":"#FFF7D6"
}

def font(size, bold=False):
    return ImageFont.truetype(FONT_BOLD if bold else FONT_REG, size)

def rr(d, box, radius, fill, outline=None, width=1):
    d.rounded_rectangle(box, radius=radius, fill=fill, outline=outline, width=width)

def text(d, xy, value, size, fill=None, bold=False, anchor=None):
    d.text(xy, value, font=font(size, bold), fill=fill or C["ink"], anchor=anchor)

def header(d):
    text(d,(48,30),"DCM : Quick Sort",30,C["ink"],True)
    text(d,(48,68),"Downloaded Content Management · v1.0.17",15,C["muted"])
    rr(d,(1030,28,1228,64),18,C["soft"])
    text(d,(1129,46),"Files · Preview · Processed",13,C["blue2"],True,"mm")

def preview(im, box):
    x0,y0,x1,y1=box
    w,h=x1-x0,y1-y0
    p=Image.new("RGB",(w,h),(127,177,218))
    d=ImageDraw.Draw(p)
    d.rectangle((0,int(h*.62),w,h),fill=(104,137,76))
    d.polygon([(0,h),(int(w*.37),int(h*.49)),(int(w*.58),int(h*.49)),(w,h)],fill=(151,146,137))
    d.rectangle((int(w*.68),int(h*.33),int(w*.86),int(h*.65)),fill=(211,194,163))
    d.rectangle((int(w*.73),int(h*.45),int(w*.79),int(h*.65)),fill=(75,83,91))
    im.paste(p,(x0,y0))

FILES = [
    "IMG-20260920-WA0003.jpg","Adobe Scan 01.pdf","R5Kzw.jpg",
    "document (17).pdf","scan_004.jpg","download"
]

def app_frame(selected=0, name="R5Kzw", processed=None, processed_sel=None,
              note="", primary=False, undo=False):
    processed = processed or []
    im=Image.new("RGB",(W,H),C["bg"])
    d=ImageDraw.Draw(im)
    header(d)
    rr(d,(36,92,1244,683),15,C["panel"],C["line"],2)

    text(d,(58,123),"Source",13,C["muted"],True)
    rr(d,(115,107,845,141),6,"#FFFFFF",C["line"])
    text(d,(129,124),r"C:\Users\Laxman\Downloads",13,C["ink"],False,"lm")
    for label,x0,x1 in [("Browse",860,963),("Reload",975,1078)]:
        rr(d,(x0,106,x1,142),6,"#F8F9FB",C["line"])
        text(d,((x0+x1)//2,124),label,13,C["ink"],False,"mm")
    text(d,(1100,124),"4 of 6",13,C["muted"],False,"lm")

    text(d,(58,163),"Type",13,C["muted"],True)
    rr(d,(99,147,233,180),6,"#FFFFFF",C["line"])
    text(d,(111,164),"Images",13,C["ink"],False,"lm")
    text(d,(258,163),"Extension",13,C["muted"],True)
    rr(d,(335,147,450,180),6,"#FFFFFF",C["line"])
    text(d,(347,164),".jpg",13,C["ink"],False,"lm")

    left=(58,202,320,473); center=(334,202,900,473); right=(914,202,1222,473)
    for box,title in [(left,"Files"),(center,"Preview"),(right,"Processed")]:
        rr(d,box,8,"#FBFCFD",C["line"])
        text(d,(box[0]+14,box[1]+21),title,14,C["ink"],True)
        d.line((box[0]+10,box[1]+40,box[2]-10,box[1]+40),fill=C["line"],width=1)

    y=left[1]+49
    for i,f in enumerate(FILES):
        if i==selected:
            rr(d,(left[0]+8,y-5,left[2]-8,y+27),5,C["sel"],"#B9D2FA")
        text(d,(left[0]+15,y+10),f,12,C["ink"],False,"lm")
        y+=37

    preview(im,(center[0]+12,center[1]+50,center[2]-12,center[3]-12))
    d=ImageDraw.Draw(im)
    rr(d,(center[0]+28,center[1]+62,center[0]+175,center[1]+94),7,"#FFFFFF")
    text(d,(center[0]+101,center[1]+78),"Selected preview",12,C["ink"],True,"mm")

    y=right[1]+50
    for i,(old,new) in enumerate(processed):
        if i==processed_sel:
            rr(d,(right[0]+8,y-7,right[2]-8,y+42),5,C["sel"],"#B9D2FA",2)
        text(d,(right[0]+14,y),old,11,C["muted"])
        text(d,(right[0]+14,y+18),"→ "+new,11,C["ink"],True)
        y+=55
    if not processed:
        text(d,((right[0]+right[2])//2,right[1]+95),"Nothing processed yet",12,C["muted"],False,"mm")

    text(d,(58,506),"Name",13,C["muted"],True)
    rr(d,(112,489,903,524),6,"#FFFFFF",C["line"])
    text(d,(126,507),name,14,C["ink"],False,"lm")
    text(d,(58,548),"Move to",13,C["muted"],True)
    rr(d,(125,531,903,566),6,"#FFFFFF",C["line"])
    text(d,(138,549),r"C:\Office\Projects\Bridge Inspection",13,C["ink"],False,"lm")
    rr(d,(914,530,1045,567),6,"#F8F9FB",C["line"])
    text(d,(979,549),"Browse",13,C["ink"],False,"mm")

    rr(d,(58,590,282,630),7,C["blue"] if primary else "#F8F9FB",C["line"])
    text(d,(170,610),"Rename + Move + Next",13,"#FFFFFF" if primary else C["ink"],True,"mm")
    rr(d,(294,590,386,630),7,"#F8F9FB",C["line"]); text(d,(340,610),"Skip",13,C["ink"],False,"mm")
    rr(d,(398,590,530,630),7,C["yellow"] if undo else "#F8F9FB","#D5B346" if undo else C["line"],2 if undo else 1)
    text(d,(464,610),"Undo Selected",13,C["ink"],undo,"mm")
    rr(d,(542,590,630,630),7,"#F8F9FB",C["line"]); text(d,(586,610),"Open",13,C["ink"],False,"mm")
    rr(d,(642,590,763,630),7,"#F8F9FB",C["line"]); text(d,(702,610),"Open Folder",13,C["ink"],False,"mm")

    rr(d,(800,588,1218,632),9,C["soft"])
    text(d,(1009,610),note,12,C["blue2"],True,"mm")
    text(d,(58,657),"DCM - Downloaded Content Management",11,C["muted"],True)
    text(d,(1220,657),"Free · Donation supported",11,C["muted"],False,"rm")
    return im

def title_frame():
    im=Image.new("RGB",(W,H),"#0F2748")
    d=ImageDraw.Draw(im)
    text(d,(W//2,155),"DCM : Quick Sort",56,"#FFFFFF",True,"mm")
    text(d,(W//2,213),"Downloaded Content Management",24,"#D7E7FF",False,"mm")
    text(d,(W//2,315),"Version 1.0.17",24,"#FFFFFF",True,"mm")
    x=155
    for label in ["Files","Preview","Processed"]:
        rr(d,(x,383,x+250,448),12,"#FFFFFF")
        text(d,(x+125,416),label,23,C["blue2"],True,"mm")
        x+=360
    text(d,(W//2,520),"Jump to any file. See what you processed. Undo any selected transaction.",20,"#D7E7FF",False,"mm")
    return im

def end_frame():
    im=Image.new("RGB",(W,H),"#0F2748")
    d=ImageDraw.Draw(im)
    text(d,(W//2,180),"DCM : Quick Sort",54,"#FFFFFF",True,"mm")
    text(d,(W//2,242),"Preview → Rename → Move → Next",29,"#D7E7FF",True,"mm")
    text(d,(W//2,342),"Selectable files and processed-history undo.",23,"#FFFFFF",False,"mm")
    rr(d,(370,465,910,528),12,C["blue"])
    text(d,(640,496),"laxmanxp.github.io/dcm-quick-sort/",19,"#FFFFFF",True,"mm")
    text(d,(W//2,603),"Free · Donation supported · No account required",17,"#CFE0FF",False,"mm")
    return im

p1=[("R5Kzw.jpg","Bridge Inspection - Site Photo.jpg")]
p3=[
    ("R5Kzw.jpg","Bridge Inspection - Site Photo.jpg"),
    ("scan_004.jpg","Equipment Serial Plate.jpg"),
    ("IMG-20260920-WA0003.jpg","Site Measurement Photo.jpg")
]

scenes=[
    (title_frame(),3.0,"01-title"),
    (app_frame(0,"IMG-20260920-WA0003",note="Select any file from the left column"),3.6,"02-select"),
    (app_frame(2,"R5Kzw",note="Preview the selected file before naming it"),3.6,"03-preview"),
    (app_frame(2,"Bridge Inspection - Site Photo",note="Give it a useful name and destination",primary=True),3.8,"04-rename"),
    (app_frame(3,"document (17)",processed=p1,note="Processed files appear immediately on the right"),3.9,"05-processed"),
    (app_frame(1,"Adobe Scan 01",processed=p3,processed_sel=0,note="Select any earlier transaction to undo it",undo=True),4.1,"06-history"),
    (app_frame(2,"R5Kzw",processed=p3[1:],note="Undo restores only the selected transaction"),4.0,"07-undo"),
    (end_frame(),3.2,"08-end")
]

segments=[]
for image,duration,name in scenes:
    png=BUILD/(name+".png")
    image.save(png)
    seg=BUILD/(name+".mp4")
    subprocess.run([
        "ffmpeg","-y","-loglevel","error","-loop","1","-i",str(png),"-t",str(duration),
        "-vf",f"scale={W}:{H},format=yuv420p","-r",str(FPS),"-an",
        "-c:v","libx264","-preset","veryfast","-crf","20",str(seg)
    ],check=True)
    segments.append((seg,duration))

xf=.28
inputs=[]
for seg,_ in segments:
    inputs += ["-i",str(seg)]
filters=[]
last="[0:v]"
offset=segments[0][1]-xf
for i in range(1,len(segments)):
    out=f"[v{i}]"
    filters.append(f"{last}[{i}:v]xfade=transition=fade:duration={xf}:offset={offset:.3f}{out}")
    last=out
    offset += segments[i][1]-xf

video=MEDIA/"DCM-QuickSort-v1.0.17-Demo.mp4"
subprocess.run([
    "ffmpeg","-y","-loglevel","error",*inputs,
    "-filter_complex",";".join(filters),"-map",last,
    "-c:v","libx264","-preset","medium","-crf","22","-pix_fmt","yuv420p",
    "-movflags","+faststart",str(video)
],check=True)

poster=MEDIA/"DCM-QuickSort-v1.0.17-Demo-Poster.png"
app_frame(2,"Bridge Inspection - Site Photo",processed=p3,processed_sel=0,
          note="Files · Preview · Processed",primary=True,undo=True).save(poster)

shutil.rmtree(BUILD)
print(video)
print(poster)
