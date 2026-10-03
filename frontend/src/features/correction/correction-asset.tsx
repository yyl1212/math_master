"use client";
import {useState} from 'react';
import {correctionUUID} from '@/lib/correction/schemas';
import type {CorrectedItem} from '@/lib/correction/types';
import styles from '@/styles/content.module.css';
export function CorrectionAsset({resultId,asset}:{resultId:string;asset:CorrectedItem['assets'][number]}){
 const[failed,setFailed]=useState(false);
 if(failed||!correctionUUID.test(resultId)||!/^[0-9a-f]{64}$/.test(asset.sha256))return <p role="status">Illustration is not available.</p>;
 return <figure className={styles.figure}><img src={`/api/v1/corrections/results/${resultId}/assets/${asset.sha256}`} alt="Question illustration" onError={()=>setFailed(true)}/></figure>;
}
