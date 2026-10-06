import {createHash} from 'node:crypto';
const hash=v=>createHash('sha256').update(typeof v==='string'?v:JSON.stringify(v)).digest('hex');
const key=r=>JSON.stringify([r.sourceId,r.originalId]);
const id=v=>typeof v==='string'&&/^[a-z][a-z0-9-]{0,63}$/.test(v);
const types=new Set(['concept','definition','axiom','theorem','corollary','method','mathematical-thinking']);
const texts=v=>Array.isArray(v)&&v.every(x=>typeof x==='string')?[...v]:[];
const string=v=>typeof v==='string'?v:'';
export function buildDraftInputs({capture,records,resolutions,legacyCatalogue}) {
  if(capture?.accepted!==true||!/^[a-f0-9]{64}$/.test(capture.snapshotId??''))throw Error('BATCH_NOT_ACCEPTED');
  if(!Array.isArray(records)||!Array.isArray(resolutions)||!Number.isSafeInteger(legacyCatalogue?.version)||legacyCatalogue.version<1)throw Error('INVALID_DRAFT_INPUT');
  const issues=[],packages=[],resolved=new Map(),unique=new Map(),conflicts=new Set();
  const captured=new Map((capture.sourceFiles??[]).map(f=>[f.path,f.sha256]));
  const concrete=new Set((capture.nodes??[]).filter(n=>n.kind==='primary'&&n.level===3).map(n=>n.id));
  for(const r of resolutions){if(resolved.has(key(r)))throw Error('DUPLICATE_MAPPING');resolved.set(key(r),r);}
  for(const record of records){
    const k=key(record),prior=unique.get(k);
    if(prior&&prior.semanticSHA!==record.semanticSHA){conflicts.add(k);issues.push({code:'RECORD_CONFLICT',sourceId:record.sourceId,originalId:record.originalId});}
    else if(!prior)unique.set(k,record);
  }
  const websiteIDs=new Map(),entries=[];
  const issue=(code,r)=>issues.push({code,sourceId:r.sourceId,originalId:r.originalId});
  for(const [k,r] of [...unique].sort(([a],[b])=>a<b?-1:a>b?1:0)){
    if(conflicts.has(k))continue;
    if(captured.get('Knowledge_JSON/'+r.path)!==r.rawSHA){issue('SOURCE_NOT_CAPTURED',r);continue;}
    const resolution=resolved.get(k);
    if(!resolution){issue('MAPPING_REQUIRED',r);continue;}
    if(!types.has(resolution.type)){issue('TYPE_MAPPING_REQUIRED',r);continue;}
    if(!Array.isArray(resolution.topicIds)||!resolution.topicIds.length||new Set(resolution.topicIds).size!==resolution.topicIds.length||resolution.topicIds.some(t=>!concrete.has(t))){issue('TOPIC_MAPPING_REQUIRED',r);continue;}
    if(!id(resolution.websiteId)||!Number.isSafeInteger(resolution.version??1)||(resolution.version??1)<1||(resolution.version??1)>2147483647||typeof resolution.workFamilyId!=='string'||!resolution.workFamilyId.trim()){issue('IDENTITY_MAPPING_REQUIRED',r);continue;}
    if(websiteIDs.has(resolution.websiteId)){issue('WEBSITE_ID_CONFLICT',r);continue;}
    websiteIDs.set(resolution.websiteId,k);
    const knowledge={id:resolution.websiteId,version:resolution.version??1,domainIds:[],topicIds:[],type:resolution.type,title:r.title,titleZh:string(resolution.titleZh),statement:r.statement,scope:string(resolution.scope),system:string(resolution.system),objectives:texts(resolution.objectives),conditions:[...r.conditions],proof:r.proofScope==='full'||r.proofScope==='complete'?r.proof:'',sources:[],relations:[]};
    // Scope, explanations and rights need an editor's explicit completion.
    // Neither source "approved" labels nor external author IDs are authority.
    const unit={id:'u-'+hash([knowledge.id,knowledge.version]).slice(0,24),version:knowledge.version,knowledge:{id:knowledge.id,version:knowledge.version},angles:[],examples:texts(r.raw?.examples),counterexamples:texts(r.raw?.counterexamples),assetIds:[]};
    const source={knowledge:{id:knowledge.id,version:knowledge.version},batchSha256:capture.snapshotId,relativePath:r.path,sha256:r.rawSHA,legacyId:r.originalId,note:'Original source '+r.sourceId+'; proof scope: '+r.proofScope+'; content and rights require website review.'};
    const assignment={knowledge:{id:knowledge.id,version:knowledge.version},topicIds:[...resolution.topicIds].sort(),sourceRefs:[{sourceId:r.sourceId,workFamilyId:resolution.workFamilyId,recordId:r.originalId,path:r.path,sha256:r.rawSHA}],sourceBatchSHA:capture.snapshotId};
    if(Buffer.byteLength(JSON.stringify({knowledge,unit,source,assignment}))>2<<20){issue('DRAFT_RECORD_TOO_LARGE',r);continue;}
    entries.push({knowledge,unit,source,assignment});
  }
  let chunk=[],size=0;
  const flush=()=>{
    if(!chunk.length)return;
    const digest=hash([capture.snapshotId,chunk]);
    const pkg={schemaVersion:1,id:'topic-import-'+digest.slice(0,24),version:1,knowledge:chunk.map(x=>x.knowledge),units:chunk.map(x=>x.unit),paths:[],assets:[]};
    packages.push({kind:'topic-draft',schemaVersion:1,draft:{catalogueVersion:legacyCatalogue.version,package:pkg,assetBytes:[],sourceMap:chunk.map(x=>x.source)},assignments:chunk.map(x=>x.assignment),sourceBatchSHA:capture.snapshotId});
    chunk=[];size=0;
  };
  for(const entry of entries){
    const bytes=Buffer.byteLength(JSON.stringify(entry));
    if(chunk.length===100||size+bytes>1800000)flush();
    chunk.push(entry);size+=bytes;
  }
  flush();return {packages,issues};
}
