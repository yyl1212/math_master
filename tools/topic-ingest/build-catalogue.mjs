import fs from 'node:fs/promises';
import {join,isAbsolute} from 'node:path';
import {createHash} from 'node:crypto';
import {parseSourceJSON} from './json.mjs';
const hash=b=>createHash('sha256').update(b).digest('hex');
const safe=p=>typeof p==='string'&&p!==''&&!isAbsolute(p)&&!p.includes('\\')&&!p.includes('\0')&&!p.split('/').some(x=>x===''||x==='.'||x==='..');
const topicID=c=>'msc-'+c.toLowerCase().replace(/xx$/,'').replace(/-$/,'');
export async function buildCatalogueArchive(snapshotDir) {
 const file=join(snapshotDir,'manifest.json');if(!(await fs.lstat(file)).isFile()||(await fs.lstat(file)).isSymbolicLink())throw Error('INVALID_CAPTURE');
 const manifest=parseSourceJSON(await fs.readFile(file));
 if(manifest.accepted!==true||manifest.schemaVersion!==1||!Array.isArray(manifest.sourceFiles)||hash(JSON.stringify(manifest.sourceFiles))!==manifest.snapshotId)throw Error('INVALID_CAPTURE');
 const files=new Map(manifest.sourceFiles.map(f=>[f.path,f]));
 async function read(path) {
  if(!safe(path)||!files.has(path))throw Error('INVALID_CAPTURE');
  let current=join(snapshotDir,'files');
  for(const part of path.split('/')){current=join(current,part);const stat=await fs.lstat(current);if(stat.isSymbolicLink())throw Error('INVALID_CAPTURE');}
  const claim=files.get(path),before=await fs.stat(current);if(!before.isFile()||before.size!==claim.sizeBytes||before.size>64<<20)throw Error('CAPTURE_BYTES_CHANGED');
  const bytes=await fs.readFile(current),after=await fs.stat(current);
  if(bytes.length!==claim.sizeBytes||hash(bytes)!==claim.sha256||before.ino!==after.ino||before.mtimeMs!==after.mtimeMs)throw Error('CAPTURE_BYTES_CHANGED');
  return parseSourceJSON(bytes);
 }
 const prefix='Materials/Collection_Metadata/MSC2020/';
 const taxonomy=await read(prefix+'MSC2020_Complete_Taxonomy.json'),mappings=await read(prefix+'MSC2020_SOURCE_KNOWLEDGE_MAPPINGS.json');
 if(typeof taxonomy.metadata?.attribution!=='string'||!taxonomy.metadata.attribution.trim()||typeof taxonomy.metadata.license!=='string'||!taxonomy.metadata.license.trim())throw Error('CLASSIFICATION_RIGHTS_MISSING');
 const nodes=taxonomy.entries.map(n=>{
  const shape={top_level:['primary',1],intermediate:['primary',2],named_alpha_leaf:['primary',3],hyphen_auxiliary:['auxiliary',2],other_bucket:['other',3]}[n.node_kind];
  if(!shape||typeof n.code!=='string'||typeof n.label_en!=='string')throw Error('INVALID_CLASSIFICATION');
  return {id:topicID(n.code),code:n.code,name:n.label_en,nameZh:typeof n.label_zh_unofficial==='string'?n.label_zh_unofficial:'',kind:shape[0],level:shape[1],parentId:n.parent_code===null?null:topicID(n.parent_code)};
 });
 const ids=new Set(nodes.map(n=>n.id));if(ids.size!==nodes.length||nodes.some(n=>n.parentId!==null&&!ids.has(n.parentId)))throw Error('INVALID_CLASSIFICATION');
 const recordIDs=new Map();
 for(const [path,claim] of files) {
  if(!path.startsWith('Knowledge_JSON/')||path.endsWith('_Index.json'))continue;
  const corpus=await read(path),values=corpus.knowledge_points??corpus.records;
  if(Array.isArray(values))recordIDs.set(path,new Set(values.map(x=>x?.id).filter(x=>typeof x==='string')));
 }
 const sourceRecordIndex=[],seen=new Set();
 for(const mapping of mappings.mappings??[])for(const path of mapping.knowledge_local_primary_paths??[]){
  const full='Knowledge_JSON/'+path,ids=recordIDs.get(full);if(!ids)continue;
  for(const recordId of mapping.knowledge_record_ids??[]) {
   if(!ids.has(recordId))throw Error('MAPPING_RECORD_MISSING');
   const ref={sourceId:mapping.source_id,workFamilyId:mapping.work_family_id,recordId,path,sha256:files.get(full).sha256};
   if(typeof ref.sourceId!=='string'||typeof ref.workFamilyId!=='string')throw Error('INVALID_SOURCE_MAPPING');
   const key=JSON.stringify(ref);if(!seen.has(key)){sourceRecordIndex.push(ref);seen.add(key);}
  }
 }
 return {manifest,nodes,sourceRecordIndex,rawClassificationSHA:taxonomy.metadata.source_checksums_sha256['MSC_2020.csv'],attribution:taxonomy.metadata.attribution,license:taxonomy.metadata.license};
}
