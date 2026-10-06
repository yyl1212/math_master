"use client";
import {UiEnum,formatUiEnum} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";

import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useEffect, useState, useRef } from "react";
import { useRouter } from "next/navigation";
import { contentRequest } from "@/lib/content/client";
import type { PublicationPage, PublicationView, SubmissionPage } from "@/lib/content/types";
import { validContentInput } from "@/lib/content/schemas";
import { TextField } from "./field-controls";
import { DiffPanel } from "./diff-panel";
import { useContentCommand } from "./command-controls";
import {PasswordDialog} from "./command-controls";
import type {PairRef,ReleaseView,TaxonomyResult,PrepareInput as TopicPrepareInput,ActivateInput as TopicActivateInput} from "@/lib/taxonomy/types";
import {prepareInputSchema,activateInputSchema} from "@/lib/taxonomy/schemas";
import styles from "@/styles/content.module.css";
export function PublicationPanel({ initial, selectedID }: {
    initial: PublicationPage;
    selectedID?: string;
}) {
 const {t,locale}=useUiI18n();

    const router = useRouter(), command = useContentCommand(), [page, setPage] = useState(initial), [selected, setSelected] = useState(selectedID ?? initial.items[0]?.id ?? ""), [approved, setApproved] = useState<SubmissionPage | null>(null), [submissionIDs, setIDs] = useState<string[]>([]), [reason, setReason] = useState(""), [stale, setStale] = useState(false);
    const candidate = page.items.find(v => v.id === selected);
    useEffect(() => { let live = true; void contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 100 } }).then(result => { if (!live)
        return; if (result.ok)
        setApproved(result.data);
    else
        command.error(uiError("content",result)); }); return () => { live = false; }; }, []);
    useEffect(() => { if (command.failure?.code === "PUBLICATION_STALE")
        setStale(true); }, [command.failure]);
    async function refresh(offset = page.offset) { const result = await contentRequest<PublicationPage>({ kind: "listPublications", query: { limit: page.limit, offset } }); if (result.ok) {
        setPage(result.data);
        setSelected(current => result.data.items.some(v => v.id === current) ? current : result.data.items[0]?.id ?? "");
        router.refresh();
    }
    else
        command.error(uiError("content",result)); }
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("publication-panel.publication.72e134",{})}/></p><h1><UiText notice={uiMessage("publication-panel.reviewed.publication.snapshots.8acf21",{})}/></h1><p>{page.head ? "Current published head: " + page.head : t("publication-panel.no.content.has.been.published.yet.e8904f",{})}</p><Link prefetch={false} href="/admin/withdrawals"><UiText notice={uiMessage("publication-panel.withdraw.a.fixed.version.2cca43",{})}/></Link><div className={styles.actions}><button className="button secondary" disabled={command.busy} onClick={() => void refresh()}><UiText notice={uiMessage("publication-panel.refresh.current.head.a29f92",{})}/></button></div>{command.controls}<div className={styles.grid}><section className={styles.card}><h2><UiText notice={uiMessage("publication-panel.approved.submissions.86299b",{})}/></h2>{approved === null ? <p><UiText notice={uiMessage("publication-panel.loading.approved.batches.e29b0b",{})}/></p> : approved.items.length === 0 ? <p><UiText notice={uiMessage("publication-panel.no.approved.submissions.fc2830",{})}/></p> : approved.items.map(s => <label className={styles.check} key={s.id}><input type="checkbox" checked={submissionIDs.includes(s.id)} disabled={command.busy || submissionIDs.length >= 20 && !submissionIDs.includes(s.id)} onChange={e => setIDs(ids => e.target.checked ? [...ids, s.id] : ids.filter(v => v !== s.id))}/><span>{s.packageId}<UiText notice={uiMessage("publication-panel.v.e40f81",{})}/>{s.packageVersion} <Link prefetch={false} href={"/review/" + s.id}><UiText notice={uiMessage("publication-panel.review.record.8f17cd",{})}/></Link></span></label>)}{approved && <div className={styles.actions}><button className="button secondary" disabled={command.busy || approved.offset === 0} onClick={async () => { const result = await contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 100, offset: Math.max(0, approved.offset - 100) } }); if (result.ok)
        setApproved(result.data);
    else
        command.error(uiError("content",result)); }}><UiText notice={uiMessage("publication-panel.previous.approved.batches.ee33bf",{})}/></button><button className="button secondary" disabled={command.busy || approved.offset + approved.limit >= approved.total} onClick={async () => { const result = await contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 100, offset: approved.offset + 100 } }); if (result.ok)
        setApproved(result.data);
    else
        command.error(uiError("content",result)); }}><UiText notice={uiMessage("publication-panel.next.approved.batches.570a44",{})}/></button></div>}<TextField label={t("publication-panel.publication.reason.8ae504",{})} value={reason} onChange={setReason} multiline/><button className="button" disabled={command.busy || !validContentInput("prepareRelease", { submissionIds: submissionIDs, expectedHead: page.head, reason })} onClick={() => void command.run({ kind: "prepareRelease" }, { submissionIds: submissionIDs, expectedHead: page.head, reason }, async (data) => { const next = data as PublicationView; await refresh(); setPage(v => ({ ...v, items: [next, ...v.items.filter(p => p.id !== next.id)], total: v.items.some(p => p.id === next.id) ? v.total : v.total + 1 })); setSelected(next.id); setStale(false); command.message(uiMessage("publication-panel.snapshot.prepared.inspect.the.fixed.difference.before.activation.13cb36",{})); })}><UiText notice={uiMessage("publication-panel.prepare.snapshot.93466e",{})}/></button></section><section className={styles.card}><h2><UiText notice={uiMessage("publication-panel.snapshot.history.d81e02",{})}/></h2>{page.items.length > 0 ? <label className={styles.field}><UiText notice={uiMessage("publication-panel.selected.snapshot.b60236",{})}/><select value={selected} disabled={command.busy} onChange={e => { setSelected(e.target.value); setStale(false); command.clearPending(); }}>{page.items.map(p => <option key={p.id} value={p.id}>{p.status} · {p.id}</option>)}</select></label> : <p><UiText notice={uiMessage("publication-panel.no.snapshots.yet.dce32a",{})}/></p>}<div className={styles.actions}><button className="button secondary" disabled={command.busy || page.offset === 0} onClick={() => void refresh(Math.max(0, page.offset - page.limit))}><UiText notice={uiMessage("publication-panel.previous.snapshots.1ead42",{})}/></button><button className="button secondary" disabled={command.busy || page.offset + page.limit >= page.total} onClick={() => void refresh(page.offset + page.limit)}><UiText notice={uiMessage("publication-panel.next.snapshots.28f079",{})}/></button></div>{candidate && <><p><UiText notice={uiMessage("publication-panel.value.catalogue.version.value.9b204b",{v0:formatUiEnum(locale,"content.enum",candidate.status),v1:uiValue(candidate.manifest.catalogueVersion)})}/></p><p className={styles.metadata}><UiText notice={uiMessage("publication-panel.manifest.sha.value.22188b",{v0:uiValue(candidate.manifestSha)})}/></p><DiffPanel diff={candidate.diff}/><details><summary><UiText notice={uiMessage("publication-panel.fixed.manifest.and.review.evidence.8aaddf",{})}/></summary><pre>{JSON.stringify(candidate.manifest, null, 2)}</pre></details>{stale && <p><UiText notice={uiMessage("publication-panel.published.content.changed.prepare.a.new.snapshot.b8bb36",{})}/></p>}<button className="button" disabled={command.busy || candidate.status !== "draft" || stale || candidate.manifest.baseHead !== page.head || !validContentInput("activateRelease", { expectedHead: page.head, expectedManifestSha: candidate.manifestSha, reason })} onClick={() => void command.run({ kind: "activateRelease", id: candidate.id }, { expectedHead: page.head, expectedManifestSha: candidate.manifestSha, reason }, async () => { await refresh(); command.message(uiMessage("publication-panel.snapshot.activated.current.head.refreshed.cd5ead",{})); })}><UiText notice={uiMessage("publication-panel.activate.snapshot.bd0268",{})}/></button></>}</section></div></section>;
}


