// Deterministic PDFium workload fixtures. No external/private PDF input.
const fs=require("node:fs");
const zlib=require("node:zlib");

function builder() {
  const chunks=[Buffer.from("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n","latin1")];
  let size=chunks[0].length;
  const offsets=[0];
  const push=b=>{if(typeof b==="string")b=Buffer.from(b,"latin1");chunks.push(b);size+=b.length;};
  const obj=(id,bytes)=>{
    offsets[id]=size;push(`${id} 0 obj\n`);
    push(bytes);push("\nendobj\n");
  };
  const save=filename=>{
    const xref=size;
    push(`xref\n0 ${offsets.length}\n0000000000 65535 f \n`);
    for(let i=1;i<offsets.length;i++)push(`${String(offsets[i]).padStart(10,"0")} 00000 n \n`);
    push(`trailer\n<< /Size ${offsets.length} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`);
    fs.writeFileSync(filename,Buffer.concat(chunks));
    return size;
  };
  return {obj,save,push};
}
function stream(data,properties="") {
  return Buffer.concat([
    Buffer.from(`<< /Length ${data.length} ${properties} >>\nstream\n`,"latin1"),
    data,
    Buffer.from("\nendstream","latin1")
  ]);
}
function makeRaster(filename){
  const w=1400,h=1800;
  const pixels=Buffer.alloc(w*h);
  let state=0x12345678;
  for(let y=0;y<h;y++){
    for(let x=0;x<w;x++){
      state=(Math.imul(state,1664525)+1013904223)>>>0;
      let value=(state>>>24);
      if((x%117)<3||(y%153)<3)value=20;
      if((x+y)%89<3)value=230;
      pixels[y*w+x]=value;
    }
  }
  const compressed=zlib.deflateSync(pixels,{level:6});
  const b=builder();
  b.obj(1,"<< /Type /Catalog /Pages 2 0 R >>");
  b.obj(2,"<< /Type /Pages /Kids [3 0 R] /Count 1 >>");
  b.obj(3,"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 700 900] /Resources << /XObject << /Scan 5 0 R >> >> /Contents 4 0 R >>");
  b.obj(4,stream(Buffer.from("q 700 0 0 900 0 0 cm /Scan Do Q\n")));
  b.obj(5,stream(compressed,`/Type /XObject /Subtype /Image /Width ${w} /Height ${h} /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode`));
  const size=b.save(filename);
  return {bytes:size,rasterPixels:w*h,compressedBytes:compressed.length};
}
function makeVector(filename){
  const commands=["q","0.94 0.96 0.99 rg","0 0 700 900 re f"];
  for(let i=0;i<7000;i++){
    const x=(i*73)%700,y=(i*47)%900;
    const r=((i*13)%100)/100,g=((i*29)%100)/100,bl=((i*43)%100)/100;
    commands.push(`${r.toFixed(2)} ${g.toFixed(2)} ${bl.toFixed(2)} rg ${x} ${y} 17 11 re f`);
  }
  commands.push("Q");
  const data=Buffer.from(commands.join("\n")+"\n","ascii");
  const b=builder();
  b.obj(1,"<< /Type /Catalog /Pages 2 0 R >>");
  b.obj(2,"<< /Type /Pages /Kids [3 0 R] /Count 1 >>");
  b.obj(3,"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 700 900] /Resources << >> /Contents 4 0 R >>");
  b.obj(4,stream(zlib.deflateSync(data,{level:6}),"/Filter /FlateDecode"));
  return {bytes:b.save(filename),vectorRectangles:7000};
}
if(require.main===module){
  if(process.argv.length!==4)throw Error("Usage: node pdfium-stress-fixture.js raster.pdf vector.pdf");
  console.log(JSON.stringify({raster:makeRaster(process.argv[2]),vector:makeVector(process.argv[3])}));
}
module.exports={makeRaster,makeVector};
