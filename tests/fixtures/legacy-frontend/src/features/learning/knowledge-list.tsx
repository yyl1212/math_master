"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import type {KnowledgeState,LearningPage} from "@/lib/learning/types";
import {LearningStatus} from "./learning-status";
import styles from "@/styles/learning.module.css";
export function KnowledgeList({page,paginate=true}:{page:LearningPage<KnowledgeState>;paginate?:boolean}){
 const {t}=useUiI18n();
return <section className={styles.panel} aria-label={t("knowledge-list.knowledge.learning.states.57e222",{})}><h2><UiText notice={uiMessage("knowledge-list.your.knowledge.map.cbbacd",{})}/></h2>{page.items.length?<ul className={styles.nodes}>{page.items.map(v=><li key={v.knowledge.id}><h3><Link prefetch={false} href={"/knowledge/"+v.knowledge.id}>{v.title}</Link></h3><LearningStatus state={v}/><Link prefetch={false} href={`/learning-history?knowledge=${v.knowledge.id}&version=${v.knowledge.version}`}><UiText notice={uiMessage("knowledge-list.review.learning.record.de4302",{})}/></Link></li>)}</ul>:<p><UiText notice={uiMessage("knowledge-list.no.reviewed.knowledge.points.are.available.yet.3cee4b",{})}/></p>}{paginate?<nav aria-label={t("knowledge-list.knowledge.pages.10ab02",{})} className={styles.actions}>{page.offset>0&&<Link prefetch={false} href={"/learn?knowledgeOffset="+Math.max(0,page.offset-page.limit)}><UiText notice={uiMessage("knowledge-list.previous.knowledge.points.4419a5",{})}/></Link>}{page.offset+page.limit<page.total&&<Link prefetch={false} href={"/learn?knowledgeOffset="+(page.offset+page.limit)}><UiText notice={uiMessage("knowledge-list.next.knowledge.points.da8759",{})}/></Link>}</nav>:<Link prefetch={false} href="/learn"><UiText notice={uiMessage("knowledge-list.explore.your.full.learning.map.d9ec05",{})}/></Link>}</section>}
