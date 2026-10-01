import { notFound } from "next/navigation";
import { getGoClient } from "@/lib/api/server-client";
import { PathView } from "@/features/reading/path-view";
export const dynamic = "force-dynamic";
export const metadata = { title: "Learning path" };
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const result = await getGoClient().getPath((await params).id);
  if (!result.ok && result.kind === "not-found") notFound();
  return <PathView result={result} />;
}
