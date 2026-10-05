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
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const query=learningPageQuery(await searchParams,"history");if(!query)return <><UiPageTitle messageKey="page.learning-history"/>{<section className="content-state" role="alert"><h1>Invalid history parameters.</h1><Link prefetch={false} href="/learning-history">Reset history view</Link></section>}</>;const actor=await learningActor();if(!actor.ok)return <><UiPageTitle messageKey="page.learning-history"/>{<LearningNotice result={actor.error}/>}</>;const client=getLearningClient();const detail=query.knowledge&&query.version?await client.readLearningKnowledge(query.knowledge,query.version):null;if(detail&&!detail.ok)return <><UiPageTitle messageKey="page.learning-history"/>{<LearningNotice result={detail}/>}</>;const history=await client.listLearningHistory({limit:20,offset:query.offset});if(!history.ok)return <><UiPageTitle messageKey="page.learning-history"/>{<LearningNotice result={history}/>}</>;return <><UiPageTitle messageKey="page.learning-history"/>{<><header className="page-heading"><h1>Learning history</h1><p>Your saved versions and assessment records.</p></header><LearningBoundary actorId={actor.id}>{detail?.ok&&<KnowledgeControls detail={detail.data}/>}<HistoryList page={history.data}/></LearningBoundary></>}</>}
