import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.corrections.id");}
import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {correctionPage,correctionPageError} from '@/features/correction/page-data';
import {CorrectionAccountProvider} from '@/features/correction/correction-account';
import {CasePanel} from '@/features/correction/case-panel';
import {CorrectionState} from '@/features/correction/status';
import {correctionServer} from '@/lib/correction/server-client';
import {CorrectionRequestError} from '@/lib/correction/types';
export const dynamic='force-dynamic';

export default async function Page({params}:{params:Promise<{id:string}>}){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;try{return <><UiPageTitle messageKey="page.review.corrections.id"/>{await withCorrectionDeadline(undefined,async signal=>{const {id}=await correctionAwait(params,signal),out=await correctionPage(async a=>{const [c,p,j]=await Promise.all([correctionServer.readCase(id,a),correctionServer.listPlans(id,{},a),correctionServer.listJobs(id,{},a)]);if(c.actorId!==p.actorId||c.actorId!==j.actorId)throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');return {actorId:c.actorId,data:{case:c.data,plans:p.data,jobs:j.data}}},true,signal);return <CorrectionAccountProvider actorId={out.actorId} management><CasePanel key={out.data.case.id} initial={out.data} topicMode={mode==="topics"}/></CorrectionAccountProvider>})}</>}catch(e){return <><UiPageTitle messageKey="page.review.corrections.id"/>{<CorrectionState error={correctionPageError(e)}/>}</>}}
