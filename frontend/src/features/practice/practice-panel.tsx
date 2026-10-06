"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import {ReportLink} from "@/features/feedback/report-link";
import Link from "next/link";
import {useState} from "react";
import {useRouter} from "next/navigation";
import type {PracticeView} from "@/lib/learning/types";
import {useLearningCommand} from "@/features/learning/pending-command";
import {CommandFeedback} from "@/features/learning/learning-status";
import {AnswerFields,emptyAnswer} from "@/features/assessment/answer-fields";
import {ResultItemPanel} from "@/features/assessment/result-panel";
import styles from "@/styles/assessment.module.css";
export function PracticePanel({view}:{view:PracticeView}){
 const {t}=useUiI18n();
const router=useRouter(),[current,setCurrent]=useState(view),[value,setValue]=useState(()=>emptyAnswer(view.question));const command=useLearningCommand<PracticeView>(r=>{setCurrent(r);router.refresh()});const active=current.summary.state==="active";return <section className={styles.panel}><h1><UiText notice={uiMessage("practice-panel.single.question.practice.7f9c5a",{})}/></h1><p><UiText notice={uiMessage("practice-panel.practice.supports.understanding.it.does.not.grant.assessment.qual.9691a0",{})}/></p><p><UiText notice={uiMessage("practice-panel.expires.at.2e4e2e",{})}/><time dateTime={current.summary.expiresAt}>{new Date(current.summary.expiresAt).toISOString().replace("T"," ").replace(".000Z"," UTC")}</time></p>{active?<form onSubmit={e=>{e.preventDefault();void command.run({kind:"answerPractice",id:current.summary.id},value)}}><AnswerFields attemptId={current.summary.id} question={current.question} value={value} disabled={command.busy} onChange={a=>{command.clear();setValue(a)}}/><div className={styles.actions}><button className="button" disabled={command.busy||!!command.pending}><UiText notice={uiMessage("practice-panel.submit.answer.b896ad",{})}/></button><button className="button secondary" type="button" disabled={command.busy||!!command.pending} onClick={()=>void command.run({kind:"revealPractice",id:current.summary.id},{})}><UiText notice={uiMessage("practice-panel.reveal.answer.848e09",{})}/></button><button className="button secondary" type="button" disabled={command.busy} onClick={()=>void command.run({kind:"abandonPractice",id:current.summary.id},{})}><UiText notice={uiMessage("practice-panel.abandon.practice.8b007f",{})}/></button></div><p><UiText notice={uiMessage("practice-panel.reveal.answer.ends.this.practice.immediately.c91381",{})}/></p></form>:<><p>{current.summary.state==="revealed"?t("practice-panel.practice.ended.answer.revealed.3e12d9",{}):"Practice "+current.summary.state+"."}</p>{current.result&&<ResultItemPanel item={current.result.item} attemptId={current.summary.id} sourceKind="practice"/>}<Link prefetch={false} href={"/knowledge/"+current.summary.knowledge.id}><UiText notice={uiMessage("practice-panel.review.the.lesson.or.start.another.practice.70e26d",{})}/></Link></>}<p><ReportLink source={{kind:'practice',id:current.summary.id}}/></p>{current.question.assets.map(a=><p key={a.id}><UiText notice={uiMessage("page.illustration.0ffea7",{})}/>{a.id} <ReportLink source={{kind:'practice',id:current.summary.id,partKind:'asset',partId:a.id}}/></p>)}<CommandFeedback command={command}/></section>}
