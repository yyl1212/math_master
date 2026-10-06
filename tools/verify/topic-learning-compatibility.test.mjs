import test from 'node:test';import assert from 'node:assert/strict';import {mkdtempSync,readFileSync,writeFileSync,mkdirSync,rmSync}from'node:fs';import{tmpdir}from'node:os';import{join}from'node:path';import{spawnSync}from'node:child_process';
import{verifyTopicCompatibility}from'./topic-learning-compatibility.mjs';
const root=new URL('../../',import.meta.url).pathname;
test('approved topic stage retains every original migration and schema',()=>{const r=verifyTopicCompatibility({root,stage:'taxonomy'});assert.equal(r.ok,true);assert.equal(r.originalMigrations,9)});
test('changed migration, original schema and unlisted file are never excused',()=>{
 const folder=mkdtempSync(join(tmpdir(),'topic-compatibility-'));try{
 const archive=spawnSync('git',['archive','bca91cc98d75af53963789f28cdda5ca96871e17'],{cwd:root,maxBuffer:64<<20});assert.equal(archive.status,0);assert.equal(spawnSync('tar',['-xf','-','-C',folder],{input:archive.stdout}).status,0);
 mkdirSync(join(folder,'api'),{recursive:true});writeFileSync(join(folder,'api/topic-learning-compatibility-baseline.json'),readFileSync(join(root,'api/topic-learning-compatibility-baseline.json')));
 for(const path of ['db/migrations/00001_content_foundation.sql','schemas/content-package.schema.json','backend/internal/content/model.go']){const full=join(folder,path),before=readFileSync(full);writeFileSync(full,Buffer.concat([before,Buffer.from('\n tampered')]));assert.throws(()=>verifyTopicCompatibility({root:folder,stage:'taxonomy'}),new RegExp(path.replaceAll('.','\\.')));writeFileSync(full,before)}
 const policyPath=join(folder,'api/topic-learning-compatibility-baseline.json'),policy=JSON.parse(readFileSync(policyPath));policy.exceptions['api/openapi.yaml'].taxonomy.sha256='0'.repeat(64);writeFileSync(policyPath,JSON.stringify(policy));assert.throws(()=>verifyTopicCompatibility({root:folder,stage:'taxonomy'}),/baseline|policy/);
 }finally{rmSync(folder,{recursive:true,force:true})}
});
