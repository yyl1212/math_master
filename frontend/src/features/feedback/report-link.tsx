"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import Link from 'next/link';
import type {ContextQuery} from '@/lib/feedback/types';
export function ReportLink({source,topic}:{source:ContextQuery;topic?:{topicId:string;taxonomyVersionId:string}}) {
  const params=new URLSearchParams();
  for(const key of ['kind','id','area','position','partKind','partId'] as const) if(source[key]!==undefined)params.set(key,String(source[key]));
  if(topic&&source.kind==="site"&&source.area==="other"){params.set("topicId",topic.topicId);params.set("taxonomyVersionId",topic.taxonomyVersionId)}
  return <Link prefetch={false} href={'/feedback/new?'+params} className="button secondary"><UiText notice={uiMessage("page.feedback.new",{})}/></Link>;
}
