// Independent QR/text checks; requires Poppler and npm dev dependencies.
const fs=require('node:fs'),os=require('node:os'),path=require('node:path'),{execFileSync}=require('node:child_process');
const jsQR=require('jsqr'),{PNG}=require('pngjs');
const prefix=process.argv[2]||'artifacts/credential';
const envelope=JSON.parse(fs.readFileSync(prefix+'.json','utf8'));
const temporary=fs.mkdtempSync(path.join(os.tmpdir(),'academy-credential-'));
try{
 execFileSync('pdftoppm',['-scale-to','1400','-png','-singlefile',prefix+'.pdf',path.join(temporary,'pdf')]);
 const text=execFileSync('pdftotext',[prefix+'.pdf','-'],{encoding:'utf8'}).replace(/\s+/g,' ');
 if(!text.includes(envelope.payload.subject_name.replace(/\s+/g,' ')))throw Error('Cyrillic subject name missing in PDF text');
 for(const file of [prefix+'.png',path.join(temporary,'pdf.png')]){
  const png=PNG.sync.read(fs.readFileSync(file));const decoded=jsQR(new Uint8ClampedArray(png.data),png.width,png.height);
  if(decoded?.data!==envelope.payload.verify_url)throw Error('QR does not match signed URL: '+file);
 }
 console.log('PASS: actual downloaded PDF Cyrillic text + PDF/PNG QR match signed JSON URL');
}finally{fs.rmSync(temporary,{recursive:true,force:true})}
