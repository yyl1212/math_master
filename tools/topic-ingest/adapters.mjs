import {createHash} from 'node:crypto';
import {isAbsolute} from 'node:path';
import {parseSourceJSON} from './json.mjs';
const hash=b=>createHash('sha256').update(b).digest('hex');
const text=v=>typeof v==='string'?v:'';
const strings=v=>Array.isArray(v)&&v.every(x=>typeof x==='string')?[...v]:[];
const pathOK=p=>typeof p==='string'&&p.length>0&&!isAbsolute(p)&&!p.includes('\\')&&!p.includes('\0')&&!p.split('/').some(x=>!x||x==='.'||x==='..');
function canonical(v) {if(Array.isArray(v))return v.map(canonical);if(v&&typeof v==='object')return Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])]));return v;}
export function normalizePrimaryFile({bytes,packageId,sourceId,path,sha256}) {
  if(!pathOK(path))throw Error('PATH_ESCAPE');
  if(!(bytes instanceof Uint8Array)||bytes.byteLength>64<<20)throw Error('SOURCE_LIMIT_EXCEEDED');
  if(typeof sha256!=='string'||hash(bytes)!==sha256)throw Error('SOURCE_DIGEST_MISMATCH');
  if(typeof sourceId!=='string'||!sourceId.trim()||sourceId.includes('\0')||typeof packageId!=='string')throw Error('SOURCE_ID_REQUIRED');
  const root=parseSourceJSON(bytes);
  if(!root||typeof root!=='object'||Array.isArray(root))throw Error('UNSUPPORTED_CORPUS');
  const forms=['knowledge_points','records','capabilities'].filter(k=>Array.isArray(root[k]));
  if(forms.length!==1)throw Error(forms.length?'AMBIGUOUS_CORPUS':'UNSUPPORTED_CORPUS');
  const records=[],auxiliary=[],issues=[];
  for(const raw of root[forms[0]]) {
    if(!raw||typeof raw!=='object'||Array.isArray(raw)||typeof raw.id!=='string'||!raw.id.trim()){issues.push({code:'RECORD_ID_REQUIRED',path});continue;}
    if(forms[0]==='capabilities'){auxiliary.push({originalId:raw.id,sourceId,title:text(raw.title),raw,path,rawSHA:sha256});continue;}
    const title=text(raw.title)||text(raw.title?.en)||text(raw.name);
    const statement=text(raw.statement)||text(raw.core_statement)||text(raw.definition);
    const originalKind=text(raw.kind)||text(raw.type)||text(raw.knowledge_type);
    let proof=text(raw.proof)||text(raw.proof?.text)||text(raw.proof_text);
    let proofScope=text(raw.proof_scope)||text(raw.proof?.scope)||'unspecified';
    if(!proof&&text(raw.proof_idea?.text)){proof=raw.proof_idea.text;proofScope='sketch';}
    const conditions=strings(raw.conditions??raw.hypotheses);
    if(!title)issues.push({code:'TITLE_MISSING',path,originalId:raw.id});
    if(!statement)issues.push({code:'STATEMENT_MISSING',path,originalId:raw.id});
    if((raw.conditions??raw.hypotheses)!=null&&(!Array.isArray(raw.conditions??raw.hypotheses)||(raw.conditions??raw.hypotheses).some(x=>typeof x!=='string')))issues.push({code:'CONDITION_FORMAT_REQUIRED',path,originalId:raw.id});
    const sourceLocations=Array.isArray(raw.provenance)?raw.provenance:Array.isArray(raw.source_locations)?raw.source_locations:raw.provenance?[raw.provenance]:[];
    records.push({originalId:raw.id,sourceId,packageId,path,title,originalKind,statement,conditions,proof,proofScope,sourceLocations,rawSHA:sha256,semanticSHA:hash(JSON.stringify(canonical(raw))),raw,sourceMetadata:root.source??root.book??null,rightsMetadata:root.license??root.rights??root.source_rights??null});
  }
  return {records,auxiliary,issues};
}
