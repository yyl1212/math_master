import {feedbackPage,feedbackListPath,feedbackPageError} from '@/features/feedback/page-data';
import {FeedbackAccountProvider} from '@/features/feedback/feedback-account';
import {TicketList} from '@/features/feedback/ticket-list';
import {FeedbackState} from '@/features/feedback/status';
import type {MetadataPage} from '@/lib/feedback/types';
export const dynamic='force-dynamic';
export const metadata={title:'Feedback review'};
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){try{const route=feedbackListPath(await searchParams,true),result=await feedbackPage<MetadataPage>(route,true);return <FeedbackAccountProvider actorId={result.actorId} review={true}><TicketList key={result.actorId+':'+route} initial={result.data} review={true} route={route}/></FeedbackAccountProvider>}catch(e){return <FeedbackState error={feedbackPageError(e)}/>}}
