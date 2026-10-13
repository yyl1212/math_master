import {it,expect} from 'vitest';
import {schemaForNotePath} from './schemas';
const id='51390b7d-a8e9-4e98-a3ae-22101c20b309',kid='k-'+'a'.repeat(56);
const summary={submissionId:id,title:'数学笔记',excerpt:'自己的理解',authorName:'learner',approvedAt:'2026-10-13T00:00:00Z',knowledgeId:kid,knowledgeTitle:'Mathematics',knowledgeTitleZh:'数学',topicKeys:['97F40'],materialChanged:false};
it('accepts only ten public summaries and rejects private fields',()=>{
 const schema=schemaForNotePath('/api/v3/shared-notes/latest');expect(schema.safeParse({items:Array(10).fill(summary)}).success).toBe(true);expect(schema.safeParse({items:Array(11).fill(summary)}).success).toBe(false);
 expect(schema.safeParse({items:[{...summary,body:'unsubmitted private body'}]}).success).toBe(false);
 expect(schemaForNotePath('/api/v3/note-sharing/health').safeParse({capability:true,schemaReady:true,body:'private'}).success).toBe(false);
});
it('counts Unicode characters rather than UTF16 code units',()=>{
 const schema=schemaForNotePath('/api/v3/shared-notes/public/'+id);expect(schema.safeParse({...summary,body:'😀'.repeat(16000)}).success).toBe(true);expect(schema.safeParse({...summary,body:'a'.repeat(16001)}).success).toBe(false);
});
it('keeps authenticated detail wrapped and strict',()=>{
 const schema=schemaForNotePath('/api/v3/shared-notes/'+id);expect(schema.safeParse({actorId:id,note:{...summary,body:'数学'}}).success).toBe(true);expect(schema.safeParse({...summary,body:'数学'}).success).toBe(false);
});
