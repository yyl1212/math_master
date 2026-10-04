import { contentPageAccess } from "@/features/content/page-access";
import { ContentState } from "@/features/content/content-state";
import { DraftReadingView } from "@/features/content/draft-reading-view";
import { readServerContent } from "@/lib/content/server-client";
import { contentUUID } from "@/lib/content/schemas";
import type { DraftView } from "@/lib/content/types";

export const dynamic = "force-dynamic";
export const metadata = { title: "Read knowledge draft" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;
    if (!contentUUID.test(id)) return <ContentState status={404}/>;
    const access = await contentPageAccess(["editor", "admin"]);
    if ("error" in access) return access.error;
    const draft = await readServerContent<DraftView>({ kind: "readDraft", id }, access.cookie);
    if (!draft.ok) return <ContentState status={draft.status} code={draft.code}/>;
    return <DraftReadingView key={id + ":" + draft.data.revision} initial={draft.data}/>;
}
