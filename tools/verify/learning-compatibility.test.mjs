import {inverseTopicAPI} from "./topic-learning-compatibility.mjs";
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync,readdirSync} from 'node:fs';
import {createHash} from 'node:crypto';
const root=new URL('../../',import.meta.url);
const read=p=>readFileSync(new URL(p,root),'utf8');
const baseline=JSON.parse(read('api/learning-compatibility-baseline.json'));
const api=JSON.parse(read('api/openapi.yaml'));
const canonical=v=>Array.isArray(v)?v.map(canonical):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])])):v;
const digest=v=>createHash('sha256').update(JSON.stringify(canonical(v))).digest('hex');
function compare(actual){actual=inverseTopicAPI(actual);for(const [section,entries] of Object.entries(baseline.sections)){const values=section==='paths'?actual.paths:actual.components[section];for(const [key,sha] of Object.entries(entries))assert.equal(digest(values[key]),sha,section+': '+key)}}
test('all 54 old paths, 147 schemas, 22 responses and three security schemes retain exact values',()=>{assert.deepEqual(baseline.counts,{paths:54,schemas:147,responses:22,securitySchemes:3});compare(api)});
test('one deliberately changed old field is detected',()=>{const changed=structuredClone(api);changed.components.schemas.QuestionIdentity.properties.version.type='string';assert.throws(()=>compare(changed),/QuestionIdentity/)});
test('original migrations and mathematical purposes remain unchanged',()=>{for(const [p,sha] of Object.entries(baseline.files))assert.equal(createHash('sha256').update(read(p)).digest('hex'),sha,p);for(const [p,names]of Object.entries(baseline.mathematicalPurposes)){const source=readdirSync(new URL(p,root)).filter(f=>f.endsWith('.go')&&!f.endsWith('_test.go')).map(f=>read(p+f)).join('\n');const actual=[...new Set([...source.matchAll(/"([a-z-]+-v\d+)"/g)].map(m=>m[1]))].sort();assert.deepEqual(actual,names)}});
test('all 21 learning operations have named strict contracts and fixed response bounds',()=>{const operations=Object.entries(api.paths).filter(([p])=>p.startsWith('/api/v1/learning/')).flatMap(([,verbs])=>Object.values(verbs));assert.equal(operations.length,21);for(const op of operations){assert(op.operationId.startsWith('learning_'));for(const status of ['400','401','403','404','405','409','413','429','503'])assert(op.responses[status]);if(op.requestBody)assert(op.requestBody.content['application/json'].schema.$ref.startsWith('#/components/schemas/Learning'))}assert.equal(api.components.schemas.LearningSubmitInput.properties.answers.minItems,5);assert.equal(api.components.schemas.LearningSubmitInput.properties.answers.maxItems,5);assert.equal(api.components.schemas.LearningSafeQuestion.oneOf.length,2)});
