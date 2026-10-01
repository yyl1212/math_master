import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { DomainView } from "@/features/catalogue/domain-view";
export const dynamic = "force-dynamic";
export const metadata = { title: "Learning domain" };
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getDomain((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  return <DomainView result={result} />;
}
