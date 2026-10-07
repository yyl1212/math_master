import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
import {ContentState} from "@/components/content-state";
import {notFound} from "next/navigation";
import {readServerTaxonomy} from "@/lib/taxonomy/server-client";
import {topicIdPattern} from "@/lib/taxonomy/protocol";
import {parseTopicDetailOffsets,sameTopicPair} from "@/lib/taxonomy/page-query";
import {TopicView} from "@/features/catalogue/topic-view";
import type {TopicPage,TopicDetail,KnowledgePage} from "@/lib/taxonomy/types";
export async function generateMetadata(){return getUiMetadata("page.topics.id")}
export const dynamic="force-dynamic";
export default async function Page({params,searchParams}:{params:Promise<{id:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 const {id}=await params;if(!topicIdPattern.test(id))notFound();const offsets=parseTopicDetailOffsets(await searchParams);if(!offsets)return <ContentState kind="unavailable"/>;
 const [detail,children,knowledge]=await Promise.all([readServerTaxonomy<TopicDetail>({kind:"readTopic",id}),readServerTaxonomy<TopicPage>({kind:"listTopics",query:{parentId:id,limit:20,offset:offsets.offset,...(offsets.childrenQ?{q:offsets.childrenQ}:{})}}),readServerTaxonomy<KnowledgePage>({kind:"listKnowledge",id,query:{limit:20,offset:offsets.knowledgeOffset,...(offsets.knowledgeQ?{q:offsets.knowledgeQ}:{})}})]);
 if(!detail.ok&&detail.code==="NOT_FOUND")notFound();if(!detail.ok||!children.ok||!knowledge.ok||!sameTopicPair(detail.data.pair,children.data.pair)||!sameTopicPair(detail.data.pair,knowledge.data.pair))return <><UiPageTitle messageKey="page.topics.id"/><ContentState kind="unavailable"/></>;
 return <><UiPageTitle messageKey="page.topics.id"/><TopicView detail={detail.data} childrenPage={children.data} knowledge={knowledge.data} childrenQ={offsets.childrenQ} knowledgeQ={offsets.knowledgeQ}/></>
}
