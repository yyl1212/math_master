"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from 'next/link';
import { CorrectionAsset } from './correction-asset';
import { SafeMarkdown } from '@/features/reading/safe-markdown';
import { useState, useContext, useEffect } from 'react';
import { correctionClient } from '@/lib/correction/client';
import { CorrectionRequestError, type ResultMetadataView, type ResultDetail } from '@/lib/correction/types';
import { CorrectionAccountContext, useCorrectionRead } from './correction-account';
import { CorrectionStatus, CorrectionState } from './status';
import styles from '@/styles/content.module.css';
export function ResultPanel({ initial,topicMode=false }: {
 topicMode?:boolean;
    initial: ResultMetadataView;
}) {
 const {t}=useUiI18n();

    const account = useContext(CorrectionAccountContext), read = useCorrectionRead(), [metadata, setMetadata] = useState(initial), [detail, setDetail] = useState<ResultDetail | null>(null), [loading, setLoading] = useState(false), [error, setError] = useState<CorrectionRequestError | null>(null);
    useEffect(() => { setMetadata(initial); setDetail(null); }, [initial]);
    async function refresh(protectedBody: boolean) {
        if (loading)
            return;
        setLoading(true);
        setError(null);
        try {
            if (protectedBody) {
                const out = await read(a => correctionClient.readOwnDetail(metadata.result.id, a));
                if (out) {
                    setDetail(out.data);
                    setMetadata({ result: out.data.result, items: [], planReason: null });
                }
            }
            else {
                const out = await read(a => correctionClient.readOwn(metadata.result.id, a));
                if (out) {
                    setMetadata(out.data);
                    setDetail(null);
                }
            }
        }
        catch (e) {
            setError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError());
        }
        finally {
            setLoading(false);
        }
    }
    const result = metadata.result, corrected = ['corrected_passed', 'corrected_failed'].includes(result.status), original = topicMode&&(result.evidence.kind==='assessment'||result.evidence.kind==='practice')?'/learning-history?archive=legacy&module='+result.evidence.kind+'&attempt='+result.evidence.id:result.evidence.kind === 'assessment' ? '/assessments/' + result.evidence.id + '/result' : result.evidence.kind === 'practice' ? '/practice/' + result.evidence.id : null;
    return <main className={styles.workbench}><h1><UiText notice={uiMessage("result-panel.learning.correction.c9d5ff",{})}/></h1><section className={styles.card}><h2><CorrectionStatus status={result.status}/></h2>{!topicMode&&result.score !== null && <p><UiText notice={uiMessage("result-panel.score.value.5.value.68a83e",{v0:uiValue(result.score),v1:uiValue(result.passed ? t("result-panel.passed.436fe7",{}) : t("audit.notPassed",{}))})}/></p>}<p><UiText notice={uiMessage("result-panel.current.validity.value.a14551",{v0:uiValue(result.validity)})}/></p>{!topicMode&&result.validity === 'restricted' && <p><UiText notice={uiMessage("result-panel.this.correction.currently.cannot.provide.a.learning.qualification.8712ec",{})}/></p>}<p>{result.reason.replaceAll('_', ' ')}</p>{result.status === 'awaiting_review' && <p><UiText notice={uiMessage("result-panel.your.original.answers.are.preserved.while.the.basis.is.independen.dfaf01",{})}/></p>}{!topicMode&&result.status === 'retake_required' && <p><UiText notice={uiMessage("result-panel.review.the.current.knowledge.and.take.a.new.assessment.when.it.be.c48272",{})}/></p>}{result.status === 'review_material' && <p><UiText notice={uiMessage("result-panel.return.to.the.updated.material.before.continuing.57ef79",{})}/></p>}{result.knowledge && <Link prefetch={false} href={'/knowledge/' + result.knowledge.id}><UiText notice={uiMessage("result-panel.review.knowledge.4b5f8a",{})}/></Link>}<div className={styles.actions}>{original && <Link prefetch={false} href={original}><UiText notice={uiMessage("result-panel.original.result.34bd06",{})}/></Link>}{result.parentResultId && <Link prefetch={false} href={'/corrections/' + result.parentResultId}><UiText notice={uiMessage("result-panel.previous.correction.c0de64",{})}/></Link>}<button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void refresh(false)}><UiText notice={uiMessage("result-panel.reload.correction.status.e893ee",{})}/></button>{corrected && <button type="button" className="button secondary" disabled={loading || account?.checking || !!detail} onClick={() => void refresh(true)}><UiText notice={uiMessage("result-panel.review.corrected.answers.83394a",{})}/></button>}</div>{error && <CorrectionState error={error} onRetry={() => void refresh(false)}/>}</section>
 {detail && <section className={styles.card}><h2><UiText notice={uiMessage("result-panel.corrected.result.def77c",{})}/></h2><p><UiText notice={uiMessage("result-panel.the.same.original.answers.were.checked.against.the.independently..edd1c9",{})}/></p>{detail.planReason && <p>{detail.planReason}</p>}<ol>{detail.items.map(item => <li className={styles.row} key={item.position}><h3><UiText notice={uiMessage("result-panel.question.value.016509",{v0:uiValue(item.position)})}/></h3><SafeMarkdown source={item.prompt} assets={[]}/>{item.choices.length > 0 && <ul>{item.choices.map(c => <li key={c.id}>{c.id}: {c.text}</li>)}</ul>}<p><UiText notice={uiMessage("result-panel.original.answer.value.80de39",{v0:uiValue(item.answer.kind === 'numeric' ? item.answer.raw : item.answer.kind === 'choice' ? item.answer.choiceId : t("audit.skipped",{}))})}/></p><p>{item.correct ? t("result-panel.correct.aca01a",{}) : t("result-panel.not.correct.f9cade",{})}</p><SafeMarkdown source={item.explanation} assets={[]}/><p className={styles.metadata}><UiText notice={uiMessage("result-panel.original.value.vvalue.effective.value.vvalue.28ac10",{v0:uiValue(item.original.id),v1:uiValue(item.original.version),v2:uiValue(item.effective.id),v3:uiValue(item.effective.version)})}/></p>{item.assets.map(asset => <CorrectionAsset key={result.id+":"+asset.id} resultId={result.id} asset={asset}/>)}</li>)}</ol></section>}
 </main>;
}
