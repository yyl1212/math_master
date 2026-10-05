import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.feedback.new");}
import {feedbackPage,feedbackSourcePath,feedbackPageError} from '@/features/feedback/page-data';
import {FeedbackAccountProvider} from '@/features/feedback/feedback-account';
import {NewFeedbackForm} from '@/features/feedback/new-form';
import {FeedbackState} from '@/features/feedback/status';
import type {Context} from '@/lib/feedback/types';
export const dynamic='force-dynamic';

export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){try{const sourcePath=feedbackSourcePath(await searchParams),result=await feedbackPage<Context>(sourcePath);return <><UiPageTitle messageKey="page.feedback.new"/>{<FeedbackAccountProvider actorId={result.actorId}><NewFeedbackForm context={result.data} sourcePath={sourcePath}/></FeedbackAccountProvider>}</>}catch(e){return <><UiPageTitle messageKey="page.feedback.new"/>{<FeedbackState error={feedbackPageError(e)}/>}</>}}
