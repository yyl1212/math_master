import test from'node:test';import assert from'node:assert/strict';import{partitionTests}from'./go-batches.mjs';
test('bounded partitions cover every selected test exactly once without hiding capacity cases',()=>{
 const names=Array.from({length:207},(_,i)=>'TestOriginal'+i).concat(['TestTopicPairAtomicFailure','TestWorkflowCapacityEnvelope','TestQuestionMaximumLegalWorkflow','TestLearningState']);
 const skip='^Test(Learning|Topic)|^Test(WorkflowCapacityEnvelope|QuestionMaximumLegalWorkflow)$';
 const result=partitionTests(names,{size:70,skip});assert.equal(result.selected.length,207);assert.equal(result.batches.length,3);assert.deepEqual(result.batches.flat(),[...names.slice(0,207)].sort());assert(result.batches.every(b=>b.length<=70));
});
test('bad, duplicated names and unbounded batches are rejected',()=>{for(const args of [[['TestGood','TestGood'],{size:70}],[['TestInjected|Other'],{size:70}],[['TestGood'],{size:81}],[['TestGood'],{size:0}]])assert.throws(()=>partitionTests(...args));});
test('example and fuzz seed tests remain in the registered test inventory',()=>{const names=['Example','FuzzValue','TestPlain'];assert.deepEqual(partitionTests(names).selected,names);});
