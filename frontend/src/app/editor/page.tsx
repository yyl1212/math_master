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
export const metadata = { title: "Content workspaces" };
export default async function Page() { const access = await contentPageAccess(["editor", "admin"]); if ("error" in access)
    return access.error; const canEdit = access.user.roles.includes("editor"), drafts = await readServerContent<DraftPage>({ kind: "listDrafts", query: { scope: canEdit ? "mine" : "all", limit: 20 } }, access.cookie); if (!drafts.ok)
    return <ContentState status={drafts.status} code={drafts.code}/>; const submissions = await readServerContent<SubmissionPage>({ kind: "listSubmissions", query: { scope: "mine", status: "pending", limit: 20 } }, access.cookie); return <><DraftList initial={drafts.data} canEdit={canEdit}/>{submissions.ok ? <SubmissionList initial={submissions.data} scope="mine"/> : <ContentState status={submissions.status} code={submissions.code}/>}</>; }
