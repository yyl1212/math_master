import {z} from "zod";
import {validContentString} from "../content/raw-json";
const text=z.string().refine(validContentString);
const id=text.regex(/^[a-z][a-z0-9-]{0,63}$/),sha=text.regex(/^[a-f0-9]{64}$/),uuid=text.regex(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
const count=z.number().int().min(0).max(Number.MAX_SAFE_INTEGER),version=count.min(1).max(2147483647);
const nodeFields={id,code:text,name:text.min(1),nameZh:text,kind:z.enum(["primary","auxiliary","other"]),level:z.number().int().min(1).max(3),parentId:id.nullable()};
const topicID=(code:string)=>"msc-"+code.toLowerCase().replace(/xx$/,"").replace(/-$/,"");
function validNode(n:z.infer<typeof rawNode>):boolean{
 if(n.id!==topicID(n.code)||n.name.trim()==="")return false;
 if(n.kind==="primary"&&n.level===1&&/^\d{2}-XX$/.test(n.code))return n.parentId===null;
 if(n.kind==="primary"&&n.level===2&&/^\d{2}[A-Z]xx$/.test(n.code))return n.parentId===topicID(n.code.slice(0,2)+"-XX");
 if(n.level===3&&/^\d{2}[A-Z]\d{2}$/.test(n.code)&&((n.kind==="primary"&&!n.code.endsWith("99"))||(n.kind==="other"&&n.code.endsWith("99"))))return n.parentId===topicID(n.code.slice(0,3)+"xx");
 return n.kind==="auxiliary"&&n.level===2&&/^\d{2}-\d{2}$/.test(n.code)&&n.parentId===topicID(n.code.slice(0,2)+"-XX");
}
const rawNode=z.object(nodeFields).strict();
export const topicNodeSchema=rawNode.refine(validNode);
export const pairSchema=z.object({knowledgeHead:uuid.nullable(),taxonomyHead:uuid.nullable(),taxonomyVersionId:sha}).strict();
export const knowledgeRefSchema=z.object({id,version,sha256:sha}).strict();
const safePath=(v:string)=>v!==""&&!v.startsWith("/")&&!v.includes(":")&&!v.includes("\\")&&v.split("/").every(s=>s!==""&&s!=="."&&s!=="..");
const boundedText=(maximum:number)=>text.refine(v=>v.trim()!==""&&new TextEncoder().encode(v).byteLength<=maximum);
export const sourceRecordRefSchema=z.object({sourceId:boundedText(128),workFamilyId:boundedText(128),recordId:boundedText(512),path:boundedText(1024).refine(safePath),sha256:sha}).strict();
const unique=(v:string[])=>new Set(v).size===v.length;
export const assignmentInputSchema=z.object({knowledge:z.object({id,version}).strict(),topicIds:z.array(text.regex(/^msc-\d{2}[a-z]\d{2}$/).refine(v=>!v.endsWith("99"))).min(1).max(16).refine(unique),sourceRefs:z.array(sourceRecordRefSchema).min(1).max(8).refine(v=>unique(v.map(r=>JSON.stringify(r)))),sourceBatchSHA:sha}).strict();
export const topicSummarySchema=z.object({...nodeFields,ancestors:z.array(topicNodeSchema).max(2),publishedKnowledgeCount:count,hasChildren:z.boolean()}).strict().refine(validNode);
export const topicDetailSchema=z.object({summary:topicSummarySchema,pair:pairSchema}).strict();
export const topicPageSchema=z.object({items:z.array(topicSummarySchema).max(100),total:count,limit:count.min(1).max(100),offset:count,pair:pairSchema}).strict().refine(v=>v.items.length<=v.limit&&v.items.length<=v.total);
export const knowledgeSummarySchema=z.object({id,version,sha256:sha,title:text,titleZh:text,topicIds:z.array(id).refine(unique)}).strict();
export const knowledgePageSchema=z.object({items:z.array(knowledgeSummarySchema).max(100),total:count,limit:count.min(1).max(100),offset:count,pair:pairSchema}).strict().refine(v=>v.items.length<=v.limit&&v.items.length<=v.total);
export const draftTopicInputSchema=z.object({expectedDraftRevision:count.min(1),expectedAssignmentRevision:count,taxonomyVersionId:sha,member:assignmentInputSchema}).strict();
export const draftTopicViewSchema=z.object({draftId:uuid,draftRevision:count.min(1),assignmentRevision:count,taxonomyVersionId:sha,members:z.array(assignmentInputSchema).max(100),digest:sha,readyToSubmit:z.boolean()}).strict();
const reason=text.refine(v=>[...v].length>=10&&[...v].length<=1000&&new TextEncoder().encode(v).byteLength<=3000&&v.trim()!=="");
export const prepareInputSchema=z.object({submissionIds:z.array(uuid).max(20).refine(unique),expectedPair:pairSchema,reason}).strict();
export const activateInputSchema=z.object({expectedPair:pairSchema,manifestSHA:sha,reason}).strict();
const change=z.object({knowledge:knowledgeRefSchema,oldTopicIds:z.array(id),newTopicIds:z.array(id)}).strict();
export const releaseViewSchema=z.object({id:uuid,status:z.enum(["draft","published"]),pair:pairSchema,manifestSHA:sha,assignmentsSHA:sha,knowledgePublicationId:uuid.nullable(),diff:z.object({added:z.array(knowledgeRefSchema),removed:z.array(knowledgeRefSchema),changedTopicMemberships:z.array(change)}).strict(),createdAt:z.iso.datetime({offset:true})}).strict();

export const releasePageSchema=z.object({items:z.array(releaseViewSchema).max(100),total:count,limit:count.min(1).max(100),offset:count.max(100000),pair:pairSchema}).strict().refine(v=>v.items.length<=v.limit&&v.items.length<=v.total);
