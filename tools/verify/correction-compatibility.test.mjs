import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync, readdirSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {compareCorrectionContracts, inverseApprovedCorrectionChanges} from './correction-compatibility.mjs';
const root=new URL('../../',import.meta.url),read=p=>readFileSync(new URL(p,root),'utf8');
const baseline=JSON.parse(read('api/correction-compatibility-baseline.json')),api=JSON.parse(read('api/openapi.yaml'));
test('P5a product baseline retains 88 paths, 220 schemas, 32 responses and three security schemes with only approved pointers',()=>{
 assert.equal(baseline.baseCommit,'dad438d13b37d1053e3bf42e7c32e3829e4518a6');assert.deepEqual(baseline.counts,{paths:88,schemas:220,responses:32,securitySchemes:3});
 for(const [s,n]of Object.entries(baseline.counts))assert.equal(Object.keys(baseline.sections[s]).length,n);
 compareCorrectionContracts(api,baseline);
});
test('all seven original migrations retain exact bytes',()=>{assert.equal(Object.keys(baseline.files).length,7);compareFiles()});
function purposeSource(p) {return readdirSync(new URL(p,root)).filter(f=>f.endsWith('.go')&&!f.endsWith('_test.go')).map(f=>read(p+f)).join('\n')}
function comparePurposes(reader=purposeSource) {for(const[p,names]of Object.entries(baseline.mathematicalPurposes))assert.deepEqual([...new Set([...reader(p).matchAll(/"([a-z-]+-v\d+)"/g)].map(m=>m[1]))].sort(),names,p)}
test('all existing mathematical digest purposes retain their exact sets',()=>comparePurposes());
test('removing an old path is rejected',()=>{const a=structuredClone(api);delete a.paths['/api/v1/learning/overview'];assert.throws(()=>compareCorrectionContracts(a,baseline),/paths/)});
test('an unapproved field is never normalized away',()=>{const a=structuredClone(api);a.components.schemas.LearningQualificationView.properties.newScore={type:'integer'};assert.throws(()=>compareCorrectionContracts(a,baseline),/LearningQualificationView/)});
test('changed fixed mathematics and bounds are rejected',()=>{for(const[k,fn]of [['LearningSubmitInput',s=>s.properties.answers.minItems=4],['LearningResultView',s=>s.oneOf[0].properties.score.maximum=6],['LearningResultView',s=>s.oneOf[0].properties.reasons.maxItems=10]]){const a=structuredClone(api);fn(a.components.schemas[k]);assert.throws(()=>compareCorrectionContracts(a,baseline),new RegExp(k))}});
test('approved fields require their exact new shape before inverse normalization',()=>{for(const fn of [a=>delete a.components.schemas.LearningQualificationView.properties.correctionId,a=>a.components.schemas.LearningRestrictionReason.enum.push('unknown-correction'),a=>a.components.schemas.LearningQualificationView.properties.correctionId={type:'string'}]){const a=structuredClone(api);fn(a);assert.throws(()=>inverseApprovedCorrectionChanges(a),/Learning/)}});
test('inverse normalization leaves the source document unchanged',()=>{const before=JSON.stringify(api);inverseApprovedCorrectionChanges(api);assert.equal(JSON.stringify(api),before)});

function compareFiles(reader=read) {for(const[p,h]of Object.entries(baseline.files))assert.equal(createHash('sha256').update(reader(p)).digest('hex'),h,p)}
test('a changed original migration fails protection',()=>{const p=Object.keys(baseline.files)[6];assert.throws(()=>compareFiles(name=>read(name)+(name===p?'\n-- incompatible\n':'')),/00007/)});
test('a changed original purpose fails protection',()=>{const p='backend/internal/assessment/';assert.throws(()=>comparePurposes(name=>purposeSource(name)+(name===p?'\nconst forgedPurpose="assessment-regraded-v9";':'')),/assessment/)});
test('all result variants and additional properties survive old full-schema checks',()=>{for(let i=0;i<3;i++){const a=structuredClone(api);a.components.schemas.LearningResultView.oneOf[i].additionalProperties=true;assert.throws(()=>compareCorrectionContracts(a,baseline),/LearningResultView/)}});
