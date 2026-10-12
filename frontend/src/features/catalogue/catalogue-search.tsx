'use client';
import Link from 'next/link';
import {useEffect,useState} from 'react';
import {TopicPicker} from '@/components/topic-picker';
import {useUiI18n} from '@/lib/i18n/provider';

export function CatalogueSearch({initialMode,initialQuery,topicKey}:{initialMode:'topics'|'knowledge';initialQuery:string;topicKey?:string}){
 const{t}=useUiI18n();const[mode,setMode]=useState(initialMode),[query,setQuery]=useState(initialQuery),[topics,setTopics]=useState(topicKey?[topicKey]:[]),[ready,setReady]=useState(false);
 useEffect(()=>{setReady(true)},[]);
 return <form action="/knowledge" className="panel ui-form ui-filter" role="search" aria-label={t('map.search.entry',{})}>
  <label>{t('map.search.target',{})}<select name="mode" value={mode} disabled={!ready} onChange={event=>setMode(event.target.value as 'topics'|'knowledge')}><option value="topics">{t('managed.topics',{})}</option><option value="knowledge">{t('map.search.knowledge',{})}</option></select></label>
  <label>{t('map.search.keyword',{})}<input name="q" type="search" maxLength={256} value={query} disabled={!ready} onChange={event=>setQuery(event.target.value)}/></label>
  {mode==='knowledge'&&<><TopicPicker value={topics} onChange={setTopics} disabled={!ready}/><input name="topicKey" type="hidden" value={topics[0]??''}/></>}
  <div className="form-actions"><button className="button" disabled={!ready}>{t('map.search.submit',{})}</button><Link prefetch={false} href={'/knowledge?mode='+mode}>{t('learn.dashboard.clear',{})}</Link></div>
 </form>;
}
