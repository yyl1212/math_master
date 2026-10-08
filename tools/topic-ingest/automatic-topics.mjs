const topicId=code=>/^\d{2}[A-Z]\d{2}$/.test(code)?'msc-'+code.toLowerCase():code;
const same=(a,b)=>a.length===b.length&&a.every((x,i)=>x===b[i]);

// Only exact, captured records are candidates. Source qualification and local
// review labels never grant website approval or learning completion.
export function automaticTopics(record,{capture,sourceMappings=[]}){
 const concrete=new Set((capture.nodes??[]).filter(n=>n.kind==='primary'&&n.level===3).map(n=>n.id));
 const matches=sourceMappings.filter(m=>m.source_id===record.sourceId&&Array.isArray(m.knowledge_record_ids)&&m.knowledge_record_ids.includes(record.originalId)&&Array.isArray(m.knowledge_local_primary_paths)&&m.knowledge_local_primary_paths.includes(record.path));
 const dot=[...new Set(matches.map(m=>typeof m.msc_code==='string'?topicId(m.msc_code):'').filter(t=>concrete.has(t)))].sort();
 let direct=[];
 for(const field of ['msc_code','msc_codes']){
  if(record.raw?.[field]===undefined)continue;
  const values=field==='msc_code'?[record.raw[field]]:record.raw[field];
  if(!Array.isArray(values)||values.some(x=>typeof x!=='string'||!concrete.has(topicId(x))))return {issue:'TOPIC_MAPPING_REQUIRED'};
  direct.push(...values.map(topicId));
 }
 direct=[...new Set(direct)].sort();
 if(dot.length&&direct.length&&!same(dot,direct))return {issue:'TOPIC_MAPPING_CONFLICT'};
 const topicIds=dot.length?dot:direct;
 if(!topicIds.length||topicIds.length>16)return {issue:'TOPIC_MAPPING_REQUIRED'};
 const sourceRefs=(capture.sourceRecordIndex??[]).filter(r=>r.sourceId===record.sourceId&&r.recordId===record.originalId&&r.path===record.path&&r.sha256===record.rawSHA);
 if(!sourceRefs.length||sourceRefs.length>8)return {issue:'SOURCE_REFERENCE_REQUIRED'};
 const families=new Set(sourceRefs.map(r=>r.workFamilyId));
 if(families.size!==1||[...families].some(x=>typeof x!=='string'||!x.trim())||matches.some(m=>!families.has(m.work_family_id)))return {issue:'SOURCE_REFERENCE_CONFLICT'};
 const unique=new Map(sourceRefs.map(r=>[JSON.stringify(r),r]));
 return {topicIds,sourceRefs:[...unique].sort(([a],[b])=>a.localeCompare(b)).map(([,r])=>({...r})),workFamilyId:sourceRefs[0].workFamilyId};
}
