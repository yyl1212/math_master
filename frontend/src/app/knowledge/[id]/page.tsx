import {ReportLink} from '@/features/feedback/report-link';
import {learningActor} from "@/features/learning/page-data";
import {getLearningClient} from "@/lib/learning/server-client";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {KnowledgeControls} from "@/features/learning/knowledge-controls";
import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { KnowledgeView } from "@/features/reading/knowledge-view";
export const dynamic = "force-dynamic";
export const metadata = { title: "Knowledge" };
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getKnowledge((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  const actor=await learningActor();const detail=result.ok&&actor.ok?await getLearningClient().readLearningKnowledge(result.data.knowledge.id,result.data.knowledge.version):null;const personal=actor.ok&&detail?<LearningBoundary actorId={actor.id}>{detail.ok?<KnowledgeControls detail={detail.data}/>:<LearningNotice result={detail}/>}</LearningBoundary>:undefined;
  return <><KnowledgeView result={result} personal={personal} />{result.ok&&<section className="panel"><h2>Lesson feedback</h2><ReportLink source={{kind:'knowledge',id:result.data.knowledge.id}}/>{result.data.units.map(unit=><p key={unit.id}>Unit {unit.id} <ReportLink source={{kind:'knowledge',id:result.data.knowledge.id,partKind:'unit',partId:unit.id}}/></p>)}{result.data.assets.map(asset=><p key={asset.id}>Illustration {asset.id} <ReportLink source={{kind:'knowledge',id:result.data.knowledge.id,partKind:'asset',partId:asset.id}}/></p>)}</section>}</>;
}
