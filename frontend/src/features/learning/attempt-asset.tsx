"use client";
import {useState} from "react";
import type {AssetRef} from "@/lib/learning/types";
import {learningUUID,learningSHA} from "@/lib/learning/schemas";
import styles from "@/styles/assessment.module.css";
export function AttemptAsset({attemptId,asset}:{attemptId:string;asset:AssetRef}){const[failed,setFailed]=useState(false);if(failed||!learningUUID.test(attemptId)||!learningSHA.test(asset.sha256))return <p role="status">Illustration is not available.</p>;return <figure className={styles.figure}><img src={`/api/v1/learning/assets/${attemptId}/${asset.sha256}`} alt="Question illustration" onError={()=>setFailed(true)}/></figure>}
