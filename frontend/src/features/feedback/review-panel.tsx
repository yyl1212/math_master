"use client";
import Link from 'next/link';
import {useContext,useState} from 'react';
import {useUiI18n} from '@/lib/i18n/provider';
import {UiEnum} from '@/lib/i18n/enums';
import {FeedbackAccountContext} from './feedback-account';
import {useFeedbackCommand} from './pending-command';
import {FeedbackCommandStatus} from './status';
import {feedbackInputSchemas} from '@/lib/feedback/schemas';
import type {Metadata,Receipt,Resolution,TransitionInput} from '@/lib/feedback/types';
import styles from '@/styles/content.module.css';
type SimpleBasis='clarified'|'service_fixed'|'duplicate'|'not_reproducible'|'out_of_scope'|'suggestion_recorded';
const edges={new:['new','processing','waiting_details','closed'],processing:['processing','waiting_details','resolved','closed'],waiting_details:['waiting_details','processing','resolved','closed'],resolved:['resolved','processing'],closed:['closed','processing']} as const;
export function ReviewPanel({ticket,onSuccess}:{ticket:Metadata;onSuccess:(receipt:Receipt,signal:AbortSignal)=>Promise<Metadata|null>}){
 const{t}=useUiI18n(),account=useContext(FeedbackAccountContext);const[status,setStatus]=useState<Metadata['status']>(ticket.status),[message,setMessage]=useState(''),[basis,setBasis]=useState<SimpleBasis>('clarified'),[duplicate,setDuplicate]=useState('');
 const command=useFeedbackCommand(account?.actorId??'',async(receipt,signal)=>{const latest=await onSuccess(receipt,signal);if(latest){setMessage('');setStatus(latest.status);setBasis(latest.status==='closed'?'suggestion_recorded':'clarified');setDuplicate('')}});
 if(!ticket.canHandle)return null;
 const allowed=edges[ticket.status] as readonly Metadata['status'][],terminal=status!==ticket.status&&(status==='resolved'||status==='closed');
 const bases:SimpleBasis[]=status==='closed'?['duplicate','not_reproducible','out_of_scope','suggestion_recorded']:ticket.target.kind==='site'?['clarified','service_fixed']:['clarified'];
 const resolution:Resolution|null=terminal?basis==='duplicate'?{kind:'duplicate',withdrawal:null,replacement:null,duplicateOf:duplicate}:{kind:basis,withdrawal:null,replacement:null,duplicateOf:null}:null;
 const input:TransitionInput={expectedSequence:ticket.sequence,status,message,resolution},locked=command.busy||!!command.pending;
 return <section className={styles.card}><h2>{t('streamline.feedback.manage',{})}</h2>
  {ticket.target.kind==='managed-knowledge'&&ticket.target.managedRef&&<p><Link prefetch={false} className="button secondary" href={'/admin/knowledge/'+ticket.target.managedRef.id}>{t('streamline.feedback.edit',{})}</Link></p>}
  <form className="ui-form" onSubmit={event=>{event.preventDefault();void command.run({route:`/api/v1/feedback/review/tickets/${ticket.id}/transition`,input})}}><fieldset disabled={locked}>
   <label className={styles.field}>{t('review-panel.report.status.22a134',{})}<select aria-label={t('review-panel.report.status.22a134',{})} value={status} onChange={event=>{const next=event.target.value as Metadata['status'];setStatus(next);setBasis(next==='closed'?'suggestion_recorded':'clarified')}}>{allowed.map(value=><option key={value} value={value}><UiEnum group="feedback.status" value={value}/></option>)}</select></label>
   <label className={styles.field}>{t('review-panel.review.reply.6bf2f2',{})}<textarea aria-label={t('review-panel.review.reply.6bf2f2',{})} required rows={5} value={message} onChange={event=>setMessage(event.target.value)}/></label>
   {terminal&&<label className={styles.field}>{t('review-panel.resolution.basis.47ef7d',{})}<select aria-label={t('review-panel.resolution.basis.47ef7d',{})} value={basis} onChange={event=>setBasis(event.target.value as SimpleBasis)}>{bases.map(value=><option key={value} value={value}><UiEnum group="feedback.basis" value={value}/></option>)}</select></label>}
   {terminal&&basis==='duplicate'&&<label className={styles.field}>{t('review-panel.duplicate.report.id.2eed31',{})}<input value={duplicate} onChange={event=>setDuplicate(event.target.value)}/></label>}
  </fieldset><button className="button" disabled={locked||!allowed.includes(status)||!feedbackInputSchemas.transition.safeParse(input).success}>{t('review-panel.save.handling.result.fe5bce',{})}</button><FeedbackCommandStatus command={command}/></form>
 </section>;
}
