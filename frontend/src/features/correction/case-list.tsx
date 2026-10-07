"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from 'next/link';
import { useContext, useState, useEffect } from 'react';
import { correctionClient } from '@/lib/correction/client';
import { caseInputSchema } from '@/lib/correction/schemas';
import { CorrectionRequestError, type Page, type CaseMetadata } from '@/lib/correction/types';
import { CorrectionAccountContext, useCorrectionRead } from './correction-account';
import { useCorrectionCommand } from './pending-command';
import { CorrectionState, CorrectionCommandStatus } from './status';
import styles from '@/styles/content.module.css';
export function CaseList({ initial,topicMode=false }: {
 topicMode?:boolean;
    initial: Page<CaseMetadata>;
}) {
 const {t}=useUiI18n();

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
    const locked = command.busy || !!command.pending || !!account?.checking, canEdit = !topicMode&&account?.roles.includes('admin');
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
    return <main className={styles.workbench}><h1><UiText notice={uiMessage("page.review.corrections",{})}/></h1><p><UiText notice={uiMessage("case-list.follow.affected.learning.evidence.from.a.registered.issue.to.an.i.e9df91",{})}/></p><button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void reload()}><UiText notice={uiMessage("case-list.reload.correction.cases.01b77f",{})}/></button>{error && <CorrectionState error={error} onRetry={() => void reload()}/>}
 <ul className={styles.list}>{page.items.map(c => <li key={c.id}><Link prefetch={false} href={'/review/corrections/' + c.id}>{c.kind === 'grading_rule' ? t("case-list.grading.rule.review.ab2fe4",{}) : t("case-list.withdrawn.source.review.966bbb",{})}</Link><p className={styles.metadata}>{c.id} · {c.hasApprovedPlan ? t("case-list.approved.basis.available.c1fcbf",{}) : t("case-list.awaiting.an.approved.basis.0eaba0",{})}</p><time dateTime={c.createdAt}>{new Date(c.createdAt).toLocaleString('en')}</time></li>)}</ul>{page.items.length === 0 && <p><UiText notice={uiMessage("case-list.no.correction.cases.yet.19127d",{})}/></p>}{page.nextCursor && <Link prefetch={false} href={'/review/corrections?'+(topicMode?'archive=legacy&':'')+'cursor='+page.nextCursor}><UiText notice={uiMessage("case-list.next.cases.f41e3e",{})}/></Link>}
 {canEdit && <section className={styles.card}><h2><UiText notice={uiMessage("case-list.register.a.correction.case.c8261a",{})}/></h2><p><UiText notice={uiMessage("case-list.a.grading.issue.uses.the.server.registration.time.withdrawn.sourc.cce17d",{})}/></p><form onSubmit={e => {
                e.preventDefault();
                if (input.success)
                    void command.run({ kind: 'createCase', input: input.data });
            }}><fieldset disabled={locked}><label className={styles.field}><UiText notice={uiMessage("case-list.case.type.858985",{})}/><select value={kind} onChange={e => setKind(e.target.value === 'grading_rule' ? 'grading_rule' : 'withdrawal')}><option value="withdrawal"><UiText notice={uiMessage("case-list.withdrawn.source.749b87",{})}/></option>{account?.roles.includes('admin') && <option value="grading_rule"><UiText notice={uiMessage("case-list.grading.rule.2115fc",{})}/></option>}</select></label>{kind === 'withdrawal' ? <><label className={styles.field}><UiText notice={uiMessage("review-panel.withdrawal.source.77a1ed",{})}/><select value={space} onChange={e => setSpace(e.target.value === 'content' ? 'content' : 'question')}><option value="content"><UiText notice={uiMessage("case-list.learning.content.ace9d7",{})}/></option><option value="question"><UiText notice={uiMessage("review-panel.question.bank.d8c022",{})}/></option></select></label><label className={styles.field}><UiText notice={uiMessage("review-panel.withdrawal.event.id.935c33",{})}/><input required value={withdrawalId} onChange={e => setWithdrawalId(e.target.value)}/></label></> : <><p><UiText notice={uiMessage("case-list.rule.version.1.exact.rational.and.single.choice.grading.80f013",{})}/></p><label className={styles.field}><UiText notice={uiMessage("case-list.grading.scope.ea1ec5",{})}/><select value={scope} onChange={e => setScope(e.target.value === 'knowledge' ? 'knowledge' : 'all')}><option value="all"><UiText notice={uiMessage("case-list.all.mathematics.evidence.9fc01d",{})}/></option><option value="knowledge"><UiText notice={uiMessage("case-list.one.knowledge.version.0c6041",{})}/></option></select></label>{scope === 'knowledge' && <><label className={styles.field}><UiText notice={uiMessage("case-list.knowledge.id.d157a6",{})}/><input value={knowledgeId} onChange={e => setKnowledgeId(e.target.value)}/></label><label className={styles.field}><UiText notice={uiMessage("case-list.knowledge.version.acf62a",{})}/><input type="number" min={1} max={2147483647} value={version} onChange={e => setVersion(Number(e.target.value))}/></label><label className={styles.field}><UiText notice={uiMessage("case-list.knowledge.sha256.46465c",{})}/><input value={sha} onChange={e => setSha(e.target.value)}/></label></>}</>}<button className="button" disabled={!input.success}><UiText notice={uiMessage("case-list.register.correction.case.abccfa",{})}/></button></fieldset></form><CorrectionCommandStatus command={command}/>{created && <Link prefetch={false} href={'/review/corrections/' + created.id}><UiText notice={uiMessage("case-list.open.correction.case.7931bb",{})}/></Link>}</section>}</main>;
}
