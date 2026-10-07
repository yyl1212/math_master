import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {TopicCorrections} from "@/features/study/topic-area";
import {contentPageAccess} from "@/features/content/page-access";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.corrections");}
import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {correctionPage,correctionPageQuery,correctionPageError} from '@/features/correction/page-data';
import {CorrectionAccountProvider} from '@/features/correction/correction-account';
import {CaseList} from '@/features/correction/case-list';
import {CorrectionState} from '@/features/correction/status';
import {correctionServer} from '@/lib/correction/server-client';
export const dynamic='force-dynamic';

export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;const params=await searchParams;if(mode==="topics"&&params.archive!=="legacy"){const access=await contentPageAccess(["editor","reviewer","admin"]);if("error"in access)return access.error;return <TopicCorrections roles={access.user.roles}/>};const {archive,...queryParams}=params;if(archive!==undefined&&archive!=="legacy")return <StudyUnavailable/>;try{return <><UiPageTitle messageKey="page.review.corrections"/>{await withCorrectionDeadline(undefined,async signal=>{const query=correctionPageQuery(queryParams),out=await correctionPage(a=>correctionServer.listCases(query,a),true,signal);return <CorrectionAccountProvider actorId={out.actorId} management><CaseList initial={out.data} topicMode={mode==="topics"}/></CorrectionAccountProvider>})}</>}catch(e){return <><UiPageTitle messageKey="page.review.corrections"/>{<CorrectionState error={correctionPageError(e)}/>}</>}}
