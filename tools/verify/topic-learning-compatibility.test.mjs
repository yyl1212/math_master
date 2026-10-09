import test from 'node:test';import assert from 'node:assert/strict';import {mkdtempSync,readFileSync,writeFileSync,mkdirSync,rmSync}from'node:fs';import{tmpdir}from'node:os';import{join}from'node:path';import{spawnSync}from'node:child_process';
import{verifyTopicCompatibility}from'./topic-learning-compatibility.mjs';
const root=new URL('../../',import.meta.url).pathname;
test('approved topic stage retains every original migration and schema',()=>{const r=verifyTopicCompatibility({root,stage:process.env.TOPIC_COMPATIBILITY_STAGE??'taxonomy'});assert.equal(r.ok,true);assert.equal(r.originalMigrations,9)});
test('changed migration, original schema and unlisted file are never excused',()=>{
 const folder=mkdtempSync(join(tmpdir(),'topic-compatibility-'));try{
 const archive=spawnSync('git',['archive','bca91cc98d75af53963789f28cdda5ca96871e17'],{cwd:root,maxBuffer:64<<20});assert.equal(archive.status,0);assert.equal(spawnSync('tar',['-xf','-','-C',folder],{input:archive.stdout}).status,0);
 mkdirSync(join(folder,'api'),{recursive:true});writeFileSync(join(folder,'api/topic-learning-compatibility-baseline.json'),readFileSync(join(root,'api/topic-learning-compatibility-baseline.json')));
 for(const path of ['db/migrations/00001_content_foundation.sql','schemas/content-package.schema.json','backend/internal/content/model.go']){const full=join(folder,path),before=readFileSync(full);writeFileSync(full,Buffer.concat([before,Buffer.from('\n tampered')]));assert.throws(()=>verifyTopicCompatibility({root:folder,stage:process.env.TOPIC_COMPATIBILITY_STAGE??'taxonomy'}),new RegExp(path.replaceAll('.','\\.')));writeFileSync(full,before)}
 const policyPath=join(folder,'api/topic-learning-compatibility-baseline.json'),policy=JSON.parse(readFileSync(policyPath));policy.exceptions['api/openapi.yaml'].taxonomy.sha256='0'.repeat(64);writeFileSync(policyPath,JSON.stringify(policy));assert.throws(()=>verifyTopicCompatibility({root:folder,stage:process.env.TOPIC_COMPATIBILITY_STAGE??'taxonomy'}),/baseline|policy/);
 }finally{rmSync(folder,{recursive:true,force:true})}
});
test('cutover API inverse only restores exact approved semantic entries',async()=>{const {inverseTopicAPI}=await import('./topic-learning-compatibility.mjs');const api=JSON.parse(readFileSync(join(root,'api/openapi.yaml')));if(!['cutover','managed'].includes(process.env.TOPIC_COMPATIBILITY_STAGE))return;const restored=inverseTopicAPI(api);assert(!restored.components.schemas.QuestionErrorCode.enum.includes('MODULE_RETIRED'));const changed=structuredClone(api);changed.components.schemas.QuestionErrorCode.enum.push('UNAPPROVED_CODE');assert(inverseTopicAPI(changed).components.schemas.QuestionErrorCode.enum.includes('UNAPPROVED_CODE'));changed.components.schemas.QuestionIdentity.properties.unapproved={type:'string'};assert.deepEqual(inverseTopicAPI(changed).components.schemas.QuestionIdentity.properties.unapproved,{type:'string'})});

test('managed compatibility is an explicit fourth stage with pinned source bytes',()=>{const r=verifyTopicCompatibility({root,stage:'managed'});assert.equal(r.ok,true);assert.equal(r.stage,'managed')});

test('managed refuses unlisted files, changed code bytes and unapproved API fields',()=>{
 const folder=mkdtempSync(join(tmpdir(),'managed-compatibility-'));try{
 const p=JSON.parse(readFileSync(join(root,'api/topic-learning-compatibility-baseline.json')));const names=new Set([...Object.keys(p.files),...Object.values(p.newFiles).flat(),'api/topic-learning-compatibility-baseline.json']);for(const path of names){mkdirSync(join(folder,path,'..'),{recursive:true});writeFileSync(join(folder,path),readFileSync(join(root,path)))}
 assert.equal(verifyTopicCompatibility({root:folder,stage:'managed'}).ok,true);
 for(const path of ['backend/internal/knowledgeadmin/validate.go','db/migrations/00012_topic_cutover.sql','db/migrations/00013_admin_knowledge.sql']){const full=join(folder,path),before=readFileSync(full);writeFileSync(full,Buffer.concat([before,Buffer.from('\n altered')]));assert.throws(()=>verifyTopicCompatibility({root:folder,stage:'managed'}),/unapproved|original migration/);writeFileSync(full,before)}
 const apiPath=join(folder,'api/openapi.yaml'),apiBefore=readFileSync(apiPath),api=JSON.parse(apiBefore);api.components.schemas.QuestionIdentity.properties.unapproved={type:'string'};writeFileSync(apiPath,JSON.stringify(api));assert.throws(()=>verifyTopicCompatibility({root:folder,stage:'managed'}),/unapproved|original API/);writeFileSync(apiPath,apiBefore);
 writeFileSync(join(folder,'unlisted-managed-code.js'),'unapproved');assert.throws(()=>verifyTopicCompatibility({root:folder,stage:'managed'}),/unlisted topic file/);
 }finally{rmSync(folder,{recursive:true,force:true})}
});
