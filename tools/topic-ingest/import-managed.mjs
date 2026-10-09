import {createHash,randomUUID} from 'node:crypto';
import {constants} from 'node:fs';
import {open} from 'node:fs/promises';
import {basename} from 'node:path';
import {parseManagedJSON} from './json.mjs';
const limit=64*1024*1024;
const sha=bytes=>createHash('sha256').update(bytes).digest('hex');
const safePath=s=>typeof s==='string'&&s.length>0&&!s.startsWith('/')&&!/^[A-Za-z]:/.test(s)&&!/[\\\0\r\n]/.test(s)&&s.split('/').every(p=>p!==''&&p!=='.'&&p!=='..');
export function canonicalManagedJSON(v){
 function canonical(x){if(x===null||typeof x==='string'||typeof x==='boolean')return x;if(typeof x==='number'){if(!Number.isFinite(x))throw Error('INVALID_CANONICAL_NUMBER');return x}if(Array.isArray(x))return x.map(canonical);if(x&&typeof x==='object')return Object.fromEntries(Object.keys(x).sort().map(k=>[k,canonical(x[k])]));throw Error('INVALID_CANONICAL_VALUE')}
 return JSON.stringify(canonical(v));
}
async function capture(path){const f=await open(path,constants.O_RDONLY|constants.O_NOFOLLOW);try{const a=await f.stat();if(!a.isFile()||a.size>limit)throw Error('UNSAFE_OR_LARGE_SOURCE');const buffer=Buffer.alloc(a.size+1);let offset=0;while(offset<buffer.length){const {bytesRead}=await f.read(buffer,offset,buffer.length-offset,offset);if(bytesRead===0)break;offset+=bytesRead};const b=buffer.subarray(0,offset);const z=await f.stat();if(a.ino!==z.ino||a.size!==z.size||a.mtimeMs!==z.mtimeMs||b.length!==a.size)throw Error('SOURCE_CHANGED');return b}finally{await f.close()}}
function verifyManifest(m,name,bytes,d){
 if(m?.format!=='math-master-knowledge-manifest'||m.schema_version!=='1.0'||m.record_digest_algorithm!=='jcs-rfc8785-sha256'||!Array.isArray(m.files)||!Array.isArray(m.records))throw Error('INVALID_MANIFEST');const names=new Set();for(const f of m.files){if(!safePath(f.path)||names.has(f.path))throw Error('UNSAFE_MANIFEST_PATH');names.add(f.path)}
 if(sha(canonicalManagedJSON({files:m.files,records:m.records}))!==m.snapshot_id)throw Error('MANIFEST_SNAPSHOT_MISMATCH');const f=m.files.find(f=>f.path===name);if(!f||f.sha256!==sha(bytes)||f.bytes!==bytes.length||f.source_id!==d.source.source_id||f.dataset_version!==d.dataset_version||f.record_count!==d.knowledge_points.length)throw Error('MANIFEST_FILE_MISMATCH');const seen=new Set();for(const r of m.records){if(!safePath(r.path)||!names.has(r.path))throw Error('UNSAFE_MANIFEST_PATH');if(r.path!==name)continue;const hit=/^\/knowledge_points\/(0|[1-9]\d*)$/.exec(r.json_pointer);if(!hit||seen.has(r.json_pointer))throw Error('MANIFEST_RECORD_MISMATCH');seen.add(r.json_pointer);const p=d.knowledge_points[Number(hit[1])];if(!p||r.id!==p.id||r.version!==p.version||r.source_id!==d.source.source_id||r.record_sha256!==sha(canonicalManagedJSON(p)))throw Error('MANIFEST_RECORD_MISMATCH')};if(seen.size!==d.knowledge_points.length)throw Error('MANIFEST_RECORD_MISMATCH')
}
const network=e=>e instanceof TypeError||e?.code==='NETWORK_ERROR';
// Files are handled sequentially; caller supplies an authenticated administrator API client.
export async function importManagedFiles({files,manifest,client,publish=true,onProgress=()=>{}}){
 if(!Array.isArray(files)||files.length===0||typeof client?.preview!=='function'||typeof client?.apply!=='function')throw Error('INVALID_IMPORT_INPUT');let m=manifest;if(typeof m==='string'){const b=await capture(m);m=parseManagedJSON(b)};const receipts=[],names=new Set();
 for(let index=0;index<files.length;index++){const path=files[index],name=basename(path);if(names.has(name))throw Error('DUPLICATE_SOURCE_FILENAME');names.add(name);const bytes=await capture(path),d=parseManagedJSON(bytes);if(d.format!=='math-master-knowledge-source'||d.schema_version!=='1.0'||!Array.isArray(d.knowledge_points)||d.knowledge_points.length>100)throw Error('INVALID_SOURCE_FORMAT');if(m)verifyManifest(m,name,bytes,d);
 const previewKey=randomUUID(),applyKey=randomUUID();onProgress({index,total:files.length,fileName:name,stage:'preview'});let preview;try{preview=await client.preview(bytes,previewKey)}catch(e){if(!network(e))throw e;preview=await client.preview(bytes,previewKey)}
 const selectedIndexes=preview.items.filter(i=>i.action==='create'||i.action==='link').map(i=>i.index);const input={selectedIndexes,publish,previewToken:preview.previewToken};onProgress({index,total:files.length,fileName:name,stage:'apply'});let receipt;try{receipt=await client.apply(preview.importId,input,applyKey)}catch(e){if(!network(e))throw e;const status=await client.readImport(preview.importId);receipt=status.receipt??await client.apply(preview.importId,input,applyKey)}
 receipts.push(receipt);onProgress({index,total:files.length,fileName:name,stage:'complete',counts:receipt.counts})
 };return receipts;
}
