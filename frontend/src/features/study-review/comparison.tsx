'use client';
import {useState} from 'react';
import {useUiI18n} from '@/lib/i18n/provider';
import {NotePreview} from '@/features/notes/note-preview';
import type {Session} from '@/lib/study-review/types';
import {ReviewSections} from './self-rating';
import styles from '@/styles/study-review.module.css';
export function Comparison({session}:{session:Session}){
 const{t}=useUiI18n(),[tab,setTab]=useState<'answer'|'reference'>('answer'),available=session.available&&!session.materialChanged&&session.reference!==null;
 return <div className={styles.comparison}><p>{t('studyReview.frozen',{})}</p><p className={styles.hint}>{t('studyReview.compareHint',{})}</p><div className={styles.comparisonTabs}><button type="button" aria-pressed={tab==='answer'} onClick={()=>setTab('answer')}>{t('studyReview.answer',{})}</button>{available&&<button type="button" aria-pressed={tab==='reference'} onClick={()=>setTab('reference')}>{t('studyReview.reference',{})}</button>}</div><div className={styles.comparisonColumns} data-tab={available?tab:'answer'}><section className={styles.answerColumn} aria-label={t('studyReview.answer',{})}><h2>{t('studyReview.answer',{})}</h2><NotePreview body={session.answer??''}/></section>{available&&session.reference&&<section className={styles.referenceColumn} aria-label={t('studyReview.reference',{})}><h2>{t('studyReview.reference',{})}</h2><ReviewSections sections={session.reference.sections}/></section>}</div></div>
}
