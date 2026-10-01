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
export const metadata = { title: "Withdraw fixed content" };
export default async function Page() { const access = await contentPageAccess(["admin"]); if ("error" in access)
    return access.error; const result = await readServerContent<PublicationPage>({ kind: "listPublications", query: { limit: 100 } }, access.cookie); if (!result.ok)
    return <ContentState status={result.status} code={result.code}/>; if (result.data.head && !result.data.items.some(p => p.id === result.data.head)) {
    const head = await readServerContent<PublicationView>({ kind: "readPublication", id: result.data.head }, access.cookie);
    if (!head.ok)
        return <ContentState status={head.status} code={head.code}/>;
    result.data.items.unshift(head.data);
} return <WithdrawalPanel initial={result.data}/>; }
