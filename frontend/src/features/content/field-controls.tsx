"use client";
import { useId, useRef, type ReactNode } from "react";
import styles from "@/styles/content.module.css";
export function TextField({ label, value, onChange, multiline = false }: {
    label: string;
    value: string;
    onChange: (v: string) => void;
    multiline?: boolean;
}) { const id = useId(); return <label className={styles.field} htmlFor={id}><span>{label}</span>{multiline ? <textarea id={id} value={value} onChange={e => onChange(e.target.value)} rows={3}/> : <input id={id} value={value} onChange={e => onChange(e.target.value)}/>}</label>; }
export function NumberField({ label, value, onChange }: {
    label: string;
    value: number;
    onChange: (v: number) => void;
}) { const id = useId(); return <label className={styles.field} htmlFor={id}><span>{label}</span><input id={id} type="number" min={1} max={2147483647} step={1} value={Number.isFinite(value) ? value : ""} onChange={e => onChange(e.target.value === "" ? 0 : Number(e.target.value))}/></label>; }
export function SelectField({ label, value, options, onChange }: {
    label: string;
    value: string;
    options: readonly string[];
    onChange: (v: string) => void;
}) { const id = useId(); return <label className={styles.field} htmlFor={id}><span>{label}</span><select id={id} value={value} onChange={e => onChange(e.target.value)}>{options.map(v => <option key={v} value={v}>{v}</option>)}</select></label>; }
export function Rows<T>({ label, items, onChange, create, children }: {
    label: string;
    items: T[];
    onChange: (v: T[]) => void;
    create: () => T;
    children: (v: T, index: number, change: (v: T) => void) => ReactNode;
}) {
    const keys = useRef<string[]>([]);
    while (keys.current.length < items.length)
        keys.current.push(crypto.randomUUID());
    if (keys.current.length > items.length)
        keys.current.length = items.length;
    return <div className={styles.rows}>{items.map((item, i) => <div className={styles.row} key={keys.current[i]}>{children(item, i, next => onChange(items.map((v, n) => n === i ? next : v)))}<button type="button" className={styles.remove} onClick={() => { keys.current.splice(i, 1); onChange(items.filter((_, n) => n !== i)); }}>Remove {label} {i + 1}</button></div>)}<button className="button secondary" type="button" onClick={() => onChange([...items, create()])}>Add {label}</button></div>;
}
export function TextList({ label, items, onChange }: {
    label: string;
    items: string[];
    onChange: (v: string[]) => void;
}) { return <Rows label={label} items={items} onChange={onChange} create={() => ""}>{(v, i, change) => <TextField label={`${label} ${i + 1}`} value={v} onChange={change} multiline/>}</Rows>; }
export function RefFields({ label, value, onChange }: {
    label: string;
    value: {
        id: string;
        version: number;
    };
    onChange: (v: {
        id: string;
        version: number;
    }) => void;
}) { return <div className={styles.grid}><TextField label={label + " ID"} value={value.id} onChange={id => onChange({ ...value, id })}/><NumberField label={label + " version"} value={value.version} onChange={version => onChange({ ...value, version })}/></div>; }
