import {feedbackPage,feedbackSourcePath,feedbackPageError} from '@/features/feedback/page-data';
import {FeedbackAccountProvider} from '@/features/feedback/feedback-account';
import {NewFeedbackForm} from '@/features/feedback/new-form';
import {FeedbackState} from '@/features/feedback/status';
import type {Context} from '@/lib/feedback/types';
export const dynamic='force-dynamic';
export const metadata={title:'Report a problem'};
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){try{const result=await feedbackPage<Context>(feedbackSourcePath(await searchParams));return <FeedbackAccountProvider actorId={result.actorId}><NewFeedbackForm context={result.data}/></FeedbackAccountProvider>}catch(e){return <FeedbackState error={feedbackPageError(e)}/>}}
