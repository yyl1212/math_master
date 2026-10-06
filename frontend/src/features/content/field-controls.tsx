"use client";
import type {UiNotice} from "@/lib/i18n/types";
import {UiText} from "@/lib/i18n/ui-text";
import {formatUiNotice,uiMessage} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import { useId, useRef, type ReactNode } from "react";
import styles from "@/styles/content.module.css";
export function TextField({ label, value, onChange, multiline = false }: {
    label: UiNotice|string;
    value: string;
    onChange: (v: string) => void;
    multiline?: boolean;
}) { const id = useId(); return <label className={styles.field} htmlFor={id}><span><UiText notice={typeof label==="string"?{kind:"literal",text:label}:label}/></span>{multiline ? <textarea id={id} value={value} onChange={e => onChange(e.target.value)} rows={3}/> : <input id={id} value={value} onChange={e => onChange(e.target.value)}/>}</label>; }
export function NumberField({ label, value, onChange }: {
    label: UiNotice|string;
    value: number;
    onChange: (v: number) => void;
}) { const id = useId(); return <label className={styles.field} htmlFor={id}><span><UiText notice={typeof label==="string"?{kind:"literal",text:label}:label}/></span><input id={id} type="number" min={1} max={2147483647} step={1} value={Number.isFinite(value) ? value : ""} onChange={e => onChange(e.target.value === "" ? 0 : Number(e.target.value))}/></label>; }
export function SelectField({ label, value, options, optionLabels, onChange }: {
    label: UiNotice|string;
    value: string;
    options: readonly string[];
    optionLabels?:Readonly<Record<string,UiNotice>>;
    onChange: (v: string) => void;
}) { const id = useId(); return <label className={styles.field} htmlFor={id}><span><UiText notice={typeof label==="string"?{kind:"literal",text:label}:label}/></span><select id={id} value={value} onChange={e => onChange(e.target.value)}>{options.map(v => <option key={v} value={v}><UiText notice={optionLabels?.[v]??{kind:"literal",text:v}}/></option>)}</select></label>; }
export function Rows<T>({ label, items, onChange, create, children }: {
    label: UiNotice|string;
    items: T[];
    onChange: (v: T[]) => void;
    create: () => T;
    children: (v: T, index: number, change: (v: T) => void) => ReactNode;
}) {
    const {locale}=useUiI18n();const labelText=typeof label==="string"?label:formatUiNotice(locale,label);
    const keys = useRef<string[]>([]);
    while (keys.current.length < items.length)
        keys.current.push(crypto.randomUUID());
    if (keys.current.length > items.length)
        keys.current.length = items.length;
    return <div className={styles.rows}>{items.map((item, i) => <div className={styles.row} key={keys.current[i]}>{children(item, i, next => onChange(items.map((v, n) => n === i ? next : v)))}<button type="button" className={styles.remove} onClick={() => { keys.current.splice(i, 1); onChange(items.filter((_, n) => n !== i)); }}><UiText notice={uiMessage("field.remove",{label:labelText,number:i+1})}/></button></div>)}<button className="button secondary" type="button" onClick={() => onChange([...items, create()])}><UiText notice={uiMessage("field.add",{label:labelText})}/></button></div>;
}
export function TextList({ label, items, onChange }: {
    label: UiNotice|string;
    items: string[];
    onChange: (v: string[]) => void;
}) { const {locale}=useUiI18n();const labelText=typeof label==="string"?label:formatUiNotice(locale,label);return <Rows label={label} items={items} onChange={onChange} create={() => ""}>{(v, i, change) => <TextField label={uiMessage("field.numbered",{label:labelText,number:i+1})} value={v} onChange={change} multiline/>}</Rows>; }
export function RefFields({ label, value, onChange }: {
    label: UiNotice|string;
    value: {
        id: string;
        version: number;
    };
    onChange: (v: {
        id: string;
        version: number;
    }) => void;
}) { const {locale}=useUiI18n();const labelText=typeof label==="string"?label:formatUiNotice(locale,label);return <div className={styles.grid}><TextField label={uiMessage("field.id",{label:labelText})} value={value.id} onChange={id => onChange({ ...value, id })}/><NumberField label={uiMessage("field.version",{label:labelText})} value={value.version} onChange={version => onChange({ ...value, version })}/></div>; }
