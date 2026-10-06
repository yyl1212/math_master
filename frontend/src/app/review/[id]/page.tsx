import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.id");}
import { contentPageAccess } from "@/features/content/page-access";
import { ContentState } from "@/features/content/content-state";
import { readServerContent } from "@/lib/content/server-client";
import type { DraftPage, DraftView, SubmissionPage, SubmissionView, PublicationPage, PublicationView } from "@/lib/content/types";
import { contentUUID } from "@/lib/content/schemas";
import { DraftList } from "@/features/content/draft-list";
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
}) { const { id } = await params; if (!contentUUID.test(id))
    return <><UiPageTitle messageKey="page.review.id"/>{<ContentState status={404}/>}</>; const access = await contentPageAccess(["reviewer", "editor", "admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.review.id"/>{access.error}</>; const result = await readServerContent<SubmissionView>({ kind: "readSubmission", id }, access.cookie); if (!result.ok)
    return <><UiPageTitle messageKey="page.review.id"/>{<ContentState status={result.status} code={result.code}/>}</>; return <><UiPageTitle messageKey="page.review.id"/>{<ReviewPanel key={id} submission={result.data} user={access.user}/>}</>; }
