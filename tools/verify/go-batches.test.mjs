import test from'node:test';import assert from'node:assert/strict';import{partitionTests}from'./go-batches.mjs';
test('bounded partitions cover every selected test exactly once without hiding capacity cases',()=>{
 const names=Array.from({length:207},(_,i)=>'TestOriginal'+i).concat(['TestTopicPairAtomicFailure','TestWorkflowCapacityEnvelope','TestQuestionMaximumLegalWorkflow','TestLearningState']);
 const skip='^Test(Learning|Topic)|^Test(WorkflowCapacityEnvelope|QuestionMaximumLegalWorkflow)$';
 const result=partitionTests(names,{size:70,skip});assert.equal(result.selected.length,207);assert.equal(result.batches.length,3);assert.deepEqual(result.batches.flat(),[...names.slice(0,207)].sort());assert(result.batches.every(b=>b.length<=70));
});
test('bad, duplicated names and unbounded batches are rejected',()=>{for(const args of [[['TestGood','TestGood'],{size:70}],[['TestInjected|Other'],{size:70}],[['TestGood'],{size:81}],[['TestGood'],{size:0}]])assert.throws(()=>partitionTests(...args));});
test('example and fuzz seed tests remain in the registered test inventory',()=>{const names=['Example','FuzzValue','TestPlain'];assert.deepEqual(partitionTests(names).selected,names);});

test('only exact newly owned review tests leave old groups; unknown and original tests remain',()=>{const result=partitionTests(['TestReviewSchemaConstraintsAndDigests','TestReviewCapacityRead10000','TestReviewUnknownLegacy','TestOriginal']);assert.deepEqual(result.selected,['TestOriginal','TestReviewUnknownLegacy']);assert.deepEqual(result.studyReview,['TestReviewCapacityRead10000','TestReviewSchemaConstraintsAndDigests']);});

test('numbered tests are assigned once to dedicated query workflow',async()=>{const {queryPaginationStoreTests}=await import('./go-batches.mjs');const p=partitionTests([...queryPaginationStoreTests,'TestNumberedUnknownFuture','TestOriginal']);assert.deepEqual(p.queryPagination,[...queryPaginationStoreTests].sort());assert.deepEqual(p.selected,['TestNumberedUnknownFuture','TestOriginal']);assert.equal(new Set([...p.selected,...p.queryPagination]).size,p.selected.length+p.queryPagination.length);});
