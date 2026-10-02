import {feedbackPage,feedbackListPath,feedbackPageError} from '@/features/feedback/page-data';
import {FeedbackAccountProvider} from '@/features/feedback/feedback-account';
import {TicketList} from '@/features/feedback/ticket-list';
import {FeedbackState} from '@/features/feedback/status';
import type {MetadataPage} from '@/lib/feedback/types';
export const dynamic='force-dynamic';
export const metadata={title:'My reports'};
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){try{const route=feedbackListPath(await searchParams,false),result=await feedbackPage<MetadataPage>(route,false);return <FeedbackAccountProvider actorId={result.actorId} review={false}><TicketList initial={result.data} review={false} route={route}/></FeedbackAccountProvider>}catch(e){return <FeedbackState error={feedbackPageError(e)}/>}}
