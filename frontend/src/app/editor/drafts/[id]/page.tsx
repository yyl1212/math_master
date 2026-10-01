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
export const metadata = { title: "Edit mathematical content" };
export default async function Page({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const { id } = await params; if (!contentUUID.test(id))
    return <ContentState status={404}/>; const access = await contentPageAccess(["editor", "admin"]); if ("error" in access)
    return access.error; const draft = await readServerContent<DraftView>({ kind: "readDraft", id }, access.cookie); if (!draft.ok)
    return <ContentState status={draft.status} code={draft.code}/>; return <DraftEditor key={id} initial={draft.data} canEdit={access.user.roles.includes("editor") && draft.data.ownerId === access.user.id}/>; }
