import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.feedback");}
import {feedbackPage,feedbackListPath,feedbackPageError} from '@/features/feedback/page-data';
import {FeedbackAccountProvider} from '@/features/feedback/feedback-account';
import {TicketList} from '@/features/feedback/ticket-list';
import {FeedbackState} from '@/features/feedback/status';
import type {MetadataPage} from '@/lib/feedback/types';
export const dynamic='force-dynamic';

export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){try{const route=feedbackListPath(await searchParams,true),result=await feedbackPage<MetadataPage>(route,true);return <><UiPageTitle messageKey="page.review.feedback"/>{<FeedbackAccountProvider actorId={result.actorId} review={true}><TicketList key={result.actorId+':'+route} initial={result.data} review={true} route={route}/></FeedbackAccountProvider>}</>}catch(e){return <><UiPageTitle messageKey="page.review.feedback"/>{<FeedbackState error={feedbackPageError(e)}/>}</>}}
