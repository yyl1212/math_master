import {learningActor} from "@/features/learning/page-data";
import {getLearningClient} from "@/lib/learning/server-client";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {JoinRoute} from "@/features/learning/path-progress";
import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { PathView } from "@/features/reading/path-view";
export const dynamic = "force-dynamic";
export const metadata = { title: "Learning path" };
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getPath((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  const actor=await learningActor();const overview=result.ok&&actor.ok?await getLearningClient().readLearningOverview():null;const current=overview?.ok&&result.ok?overview.data.availablePaths.find(p=>p.path.id===result.data.path.id&&p.path.version===result.data.path.version):null;const personal=actor.ok&&overview?<LearningBoundary actorId={actor.id}>{!overview.ok?<LearningNotice result={overview}/>:current&&overview.data.knowledgeHead?<JoinRoute path={current} knowledgeHead={overview.data.knowledgeHead}/>:null}</LearningBoundary>:undefined;
  return <PathView result={result} personal={personal} />;
}
