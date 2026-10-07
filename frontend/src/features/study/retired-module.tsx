"use client";
import Link from "next/link";
import {useUiI18n} from "@/lib/i18n/provider";
export function RetiredModule({archiveHref="/learning-history?archive=legacy"}:{archiveHref?:string}){const{t}=useUiI18n();return <section className="content-state"><h1>{t("study.retired.title",{})}</h1><p>{t("study.retired.body",{})}</p><div className="actions"><Link prefetch={false} href="/learn">{t("study.mode.learn",{})}</Link><Link prefetch={false} href="/knowledge">{t("nav.knowledgeMap",{})}</Link><Link prefetch={false} href={archiveHref}>{t("study.archive.title",{})}</Link></div></section>}
