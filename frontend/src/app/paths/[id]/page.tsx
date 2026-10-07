import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {RetiredModule} from "@/features/study/retired-module";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.paths.id");}
import {ReportLink} from '@/features/feedback/report-link';
import {learningActor} from "@/features/learning/page-data";
import {getLearningClient} from "@/lib/learning/server-client";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {JoinRoute} from "@/features/learning/path-progress";
import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { PathView } from "@/features/reading/path-view";
export const dynamic = "force-dynamic";

async function LegacyPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getPath((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  const actor=await learningActor();const overview=result.ok&&actor.ok?await getLearningClient().readLearningOverview():null;const current=overview?.ok&&result.ok?overview.data.availablePaths.find(p=>p.path.id===result.data.path.id&&p.path.version===result.data.path.version):null;const personal=actor.ok&&overview?<LearningBoundary actorId={actor.id}>{!overview.ok?<LearningNotice result={overview}/>:current&&overview.data.knowledgeHead?<JoinRoute path={current} knowledgeHead={overview.data.knowledgeHead}/>:null}</LearningBoundary>:undefined;
  return <><UiPageTitle messageKey="page.paths.id"/>{<><PathView result={result} personal={personal}/>{result.ok&&<section className="panel"><h2><UiText notice={uiMessage("page.route.feedback.729fbd",{})}/></h2><ReportLink source={{kind:'path',id:result.data.path.id}}/></section>}</>}</>;
}

export default async function Page(props:Parameters<typeof LegacyPage>[0]){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;if(mode==="topics")return <RetiredModule/>;return <LegacyPage {...props}/>}
