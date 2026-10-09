import {getContentMode} from "@/lib/knowledge-admin/mode";
import {CurrentCatalogue} from "@/features/catalogue/current-catalogue";
import {ContentState} from "@/components/content-state";
import {readExperienceMode} from "@/lib/study/mode";
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
 const content=await getContentMode();if(content===null)return <ContentState kind="unavailable"/>;if(content.mode==="managed")return <CurrentCatalogue/>;const mode=await readExperienceMode();if(mode===null)return <ContentState kind="unavailable"/>;
 const topics=await readServerTaxonomy<TopicPage>({kind:"listTopics",query:{level:1,limit:100}});
 if(mode==="topics"||topics.ok||topics.code!=="TAXONOMY_NOT_CONFIGURED")return <><UiPageTitle messageKey="page.home"/><KnowledgeMap topics={topics} q="" level={1}/></>;
 const result=await getGoClient().listDomains({q:"",limit:100,offset:0});return <><UiPageTitle messageKey="page.home"/><LearningHub result={result}/></>
}
