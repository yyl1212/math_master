import {feedbackPage,feedbackTicketPath,feedbackPageError} from '@/features/feedback/page-data';
import {FeedbackAccountProvider} from '@/features/feedback/feedback-account';
import {FeedbackTicket} from '@/features/feedback/discussion-panel';
import {FeedbackState} from '@/features/feedback/status';
import type {Metadata} from '@/lib/feedback/types';
export const dynamic='force-dynamic';
export const metadata={title:'Report details'};
export default async function Page({params}:{params:Promise<{id:string}>}){try{const {id}=await params,result=await feedbackPage<Metadata>(feedbackTicketPath(id,true),true);return <FeedbackAccountProvider actorId={result.actorId} review={true}><FeedbackTicket key={result.data.id} initial={result.data} review={true}/></FeedbackAccountProvider>}catch(e){return <FeedbackState error={feedbackPageError(e)}/>}}
