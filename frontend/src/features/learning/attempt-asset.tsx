"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import {useState} from "react";
import type {AssetRef} from "@/lib/learning/types";
import {learningUUID,learningSHA} from "@/lib/learning/schemas";
import styles from "@/styles/assessment.module.css";
export function AttemptAsset({attemptId,asset}:{attemptId:string;asset:AssetRef}){
 const {t}=useUiI18n();
const[failed,setFailed]=useState(false);if(failed||!learningUUID.test(attemptId)||!learningSHA.test(asset.sha256))return <p role="status"><UiText notice={uiMessage("safe-markdown.illustration.is.not.available.ef0518",{})}/></p>;return <figure className={styles.figure}><img src={`/api/v1/learning/assets/${attemptId}/${asset.sha256}`} alt={t("attempt-asset.question.illustration.b088b7",{})} onError={()=>setFailed(true)}/></figure>}
