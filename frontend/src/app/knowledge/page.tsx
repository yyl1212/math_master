import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.knowledge");}
import {learningActor} from "@/features/learning/page-data";
import {getLearningClient} from "@/lib/learning/server-client";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {KnowledgeList} from "@/features/learning/knowledge-list";
import Link from "next/link";
import { parseCatalogueQuery } from "@/features/catalogue/query";
import { KnowledgeMap } from "@/features/catalogue/knowledge-map";
import { getGoClient } from "@/lib/api/server-client";
export const dynamic = "force-dynamic";

export default async function Page({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const query = parseCatalogueQuery(await searchParams);
  if (!query.ok)
    return <><UiPageTitle messageKey="page.knowledge"/>{(
      <section className="content-state" role="alert">
        <h1><UiText notice={uiMessage("page.invalid.search.parameters.29e585",{})}/></h1>
        <p><UiText notice={uiMessage("page.please.shorten.your.search.or.reset.the.filters.bd6ccd",{})}/></p>
        <Link prefetch={false} href="/knowledge" className="button secondary"><UiText notice={uiMessage("page.reset.search.92df98",{})}/></Link>
      </section>
    )}</>;
  const result = await getGoClient().listDomains({
    q: query.q,
    limit: 100,
    offset: 0,
  });
  const actor=await learningActor();const states=actor.ok?await getLearningClient().listLearningKnowledge({limit:20,offset:0}):null;const personal=actor.ok&&states?<LearningBoundary actorId={actor.id}>{states.ok?<KnowledgeList page={states.data} paginate={false}/>:<LearningNotice result={states}/>}</LearningBoundary>:undefined;
  return <><UiPageTitle messageKey="page.knowledge"/>{<KnowledgeMap result={result} q={query.q} status={query.status} personal={personal} />}</>;
}