// 新管理代理在 A7 接线；此组件只消费准确 PairRef 与受保护命令回调。
export function TopicPublicationPanel({pair,initial,onPrepare,onActivate,onRefresh}:{pair:PairRef;initial?:ReleaseView;onPrepare:(input:TopicPrepareInput,key:string)=>Promise<TaxonomyResult<ReleaseView>>;onActivate:(id:string,input:TopicActivateInput,key:string)=>Promise<TaxonomyResult<ReleaseView>>;onRefresh?:()=>Promise<PairRef>}){
 const {t}=useUiI18n(),[current,setCurrent]=useState(pair),[version,setVersion]=useState(pair.taxonomyVersionId),[ids,setIDs]=useState(""),[reason,setReason]=useState(""),[candidate,setCandidate]=useState(initial),[busy,setBusy]=useState(false),[notice,setNotice]=useState<"prepared"|"published"|"failed"|"stale"|null>(null),[verify,setVerify]=useState(false),[stale,setStale]=useState(false);
 const lock=useRef(false),trigger=useRef<HTMLElement|null>(null),pending=useRef<{signature:string;key:string}|null>(null);
 const expected={...current,taxonomyVersionId:version},prepare={submissionIds:ids.split(",").map(v=>v.trim()).filter(Boolean),expectedPair:expected,reason};
 const activation=candidate?{expectedPair:candidate.pair,manifestSHA:candidate.manifestSHA,reason}:null;
 async function execute(kind:"prepare"|"activate"){
  if(lock.current)return;const input=kind==="prepare"?prepare:activation;if(!input)return;
  if(!(kind==="prepare"?prepareInputSchema:activateInputSchema).safeParse(input).success)return;
  const signature=JSON.stringify({kind,id:kind==="activate"?candidate?.id:"",input});if(pending.current?.signature!==signature)pending.current={signature,key:crypto.randomUUID()};
  lock.current=true;setBusy(true);setNotice(null);trigger.current=document.activeElement as HTMLElement;
  try{const result=kind==="prepare"?await onPrepare(input as TopicPrepareInput,pending.current.key):await onActivate(candidate!.id,input as TopicActivateInput,pending.current.key);
   if(result.ok){setCandidate(result.data);setStale(false);setNotice(kind==="prepare"?"prepared":"published");pending.current=null;}
   else if(result.code==="REAUTH_REQUIRED"){setVerify(true);}else if(result.code==="PUBLICATION_STALE"){setStale(true);setNotice("stale");}else setNotice("failed");
  }catch{setNotice("failed")}finally{lock.current=false;setBusy(false)}
 }
 function close(){setVerify(false);setTimeout(()=>trigger.current?.focus(),0)}
 return <section className={styles.workbench}><h1>{t("topic.publication.title",{})}</h1><p className={styles.metadata}>{current.knowledgeHead??"—"} · {current.taxonomyHead??"—"}</p><fieldset disabled={busy}><TextField label={t("topic.publication.version",{})} value={version} onChange={setVersion}/><TextField label={t("topic.publication.submissions",{})} value={ids} onChange={setIDs}/><TextField label={t("publication-panel.publication.reason.8ae504",{})} value={reason} onChange={setReason} multiline/><button type="button" className="button" disabled={!prepareInputSchema.safeParse(prepare).success} onClick={()=>void execute("prepare")}>{t("topic.publication.prepare",{})}</button>{onRefresh&&<button type="button" className="button secondary" onClick={async()=>{if(lock.current)return;lock.current=true;setBusy(true);try{const next=await onRefresh();setCurrent(next);setVersion(next.taxonomyVersionId);setCandidate(undefined);setStale(false);pending.current=null;setNotice(null)}catch{setNotice("failed")}finally{lock.current=false;setBusy(false)}}}>{t("topic.publication.refresh",{})}</button>}</fieldset>{candidate&&<section className={styles.card}><p className={styles.metadata}>{candidate.id} · {candidate.manifestSHA}</p><DiffPanel diff={{added:0,replaced:0,removed:0,changes:[]}} topics={candidate.diff}/><button type="button" className="button" disabled={busy||stale||candidate.status!=="draft"||!activation||!activateInputSchema.safeParse(activation).success} onClick={()=>void execute("activate")}>{t("topic.publication.activate",{})}</button></section>}{notice&&<p role={notice==="failed"||notice==="stale"?"alert":"status"}>{t(notice==="prepared"?"topic.publication.prepared":notice==="published"?"topic.publication.published":notice==="stale"?"topic.publication.stale":"topic.publication.failed",{})}</p>}{verify&&<PasswordDialog close={close} verified={close}/>}</section>
}
