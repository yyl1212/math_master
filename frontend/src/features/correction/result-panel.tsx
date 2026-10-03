"use client";
import Link from 'next/link';
import { AttemptAsset } from '@/features/learning/attempt-asset';
import { SafeMarkdown } from '@/features/reading/safe-markdown';
import { useState, useContext, useEffect } from 'react';
import { correctionClient } from '@/lib/correction/client';
import { CorrectionRequestError, type ResultMetadataView, type ResultDetail } from '@/lib/correction/types';
import { CorrectionAccountContext, useCorrectionRead } from './correction-account';
import { CorrectionStatus, CorrectionState } from './status';
import styles from '@/styles/content.module.css';
export function ResultPanel({ initial }: {
    initial: ResultMetadataView;
}) {
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
    const result = metadata.result, corrected = ['corrected_passed', 'corrected_failed'].includes(result.status), original = result.evidence.kind === 'assessment' ? '/assessments/' + result.evidence.id + '/result' : result.evidence.kind === 'practice' ? '/practice/' + result.evidence.id : null;
    return <main className={styles.workbench}><h1>Learning correction</h1><section className={styles.card}><h2><CorrectionStatus status={result.status}/></h2>{result.score !== null && <p>Score: {result.score}/5 · {result.passed ? 'Passed' : 'Not passed'}</p>}<p>Current validity: {result.validity}</p>{result.validity === 'restricted' && <p>This correction currently cannot provide a learning qualification.</p>}<p>{result.reason.replaceAll('_', ' ')}</p>{result.status === 'awaiting_review' && <p>Your original answers are preserved while the basis is independently reviewed.</p>}{result.status === 'retake_required' && <p>Review the current knowledge and take a new assessment when it becomes available.</p>}{result.status === 'review_material' && <p>Return to the updated material before continuing.</p>}{result.knowledge && <Link prefetch={false} href={'/knowledge/' + result.knowledge.id}>Review knowledge</Link>}<div className={styles.actions}>{original && <Link prefetch={false} href={original}>Original result</Link>}{result.parentResultId && <Link prefetch={false} href={'/corrections/' + result.parentResultId}>Previous correction</Link>}<button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void refresh(false)}>Reload correction status</button>{corrected && <button type="button" className="button secondary" disabled={loading || account?.checking || !!detail} onClick={() => void refresh(true)}>Review corrected answers</button>}</div>{error && <CorrectionState error={error} onRetry={() => void refresh(false)}/>}</section>
 {detail && <section className={styles.card}><h2>Corrected result</h2><p>The same original answers were checked against the independently approved basis.</p>{detail.planReason && <p>{detail.planReason}</p>}<ol>{detail.items.map(item => <li className={styles.row} key={item.position}><h3>Question {item.position}</h3><SafeMarkdown source={item.prompt} assets={[]}/>{item.choices.length > 0 && <ul>{item.choices.map(c => <li key={c.id}>{c.id}: {c.text}</li>)}</ul>}<p>Original answer: {item.answer.kind === 'numeric' ? item.answer.raw : item.answer.kind === 'choice' ? item.answer.choiceId : 'Skipped'}</p><p>{item.correct ? 'Correct' : 'Not correct'}</p><SafeMarkdown source={item.explanation} assets={[]}/><p className={styles.metadata}>Original {item.original.id} v{item.original.version} → Effective {item.effective.id} v{item.effective.version}</p>{item.assets.map(asset => <AttemptAsset key={asset.id} attemptId={result.evidence.id} asset={asset}/>)}</li>)}</ol></section>}
 </main>;
}
