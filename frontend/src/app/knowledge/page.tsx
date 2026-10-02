import {learningActor} from "@/features/learning/page-data";
import {getLearningClient} from "@/lib/learning/server-client";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {KnowledgeList} from "@/features/learning/knowledge-list";
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
  const actor=await learningActor();const states=actor.ok?await getLearningClient().listLearningKnowledge({limit:20,offset:0}):null;const personal=actor.ok&&states?<LearningBoundary actorId={actor.id}>{states.ok?<KnowledgeList page={states.data} paginate={false}/>:<LearningNotice result={states}/>}</LearningBoundary>:undefined;
  return <KnowledgeMap result={result} q={query.q} status={query.status} personal={personal} />;
}
