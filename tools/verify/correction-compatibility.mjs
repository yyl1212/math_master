import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
export const canonical=v=>Array.isArray(v)?v.map(canonical):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])])):v;
export const digest=v=>createHash('sha256').update(JSON.stringify(canonical(v))).digest('hex');
const originalReasons=['knowledge-updated','knowledge-withdrawn','unit-withdrawn','asset-withdrawn','template-withdrawn','instance-withdrawn','blueprint-withdrawn','exposed-after-creation'];
const originalBlueprintReasons=['no-blueprint','no-instances','insufficient-coverage','exposure-cooldown','recent-assessment-exclusion'];
const uuid={type:'string',pattern:'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'};
// Every modified JSON pointer is asserted before reversing it. Other fields
// survive into the old full-schema digest and therefore cannot be hidden.
export function inverseApprovedCorrectionChanges(actual) {
 const api=structuredClone(actual),s=api.components.schemas;
 const name='LearningQualificationView',q=s[name];
 assert.deepEqual(q.properties.correctionId,{oneOf:[uuid,{type:'null'}]},name+' /properties/correctionId');
 assert.deepEqual(q.required,['knowledge','kind','evidenceAttemptId','completedEventId','validity','correctionId'],name+' /required');
 delete q.properties.correctionId;q.required.pop();
 assert.deepEqual(s.LearningRestrictionReason.enum,[...originalReasons,'grading-issue'],'LearningRestrictionReason /enum');s.LearningRestrictionReason.enum=[...originalReasons];
 function reasons(value,label,old) {
  assert.deepEqual(value.items.enum,[...old,'grading-issue'],label+' /reasons/items/enum');
  assert.equal(value.maxItems,old.length+1,label+' /reasons/maxItems');
  value.items.enum=[...old];value.maxItems=old.length;
 }
 for(const k of ['LearningPrerequisiteState','LearningPathNode','LearningResultItem'])reasons(s[k].properties.reasons,k,originalReasons);
 reasons(s.LearningBlueprintOption.properties.reasons,'LearningBlueprintOption',originalBlueprintReasons);
 assert.equal(s.LearningResultView.oneOf.length,3,'LearningResultView /oneOf');
 for(const [i,v]of s.LearningResultView.oneOf.entries())reasons(v.properties.reasons,'LearningResultView /oneOf/'+i,originalReasons);
 return api;
}
export function compareCorrectionContracts(actual,baseline) {
 const restored=inverseApprovedCorrectionChanges(actual);
 for(const[section,entries]of Object.entries(baseline.sections)) {
  const values=section==='paths'?restored.paths:restored.components[section];
  for(const[key,sha]of Object.entries(entries)){assert(Object.hasOwn(values,key),section+': '+key);assert.equal(digest(values[key]),sha,section+': '+key);}
 }
}
