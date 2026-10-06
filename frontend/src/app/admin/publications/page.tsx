import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.admin.publications");}
import { contentPageAccess } from "@/features/content/page-access";
import { ContentState } from "@/features/content/content-state";
import { readServerContent } from "@/lib/content/server-client";
import type { DraftPage, DraftView, SubmissionPage, SubmissionView, PublicationPage, PublicationView } from "@/lib/content/types";
import { contentUUID } from "@/lib/content/schemas";
import { DraftList } from "@/features/content/draft-list";
import { DraftEditor } from "@/features/content/draft-editor";
import { SubmissionList } from "@/features/content/submission-list";
import { ReviewPanel } from "@/features/content/review-panel";
import {TopicPublicationWorkspace} from "@/features/content/topic-workspace";
import {readServerTopicManagement} from "@/lib/taxonomy/management-server-client";
import type {ReleasePage} from "@/lib/taxonomy/types";
import { PublicationPanel } from "@/features/content/publication-panel";
import { WithdrawalPanel } from "@/features/content/withdrawal-panel";
export const dynamic = "force-dynamic";

export default async function Page() { const access = await contentPageAccess(["admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.admin.publications"/>{access.error}</>; const topics=await readServerTopicManagement<ReleasePage>({kind:"listReleases",query:{limit:20}},access.cookie);if(topics.ok)return <><UiPageTitle messageKey="page.admin.publications"/><TopicPublicationWorkspace initial={topics.data}/></>;if(topics.code!=="TAXONOMY_NOT_CONFIGURED")return <ContentState status={topics.status}/>; const result = await readServerContent<PublicationPage>({ kind: "listPublications", query: { limit: 100 } }, access.cookie); if (!result.ok)
    return <><UiPageTitle messageKey="page.admin.publications"/>{<ContentState status={result.status} code={result.code}/>}</>; return <><UiPageTitle messageKey="page.admin.publications"/>{<PublicationPanel initial={result.data}/>}</>; }
