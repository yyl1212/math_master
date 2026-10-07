"use client";
import Link from "next/link";
import {useUiI18n} from "@/lib/i18n/provider";
import type {ContentReminder} from "@/lib/study/types";
export function ContentReminders({items}:{items:ContentReminder[]}){const{t}=useUiI18n();const pending=items.filter(item=>!item.reviewed);if(!pending.length)return null;return <section className="panel"><h2>{t("study.reminders.title",{})}</h2><p>{t("study.reminders.readonly",{})}</p><ul>{pending.map(item=><li key={item.changeId}><p>{t(item.kind==="updated"?"study.materialChanged":"study.withdrawn",{})}</p>{item.kind==="updated"&&item.currentRef?<Link prefetch={false} href={"/knowledge/"+item.knowledgeId}>{item.knowledgeId} v{item.currentRef.version}</Link>:<span>{item.knowledgeId}</span>} <time dateTime={item.recordedAt}>{new Date(item.recordedAt).toISOString().slice(0,16).replace("T"," ")} UTC</time></li>)}</ul><Link prefetch={false} href="/learn?mode=review">{t("study.review.search",{})}</Link></section>}
