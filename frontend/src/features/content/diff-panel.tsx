import type { Diff } from "@/lib/content/types";
import styles from "@/styles/content.module.css";
export function DiffPanel({ diff }: {
    diff: Diff;
}) { return <div><p>Added: {diff.added} · Replaced: {diff.replaced} · Removed: {diff.removed}</p>{diff.changes.length > 0 && <table className={styles.diff}><thead><tr><th>Member</th><th>Before</th><th>After</th><th>Reason</th></tr></thead><tbody>{diff.changes.map(c => <tr key={c.kind + "/" + c.id}><td>{c.kind}: {c.id}</td><td>{c.before ? "v" + c.before.version : "—"}</td><td>{c.after ? "v" + c.after.version : "—"}</td><td>{c.reason}</td></tr>)}</tbody></table>}</div>; }
