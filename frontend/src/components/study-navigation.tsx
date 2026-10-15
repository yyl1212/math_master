'use client';

import Link from 'next/link';
import {useUiI18n} from '@/lib/i18n/provider';
import {historyHref} from '@/lib/knowledge-admin/study-navigation';
import styles from '@/styles/study-navigation.module.css';

const sections=[
 {id:'knowledge',href:'/learn',label:'learn.workspace.knowledge'},
 {id:'notes',href:'/learn?view=notes',label:'notes.mine'},
 {id:'review',href:'/learn/review',label:'studyReview.workspace'},
 {id:'review-history',href:'/learn/review/history',label:'studyReview.history'},
 {id:'learning-history',href:historyHref(),label:'managed.history'},
] as const;

export function StudyNavigation({active}:{active:typeof sections[number]['id']}){
 const{t}=useUiI18n();
 return <nav className={styles.navigation} aria-label={t('learn.workspace.sections',{})}>
  {sections.map(section=><Link key={section.id} prefetch={false} href={section.href} aria-current={active===section.id?'page':undefined}>{t(section.label,{})}</Link>)}
 </nav>;
}
