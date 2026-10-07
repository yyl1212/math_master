import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.corrections.id");}
import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {correctionPage,correctionPageError} from '@/features/correction/page-data';
import {CorrectionAccountProvider} from '@/features/correction/correction-account';
import {ResultPanel} from '@/features/correction/result-panel';
import {CorrectionState} from '@/features/correction/status';
import {correctionServer} from '@/lib/correction/server-client';
export const dynamic='force-dynamic';

export default async function Page({params}:{params:Promise<{id:string}>}){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;try{return <><UiPageTitle messageKey="page.corrections.id"/>{await withCorrectionDeadline(undefined,async signal=>{const {id}=await correctionAwait(params,signal),out=await correctionPage(a=>correctionServer.readOwn(id,a),false,signal);return <CorrectionAccountProvider actorId={out.actorId}><ResultPanel key={out.data.result.id} initial={out.data} topicMode={mode==="topics"}/></CorrectionAccountProvider>})}</>}catch(e){return <><UiPageTitle messageKey="page.corrections.id"/>{<CorrectionState error={correctionPageError(e)}/>}</>}}
