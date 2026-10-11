import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.corrections.id.plans.planId.version");}
import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {correctionPage,correctionPageError} from '@/features/correction/page-data';
import {CorrectionAccountProvider} from '@/features/correction/correction-account';
import {PlanEditor} from '@/features/correction/plan-editor';
import {CorrectionState} from '@/features/correction/status';
import {correctionServer} from '@/lib/correction/server-client';
import {CorrectionRequestError} from '@/lib/correction/types';
export const dynamic='force-dynamic';

export default async function Page({params}:{params:Promise<{id:string;planId:string;version:string}>}){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;try{return <><UiPageTitle messageKey="page.review.corrections.id.plans.planId.version"/>{await withCorrectionDeadline(undefined,async signal=>{const {id,planId,version}=await correctionAwait(params,signal);if(!/^[1-9][0-9]*$/.test(version))throw new CorrectionRequestError('INVALID_REQUEST');const out=await correctionPage(a=>correctionServer.readPlan({id:planId,version:Number(version)},a),true,signal);if(out.data.plan.caseId!==id)throw new CorrectionRequestError('NOT_FOUND');return <CorrectionAccountProvider actorId={out.actorId} management><PlanEditor key={out.data.plan.ref.id+':'+out.data.plan.ref.version} initial={out.data} topicMode={mode==="topics"}/></CorrectionAccountProvider>})}</>}catch(e){return <><UiPageTitle messageKey="page.review.corrections.id.plans.planId.version"/>{<CorrectionState error={correctionPageError(e)}/>}</>}}
