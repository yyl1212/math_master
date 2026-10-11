'use client';
import {useUiI18n} from '@/lib/i18n/provider';
import type {CurrentInput} from '@/lib/knowledge-admin/types';
type Sources=CurrentInput['sources'];type Explanations=CurrentInput['point']['explanations'];
export function SourceFields({value,onChange}:{value:Sources;onChange:(value:Sources)=>void}){
 const{t}=useUiI18n();const update=(index:number,patch:Partial<Sources[number]>)=>onChange(value.map((source,n)=>n===index?{...source,...patch}:source));
 return <section><h2>{t('managed.source',{})}</h2>{value.map((source,index)=><fieldset key={index} className="managed-fields"><legend>{t('streamline.source.item',{number:index+1})}</legend>
  <label>{t('streamline.source.id',{})}<input aria-label={t('streamline.source.id',{})} required value={source.sourceId} onChange={e=>update(index,{sourceId:e.target.value})}/></label>
  <label>{t('streamline.source.title',{})}<input aria-label={t('streamline.source.title',{})} required value={source.title} onChange={e=>update(index,{title:e.target.value})}/></label>
  <label>{t('streamline.source.citation',{})}<textarea aria-label={t('streamline.source.citation',{})} required rows={3} value={source.citation} onChange={e=>update(index,{citation:e.target.value})}/></label>
  <label>{t('streamline.source.url',{})}<input aria-label={t('streamline.source.url',{})} type="url" value={source.url??''} onChange={e=>{const {url:unused,...rest}=source;onChange(value.map((item,n)=>n===index?{...rest,...(e.target.value?{url:e.target.value}:{})}:item));}}/></label>
  <button type="button" className="button secondary" disabled={value.length<=1} onClick={()=>onChange(value.filter((_,n)=>n!==index))}>{t('streamline.remove',{})}</button>
 </fieldset>)}<button type="button" className="button secondary" onClick={()=>onChange([...value,{sourceId:'',title:'',citation:''}])}>{t('streamline.source.add',{})}</button></section>;
}
export function ExplanationFields({value,onChange}:{value:Explanations;onChange:(value:Explanations)=>void}){
 const{t}=useUiI18n();const kinds=['intuition','formal','application','connection'] as const;
 return <section><h2>{t('streamline.explanation.title',{})}</h2>{value.map((item,index)=><fieldset key={index} className="managed-fields"><legend>{t('streamline.explanation.item',{number:index+1})}</legend>
  <label>{t('streamline.explanation.kind',{})}<select aria-label={t('streamline.explanation.kind',{})} value={item.kind} onChange={e=>onChange(value.map((entry,n)=>n===index?{...entry,kind:e.target.value as typeof item.kind}:entry))}>{kinds.map(kind=><option key={kind} value={kind}>{t(`streamline.explanation.${kind}`,{})}</option>)}</select></label>
  <label>{t('streamline.explanation.body',{})}<textarea aria-label={t('streamline.explanation.body',{})} rows={5} required value={item.body} onChange={e=>onChange(value.map((entry,n)=>n===index?{...entry,body:e.target.value}:entry))}/></label>
  <button type="button" className="button secondary" disabled={value.length<=1} onClick={()=>onChange(value.filter((_,n)=>n!==index))}>{t('streamline.remove',{})}</button>
 </fieldset>)}<button type="button" className="button secondary" disabled={value.length>=16} onClick={()=>onChange([...value,{kind:'intuition',body:''}])}>{t('streamline.explanation.add',{})}</button></section>;
}
