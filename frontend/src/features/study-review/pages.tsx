import 'server-only';
import {headers} from 'next/headers';
import {readServerSession} from '@/lib/auth/server-client';
import {readReview} from '@/lib/study-review/server-client';
import type {Overview,QueuePage,HistoryPage} from '@/lib/study-review/types';
import {StudyAccountBoundary} from '@/features/study/account-boundary';
import {ReviewQueue} from './queue';
import {ReviewHistory} from './history';
import {ReviewUnavailable} from './unavailable';
type PageProps={searchParams:Promise<Record<string,string|string[]|undefined>>};
async function actor(){const cookie=(await headers()).get('cookie')??'',result=await readServerSession(cookie);if(!result.ok)return {ok:false as const,code:result.code};if(!result.data)return {ok:false as const,code:'AUTHENTICATION_REQUIRED'};if(result.data.mustChangePassword)return {ok:false as const,code:'PASSWORD_CHANGE_REQUIRED'};return {ok:true as const,cookie,user:result.data}}
function filters(values:Record<string,string|string[]|undefined>,keys:string[]){const q=new URLSearchParams(),out:Record<string,string>={};for(const [key,value] of Object.entries(values)){if(!keys.includes(key)||Array.isArray(value))return null;if(value){q.set(key,value);out[key]=value}}return {q,out}}
export async function ReviewWorkspacePage({searchParams}:PageProps){
 const a=await actor();if(!a.ok)return <ReviewUnavailable code={a.code}/>;
 const f=filters(await searchParams,['bucket','q','topicKey','limit','offset']);if(!f)return <ReviewUnavailable code="INVALID_REQUEST"/>;if(!f.out.bucket){f.out.bucket='due';f.q.set('bucket','due')}
 // RC private reads use identity locks: finish overview before starting the queue.
 const overview=await readReview<Overview>('/api/v3/study/review/overview',a.cookie);if(!overview.ok)return <ReviewUnavailable code={overview.code}/>;
 const queue=await readReview<QueuePage>('/api/v3/study/review/queue'+(f.q.size?'?'+f.q:''),a.cookie);if(!queue.ok)return <ReviewUnavailable code={queue.code}/>;
 if(overview.data.actorId!==a.user.id||queue.data.actorId!==a.user.id)return <ReviewUnavailable code="AUTHENTICATION_REQUIRED"/>;
 return <StudyAccountBoundary actorId={a.user.id}><ReviewQueue overview={overview.data} page={queue.data} filters={f.out}/></StudyAccountBoundary>;
}
export async function ReviewHistoryPage({searchParams}:PageProps){
 const a=await actor();if(!a.ok)return <ReviewUnavailable code={a.code}/>;const f=filters(await searchParams,['knowledgeId','topicKey','mode','rating','limit','cursor']);if(!f)return <ReviewUnavailable code="INVALID_REQUEST"/>;
 const page=await readReview<HistoryPage>('/api/v3/study/review/history'+(f.q.size?'?'+f.q:''),a.cookie);if(!page.ok)return <ReviewUnavailable code={page.code}/>;if(page.data.actorId!==a.user.id)return <ReviewUnavailable code="AUTHENTICATION_REQUIRED"/>;
 return <StudyAccountBoundary actorId={a.user.id}><ReviewHistory page={page.data} filters={f.out}/></StudyAccountBoundary>;
}
