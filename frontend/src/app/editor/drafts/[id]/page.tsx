import {redirectLegacyManagement} from "@/lib/knowledge-admin/mode";
import {readExperienceMode} from "@/lib/study/mode";
import {StudyUnavailable} from "@/features/study/pages";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.editor.drafts.id");}
import { contentPageAccess } from "@/features/content/page-access";
import { ContentState } from "@/features/content/content-state";
import { readServerContent } from "@/lib/content/server-client";
import type { DraftPage, DraftView, SubmissionPage, SubmissionView, PublicationPage, PublicationView } from "@/lib/content/types";
import { contentUUID } from "@/lib/content/schemas";
import { DraftList } from "@/features/content/draft-list";
import {TopicDraftEditor} from "@/features/content/topic-workspace";
import {readServerTopicManagement} from "@/lib/taxonomy/management-server-client";
import type {DraftTopicView} from "@/lib/taxonomy/types";
import { DraftEditor } from "@/features/content/draft-editor";
import { SubmissionList } from "@/features/content/submission-list";
import { ReviewPanel } from "@/features/content/review-panel";
import { PublicationPanel } from "@/features/content/publication-panel";
import { WithdrawalPanel } from "@/features/content/withdrawal-panel";
export const dynamic = "force-dynamic";

export default async function Page({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const mode=await readExperienceMode();if(mode===null)return <StudyUnavailable/>;const { id } = await params; if (!contentUUID.test(id))
    return <><UiPageTitle messageKey="page.editor.drafts.id"/>{<ContentState status={404}/>}</>; await redirectLegacyManagement(); const access = await contentPageAccess(["editor", "admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.editor.drafts.id"/>{access.error}</>; const draft = await readServerContent<DraftView>({ kind: "readDraft", id }, access.cookie); if (!draft.ok)
    return <><UiPageTitle messageKey="page.editor.drafts.id"/>{<ContentState status={draft.status} code={draft.code}/>}</>; const topics=await readServerTopicManagement<DraftTopicView>({kind:"readDraft",id},access.cookie);if(topics.ok)return <><UiPageTitle messageKey="page.editor.drafts.id"/><TopicDraftEditor topicMode={mode==="topics"} initial={draft.data} topics={topics.data} canEdit={access.user.roles.includes("editor")&&draft.data.ownerId===access.user.id}/></>;if(mode==="topics"||topics.code!=="TAXONOMY_NOT_CONFIGURED")return <ContentState status={topics.status}/>; return <><UiPageTitle messageKey="page.editor.drafts.id"/>{<DraftEditor topicMode={false} key={id} initial={draft.data} canEdit={access.user.roles.includes("editor") && draft.data.ownerId === access.user.id}/>}</>; }
