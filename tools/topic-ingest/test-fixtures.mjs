import fs from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createHash} from 'node:crypto';
export const hash = b => createHash('sha256').update(b).digest('hex');
export const metadataNames = ['MSC2020_Complete_Taxonomy.json','MSC2020_SOURCE_KNOWLEDGE_MAPPINGS.json','MSC2020_DUAL_SOURCE_COVERAGE_MATRIX.json','INDEPENDENT_SOURCE_REGISTRY_CURRENT.json'];
export async function acceptedFixture(t) {
  const base = await fs.mkdtemp(join(tmpdir(),'math-topic-capture-'));
  const sourceRoot = join(base,'source'), outDir = join(base,'out');
  const metadataDir = join(sourceRoot,'Materials/Collection_Metadata/MSC2020');
  const knowledgeDir = join(sourceRoot,'Knowledge_JSON');
  await fs.mkdir(metadataDir,{recursive:true}); await fs.mkdir(join(knowledgeDir,'Fixture'),{recursive:true});
  const csv = Buffer.from('fixture classification\n13-XX\tCommutative algebra\n13Cxx\tModules\n13C60\tModule categories\n');
  const csvSHA = hash(csv);
  const taxonomy = {metadata:{taxonomy:'MSC2020',attribution:'Original technical fixture',license:'Original fixture',source_checksums_sha256:{'MSC_2020.csv':csvSHA}},entries:[
    {code:'13-XX',label_en:'Commutative algebra',node_kind:'top_level',parent_code:null,top_level_code:'13-XX',is_terminal:false},
    {code:'13Cxx',label_en:'Modules',node_kind:'intermediate',parent_code:'13-XX',top_level_code:'13-XX',is_terminal:false},
    {code:'13C60',label_en:'Module categories',node_kind:'named_alpha_leaf',parent_code:'13Cxx',top_level_code:'13-XX',is_terminal:true}
  ]};
  const taxonomyBytes = Buffer.from(JSON.stringify(taxonomy));
  const common = {schema_version:'1.0',candidate_batch:7,staged_only:false,formal_integration_performed:true,classification_raw_sha256:csvSHA};
  const corpus = Buffer.from(JSON.stringify({schema_version:'1',source_id:'fixture-source',knowledge_points:[{id:'record-1',title:'A fixture concept',kind:'definition',statement:'An original test definition.'}]}));
  const primary = 'Fixture/knowledge.json';
  const index = {packages:[{package:'fixture',primary_branch:'Algebra',primary_knowledge_local_relative_paths:[primary],json_files:[{filename:'knowledge.json',local_relative_path:primary,sha256:hash(corpus),recommended_knowledge_entry:true}]}]};
  const sourceRecord = {msc_code:'13C60',source_id:'fixture-source',work_family_id:'fixture-work',knowledge_local_primary_paths:[primary],knowledge_record_ids:['record-1']};
  const metas = {
    [metadataNames[0]]:taxonomyBytes,
    [metadataNames[1]]:Buffer.from(JSON.stringify({...common,summary:{all_formal_records:3},sources:[],mappings:[sourceRecord]})),
    [metadataNames[2]]:Buffer.from(JSON.stringify({...common,last_completed_batch:7,summary:{all_formal_records:3},canonical_taxonomy_reference:{sha256:hash(taxonomyBytes)},entries:[]})),
    [metadataNames[3]]:Buffer.from(JSON.stringify({schema_version:'1.0',candidate_batch:7,staged_only:false,formal_integration_performed:true,sources:[],counting_note:'Original technical fixture.'}))
  };
  for (const [name,bytes] of Object.entries(metas)) await fs.writeFile(join(metadataDir,name),bytes);
  await fs.writeFile(join(metadataDir,'MSC_2020.official.csv'),csv);
  await fs.writeFile(join(knowledgeDir,'Knowledge_JSON_Index.json'),JSON.stringify(index));
  await fs.writeFile(join(knowledgeDir,primary),corpus);
  const clean = async () => fs.rm(base,{recursive:true,force:true});
  t?.after(clean);
  return {sourceRoot,outDir,selectedPrimaryPaths:[primary],base,metadataDir,knowledgeDir,primary,clean};
}
export async function stagedFixture(t) {
  const f = await acceptedFixture(t);
  const p=join(f.metadataDir,metadataNames[1]),m=JSON.parse(await fs.readFile(p));
  m.staged_only=true; await fs.writeFile(p,JSON.stringify(m));return f;
}
export async function changingReadFixture(t) { return acceptedFixture(t); }
