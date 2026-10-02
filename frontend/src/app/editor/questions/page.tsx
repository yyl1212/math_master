import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { DraftList } from "@/features/question/draft-list";
import type { DraftPage } from "@/lib/question/types";
export const metadata = { title: "Question workspaces" };
export default async function Page() { const access = await questionPageAccess(["editor", "admin"]); if ("error" in access)
    return access.error; const canEdit = access.user.roles.includes("editor"), r = await readServerQuestion<DraftPage>({ kind: "listDrafts", query: { scope: canEdit ? "mine" : "all", limit: 20 } }, access.cookie); if (!r.ok)
    return <QuestionState status={r.status} code={r.code}/>; return <DraftList initial={r.data} canEdit={canEdit}/>; }
