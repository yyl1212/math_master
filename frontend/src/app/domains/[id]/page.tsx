import {getContentMode} from "@/lib/knowledge-admin/mode";
import {CurrentCatalogue} from "@/features/catalogue/current-catalogue";
import {readExperienceMode} from "@/lib/study/mode";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
import {notFound} from "next/navigation";
import {getGoClient} from "@/lib/api/server-client";
import {DomainView,TopicDomainView} from "@/features/catalogue/domain-view";
import {ContentState} from "@/components/content-state";
import {readServerTaxonomy} from "@/lib/taxonomy/server-client";
import {resolveLegacyDomain} from "@/lib/taxonomy/legacy-aliases";
import type {TopicPage} from "@/lib/taxonomy/types";
export async function generateMetadata(){return getUiMetadata("page.domains.id")}
export const dynamic="force-dynamic";
export default async function Page({params}:{params:Promise<{id:string}>}){
 const content=await getContentMode();if(content===null)return <ContentState kind="unavailable"/>;if(content.mode==="managed")return <CurrentCatalogue/>;const mode=await readExperienceMode();if(mode===null)return <ContentState kind="unavailable"/>;
 const {id}=await params,aliases=resolveLegacyDomain(id);if(!aliases)notFound();const roots=await readServerTaxonomy<TopicPage>({kind:"listTopics",query:{level:1,limit:100}});
 if(roots.ok){const details=roots.data.items.filter(n=>aliases.includes(n.id)).map(summary=>({summary,pair:roots.data.pair}));return <><UiPageTitle messageKey="page.domains.id"/><TopicDomainView details={details}/></>}
 if(mode==="topics"||roots.code!=="TAXONOMY_NOT_CONFIGURED")return <><UiPageTitle messageKey="page.domains.id"/><ContentState kind="unavailable"/></>;
 const result=await getGoClient().getDomain(id);if(!result.ok&&result.kind==="not-found")notFound();return <><UiPageTitle messageKey="page.domains.id"/><DomainView result={result}/></>
}
