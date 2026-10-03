"use client";
import Link from 'next/link';
import { useContext, useState, useEffect } from 'react';
import { correctionClient } from '@/lib/correction/client';
import { caseInputSchema } from '@/lib/correction/schemas';
import { CorrectionRequestError, type Page, type CaseMetadata } from '@/lib/correction/types';
import { CorrectionAccountContext, useCorrectionRead } from './correction-account';
import { useCorrectionCommand } from './pending-command';
import { CorrectionState, CorrectionCommandStatus } from './status';
import styles from '@/styles/content.module.css';
export function CaseList({ initial }: {
    initial: Page<CaseMetadata>;
}) {
    const account = useContext(CorrectionAccountContext), read = useCorrectionRead(), [page, setPage] = useState(initial), [error, setError] = useState<CorrectionRequestError | null>(null), [loading, setLoading] = useState(false), [created, setCreated] = useState<CaseMetadata | null>(null);
    const [kind, setKind] = useState<'withdrawal' | 'grading_rule'>(account?.roles.includes('admin') ? 'grading_rule' : 'withdrawal'), [space, setSpace] = useState<'content' | 'question'>('question'), [withdrawalId, setWithdrawalId] = useState(''), [scope, setScope] = useState<'all' | 'knowledge'>('all'), [knowledgeId, setKnowledgeId] = useState(''), [version, setVersion] = useState(1), [sha, setSha] = useState('');
    useEffect(() => { setPage(initial); }, [initial]);
    const command = useCorrectionCommand(account?.actorId ?? '', async (receipt, signal) => {
        if (!receipt.case)
            throw new CorrectionRequestError();
        const latest = await read(a => correctionClient.readCase(receipt.case!.id, a), signal);
        if (!latest || signal.aborted)
            throw new CorrectionRequestError();
        setCreated(latest.data);
        setPage(v => ({ ...v, items: [latest.data, ...v.items.filter(i => i.id !== latest.data.id)] }));
    });
    const input = caseInputSchema.safeParse(kind === 'withdrawal' ? { kind, withdrawal: { space, id: withdrawalId }, rule: null } : { kind, withdrawal: null, rule: scope === 'all' ? { ruleVersion: 1, kind: scope, knowledge: null } : { ruleVersion: 1, kind: scope, knowledge: { id: knowledgeId, version, sha256: sha } } });
    const locked = command.busy || !!command.pending || !!account?.checking, canEdit = account?.roles.includes('admin');
    const reload = async () => {
        if (loading)
            return;
        setLoading(true);
        setError(null);
        try {
            const out = await read(a => correctionClient.listCases({}, a));
            if (out)
                setPage(out.data);
        }
        catch (e) {
            setError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError());
        }
        finally {
            setLoading(false);
        }
    };
    return <main className={styles.workbench}><h1>Correction cases</h1><p>Follow affected learning evidence from a registered issue to an independently reviewed correction.</p><button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void reload()}>Reload correction cases</button>{error && <CorrectionState error={error} onRetry={() => void reload()}/>}
 <ul className={styles.list}>{page.items.map(c => <li key={c.id}><Link prefetch={false} href={'/review/corrections/' + c.id}>{c.kind === 'grading_rule' ? 'Grading rule review' : 'Withdrawn source review'}</Link><p className={styles.metadata}>{c.id} · {c.hasApprovedPlan ? 'Approved basis available' : 'Awaiting an approved basis'}</p><time dateTime={c.createdAt}>{new Date(c.createdAt).toLocaleString('en')}</time></li>)}</ul>{page.items.length === 0 && <p>No correction cases yet.</p>}{page.nextCursor && <Link prefetch={false} href={'/review/corrections?cursor=' + page.nextCursor}>Next cases</Link>}
 {canEdit && <section className={styles.card}><h2>Register a correction case</h2><p>A grading issue uses the server registration time. Withdrawn sources must reference an actual withdrawal event.</p><form onSubmit={e => {
                e.preventDefault();
                if (input.success)
                    void command.run({ kind: 'createCase', input: input.data });
            }}><fieldset disabled={locked}><label className={styles.field}>Case type<select value={kind} onChange={e => setKind(e.target.value === 'grading_rule' ? 'grading_rule' : 'withdrawal')}><option value="withdrawal">Withdrawn source</option>{account?.roles.includes('admin') && <option value="grading_rule">Grading rule</option>}</select></label>{kind === 'withdrawal' ? <><label className={styles.field}>Withdrawal source<select value={space} onChange={e => setSpace(e.target.value === 'content' ? 'content' : 'question')}><option value="content">Learning content</option><option value="question">Question bank</option></select></label><label className={styles.field}>Withdrawal event ID<input required value={withdrawalId} onChange={e => setWithdrawalId(e.target.value)}/></label></> : <><p>Rule version 1 · Exact rational and single choice grading.</p><label className={styles.field}>Grading scope<select value={scope} onChange={e => setScope(e.target.value === 'knowledge' ? 'knowledge' : 'all')}><option value="all">All mathematics evidence</option><option value="knowledge">One knowledge version</option></select></label>{scope === 'knowledge' && <><label className={styles.field}>Knowledge ID<input value={knowledgeId} onChange={e => setKnowledgeId(e.target.value)}/></label><label className={styles.field}>Knowledge version<input type="number" min={1} max={2147483647} value={version} onChange={e => setVersion(Number(e.target.value))}/></label><label className={styles.field}>Knowledge SHA256<input value={sha} onChange={e => setSha(e.target.value)}/></label></>}</>}<button className="button" disabled={!input.success}>Register correction case</button></fieldset></form><CorrectionCommandStatus command={command}/>{created && <Link prefetch={false} href={'/review/corrections/' + created.id}>Open correction case</Link>}</section>}</main>;
}
