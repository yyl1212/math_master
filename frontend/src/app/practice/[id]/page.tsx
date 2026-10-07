import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {RetiredModule} from "@/features/study/retired-module";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.practice.id");}
import {notFound,redirect} from "next/navigation";import {learningActor} from "@/features/learning/page-data";import {getLearningClient} from "@/lib/learning/server-client";import {learningUUID} from "@/lib/learning/schemas";import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";import {PracticePanel} from "@/features/practice/practice-panel";
export const dynamic="force-dynamic";
async function LegacyPage({params}:{params:Promise<{id:string}>}){const{id}=await params;if(!learningUUID.test(id))notFound();const actor=await learningActor();if(!actor.ok)return <><UiPageTitle messageKey="page.practice.id"/>{<LearningNotice result={actor.error}/>}</>;const result=await getLearningClient().readPractice(id);if(!result.ok)return <><UiPageTitle messageKey="page.practice.id"/>{<LearningNotice result={result}/>}</>;return <><UiPageTitle messageKey="page.practice.id"/>{<LearningBoundary actorId={actor.id}><PracticePanel key={id} view={result.data}/></LearningBoundary>}</>}

export default async function Page(props:Parameters<typeof LegacyPage>[0]){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;if(mode==="topics")return <RetiredModule archiveHref={"/learning-history?archive=legacy&module=practice&attempt="+(await props.params).id}/>;return <LegacyPage {...props}/>}
