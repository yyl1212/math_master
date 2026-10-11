import {getContentMode} from "@/lib/knowledge-admin/mode";
import {readManaged} from "@/lib/knowledge-admin/server-client";
import type {PublicKnowledge} from "@/lib/knowledge-admin/types";
import {CurrentKnowledgeView} from "@/features/reading/current-knowledge-view";
import {ManagedPersonal} from "@/features/study/managed-pages";
import {ContentState as CurrentState} from "@/components/content-state";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.knowledge.id");}
import {ReportLink} from '@/features/feedback/report-link';
import {learningActor} from "@/features/learning/page-data";
import {getLearningClient} from "@/lib/learning/server-client";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {KnowledgeControls} from "@/features/learning/knowledge-controls";
import {StudyAccountBoundary} from "@/features/study/account-boundary";
import {KnowledgeStudyControls} from "@/features/study/knowledge-controls";
import {readServerStudy} from "@/lib/study/server-client";
import type {Overview,StudyDetail,NoteView} from "@/lib/study/types";
import {headers} from "next/headers";
import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { KnowledgeView } from "@/features/reading/knowledge-view";
export const dynamic = "force-dynamic";

export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const content=await getContentMode();if(content===null)return <CurrentState kind="unavailable"/>;
  if(content.mode==="managed"){const{id}=await params;const current=await readManaged<PublicKnowledge>("/api/v3/knowledge/"+id);if(!current.ok){if(current.status===404)return <><CurrentState kind="not-found"/><ManagedPersonal id={id}/></>;return <CurrentState kind="unavailable"/>};return <CurrentKnowledgeView knowledge={current.data} personal={<ManagedPersonal id={id}/>}/>}
  const result = await getGoClient().getKnowledge((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  const mode=await readExperienceMode();
  const actor=await learningActor();let personal:React.ReactNode;
  if(mode===null){personal=<StudyUnavailable/>}else if(mode==="topics"&&result.ok&&actor.ok){const cookie=(await headers()).get("cookie")??"";const[study,note]=await Promise.all([readServerStudy<StudyDetail>({kind:"readKnowledge",id:result.data.knowledge.id},cookie),readServerStudy<NoteView>({kind:"readNote",id:result.data.knowledge.id},cookie)]);personal=study.ok?<StudyAccountBoundary actorId={actor.id}><KnowledgeStudyControls detail={study.data} readingKnowledge={{id:result.data.knowledge.id,version:result.data.knowledge.version}} initialNote={note.ok?note.data:undefined}/></StudyAccountBoundary>:<StudyUnavailable/>;}else if(mode==="legacy"){const detail=result.ok&&actor.ok?await getLearningClient().readLearningKnowledge(result.data.knowledge.id,result.data.knowledge.version):null;personal=actor.ok&&detail?<LearningBoundary actorId={actor.id}>{detail.ok?<KnowledgeControls detail={detail.data}/>:<LearningNotice result={detail}/>}</LearningBoundary>:undefined;}
  return <><UiPageTitle messageKey="page.knowledge.id"/>{<><KnowledgeView result={result} personal={personal} />{result.ok&&<section className="panel"><h2><UiText notice={uiMessage("page.lesson.feedback.14669c",{})}/></h2><ReportLink source={{kind:'knowledge',id:result.data.knowledge.id}}/>{result.data.units.map(unit=><p key={unit.id}><UiText notice={uiMessage("page.unit.c57306",{})}/>{unit.id} <ReportLink source={{kind:'knowledge',id:result.data.knowledge.id,partKind:'unit',partId:unit.id}}/></p>)}{result.data.assets.map(asset=><p key={asset.id}><UiText notice={uiMessage("page.illustration.0ffea7",{})}/>{asset.id} <ReportLink source={{kind:'knowledge',id:result.data.knowledge.id,partKind:'asset',partId:asset.id}}/></p>)}</section>}</>}</>;
}
