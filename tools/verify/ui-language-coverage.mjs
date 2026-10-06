export function validateUiCoverage(manifest,policy){
 if(manifest?.schemaVersion!==1)throw Error('invalid coverage schema');
 const keys=new Set(policy.keys),scenarios=new Set(policy.scenarios);
 for(const group of ['pages','components']){
  if(!Array.isArray(manifest[group]))throw Error('missing coverage group');const seen=new Set();
  for(const row of manifest[group]){
   if(!row||typeof row.file!=='string'||seen.has(row.file))throw Error('duplicate or invalid coverage file');seen.add(row.file);
   if(row.complete!==true)throw Error('incomplete coverage: '+row.file);
   if(typeof row.module!=='string'||!row.module)throw Error('missing module coverage');
   for(const name of ['messageKeys','rawDataAreas','scenarios']){
    const noText=name==='messageKeys'&&group==='components'&&row.noOwnText===true&&(policy.noOwnText??[]).includes(row.file);
    if(!Array.isArray(row[name])||(!row[name].length&&!noText)||row[name].some(v=>typeof v!=='string'||!v))throw Error('missing '+name+' coverage: '+row.file);
   }
   if(row.messageKeys.some(k=>!keys.has(k))||row.scenarios.some(s=>!scenarios.has(s)))throw Error('unknown key or scenario coverage: '+row.file);
  }
  for(const file of policy[group])if(!seen.has(file))throw Error('missing coverage: '+file);
  for(const file of seen)if(!policy[group].includes(file))throw Error('unknown coverage file: '+file);
 }
 return true;
}
