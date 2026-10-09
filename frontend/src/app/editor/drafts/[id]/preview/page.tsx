import {redirectLegacyManagement} from "@/lib/knowledge-admin/mode";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.editor.drafts.id.preview");}
import { contentPageAccess } from "@/features/content/page-access";
import { ContentState } from "@/features/content/content-state";
import { DraftReadingView } from "@/features/content/draft-reading-view";
import { readServerContent } from "@/lib/content/server-client";
import { contentUUID } from "@/lib/content/schemas";
import type { DraftView } from "@/lib/content/types";

export const dynamic = "force-dynamic";


export default async function Page({ params }: { params: Promise<{ id: string }> }) {
    const { id } = await params;
    if (!contentUUID.test(id)) return <><UiPageTitle messageKey="page.editor.drafts.id.preview"/>{<ContentState status={404}/>}</>;
    await redirectLegacyManagement(); const access = await contentPageAccess(["editor", "admin"]);
    if ("error" in access) return <><UiPageTitle messageKey="page.editor.drafts.id.preview"/>{access.error}</>;
    const draft = await readServerContent<DraftView>({ kind: "readDraft", id }, access.cookie);
    if (!draft.ok) return <><UiPageTitle messageKey="page.editor.drafts.id.preview"/>{<ContentState status={draft.status} code={draft.code}/>}</>;
    return <><UiPageTitle messageKey="page.editor.drafts.id.preview"/>{<DraftReadingView key={id + ":" + draft.data.revision} initial={draft.data}/>}</>;
}
