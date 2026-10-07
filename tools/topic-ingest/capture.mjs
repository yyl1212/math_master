import fs from 'node:fs/promises';
import {constants} from 'node:fs';
import {resolve,dirname,basename,join,relative,isAbsolute} from 'node:path';
import {createHash} from 'node:crypto';
import {isDeepStrictEqual} from 'node:util';
import {readIndexDeclarations,indexNames} from '../content-ingest/source-index.mjs';
import {parseSourceJSON} from './json.mjs';

const META='Materials/Collection_Metadata/MSC2020';
const NAMES=['MSC2020_Complete_Taxonomy.json','MSC2020_SOURCE_KNOWLEDGE_MAPPINGS.json','MSC2020_DUAL_SOURCE_COVERAGE_MATRIX.json','INDEPENDENT_SOURCE_REGISTRY_CURRENT.json'];
const METADATA_LIMIT=8<<20,PRIMARY_LIMIT=64<<20;
const hash=b=>createHash('sha256').update(b).digest('hex');
const sha=v=>typeof v==='string'&&/^[a-f0-9]{64}$/.test(v);
const safe=p=>typeof p==='string'&&p.length>0&&!p.includes('\0')&&!p.includes('\\')&&!isAbsolute(p)&&!p.split('/').some(x=>!x||x==='.'||x==='..');
const fail=code=>{throw new Error(code);};
const inside=(root,p)=>{const r=relative(root,p);return !r||(!isAbsolute(r)&&r!=='..'&&!r.startsWith('../'));};
const fingerprint=s=>[s.dev,s.ino,s.size,s.mtimeNs,s.ctimeNs].join(':');
const revisionNoticeKey='current_native_record_revision_notice';
function validateRevisionNotice(n){
  const keys=['revision_id','knowledge_record_id','approved_local_correction_applied','native_primary_relative_path','new_primary_sha256','new_primary_bytes','source_notice_relative_path','new_qualified_source_rows','new_dual_source_credit','direct_qualification_remains_excluded'];
  if(!n||Array.isArray(n)||typeof n!=='object'||Object.keys(n).sort().join(',')!==keys.sort().join(','))fail('INVALID_NATIVE_REVISION_NOTICE');
  if(['revision_id','knowledge_record_id'].some(k=>typeof n[k]!=='string'||n[k].length===0||Buffer.byteLength(n[k])>256)||typeof n.approved_local_correction_applied!=='boolean'||!safe(n.native_primary_relative_path)||!safe(n.source_notice_relative_path)||!sha(n.new_primary_sha256)||!Number.isSafeInteger(n.new_primary_bytes)||n.new_primary_bytes<0||n.new_primary_bytes>PRIMARY_LIMIT||n.new_qualified_source_rows!==0||n.new_dual_source_credit!==0||n.direct_qualification_remains_excluded!==true)fail('INVALID_NATIVE_REVISION_NOTICE');
}
const fields={
  [NAMES[0]]:new Set(['metadata','entries']),
  [NAMES[1]]:new Set(['schema_version','updated_utc','classification_raw_sha256','summary','sources','mappings','scope_note','taxonomy_attribution','taxonomy_license','limited_support_noncounting_candidates','auxiliary_semantic_mappings','staged_only','candidate_batch','candidate_scope','formal_integration_performed','transaction_created_utc','local_only_storage_current']),
  [NAMES[2]]:new Set(['schema_version','created_utc','classification_raw_sha256','canonical_taxonomy_reference','scope','acceptance_rule','workflows','summary','entries','updated_utc','last_completed_batch','taxonomy_attribution','taxonomy_license','formal_integration_performed','staged_only','candidate_batch','candidate_scope','transaction_created_utc','last_completed_batch_scope','local_only_storage_current']),
  [NAMES[3]]:new Set(['schema_version','created_utc','catalog_snapshot_sha256','catalog_book_id_count','counting_note','sources','updated_utc','reviewed_source_families','remaining_catalog_sources_not_assessed','reviewed_bibliographic_source_ids','catalog_baseline_work_count','staged_new_bibliographic_work_count','batch3_installed_bibliographic_work_count','formal_integration_performed','staged_only','candidate_batch','candidate_scope','transaction_created_utc','local_only_storage_current'])
};
function metadata(bytes,name) {
  const data=parseSourceJSON(bytes);
  if(!data||Array.isArray(data)||typeof data!=='object')fail('INVALID_SOURCE_JSON');
  for(const key of Object.keys(data)){
    const historical=name===NAMES[3]&&(/^previous_catalog_snapshot_before_batch\d+$/.test(key)||key==='previous_catalog_snapshot_before_source8_reconciliation');
    const revisionNotice=name!==NAMES[0]&&key===revisionNoticeKey;
    if(revisionNotice)validateRevisionNotice(data[key]);
    if(!fields[name].has(key)&&!historical&&!revisionNotice)fail('UNKNOWN_METADATA_FIELD');
    if(historical){
      const pin=data[key];
      if(key==='previous_catalog_snapshot_before_source8_reconciliation'){
        if(!pin||Array.isArray(pin)||typeof pin!=='object'||Object.keys(pin).sort().join(',')!=='book_id_count,sha256'||!sha(pin.sha256)||!Number.isSafeInteger(pin.book_id_count)||pin.book_id_count<0)fail('INVALID_SOURCE_JSON');
      }else if(!sha(pin))fail('INVALID_SOURCE_JSON');
    }
  }
  return data;
}
async function regular(root,path,limit,signal) {
  if(signal?.aborted)fail('CAPTURE_ABORTED');
  if(!safe(path))fail('PATH_ESCAPE');
  let current=root;
  for(const part of path.split('/')){
    current=join(current,part);const st=await fs.lstat(current);
    if(st.isSymbolicLink())fail('SOURCE_SYMLINK');
  }
  const before=await fs.lstat(current,{bigint:true});
  if(!before.isFile())fail('SOURCE_NOT_REGULAR');
  if(before.size>BigInt(limit))fail('SOURCE_LIMIT_EXCEEDED');
  const handle=await fs.open(current,constants.O_RDONLY|(constants.O_NOFOLLOW??0));
  let bytes;
  try {
    const opened=await handle.stat({bigint:true});
    if(fingerprint(opened)!==fingerprint(before))fail('SOURCE_CHANGED_RETRY');
    const maximum=Number(before.size)+1,buffer=Buffer.alloc(maximum);let count=0;
    while(count<maximum) {
      if(signal?.aborted)fail('CAPTURE_ABORTED');
      const part=await handle.read(buffer,count,Math.min(maximum-count,64<<10),count);
      if(!part.bytesRead)break;count+=part.bytesRead;
    }
    if(count>limit)fail('SOURCE_LIMIT_EXCEEDED');
    if(fingerprint(await handle.stat({bigint:true}))!==fingerprint(before))fail('SOURCE_CHANGED_RETRY');
    bytes=buffer.subarray(0,count);
  } finally {await handle.close();}
  if(fingerprint(await fs.lstat(current,{bigint:true}))!==fingerprint(before)||!inside(root,await fs.realpath(current)))fail('SOURCE_CHANGED_RETRY');
  return {bytes,sizeBytes:bytes.length,identity:fingerprint(before),path,limit,sha256:hash(bytes)};
}
export async function captureTopicBatch({sourceRoot,outDir,selectedPrimaryPaths=[],previousManifestPath,signal}) {
  if(signal?.aborted)fail('CAPTURE_ABORTED');
  if(typeof sourceRoot!=='string'||typeof outDir!=='string'||!Array.isArray(selectedPrimaryPaths)||selectedPrimaryPaths.length>1000||new Set(selectedPrimaryPaths).size!==selectedPrimaryPaths.length)fail('INVALID_SELECTION');
  if(selectedPrimaryPaths.some(p=>!safe(p)||!p.endsWith('.json')))fail('PATH_ESCAPE');
  if((await fs.lstat(sourceRoot)).isSymbolicLink())fail('SOURCE_SYMLINK');
  const root=await fs.realpath(sourceRoot),parent=await fs.realpath(dirname(resolve(outDir))),out=join(parent,basename(resolve(outDir)));
  if(inside(root,out))fail('OUTPUT_IN_SOURCE');
  try {await fs.lstat(out);fail('OUTPUT_EXISTS');}catch(e){if(e.code!=='ENOENT')throw e;}
  const captures=new Map(),remember=async(path,limit=METADATA_LIMIT)=>{const c=await regular(root,path,limit,signal);captures.set(path,c);return c;};
  const metaBytes=await Promise.all(NAMES.map(n=>remember(META+'/'+n)));
  const [taxonomy,mappings,matrix,registry]=metaBytes.map((c,i)=>metadata(c.bytes,NAMES[i]));
  const notices=[mappings,matrix,registry].map(m=>m[revisionNoticeKey]);
  if(notices.some(n=>n!==undefined)&&!notices.every(n=>isDeepStrictEqual(n,notices[0])))fail('NATIVE_REVISION_NOTICE_MISMATCH');
  for(const m of [mappings,matrix,registry])if(m.staged_only!==false||m.formal_integration_performed!==true)fail('BATCH_NOT_ACCEPTED');
  const batch=mappings.candidate_batch;
  if(!Number.isSafeInteger(batch)||batch<1||matrix.candidate_batch!==batch||registry.candidate_batch!==batch||matrix.last_completed_batch!==batch)fail('BATCH_MISMATCH');
  const csv=await remember(META+'/MSC_2020.official.csv');
  if(!sha(taxonomy.metadata?.source_checksums_sha256?.['MSC_2020.csv'])||taxonomy.metadata.source_checksums_sha256['MSC_2020.csv']!==csv.sha256||mappings.classification_raw_sha256!==csv.sha256||matrix.classification_raw_sha256!==csv.sha256)fail('CLASSIFICATION_DIGEST_MISMATCH');
  if(matrix.canonical_taxonomy_reference?.sha256!==metaBytes[0].sha256)fail('CLASSIFICATION_DIGEST_MISMATCH');
  if(!isDeepStrictEqual(mappings.summary,matrix.summary))fail('COVERAGE_SUMMARY_MISMATCH');
  const indexes=new Map();
  for(const name of indexNames) {
    try {const c=await remember('Knowledge_JSON/'+name);parseSourceJSON(c.bytes);indexes.set(name,c.bytes);}
    catch(e){if(e.code!=='ENOENT'||name===indexNames[0])throw e;}
  }
  const audit=readIndexDeclarations(indexes),declarations=new Map(audit.declarations.map(x=>[x.path,x.claims]));
  const primary=new Set(audit.primaryFiles);
  if(audit.declarations.some(x=>!safe(x.path)))fail('PATH_ESCAPE');
  const issues=[];
  let before=[];
  if(previousManifestPath) {
    const priorRoot=await fs.realpath(dirname(resolve(previousManifestPath)));
    const c=await regular(priorRoot,basename(previousManifestPath),METADATA_LIMIT,signal),old=parseSourceJSON(c.bytes);
    const ids=new Set();
    if(old.schemaVersion!==1||old.accepted!==true||!Array.isArray(old.sourceFiles))fail('INVALID_PREVIOUS');
    for(const f of old.sourceFiles){if(!safe(f.path)||!sha(f.sha256)||!Number.isSafeInteger(f.sizeBytes)||f.sizeBytes<0||ids.has(f.path))fail('INVALID_PREVIOUS');ids.add(f.path);}
    if(hash(JSON.stringify(old.sourceFiles))!==old.snapshotId)fail('INVALID_PREVIOUS');before=old.sourceFiles;
  }
  let created=false;
  try {
    await fs.mkdir(out,{mode:0o700});created=true;
    const copy=async c=>{
      if(signal?.aborted)fail('CAPTURE_ABORTED');
      const file=join(out,'files',c.path);
      await fs.mkdir(dirname(file),{recursive:true,mode:0o700});
      await fs.writeFile(file,c.bytes,{flag:'wx',mode:0o600});
    };
    for(const c of captures.values())await copy(c);
    // Copy each selected corpus immediately. Keep identities and hashes, not
    // all 1,000 possible 64-MiB corpora resident at the same time.
    const selectedCaptures=[];
    for(const p of [...selectedPrimaryPaths].sort()) {
      if(!primary.has(p)||!declarations.has(p)){issues.push({code:'SELECTED_FILE_UNDECLARED',path:p});continue;}
      const claims=declarations.get(p);
      if(claims.some(x=>!sha(x.sha256))||new Set(claims.map(x=>x.sha256)).size!==1){issues.push({code:'INDEX_DIGEST_CONFLICT',path:p});continue;}
      let c;
      try{c=await regular(root,'Knowledge_JSON/'+p,PRIMARY_LIMIT,signal);}
      catch(e){if(e.code==='ENOENT'){issues.push({code:'INDEX_FILE_MISSING',path:p});continue;}throw e;}
      if(claims.some(x=>x.sha256!==c.sha256)){issues.push({code:'INDEX_DIGEST_MISMATCH',path:p});continue;}
      parseSourceJSON(c.bytes);await copy(c);
      const {bytes,...identity}=c;selectedCaptures.push(identity);
    }
    const all=[...captures.values(),...selectedCaptures].sort((a,b)=>a.path<b.path?-1:a.path>b.path?1:0);
    for(const c of all) {
      const check=await regular(root,c.path,c.limit,signal);
      if(check.identity!==c.identity||check.sha256!==c.sha256)fail('SOURCE_CHANGED_RETRY');
    }
    const sourceFiles=all.map(c=>({path:c.path,sizeBytes:c.sizeBytes,sha256:c.sha256}));
    const old=new Map(before.map(f=>[f.path,f.sha256])),now=new Map(sourceFiles.map(f=>[f.path,f.sha256]));
    const diff={added:sourceFiles.filter(f=>!old.has(f.path)).map(f=>f.path),changed:sourceFiles.filter(f=>old.has(f.path)&&old.get(f.path)!==f.sha256).map(f=>f.path),missing:before.filter(f=>!now.has(f.path)).map(f=>f.path).sort()};
    const manifest={schemaVersion:1,batch,sourceFiles,snapshotId:hash(JSON.stringify(sourceFiles)),diff,issues,accepted:true};
    await fs.writeFile(join(out,'manifest.json'),JSON.stringify(manifest,null,2)+'\n',{flag:'wx',mode:0o600});
    return manifest;
  }catch(e){if(created)await fs.rm(out,{recursive:true,force:true});throw e;}
}
