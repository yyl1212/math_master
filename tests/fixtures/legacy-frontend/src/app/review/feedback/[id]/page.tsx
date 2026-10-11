import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.feedback.id");}
import {feedbackPage,feedbackTicketPath,feedbackPageError} from '@/features/feedback/page-data';
import {FeedbackAccountProvider} from '@/features/feedback/feedback-account';
import {FeedbackTicket} from '@/features/feedback/discussion-panel';
import {FeedbackState} from '@/features/feedback/status';
import type {Metadata} from '@/lib/feedback/types';
export const dynamic='force-dynamic';

export default async function Page({params}:{params:Promise<{id:string}>}){try{const {id}=await params,result=await feedbackPage<Metadata>(feedbackTicketPath(id,true),true);return <><UiPageTitle messageKey="page.review.feedback.id"/>{<FeedbackAccountProvider actorId={result.actorId} review={true}><FeedbackTicket key={result.data.id} initial={result.data} review={true}/></FeedbackAccountProvider>}</>}catch(e){return <><UiPageTitle messageKey="page.review.feedback.id"/>{<FeedbackState error={feedbackPageError(e)}/>}</>}}
