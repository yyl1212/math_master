"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import type { Diff } from "@/lib/content/types";
import styles from "@/styles/content.module.css";
export function DiffPanel({ diff }: {
    diff: Diff;
}) { return <div><p><UiText notice={uiMessage("diff-panel.added.value.replaced.value.removed.value.83f44a",{v0:uiValue(diff.added),v1:uiValue(diff.replaced),v2:uiValue(diff.removed)})}/></p>{diff.changes.length > 0 && <table className={styles.diff}><thead><tr><th><UiText notice={uiMessage("diff-panel.member.7c968f",{})}/></th><th><UiText notice={uiMessage("diff-panel.before.9bb725",{})}/></th><th><UiText notice={uiMessage("diff-panel.after.7b68fe",{})}/></th><th><UiText notice={uiMessage("admin-users.reason.f81ab8",{})}/></th></tr></thead><tbody>{diff.changes.map(c => <tr key={c.kind + "/" + c.id}><td>{c.kind}: {c.id}</td><td>{c.before ? "v" + c.before.version : "—"}</td><td>{c.after ? "v" + c.after.version : "—"}</td><td>{c.reason}</td></tr>)}</tbody></table>}</div>; }
