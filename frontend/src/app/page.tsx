import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
import {getGoClient} from "@/lib/api/server-client";
import {LearningHub} from "@/components/learning-hub";
import {KnowledgeMap} from "@/features/catalogue/knowledge-map";
import {readServerTaxonomy} from "@/lib/taxonomy/server-client";
import type {TopicPage} from "@/lib/taxonomy/types";
export async function generateMetadata(){return getUiMetadata("page.home")}
export const dynamic="force-dynamic";
export default async function Page(){
 const topics=await readServerTaxonomy<TopicPage>({kind:"listTopics",query:{level:1,limit:100}});
 if(topics.ok||topics.code!=="TAXONOMY_NOT_CONFIGURED")return <><UiPageTitle messageKey="page.home"/><KnowledgeMap topics={topics} q="" level={1}/></>;
 const result=await getGoClient().listDomains({q:"",limit:100,offset:0});return <><UiPageTitle messageKey="page.home"/><LearningHub result={result}/></>
}
