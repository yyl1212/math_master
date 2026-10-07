"use client";
import {ContentReminders} from "./content-reminders";
import Link from "next/link";
import {useUiI18n} from "@/lib/i18n/provider";
import {UiText} from "@/lib/i18n/ui-text";
import {uiError} from "@/lib/i18n/errors";
import type {Overview as OverviewData,StudyResult} from "@/lib/study/types";
export function Overview({data}:{data:OverviewData}){const{t}=useUiI18n();return <section className="panel"><h2>{t("study.overview",{})}</h2><div className="metrics"><p>{t("study.total",{})} <strong>{data.total}</strong></p><p>{t("study.state.completed",{})} <strong>{data.completed}</strong></p><p>{t("study.state.learning",{})} <strong>{data.learning}</strong></p><p>{t("study.state.reviewing",{})} <strong>{data.reviewing}</strong></p></div><p>{t("study.progress.rule",{})}</p>{data.unavailable>0&&<p>{t("study.withdrawn",{})} {data.unavailable}</p>}{data.unclassified>0&&<p>{t("study.unclassified",{})} {data.unclassified}</p>}<ContentReminders items={data.reminders}/><Link prefetch={false} href="/learn?mode=review">{t("study.review.search",{})}</Link></section>}
export function StudyNotice({result}:{result:Extract<StudyResult<never>,{ok:false}>}){return <section className="content-state" role="alert"><UiText notice={uiError("study",result)}/></section>}
