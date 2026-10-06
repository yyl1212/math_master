"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import {useState} from 'react';
import {correctionUUID} from '@/lib/correction/schemas';
import type {CorrectedItem} from '@/lib/correction/types';
import styles from '@/styles/assessment.module.css';
export function CorrectionAsset({resultId,asset}:{resultId:string;asset:CorrectedItem['assets'][number]}){
 const {t}=useUiI18n();

 const[failed,setFailed]=useState(false);
 if(failed||!correctionUUID.test(resultId)||!/^[0-9a-f]{64}$/.test(asset.sha256))return <p role="status"><UiText notice={uiMessage("safe-markdown.illustration.is.not.available.ef0518",{})}/></p>;
 return <figure className={styles.figure}><img src={`/api/v1/corrections/results/${resultId}/assets/${asset.sha256}`} alt={t("attempt-asset.question.illustration.b088b7",{})} onError={()=>setFailed(true)}/></figure>;
}
