import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.domains.id");}
import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { DomainView } from "@/features/catalogue/domain-view";
export const dynamic = "force-dynamic";

export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getDomain((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  return <><UiPageTitle messageKey="page.domains.id"/>{<DomainView result={result} />}</>;
}
