'use client';
import {Pagination} from '@/components/pagination';
import type {Numbered} from '@/lib/pagination/schemas';
import Link from 'next/link';
import {useEffect,useState} from 'react';
import {useUiI18n} from '@/lib/i18n/provider';
import {LocalizedName} from '@/lib/i18n/localized-name';
import {TopicPicker} from '@/components/topic-picker';
import {historyHref} from '@/lib/knowledge-admin/study-navigation';
import {LearningHeader} from './managed-learning-view';
import styles from '@/styles/managed-learning.module.css';
import type {HistoryPage} from '@/lib/knowledge-admin/types';
import type {HistoryPage as LegacyHistoryPage} from '@/lib/study/types';
const labels={started:'study.event.started',completed:'study.event.completed','review-started':'study.event.reviewStarted','review-finished':'study.event.reviewFinished','note-saved':'study.event.noteSaved','note-deleted':'study.event.noteDeleted'} as const;
function EventTime({value}:{value:string}){const{locale}=useUiI18n();const[zone,setZone]=useState('UTC');useEffect(()=>{setZone(Intl.DateTimeFormat().resolvedOptions().timeZone)},[]);return <time dateTime={value}>{new Date(value).toLocaleString(locale,{timeZone:zone})}</time>}
export function ManagedHistory({page,legacy,filters={}}:{page:HistoryPage|Numbered<HistoryPage>;legacy?:LegacyHistoryPage|Numbered<LegacyHistoryPage>;filters?:Record<string,string>}){
 const{t}=useUiI18n(),[topics,setTopics]=useState(filters.topicKey?[filters.topicKey]:[]);const paging=(key:string,cursor:string)=>historyHref({...filters,[key]:cursor});
 return <div className={styles.page}><LearningHeader history/><section className={styles.knowledge}><h2>{t('managed.history',{})}</h2><p className={styles.hint}>{t('managed.historyTruth',{})}</p>
  <form action="/learn" className={"ui-form ui-filter "+styles.historyFilters}><input type="hidden" name="view" value="history"/><TopicPicker value={topics} onChange={setTopics}/><input type="hidden" name="topicKey" value={topics[0]??''}/>{['knowledgeId','from','to'].map(key=>filters[key]&&<input key={key} type="hidden" name={key} value={filters[key]}/>)}<label>{t('managed.action',{})}<select name="kind" defaultValue={filters.kind??''}><option value="">{t('managed.all',{})}</option>{Object.entries(labels).map(([kind,label])=><option key={kind} value={kind}>{t(label,{})}</option>)}</select></label><div className="form-actions"><button className="button">{t('managed.search',{})}</button><Link prefetch={false} href={historyHref()} onClick={event=>{setTopics([]);event.currentTarget.closest('form')?.reset()}}>{t('learn.dashboard.clear',{})}</Link></div></form>
  <ol className={styles.timeline}>{page.items.map(event=><li key={'managed:'+event.id} style={{paddingBlock:14,borderBottom:'1px solid var(--border)',overflowWrap:'anywhere'}}><EventTime value={event.recordedAt}/><p>{t(labels[event.kind],{})} · {event.available?<Link prefetch={false} href={'/knowledge/'+event.knowledge.id}><LocalizedName english={event.title} chinese={event.titleZh??''}/></Link>:event.title||event.titleZh?<LocalizedName english={event.title} chinese={event.titleZh??''}/>:t('streamline.history.unavailable',{})}</p><details><summary>{t('streamline.history.details',{})}</summary><p>{event.knowledge.id} · {event.topicKeys.join(' · ')}</p></details></li>)}</ol>
  {page.items.length===0&&<p>{t('managed.empty',{})}</p>}{'total' in page?<Pagination meta={page} label={t('managed.history',{})} navigation={{kind:'link',pathname:'/learn',query:{...filters,view:'history'},pageMode:true,pageKey:'page',cursorKey:'cursor'}}/>:'nextCursor' in page&&page.nextCursor&&<Link prefetch={false} href={paging('cursor',page.nextCursor)}>{t('managed.next',{})}</Link>}
  {legacy&&<details><summary>{t('managed.legacyHistory',{})}</summary><p>{t('streamline.history.early',{})}</p><ol className={styles.timeline}>{legacy.items.map(event=><li key={'legacy:'+event.id}><EventTime value={event.occurredAt}/><p>{t(labels[event.kind],{})} · {event.title||event.titleZh?<LocalizedName english={event.title} chinese={event.titleZh??''}/>:t('streamline.history.archived',{})}</p><details><summary>{t('streamline.history.details',{})}</summary>{event.knowledge.id}</details></li>)}</ol>{'total' in legacy?<Pagination meta={legacy} label={t('managed.legacyHistory',{})} navigation={{kind:'link',pathname:'/learn',query:{...filters,view:'history'},pageMode:true,pageKey:'legacyPage',cursorKey:'legacyCursor'}}/>:'nextCursor' in legacy&&legacy.nextCursor&&<Link prefetch={false} href={paging('legacyCursor',legacy.nextCursor)}>{t('managed.next',{})}</Link>}</details>}
 </section></div>;
}
