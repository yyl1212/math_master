'use client';
import {useState,useEffect} from 'react';
import Link from 'next/link';
import {useUiI18n} from '@/lib/i18n/provider';
import {SafeMarkdown} from '@/features/reading/safe-markdown';
import {LocalizedName} from '@/lib/i18n/localized-name';
import {PageHeading} from '@/components/page-heading';
import {UiPageTitle} from '@/components/ui-page-title';
import type {Session,Rating,Schedule,Section} from '@/lib/study-review/types';
import {useReviewCommand,type ReviewCommandFlow} from './use-review-command';
import styles from '@/styles/study-review.module.css';
const sectionKeys={statement:'managed.statement',conditions:'managed.conditions',scope:'managed.scope',system:'managed.system',equations:'studyReview.section.equations',proof:'managed.proof','proof-scope':'managed.proofScope'} as const;
const ratingKey={forgotten:'studyReview.forgotten',uncertain:'studyReview.uncertainRating',familiar:'studyReview.familiar'} as const;
export function ReviewCommandNotice({flow}:{flow:ReviewCommandFlow}){const{t}=useUiI18n();return <>{flow.notice&&<p role="status" className={styles.notice}>{t(flow.notice,{})}</p>}{flow.unconfirmed&&<button type="button" className="button secondary" disabled={flow.busy} onClick={()=>void flow.retry()}>{t(flow.confirmedRead?'studyReview.reread':'studyReview.retry',{})}</button>}{flow.activeId&&<Link prefetch={false} href={'/learn/review/'+flow.activeId}>{t('studyReview.continue',{})}</Link>}</>}
export function ScheduleSummary({schedule}:{schedule:Schedule|null}){const{t,locale}=useUiI18n();if(!schedule)return <p>{t('studyReview.unscheduled',{})}</p>;return <p><span>{t('studyReview.stage',{stage:schedule.stage})}</span>{' · '}<time dateTime={schedule.dueAt}>{new Intl.DateTimeFormat(locale,{dateStyle:'medium',timeStyle:'short'}).format(new Date(schedule.dueAt))}</time></p>}
export function ReviewSections({sections}:{sections:Section[]}){const{t}=useUiI18n();return <div className={styles.material}>{sections.map(s=><section key={s.kind}><h3>{t(sectionKeys[s.kind],{})}</h3>{s.items.map((source,i)=><SafeMarkdown presentation="study" key={i} source={source} assets={[]}/>)}</section>)}</div>}
export function SelfRating({session,onGrade,busy}:{session:Session;onGrade:(rating:Rating)=>void;busy:boolean}){
 const{t}=useUiI18n();if(session.status!=='compared')return null;
 return <section className={styles.rating} aria-labelledby="review-rating"><h2 id="review-rating">{t('studyReview.selfRating',{})}</h2><p className={styles.hint}>{t('studyReview.selfRatingHint',{})}</p>{session.unanswered&&<p>{t('studyReview.emptyAnswerHint',{})}</p>}<div className={styles.ratingActions}>{(['forgotten','uncertain','familiar'] as const).map(rating=><button type="button" className="button secondary" key={rating} disabled={busy||session.unanswered&&rating!=='forgotten'||!session.available||session.materialChanged} onClick={()=>onGrade(rating)}>{t(ratingKey[rating],{})}</button>)}</div></section>;
}
export function ReadingReview({initial}:{initial:Session}){
 const{t}=useUiI18n(),[session,setSession]=useState<Session|null>(initial),flow=useReviewCommand(initial.actorId,setSession,()=>setSession(null));
 useEffect(()=>{setSession(initial)},[initial]);
 if(!session||!flow.authorized)return <p role="status">{t('studyReview.signedOut',{})}</p>;
 return <ReadingReviewBody session={session} flow={flow}/>;
}
export function ReadingReviewBody({session,flow}:{session:Session;flow:ReviewCommandFlow}){
 const{t}=useUiI18n();
 const active=session.available&&!session.materialChanged&&!flow.contentBlocked;
 return <div className={styles.session}><UiPageTitle messageKey="page.learn.review.id"/><PageHeading title={<LocalizedName english={session.title} chinese={session.titleZh}/>} description={t('studyReview.read',{})} actions={<Link prefetch={false} href="/learn/review">{t('studyReview.back',{})}</Link>}/>
 {!session.available&&<p role="status">{t('studyReview.availableFalse',{})}</p>}{session.materialChanged&&<p role="status">{t('studyReview.changedHint',{})}</p>}
 {active&&session.reference&&<section className={styles.card} aria-label={t('studyReview.reference',{})}><h2>{t('studyReview.reference',{})}</h2><ReviewSections sections={session.reference.sections}/></section>}
 {active&&session.status==='working'&&<button type="button" className="button" disabled={flow.busy||flow.unconfirmed} onClick={()=>void flow.run(`/api/v3/study/review/sessions/${session.sessionId}/compare`,'POST',{expectedSequence:session.sequence,body:'',unanswered:false},session.knowledge)}>{t('studyReview.completeReading',{})}</button>}
 {active&&<SelfRating session={session} busy={flow.busy||flow.unconfirmed} onGrade={rating=>void flow.run(`/api/v3/study/review/sessions/${session.sessionId}/grade`,'POST',{expectedSequence:session.sequence,rating},session.knowledge)}/>}
 {session.status==='completed'&&<section className={styles.card}><h2>{t('studyReview.done',{})}</h2>{session.grade&&<p>{t(ratingKey[session.grade.rating],{})}</p>}<h3>{t('studyReview.thenSchedule',{})}</h3><ScheduleSummary schedule={session.grade?.schedule??null}/><h3>{t('studyReview.currentSchedule',{})}</h3><ScheduleSummary schedule={session.schedule}/><Link prefetch={false} className="button" href="/learn/review">{t('studyReview.next',{})}</Link></section>}
 <ReviewCommandNotice flow={flow}/>
 </div>;
}
