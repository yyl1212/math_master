import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {RetiredModule} from "@/features/study/retired-module";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.questions.id");}
import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { ReviewPanel } from "@/features/question/review-panel";
import type { SubmissionView, InstancePage } from "@/lib/question/types";
import { questionUUID } from "@/lib/question/schemas";

async function LegacyPage({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const access = await questionPageAccess(["editor", "reviewer", "admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.review.questions.id"/>{access.error}</>; const { id } = await params; if (!questionUUID.test(id))
    return <><UiPageTitle messageKey="page.review.questions.id"/>{<QuestionState status={404}/>}</>; const r = await readServerQuestion<SubmissionView>({ kind: "readSubmission", id }, access.cookie); if (!r.ok)
    return <><UiPageTitle messageKey="page.review.questions.id"/>{<QuestionState status={r.status} code={r.code}/>}</>; const instances = await readServerQuestion<InstancePage>({ kind: "listInstances", id, query: { limit: 20 } }, access.cookie); if (!instances.ok)
    return <><UiPageTitle messageKey="page.review.questions.id"/>{<QuestionState status={instances.status} code={instances.code}/>}</>; const bound = new Map(r.data.frozen.instanceIdentities.map(i => [i.id + ":" + i.version, i.sha256])); if (instances.data.total !== bound.size || instances.data.items.some(i => bound.get(i.identity.id + ":" + i.identity.version) !== i.identity.sha256))
    return <><UiPageTitle messageKey="page.review.questions.id"/>{<QuestionState status={503}/>}</>; return <><UiPageTitle messageKey="page.review.questions.id"/>{<ReviewPanel submission={r.data} initialInstances={instances.data} user={access.user}/>}</>; }

export default async function Page(props:Parameters<typeof LegacyPage>[0]){const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;if(mode==="topics")return <RetiredModule/>;return <LegacyPage {...props}/>}
