import {numberedFields} from '../pagination/schemas';
import {z} from 'zod';
const uuid=z.string().regex(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/),kid=z.string().regex(/^k-[0-9a-f]{56}$/),time=z.iso.datetime({offset:true}),seq=z.number().int().min(0).max(Number.MAX_SAFE_INTEGER);
const text=(max:number)=>z.string().refine(v=>v.isWellFormed()&&Array.from(v).length<=max);
const status=z.enum(['unshared','pending','shared','returned','withdrawn','hidden']);
const action=z.enum(['submit','withdraw','approve','return','hide']);
const head=z.object({actorId:uuid,knowledgeId:kid,status,sequence:seq,submissionId:uuid.nullable(),approvedAt:time.nullable(),lastDecisionReason:text(1000)}).strict();
const summaryFields={submissionId:uuid,title:text(120).refine(v=>v.trim().length>0),excerpt:text(160),authorName:z.string().regex(/^[a-z0-9_]{3,32}$/),approvedAt:time,knowledgeId:kid,knowledgeTitle:z.string(),knowledgeTitleZh:z.string(),topicKeys:z.array(z.string()),materialChanged:z.boolean()};
const summary=z.object(summaryFields).strict();
const body=text(16000).refine(v=>new TextEncoder().encode(v).length<=65536);
const detail=z.object({...summaryFields,body}).strict();
const cursor=z.string().max(512).nullable();
const own=z.object({knowledgeId:kid,knowledgeTitle:z.string(),knowledgeTitleZh:z.string(),topicKeys:z.array(z.string()),excerpt:text(160),updatedAt:time,available:z.boolean(),noteRevision:seq.min(1),share:head}).strict();
const ownPage=z.object({actorId:uuid,items:z.array(own).max(50),nextCursor:cursor}).strict();
const sharedPage=z.object({actorId:uuid,items:z.array(summary).max(50),nextCursor:cursor}).strict();
const adminFields={submissionId:uuid,authorName:z.string().regex(/^[a-z0-9_]{3,32}$/),knowledgeId:kid,title:text(120),submittedAt:time,status,sequence:seq};
const adminSummary=z.object(adminFields).strict();
const ref=z.object({id:kid,contentSha256:z.string().regex(/^[a-f0-9]{64}$/),sourceKind:z.literal('managed')}).strict();
const event=z.object({action,actorName:z.string().regex(/^[a-z0-9_]{3,32}$/),reason:text(1000),sequence:seq,recordedAt:time}).strict();
const adminDetail=z.object({actorId:uuid,...adminFields,body,noteRevision:seq.min(1),noteKnowledgeRef:ref,knowledgeTitle:z.string(),knowledgeTitleZh:z.string(),topicKeys:z.array(z.string()),approvedAt:time.nullable(),events:z.array(event).max(50),nextEventsCursor:cursor}).strict();
const receipt=z.object({submissionId:uuid,knowledgeId:kid,action,status,sequence:seq}).strict();
const health=z.object({capability:z.boolean(),schemaReady:z.boolean()}).strict();
export type NoteRoute={kind:string;private:boolean;method:'GET'|'POST';list:boolean};
const noteKid='k-[0-9a-f]{56}',noteUuid='[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}';
export function routeForNote(path:string):NoteRoute|null{
 const make=(kind:string,privateRoute=false,method:'GET'|'POST'='GET',list=false)=>({kind,private:privateRoute,method,list});
 if(path==='/api/v3/note-sharing/health')return make('health');if(path==='/api/v3/shared-notes/latest')return make('latest',false,'GET',true);
 if(path==='/api/v3/study/notes')return make('own',true,'GET',true);if(path==='/api/v3/shared-notes')return make('shared',true,'GET',true);if(path==='/api/v3/admin/note-shares')return make('admin',true,'GET',true);
 if(new RegExp('^/api/v3/shared-notes/public/'+noteUuid+'$').test(path))return make('public-detail');
 if(new RegExp('^/api/v3/shared-notes/'+noteUuid+'$').test(path))return make('member-detail',true);
 if(new RegExp('^/api/v3/admin/note-shares/'+noteUuid+'$').test(path))return make('admin-detail',true);
 if(new RegExp('^/api/v3/admin/note-shares/'+noteUuid+'/(approve|return|hide)$').test(path))return make('receipt',true,'POST');
 if(new RegExp('^/api/v3/study/knowledge/'+noteKid+'/note-sharing$').test(path))return make('head',true);
 if(new RegExp('^/api/v3/study/knowledge/'+noteKid+'/note-sharing/(submit|withdraw)$').test(path))return make('receipt',true,'POST');return null;
}
export function schemaForNotePath(path:string,method='GET'){const url=new URL(path,'http://localhost');path=url.pathname;const numbered=url.searchParams.get('pagination')==='page';if(numbered){if(method!=='GET')return z.never();if(path==='/api/v3/study/notes')return ownPage.omit({nextCursor:true}).extend(numberedFields()).strict();if(path==='/api/v3/shared-notes')return sharedPage.omit({nextCursor:true}).extend(numberedFields()).strict();if(path==='/api/v3/admin/note-shares')return z.object({actorId:uuid,items:z.array(adminSummary).max(50),...numberedFields()}).strict();return z.never();}
 const route=routeForNote(path);if(!route||route.method!==method)return z.never();
 switch(route.kind){case 'health':return health;case 'latest':return z.object({items:z.array(summary).max(10)}).strict();case 'own':return ownPage;case 'head':return head;case 'shared':return sharedPage;case 'public-detail':return detail;case 'member-detail':return z.object({actorId:uuid,note:detail}).strict();case 'admin':return z.object({actorId:uuid,items:z.array(adminSummary).max(50),nextCursor:cursor}).strict();case 'admin-detail':return adminDetail;case 'receipt':return receipt;default:return z.never()}
}
export const noteProblem=z.object({error:z.object({code:z.string(),message:z.string(),requestId:z.string(),path:z.string().optional()}).strict()}).strict();
export function isNoteSharingPath(path:string){return path==='/api/v3/study/notes'||/^\/api\/v3\/(?:shared-notes|note-sharing|admin\/note-shares)(?:\/|$)/.test(path)||new RegExp('^/api/v3/study/knowledge/'+noteKid+'/note-sharing(?:/|$)').test(path)}
