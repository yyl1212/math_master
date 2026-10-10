import {it,expect} from 'vitest';import {managedRefSchema,modeSchema,previewSchema,schemaForPath} from './schemas';
it('requires managed reference without a business version',()=>{const r={id:'k-'+'a'.repeat(56),contentSha256:'b'.repeat(64),sourceKind:'managed'};expect(managedRefSchema.safeParse(r).success).toBe(true);expect(managedRefSchema.safeParse({...r,version:1}).success).toBe(false);expect(managedRefSchema.safeParse({...r,sourceKind:'legacy'}).success).toBe(false)});
it('does not disguise an activated capability as legacy mode',()=>{expect(modeSchema.safeParse({mode:'legacy',capability:false}).success).toBe(true);expect(modeSchema.safeParse({mode:'legacy',capability:true}).success).toBe(false)});
it('retains precise upload counts and conflict locations',()=>{const p={importId:'51390b7d-a8e9-4e98-a3ae-22101c20b309',inputSha256:'a'.repeat(64),previewToken:'b1390b7d-a8e9-4e98-a3ae-22101c20b309',items:[{index:0,externalId:'source-raw-id',action:'conflict',topicKeys:['97F40'],errorCode:'ID_CONTENT_CONFLICT',errorPath:'/statement'}],counts:{createdKnowledge:0,linkedTopics:0,skippedItems:0,conflicts:1,invalidItems:0}};expect(previewSchema.parse(p).items[0].externalId).toBe('source-raw-id')});

it("uses the detail contract for manual creation",()=>{expect(schemaForPath("/api/v3/admin/knowledge","POST").safeParse({items:[],total:0,limit:20,offset:0}).success).toBe(false)});

it('accepts explicit bilingual topic names and legacy topic responses but rejects unknown fields',()=>{
 const topic={topicKey:'15-XX',title:'线性代数',titleEn:'Linear algebra',kind:'primary',knowledgeCount:0,items:{items:[],total:0,limit:20,offset:0}};
 const schema=schemaForPath('/api/v3/topics/15-XX');expect(schema.parse(topic)).toMatchObject({title:'线性代数',titleEn:'Linear algebra'});
 const {titleEn,...legacy}=topic;expect(schema.safeParse(legacy).success).toBe(true);
 expect(schema.safeParse({...topic,unknown:'no'}).success).toBe(false);
 expect(schema.safeParse({...topic,titleEn:42}).success).toBe(false);
});
