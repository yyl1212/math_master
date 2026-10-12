'use client';

import Link from 'next/link';
import type {Overview,ProgressPage,StudyPage,Topic} from '@/lib/knowledge-admin/types';
import {useUiI18n} from '@/lib/i18n/provider';
import {LocalizedName} from '@/lib/i18n/localized-name';
import {UiPageTitle} from '@/components/ui-page-title';
import styles from '@/styles/managed-learning.module.css';

export type LearningTheme=ProgressPage['items'][number]&Pick<Topic,'title'|'titleEn'>;
type Filters=Record<string,string>;
const states=['unlearned','learning','completed','reviewing'] as const;
const stateKey=(state:typeof states[number])=>state==='completed'?'managed.learned':`managed.${state}` as const;
export function learningHref(filters:Filters,changes:Filters={}){
 const query=new URLSearchParams();
 for(const key of ['q','topicKey','state','mode','offset']){const value={...filters,...changes}[key];if(value)query.set(key,value);}
 return '/learn'+(query.size?'?'+query:'');
}
const primary=(key:string)=>key==='project:other'||key==='project-other'?'project:other':key.replace(/^msc-/,'').slice(0,2);

export function ManagedLearningView({overview,list,themes,filters,selectedTopic}:{overview:Overview;list:StudyPage;themes:LearningTheme[];filters:Filters;selectedTopic?:Pick<Topic,'topicKey'|'title'|'titleEn'>}){
 const{t,locale}=useUiI18n();
 const name=(theme:Pick<Topic,'title'|'titleEn'>)=>locale==='zh-CN'?theme.title||theme.titleEn:theme.titleEn||theme.title;
 const done=overview.completed+overview.reviewing;
 const progress=(completed:number,total:number)=>t('learn.dashboard.completedCount',{completed,total});
 const selected=themes.find(theme=>theme.topicKey===filters.topicKey)??selectedTopic;
 const unlearned=Math.max(0,overview.total-overview.completed-overview.learning-overview.reviewing);
 const counts={unlearned,learning:overview.learning,completed:overview.completed,reviewing:overview.reviewing};
 const reset=filters.mode==='review'?'/learn?mode=review':'/learn';
 const hasFilters=Boolean(filters.q||filters.topicKey||filters.state);
 return <div className={styles.page}>
  <UiPageTitle messageKey="page.learn"/>
  <header className={styles.header}>
   <div><h1>{t('managed.learn',{})}</h1><p>{t('learn.dashboard.intro',{})}</p></div>
   <div className={styles.actions}>
    <Link prefetch={false} className="button secondary" href={learningHref(filters,{mode:filters.mode==='review'?'':'review',offset:''})}>{t(filters.mode==='review'?'learn.dashboard.backToLearning':'managed.review',{})}</Link>
    <Link prefetch={false} href="/learning-history">{t('managed.history',{})}</Link>
   </div>
  </header>
  <section className={styles.overview} aria-labelledby="learning-overview">
   <div className={styles.sectionHeading}><h2 id="learning-overview">{t('managed.overview',{})}</h2><span>{t('learn.dashboard.total',{total:overview.total})}</span></div>
   <dl className={styles.stats}>{states.map(state=><div key={state} data-state={state}><dt>{t(stateKey(state),{})}</dt><dd>{counts[state]}</dd></div>)}</dl>
   <div className={styles.overallProgress}><span>{progress(done,overview.total)}</span><span>{overview.total?Math.round(done/overview.total*100):0}%</span></div>
   <progress className={styles.progress} max={Math.max(1,overview.total)} value={done} aria-label={t('learn.dashboard.overallProgress',{})}/>
   <p className={styles.hint}>{t('study.progress.rule',{})}</p>
   {overview.materialChanged>0&&<p className={styles.notice}>{t('managed.changed',{})} ({overview.materialChanged})</p>}
  </section>
  <div className={styles.layout}>
   <nav className={styles.themes} aria-labelledby="learning-themes">
    <h2 id="learning-themes">{t('learn.dashboard.themes',{})}</h2>
    <p className={styles.hint}>{t('learn.dashboard.themeHint',{})}</p>
    <Link prefetch={false} className={styles.allThemes} aria-current={!filters.topicKey?'page':undefined} href={learningHref(filters,{topicKey:'',offset:''})}>{t('learn.dashboard.allThemes',{})}<span>{overview.total}</span></Link>
    <div className={styles.themeList}>{themes.map(theme=><Link prefetch={false} key={theme.topicKey} href={learningHref(filters,{topicKey:theme.topicKey,offset:''})} className={styles.theme} aria-current={filters.topicKey&&primary(filters.topicKey)===primary(theme.topicKey)?'page':undefined}>
     <strong><LocalizedName english={theme.titleEn} chinese={theme.title}/></strong>
     <div className={styles.themeProgress}><span>{progress(theme.completed+theme.reviewing,theme.total)}</span><span>{Math.round((theme.completed+theme.reviewing)/theme.total*100)}%</span></div>
     <progress className={styles.progress} max={theme.total} value={theme.completed+theme.reviewing} aria-label={t('learn.dashboard.themeProgress',{theme:name(theme)??'',completed:theme.completed+theme.reviewing,total:theme.total})}/>
     {theme.learning>0&&<span className={styles.hint}>{t('learn.dashboard.learningCount',{count:theme.learning})}</span>}
    </Link>)}</div>
    {themes.length===0&&<p className={styles.hint}>{t('learn.dashboard.noThemes',{})}</p>}
    <Link prefetch={false} className={styles.mapLink} href="/knowledge">{t('learn.dashboard.browseMap',{})} →</Link>
   </nav>
   <section className={styles.knowledge} aria-labelledby="learning-knowledge">
    <div className={styles.sectionHeading}><h2 id="learning-knowledge">{t(filters.mode==='review'?'study.review.search':'study.knowledge',{})}</h2><span>{t('learn.dashboard.total',{total:list.total})}</span></div>
    {selected&&<p className={styles.selected}><LocalizedName english={selected.titleEn} chinese={selected.title}/></p>}
    {filters.mode==='review'&&<p className={styles.hint}>{t('learn.dashboard.reviewHint',{})}</p>}
    <form action="/learn" className={styles.filters} key={JSON.stringify(filters)}>
     <label className={styles.search}>{t('learn.dashboard.search',{})}<input name="q" type="search" defaultValue={filters.q??''} placeholder={t('learn.dashboard.searchPlaceholder',{})}/></label>
     <label>{t('learn.dashboard.theme',{})}<select name="topicKey" defaultValue={filters.topicKey??''}><option value="">{t('learn.dashboard.allThemes',{})}</option>{themes.map(theme=><option key={theme.topicKey} value={theme.topicKey}>{name(theme)}</option>)}{selected&&!themes.some(theme=>theme.topicKey===selected.topicKey)&&<option value={selected.topicKey}>{name(selected)}</option>}</select></label>
     <label>{t('learn.dashboard.state',{})}<select name="state" defaultValue={filters.state??''}><option value="">{t('managed.all',{})}</option>{states.map(state=><option key={state} value={state}>{t(stateKey(state),{})}</option>)}</select></label>
     {filters.mode==='review'&&<input type="hidden" name="mode" value="review"/>}
     <button className="button">{t('learn.dashboard.apply',{})}</button>
     {hasFilters&&<Link prefetch={false} className={styles.clear} href={reset}>{t('learn.dashboard.clear',{})}</Link>}
    </form>
    <div className={styles.knowledgeList}>{list.items.map(detail=>{
     const point=detail.currentKnowledge?.point;
     const memberships=themes.filter(theme=>detail.currentKnowledge?.topicKeys.some(key=>primary(key)===primary(theme.topicKey)));
     return <article className={styles.knowledgeRow} key={detail.record.knowledgeId}>
      <div className={styles.rowMain}><div className={styles.rowHeading}><h3><Link prefetch={false} href={`/knowledge/${detail.record.knowledgeId}`}>{point?<LocalizedName english={point.title} chinese={point.title_zh}/>:t('managed.unavailable',{})}</Link></h3><span className={styles.badge} data-state={detail.record.state}>{t(stateKey(detail.record.state),{})}</span></div>
       <div className={styles.metadata}>{memberships.map(theme=><span key={theme.topicKey}><LocalizedName english={theme.titleEn} chinese={theme.title}/></span>)}{point&&<><span>{t(`managed.type.${point.type}`,{})}</span><span>{t('learn.dashboard.difficulty',{level:point.learning_difficulty.difficulty_level})}</span></>}</div>
       {detail.materialChanged&&<p className={styles.notice}>{t('managed.changed',{})}</p>}
      </div>
      <Link prefetch={false} className={styles.readLink} href={`/knowledge/${detail.record.knowledgeId}`}>{t(detail.record.state==='learning'?'learn.dashboard.continue':'learn.dashboard.read',{})} →</Link>
     </article>;
    })}</div>
    {list.items.length===0&&<div className={styles.empty}><p>{t(overview.total===0?'learn.dashboard.noKnowledge':'study.empty.search',{})}</p></div>}
    {(list.offset>0||list.offset+list.limit<list.total)&&<nav className={styles.paging} aria-label={t('learn.dashboard.pages',{})}>
     <span>{t('learn.dashboard.showing',{from:list.total?list.offset+1:0,to:Math.min(list.offset+list.items.length,list.total),total:list.total})}</span>
     <div>{list.offset>0&&<Link prefetch={false} href={learningHref(filters,{offset:String(Math.max(0,list.offset-list.limit))})}>{t('managed.previous',{})}</Link>}{list.offset+list.limit<list.total&&<Link prefetch={false} href={learningHref(filters,{offset:String(list.offset+list.limit)})}>{t('managed.next',{})}</Link>}</div>
    </nav>}
   </section>
  </div>
 </div>;
}
