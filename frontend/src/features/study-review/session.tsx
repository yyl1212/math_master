'use client';
import {useEffect,useRef,useState} from 'react';
import Link from 'next/link';
import {useUiI18n} from '@/lib/i18n/provider';
import {LocalizedName} from '@/lib/i18n/localized-name';
import {PageHeading} from '@/components/page-heading';
import {UiPageTitle} from '@/components/ui-page-title';
import {NoteLeaveGuard} from '@/features/notes/note-leave-guard';
import type {Session} from '@/lib/study-review/types';
import {isReviewAnswerBlank} from '@/lib/study-review/answer';
import {ReadingReviewBody,ReviewSections,SelfRating,ReviewCommandNotice,ScheduleSummary} from './self-rating';
import {RecallEditor,answerOverLimit} from './recall-editor';
import {Comparison} from './comparison';
import {useReviewCommand} from './use-review-command';
import styles from '@/styles/study-review.module.css';
const gradeKeys={forgotten:'studyReview.forgotten',uncertain:'studyReview.uncertainRating',familiar:'studyReview.familiar'} as const;
export function SessionPanel({initial}:{initial:Session}){
 const{t}=useUiI18n(),[session,setSession]=useState<Session|null>(initial),[draft,setDraft]=useState(initial.answer??''),[saved,setSaved]=useState(initial.answer??''),[draftSaved,setDraftSaved]=useState(false);
 const draftRef=useRef(draft),savedRef=useRef(saved);draftRef.current=draft;savedRef.current=saved;
 const update=(next:Session)=>{setSession(next);if(next.status==='working'){const body=next.answer??'';if(draftRef.current===savedRef.current)setDraft(body);setSaved(body)}};
 const flow=useReviewCommand(initial.actorId,update,()=>{setSession(null);setDraft('');setSaved('')});
 useEffect(()=>{setSession(initial);if(draftRef.current===savedRef.current)setDraft(initial.answer??'');setSaved(initial.answer??'')},[initial]);
 if(!session||!flow.authorized)return <p role="status">{t('studyReview.signedOut',{})}</p>;
 const active=session.available&&!session.materialChanged&&!flow.contentBlocked,unfinished=session.status==='working'||session.status==='compared',dirty=session.mode==='recall'&&session.status==='working'&&draft!==saved;
 const locked=flow.busy||flow.unconfirmed,over=answerOverLimit(draft),blank=isReviewAnswerBlank(draft);
 const base='/api/v3/study/review/sessions/'+session.sessionId;
 const runDraft=async()=>{setDraftSaved(false);const current=await flow.run(base+'/draft','PUT',{expectedSequence:session.sequence,expectedDraftRevision:session.draftRevision,body:draft},session.knowledge);if(current?.status==='working')setDraftSaved(true)};
 const abandon=()=>{if(dirty&&!window.confirm(t('studyReview.restartHint',{})))return;void flow.run(base+'/abandon','POST',{expectedSequence:session.sequence},session.knowledge)};
 const refresh=()=>flow.refresh(session.sessionId);
 const projected={...session,prompt:active?session.prompt:[],reference:active?session.reference:null};
 return <div className={styles.session}><NoteLeaveGuard dirty={dirty} actorId={session.actorId} onDiscard={()=>{setDraft('');setSaved('')}}/>
 {session.mode==='read'?<ReadingReviewBody session={projected} flow={flow}/>:<><UiPageTitle messageKey="page.learn.review.id"/><PageHeading title={<LocalizedName english={session.title} chinese={session.titleZh}/>} description={t(session.target==='proof'?'studyReview.recall.proof':'studyReview.recall.statement',{})} actions={<Link prefetch={false} href="/learn/review">{t('studyReview.back',{})}</Link>}/>
 {!session.available&&<p role="status">{t('studyReview.availableFalse',{})}</p>}{session.materialChanged&&<p role="status">{t('studyReview.changedHint',{})}</p>}
 {session.status==='working'&&<><p>{t(session.target==='proof'?'studyReview.task.proof':'studyReview.task.statement',{})}</p>{active&&session.target==='proof'&&<section className={styles.card} aria-label={t('studyReview.prompt',{})}><h2>{t('studyReview.prompt',{})}</h2><ReviewSections sections={session.prompt}/></section>}
 <RecallEditor value={draft} onChange={value=>{setDraft(value);setDraftSaved(false)}} dirty={dirty} disabled={!active||locked}/>{draftSaved&&<p role="status">{t('studyReview.draftSaved',{})}</p>}<div className={styles.actions}><button type="button" className="button secondary" disabled={!active||locked||over} onClick={()=>void runDraft()}>{t('studyReview.saveDraft',{})}</button><button type="button" className="button" disabled={!active||locked||over||blank} onClick={()=>void flow.run(base+'/compare','POST',{expectedSequence:session.sequence,body:draft,unanswered:false},session.knowledge)}>{t('studyReview.compare',{})}</button>{blank&&<button type="button" className="button secondary" disabled={!active||locked||over} onClick={()=>void flow.run(base+'/compare','POST',{expectedSequence:session.sequence,body:draft,unanswered:true},session.knowledge)}>{t('studyReview.unanswered',{})}</button>}</div></>}
 {session.status!=='working'&&<Comparison session={projected}/>} {active&&<SelfRating session={session} busy={locked} onGrade={rating=>void flow.run(base+'/grade','POST',{expectedSequence:session.sequence,rating},session.knowledge)}/>}
 {session.status==='completed'&&<section className={styles.card}><h2>{t('studyReview.done',{})}</h2>{session.grade&&<p>{t(gradeKeys[session.grade.rating],{})}</p>}<h3>{t('studyReview.thenSchedule',{})}</h3><ScheduleSummary schedule={session.grade?.schedule??null}/><h3>{t('studyReview.currentSchedule',{})}</h3><ScheduleSummary schedule={session.schedule}/><Link prefetch={false} className="button" href="/learn/review">{t('studyReview.next',{})}</Link></section>}
 <ReviewCommandNotice flow={flow}/></>}
 <div className={styles.actions}>{unfinished&&<button type="button" className="button secondary" disabled={locked} onClick={abandon}>{t('studyReview.abandon',{})}</button>}<button type="button" className="button secondary" disabled={locked} onClick={()=>void refresh()}>{t('studyReview.refresh',{})}</button>{session.status==='abandoned'&&<Link prefetch={false} className="button" href="/learn/review">{t('studyReview.restart',{})}</Link>}</div>{(!active||session.status==='abandoned')&&<p className={styles.hint}>{t('studyReview.restartHint',{})}</p>}
 </div>;
}
