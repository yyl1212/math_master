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
import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { KnowledgeView } from "@/features/reading/knowledge-view";
export const dynamic = "force-dynamic";

export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getKnowledge((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  const actor=await learningActor();const detail=result.ok&&actor.ok?await getLearningClient().readLearningKnowledge(result.data.knowledge.id,result.data.knowledge.version):null;let personal:React.ReactNode=actor.ok&&detail?<LearningBoundary actorId={actor.id}>{detail.ok?<KnowledgeControls detail={detail.data}/>:<LearningNotice result={detail}/>}</LearningBoundary>:undefined;
  if(result.ok&&actor.ok){const cookie=(await headers()).get("cookie")??"";const overview=await readServerStudy<Overview>({kind:"overview"},cookie);if(overview.ok&&overview.data.mode==="topics"){const study=await readServerStudy<StudyDetail>({kind:"readKnowledge",id:result.data.knowledge.id},cookie);const note=await readServerStudy<NoteView>({kind:"readNote",id:result.data.knowledge.id},cookie);personal=study.ok?<StudyAccountBoundary actorId={actor.id}><KnowledgeStudyControls detail={study.data} readingKnowledge={{id:result.data.knowledge.id,version:result.data.knowledge.version}} initialNote={note.ok?note.data:undefined}/></StudyAccountBoundary>:undefined;}}

  return <><UiPageTitle messageKey="page.knowledge.id"/>{<><KnowledgeView result={result} personal={personal} />{result.ok&&<section className="panel"><h2><UiText notice={uiMessage("page.lesson.feedback.14669c",{})}/></h2><ReportLink source={{kind:'knowledge',id:result.data.knowledge.id}}/>{result.data.units.map(unit=><p key={unit.id}><UiText notice={uiMessage("page.unit.c57306",{})}/>{unit.id} <ReportLink source={{kind:'knowledge',id:result.data.knowledge.id,partKind:'unit',partId:unit.id}}/></p>)}{result.data.assets.map(asset=><p key={asset.id}><UiText notice={uiMessage("page.illustration.0ffea7",{})}/>{asset.id} <ReportLink source={{kind:'knowledge',id:result.data.knowledge.id,partKind:'asset',partId:asset.id}}/></p>)}</section>}</>}</>;
}
