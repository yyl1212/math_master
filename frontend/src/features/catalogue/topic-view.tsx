"use client";
import {ReportLink} from "@/features/feedback/report-link";
import Link from "next/link";
import {useState} from "react";
import Form from "next/form";
import {useUiI18n} from "@/lib/i18n/provider";
import {ContentState} from "@/components/content-state";
import type {TaxonomyResult,TopicPage,TopicDetail,KnowledgePage,TopicNode,TopicSummary} from "@/lib/taxonomy/types";
import styles from "@/styles/catalogue.module.css";
function topicName(n:TopicNode,locale:string){return locale==="zh-CN"&&n.nameZh?n.nameZh:n.name}
function TopicAncestry({topic}:{topic:TopicSummary}){
 const {t,locale}=useUiI18n();
 if(topic.ancestors.length===0)return null;
 return <nav aria-label={t("topic.map.hierarchy",{})}><ol className={styles.ancestry}>{topic.ancestors.map(a=><li key={a.id}><Link prefetch={false} href={"/topics/"+a.id}>{a.code} {topicName(a,locale)}</Link><span aria-hidden="true"> › </span></li>)}<li>{topic.code} {topicName(topic,locale)}</li></ol></nav>;
}
export function TopicMap({result,q,level,kind="primary"}:{result:TaxonomyResult<TopicPage>;q:string;level?:number;kind?:string}){
 const {t,locale}=useUiI18n(),[selectedLevel,setLevel]=useState(level??0),[selectedKind,setKind]=useState(kind);
 function url(offset:number){const query=new URLSearchParams();if(q)query.set("q",q);if(level)query.set("level",String(level));if(kind!=="primary")query.set("kind",kind);if(offset)query.set("offset",String(offset));return "/knowledge"+(query.size?"?"+query:"")}
 return <><div className="page-heading"><p className="eyebrow">MSC2020</p><h1>{t("nav.knowledgeMap",{})}</h1><p>{t("topic.map.intro",{})}</p></div><Form role="search" action="/knowledge" prefetch={false} className={styles.toolbar}><div className={styles.search}><label htmlFor="topic-search">{t("topic.map.search",{})}</label><div className={styles.searchInput}><input type="search" id="topic-search" name="q" defaultValue={q} maxLength={512}/></div></div><div className={styles.filter}><label htmlFor="topic-level">{t("topic.map.level",{})}</label><select id="topic-level" name="level" value={selectedLevel} onChange={e=>setLevel(Number(e.target.value))}><option value="0">{t("topic.map.allLevels",{})}</option><option value="1">{t("topic.map.level1",{})}</option><option value="2">{t("topic.map.level2",{})}</option><option value="3">{t("topic.map.level3",{})}</option></select></div><div className={styles.filter}><label htmlFor="topic-kind">{t("topic.map.kind",{})}</label><select id="topic-kind" name="kind" value={selectedKind} onChange={e=>{setKind(e.target.value);setLevel(e.target.value==="auxiliary"?2:e.target.value==="other"?3:1)}}><option value="primary">{t("topic.map.primary",{})}</option><option value="auxiliary">{t("topic.map.auxiliary",{})}</option><option value="other">{t("topic.map.other",{})}</option></select></div><button className="button" type="submit">{t("knowledge-map.search.49c266",{})}</button></Form>{!result.ok?<ContentState kind={result.code==="NOT_FOUND"?"not-found":"unavailable"}/>:<><div className={styles.results}><span>{result.data.total}</span></div>{result.data.items.length===0?<p>{t("topic.map.noResults",{})}</p>:<div className="domain-grid">{result.data.items.map(n=><article className="domain-card" key={n.id}><div className="card-top"><code>{n.code}</code><span>{n.level}</span></div><h2 className={styles.cardTitle}><Link prefetch={false} href={"/topics/"+n.id}>{topicName(n,locale)}</Link></h2>{n.nameZh&&locale==="en"&&<p className="zh" lang="zh-CN">{n.nameZh}</p>}<TopicAncestry topic={n}/><div className="card-bottom"><span>{t(n.publishedKnowledgeCount===1?"public.publishedOne":"public.publishedMany",{count:n.publishedKnowledgeCount})}</span></div></article>)}</div>}<nav className={styles.toolbar} aria-label={t("topic.map.explore",{})}>{result.data.offset>0&&<Link prefetch={false} className="button secondary" href={url(Math.max(0,result.data.offset-result.data.limit))}>{t("topic.map.previous",{})}</Link>}{result.data.offset+result.data.limit<result.data.total&&<Link prefetch={false} className="button secondary" href={url(result.data.offset+result.data.limit)}>{t("topic.map.next",{})}</Link>}</nav></>}<p className={styles.mapNote}>{t("topic.map.note",{})}</p></>
}
export function TopicView({detail,childrenPage,knowledge,childrenQ="",knowledgeQ=""}:{detail:TopicDetail;childrenPage:TopicPage;knowledge:KnowledgePage;childrenQ?:string;knowledgeQ?:string}){
 const {t,locale}=useUiI18n(),n=detail.summary,action="/topics/"+n.id;
 function url(key:"offset"|"knowledgeOffset",value:number){
  const query=new URLSearchParams();
  if(childrenQ)query.set("childrenQ",childrenQ);
  if(knowledgeQ)query.set("knowledgeQ",knowledgeQ);
  const offset=key==="offset"?value:childrenPage.offset,knowledgeOffset=key==="knowledgeOffset"?value:knowledge.offset;
  if(offset)query.set("offset",String(offset));
  if(knowledgeOffset)query.set("knowledgeOffset",String(knowledgeOffset));
  return action+(query.size?"?"+query:"");
 }
 return <>
  <nav className="breadcrumbs" aria-label={t("domain-view.breadcrumb.2bd873",{})}><Link prefetch={false} href="/knowledge">{t("nav.knowledgeMap",{})}</Link>{n.ancestors.map(a=><span key={a.id}> / <Link prefetch={false} href={"/topics/"+a.id}>{topicName(a,locale)}</Link></span>)}</nav><div className="page-heading"><p className="eyebrow"><code>{n.code}</code></p><h1>{topicName(n,locale)}</h1><p>{t(n.publishedKnowledgeCount===1?"public.publishedOne":"public.publishedMany",{count:n.publishedKnowledgeCount})}</p></div>
  <div className={styles.detailGrid}><div className={styles.detailMain}>
   <section className="panel"><h2>{t("topic.map.children",{})}</h2>
    <Form role="search" aria-label={t("topic.map.searchChildren",{})} action={action} prefetch={false} className={styles.toolbar}>
     <div className={styles.search}><label htmlFor="children-search">{t("topic.map.searchChildren",{})}</label><div className={styles.searchInput}><input key={childrenQ} id="children-search" type="search" name="childrenQ" defaultValue={childrenQ} maxLength={512}/></div></div>
     {knowledgeQ&&<input type="hidden" name="knowledgeQ" value={knowledgeQ}/>}{knowledge.offset>0&&<input type="hidden" name="knowledgeOffset" value={knowledge.offset}/>}
     <button className="button" type="submit">{t("knowledge-map.search.49c266",{})}</button>
    </Form>
    <p className={styles.results}>{childrenPage.total}</p>
    {childrenPage.items.length===0?<p>{t("topic.map.noResults",{})}</p>:<div className={styles.topics}>{childrenPage.items.map(c=><article key={c.id}><code>{c.code}</code><h3><Link prefetch={false} href={"/topics/"+c.id}>{topicName(c,locale)}</Link></h3><p>{t(c.publishedKnowledgeCount===1?"public.publishedOne":"public.publishedMany",{count:c.publishedKnowledgeCount})}</p></article>)}</div>}
    <nav className={styles.toolbar}>{childrenPage.offset>0&&<Link prefetch={false} href={url("offset",Math.max(0,childrenPage.offset-childrenPage.limit))}>{t("topic.map.previous",{})}</Link>}{childrenPage.offset+childrenPage.limit<childrenPage.total&&<Link prefetch={false} href={url("offset",childrenPage.offset+childrenPage.limit)}>{t("topic.map.next",{})}</Link>}</nav>
   </section>
   <section className="panel"><h2>{t("topic.map.knowledge",{})}</h2>
    <Form role="search" aria-label={t("topic.map.searchKnowledge",{})} action={action} prefetch={false} className={styles.toolbar}>
     <div className={styles.search}><label htmlFor="knowledge-search">{t("topic.map.searchKnowledge",{})}</label><div className={styles.searchInput}><input key={knowledgeQ} id="knowledge-search" type="search" name="knowledgeQ" defaultValue={knowledgeQ} maxLength={512}/></div></div>
     {childrenQ&&<input type="hidden" name="childrenQ" value={childrenQ}/>}{childrenPage.offset>0&&<input type="hidden" name="offset" value={childrenPage.offset}/>}
     <button className="button" type="submit">{t("knowledge-map.search.49c266",{})}</button>
    </Form>
    <p className={styles.results}>{knowledge.total}</p>
    {knowledge.items.length===0?<p>{t(knowledgeQ?"topic.map.noKnowledgeResults":"topic.map.empty",{})}</p>:<div className={styles.paths}>{knowledge.items.map(k=><Link prefetch={false} className={styles.pathCard} key={k.id} href={"/knowledge/"+k.id}><div><h3>{locale==="zh-CN"&&k.titleZh?k.titleZh:k.title}</h3><p>v{k.version}</p></div><span aria-hidden="true">↗</span></Link>)}</div>}
    <nav className={styles.toolbar}>{knowledge.offset>0&&<Link prefetch={false} href={url("knowledgeOffset",Math.max(0,knowledge.offset-knowledge.limit))}>{t("topic.map.previousKnowledge",{})}</Link>}{knowledge.offset+knowledge.limit<knowledge.total&&<Link prefetch={false} href={url("knowledgeOffset",knowledge.offset+knowledge.limit)}>{t("topic.map.nextKnowledge",{})}</Link>}</nav>
   </section>
  </div><aside className={styles.detailSide}><section className="panel"><p>{t("topic.map.note",{})}</p><ReportLink source={{kind:"site",area:"other"}} topic={{topicId:n.id,taxonomyVersionId:detail.pair.taxonomyVersionId}}/></section></aside></div>
 </>;
}
