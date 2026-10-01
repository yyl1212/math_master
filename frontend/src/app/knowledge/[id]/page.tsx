import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { KnowledgeView } from "@/features/reading/knowledge-view";
export const dynamic = "force-dynamic";
export const metadata = { title: "Knowledge" };
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getKnowledge((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  return <KnowledgeView result={result} />;
}
