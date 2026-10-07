import {readExperienceMode} from "@/lib/study/mode";
import {LegacyStudyArchive} from "@/features/study/legacy-archive";
import {StudyUnavailable} from "@/features/study/pages";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.assessments.id.result");}
import {notFound} from "next/navigation";import {learningActor} from "@/features/learning/page-data";import {getLearningClient} from "@/lib/learning/server-client";import {learningUUID} from "@/lib/learning/schemas";import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";import {ResultPanel} from "@/features/assessment/result-panel";export const dynamic="force-dynamic";async function LegacyPage({params}:{params:Promise<{id:string}>}){const{id}=await params;if(!learningUUID.test(id))notFound();const actor=await learningActor();if(!actor.ok)return <><UiPageTitle messageKey="page.assessments.id.result"/>{<LearningNotice result={actor.error}/>}</>;const result=await getLearningClient().readAssessmentResult(id);if(!result.ok)return <><UiPageTitle messageKey="page.assessments.id.result"/>{<LearningNotice result={result}/>}</>;return <><UiPageTitle messageKey="page.assessments.id.result"/>{<LearningBoundary actorId={actor.id}><ResultPanel result={result.data}/></LearningBoundary>}</>}

export default async function Page(props:Parameters<typeof LegacyPage>[0]){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;if(mode==="topics")return <LegacyStudyArchive searchParams={Promise.resolve({archive:"legacy",module:"assessment",attempt:(await props.params).id})}/>;return <LegacyPage {...props}/>}
