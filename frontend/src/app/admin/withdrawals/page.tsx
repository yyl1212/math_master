import {redirectLegacyManagement} from "@/lib/knowledge-admin/mode";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.admin.withdrawals");}
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

export default async function Page() { await redirectLegacyManagement(); const access = await contentPageAccess(["admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.admin.withdrawals"/>{access.error}</>; const result = await readServerContent<PublicationPage>({ kind: "listPublications", query: { limit: 100 } }, access.cookie); if (!result.ok)
    return <><UiPageTitle messageKey="page.admin.withdrawals"/>{<ContentState status={result.status} code={result.code}/>}</>; if (result.data.head && !result.data.items.some(p => p.id === result.data.head)) {
    const head = await readServerContent<PublicationView>({ kind: "readPublication", id: result.data.head }, access.cookie);
    if (!head.ok)
        return <><UiPageTitle messageKey="page.admin.withdrawals"/>{<ContentState status={head.status} code={head.code}/>}</>;
    result.data.items.unshift(head.data);
} return <><UiPageTitle messageKey="page.admin.withdrawals"/>{<WithdrawalPanel initial={result.data}/>}</>; }
