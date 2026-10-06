import {readExperienceMode} from "@/lib/study/mode";
import {StudyHistoryPage,StudyUnavailable} from "@/features/study/pages";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.learning-history");}
import Link from "next/link";
import {getLearningClient} from "@/lib/learning/server-client";
import {learningActor,learningPageQuery} from "@/features/learning/page-data";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {KnowledgeControls} from "@/features/learning/knowledge-controls";
import {HistoryList} from "@/features/learning/history-list";
export const dynamic="force-dynamic";
async function LegacyHistoryPage({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const query=learningPageQuery(await searchParams,"history");if(!query)return <><UiPageTitle messageKey="page.learning-history"/>{<section className="content-state" role="alert"><h1><UiText notice={uiMessage("page.invalid.history.parameters.7f8f9a",{})}/></h1><Link prefetch={false} href="/learning-history"><UiText notice={uiMessage("page.reset.history.view.bc760c",{})}/></Link></section>}</>;const actor=await learningActor();if(!actor.ok)return <><UiPageTitle messageKey="page.learning-history"/>{<LearningNotice result={actor.error}/>}</>;const client=getLearningClient();const detail=query.knowledge&&query.version?await client.readLearningKnowledge(query.knowledge,query.version):null;if(detail&&!detail.ok)return <><UiPageTitle messageKey="page.learning-history"/>{<LearningNotice result={detail}/>}</>;const history=await client.listLearningHistory({limit:20,offset:query.offset});if(!history.ok)return <><UiPageTitle messageKey="page.learning-history"/>{<LearningNotice result={history}/>}</>;return <><UiPageTitle messageKey="page.learning-history"/>{<><header className="page-heading"><h1><UiText notice={uiMessage("page.learning-history",{})}/></h1><p><UiText notice={uiMessage("page.your.saved.versions.and.assessment.records.85765b",{})}/></p></header><LearningBoundary actorId={actor.id}>{detail?.ok&&<KnowledgeControls detail={detail.data}/>}<HistoryList page={history.data}/></LearningBoundary></>}</>}

export default async function Page(props:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;if(mode==="topics")return <StudyHistoryPage {...props}/>;return <LegacyHistoryPage {...props}/>;}
