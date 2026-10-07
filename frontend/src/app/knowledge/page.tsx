import {readExperienceMode} from "@/lib/study/mode";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
import {getGoClient} from "@/lib/api/server-client";
import {KnowledgeMap} from "@/features/catalogue/knowledge-map";
import {ContentState} from "@/components/content-state";
import {parseCatalogueQuery} from "@/features/catalogue/query";
import {parseTopicPageQuery} from "@/lib/taxonomy/page-query";
import {readServerTaxonomy} from "@/lib/taxonomy/server-client";
import type {TopicPage} from "@/lib/taxonomy/types";
export async function generateMetadata(){return getUiMetadata("page.knowledge")}
export const dynamic="force-dynamic";
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){
 const mode=await readExperienceMode();if(mode===null)return <ContentState kind="unavailable"/>;
 const params=await searchParams,query=parseTopicPageQuery(params);if(!query.ok)return <><UiPageTitle messageKey="page.knowledge"/><ContentState kind="unavailable"/></>;
 const topics=await readServerTaxonomy<TopicPage>({kind:"listTopics",query:query.query});
 if(mode==="legacy"&&!topics.ok&&topics.code==="TAXONOMY_NOT_CONFIGURED"){
  const old=parseCatalogueQuery(Object.fromEntries(Object.entries(params).filter(([key])=>["q","status"].includes(key))));if(!old.ok)return <ContentState kind="unavailable"/>;
  const result=await getGoClient().listDomains({q:old.q,limit:100,offset:0});return <><UiPageTitle messageKey="page.knowledge"/><KnowledgeMap result={result} q={old.q} status={old.status}/></>
 }
 return <><UiPageTitle messageKey="page.knowledge"/><KnowledgeMap topics={topics} q={query.q} level={query.level} kind={query.kind}/></>
}
