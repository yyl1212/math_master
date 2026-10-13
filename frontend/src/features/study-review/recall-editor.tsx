'use client';
import {useEffect,useRef,useState} from 'react';
import {useUiI18n} from '@/lib/i18n/provider';
import {NoteToolbar} from '@/features/notes/note-toolbar';
import {NotePreview} from '@/features/notes/note-preview';
import styles from '@/styles/study-review.module.css';
export function answerOverLimit(value:string){return !value.isWellFormed()||value.includes('\0')||Array.from(value).length>16000||new TextEncoder().encode(value).length>65536}
export function RecallEditor({value,onChange,disabled,dirty}:{value:string;onChange:(value:string)=>void;disabled:boolean;dirty:boolean}){
 const{t}=useUiI18n(),[preview,setPreview]=useState(false),[selection,setSelection]=useState({start:0,end:0}),input=useRef<HTMLTextAreaElement>(null);
 // The existing guard records focusin. A first edit can make the guard dirty after
 // the textarea was already focused, so announce that focused element once.
 useEffect(()=>{if(dirty&&input.current===document.activeElement)input.current?.dispatchEvent(new FocusEvent('focusin',{bubbles:true}))},[dirty]);
 return <div className={styles.editor}><div className={styles.tabs}><button type="button" aria-pressed={!preview} onClick={()=>setPreview(false)}>{t('studyReview.edit',{})}</button><button type="button" aria-pressed={preview} onClick={()=>setPreview(true)}>{t('studyReview.preview',{})}</button></div>
 {!preview&&<><NoteToolbar value={value} onChange={onChange} selection={selection} disabled={disabled}/><label>{t('studyReview.answer',{})}<textarea ref={input} value={value} disabled={disabled} rows={12} onChange={event=>onChange(event.target.value)} onSelect={event=>setSelection({start:event.currentTarget.selectionStart,end:event.currentTarget.selectionEnd})}/></label></>}
 {preview&&<section aria-label={t('studyReview.preview',{})}><NotePreview body={value}/></section>}
 <p className={styles.hint}>{t('studyReview.characters',{count:Array.from(value).length})}</p>{dirty&&<p>{t('studyReview.unsaved',{})}</p>}{answerOverLimit(value)&&<p role="alert">{t('studyReview.draftLimit',{})}</p>}</div>
}
