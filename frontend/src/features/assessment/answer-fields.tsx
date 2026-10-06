"use client";
import {useUiI18n} from "@/lib/i18n/provider";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import type {SafeQuestion,Answer} from "@/lib/learning/types";
import {SafeMarkdown} from "@/features/reading/safe-markdown";
import {AttemptAsset} from "@/features/learning/attempt-asset";
import styles from "@/styles/assessment.module.css";
export const emptyAnswer=(q:SafeQuestion):Answer=>q.type==="numeric"?{kind:"numeric",raw:""}:{kind:"choice",choiceId:""};
export function AnswerFields({attemptId,question:q,value,onChange,disabled}:{attemptId:string;question:SafeQuestion;value:Answer;onChange:(answer:Answer)=>void;disabled:boolean}){const {t}=useUiI18n();const name="answer-"+q.instance.id;return <fieldset className={styles.question} disabled={disabled}><legend><UiText notice={uiMessage("result-panel.question.value.016509",{v0:uiValue(q.position)})}/></legend><SafeMarkdown source={q.prompt} assets={[]}/>{q.assets.map(a=><AttemptAsset key={a.sha256} attemptId={attemptId} asset={a}/>)}{q.type==="numeric"?<><label htmlFor={name}><UiText notice={uiMessage("answer-fields.answer.for.question.value.273662",{v0:uiValue(q.position)})}/></label><input id={name} name={name} type="text" autoComplete="off" inputMode="text" value={value.kind==="numeric"?value.raw:""} onChange={e=>onChange({kind:"numeric",raw:e.target.value})}/><p className={styles.hint}><UiText notice={uiMessage("answer-fields.value.at.most.128.characters.your.spelling.is.preserved.4b1382",{v0:t(q.answerFormat==="percentage"?"learning.answer.percentage":"learning.answer.number",{})})}/></p></>:<div role="group" aria-label={t("answer-fields.answer.for.question.value.273662",{v0:q.position})}>{q.choices.map(c=><label key={c.id}><input type="radio" name={name} value={c.id} checked={value.kind==="choice"&&value.choiceId===c.id} onChange={()=>onChange({kind:"choice",choiceId:c.id})}/>{c.text}</label>)}</div>}</fieldset>}
