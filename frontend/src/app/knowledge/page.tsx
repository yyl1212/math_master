import Link from "next/link";
import { parseCatalogueQuery } from "@/features/catalogue/query";
import { KnowledgeMap } from "@/features/catalogue/knowledge-map";
import { getGoClient } from "@/lib/api/server-client";
export const dynamic = "force-dynamic";
export const metadata = { title: "Knowledge Map" };
export default async function Page({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const query = parseCatalogueQuery(await searchParams);
  if (!query.ok)
    return (
      <section className="content-state" role="alert">
        <h1>Invalid search parameters.</h1>
        <p>Please shorten your search or reset the filters.</p>
        <Link prefetch={false} href="/knowledge" className="button secondary">
          Reset search
        </Link>
      </section>
    );
  const result = await getGoClient().listDomains({
    q: query.q,
    limit: 100,
    offset: 0,
  });
  return <KnowledgeMap result={result} q={query.q} status={query.status} />;
}
