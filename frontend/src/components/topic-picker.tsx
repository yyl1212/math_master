'use client';
import {useEffect,useState} from 'react';
import {managedRequest} from '@/lib/knowledge-admin/client';
import type {Topic,TopicPage} from '@/lib/knowledge-admin/types';
import {useUiI18n} from '@/lib/i18n/provider';

type Props={value:string[];onChange:(value:string[])=>void;multiple?:boolean;leafOnly?:boolean;disabled?:boolean};
export function TopicPicker({value,onChange,multiple=false,leafOnly=false,disabled=false}:Props){
 const{locale,t}=useUiI18n();const[roots,setRoots]=useState<Topic[]>([]),[children,setChildren]=useState<Topic[]>([]),[leaves,setLeaves]=useState<Topic[]>([]),[names,setNames]=useState<Record<string,Topic>>({}),[error,setError]=useState(false);
 const first=value[0]??'',stem=first.replace(/^msc-/i,'').toUpperCase().replace(/-?XX$/,'');
 const[root,setRoot]=useState(first==='project:other'?first:/^\d{2}/.test(stem)?stem.slice(0,2)+'-XX':''),[secondary,setSecondary]=useState(stem.length>=3&&first!=='project:other'?stem.slice(0,3)+'xx':'');
 const title=(item:Topic)=>locale==='zh-CN'?item.title||item.titleEn:item.titleEn||item.title;
 const remember=(items:Topic[])=>setNames(old=>({...old,...Object.fromEntries(items.map(item=>[item.topicKey,item]))}));
 useEffect(()=>{let live=true;managedRequest<TopicPage>('/api/v3/topics?limit=100').then(page=>{if(live){setRoots(page.items);remember(page.items)}}).catch(()=>{if(live)setError(true)});return()=>{live=false}},[]);
 useEffect(()=>{let live=true;setChildren([]);if(!root||root==='project:other')return;managedRequest<TopicPage>('/api/v3/topics?'+new URLSearchParams({topicKey:root,limit:'100'})).then(page=>{if(live){setChildren(page.items);remember(page.items)}}).catch(()=>{if(live)setError(true)});return()=>{live=false}},[root]);
 useEffect(()=>{let live=true;setLeaves([]);if(!secondary)return;managedRequest<TopicPage>('/api/v3/topics?'+new URLSearchParams({topicKey:secondary,limit:'100'})).then(page=>{if(live){setLeaves(page.items);remember(page.items)}}).catch(()=>{if(live)setError(true)});return()=>{live=false}},[secondary]);
 const signature=value.join('|');
 useEffect(()=>{if(multiple)return;const key=value[0]??'',prefix=key.replace(/^msc-/i,'').toUpperCase().replace(/-?XX$/,'');setRoot(key==='project:other'?key:/^\d{2}/.test(prefix)?prefix.slice(0,2)+'-XX':'');setSecondary(key!=='project:other'&&prefix.length>=3?prefix.slice(0,3)+'xx':'');},[signature,multiple]);
 useEffect(()=>{let live=true;const selected=signature?signature.split('|'):[];void(async()=>{for(let i=0;i<selected.length&&live;i+=8){await Promise.all(selected.slice(i,i+8).map(async key=>{try{const topic=await managedRequest<Topic>('/api/v3/topics/'+encodeURIComponent(key));if(live)remember([topic])}catch{if(live)setError(true)}}))}})();return()=>{live=false}},[signature]);
 function choose(key:string){if(!key){onChange([]);return}if(multiple){if(value.includes(key)||value.length>=32)return;onChange([...value,key])}else onChange([key]);}
 return <fieldset disabled={disabled} style={{minWidth:0,border:0,padding:0,margin:0}}><legend>{t('managed.topics',{})}</legend>
  <div style={{display:'grid',gridTemplateColumns:'repeat(auto-fit,minmax(min(100%,180px),1fr))',gap:12}}>
   <label>{t('topic.map.level1',{})}<select aria-label={t('topic.map.level1',{})} value={root} onChange={e=>{const next=e.target.value;setRoot(next);setSecondary('');setLeaves([]);if(!leafOnly||next==='project:other'||!next)choose(next)}}><option value="">{t('managed.all',{})}</option>{roots.map(item=><option key={item.topicKey} value={item.topicKey}>{title(item)}</option>)}</select></label>
   <label>{t('topic.map.level2',{})}<select aria-label={t('topic.map.level2',{})} disabled={!root||root==='project:other'} value={secondary} onChange={e=>{const next=e.target.value;setSecondary(next);if(!leafOnly)choose(next||root)}}><option value="">{t('managed.all',{})}</option>{children.map(item=><option key={item.topicKey} value={item.topicKey}>{title(item)}</option>)}</select></label>
   <label>{t('topic.map.level3',{})}<select aria-label={t('topic.map.level3',{})} disabled={!secondary} value={multiple?'':value.find(key=>leaves.some(item=>item.topicKey===key))??''} onChange={e=>{if(e.target.value)choose(e.target.value);else if(!leafOnly)choose(secondary)}}><option value="">{t(multiple?'streamline.topic.add':'managed.all',{})}</option>{leaves.map(item=><option key={item.topicKey} value={item.topicKey}>{title(item)}</option>)}</select></label>
  </div>
  {multiple&&<ul>{value.map(key=><li key={key}>{names[key]?title(names[key]):t('streamline.topic.loading',{})} <button type="button" className="button secondary" disabled={disabled} onClick={()=>onChange(value.filter(item=>item!==key))}>{t('streamline.remove',{})}</button></li>)}</ul>}
  {error&&<p role="alert">{t('managed.service',{})}</p>}
 </fieldset>;
}
