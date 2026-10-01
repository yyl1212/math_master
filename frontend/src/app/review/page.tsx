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
export const metadata = { title: "Independent review" };
export default async function Page() { const access = await contentPageAccess(["reviewer", "editor", "admin"]); if ("error" in access)
    return access.error; const scope = access.user.roles.includes("reviewer") ? "review" : access.user.roles.includes("admin") ? "all" : "mine"; const result = await readServerContent<SubmissionPage>({ kind: "listSubmissions", query: { scope, status: "pending", limit: 20 } }, access.cookie); if (!result.ok)
    return <ContentState status={result.status} code={result.code}/>; return <SubmissionList initial={result.data} scope={scope}/>; }
