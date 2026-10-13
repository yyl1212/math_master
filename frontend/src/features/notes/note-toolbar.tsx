'use client';
import {useUiI18n} from '@/lib/i18n/provider';import styles from '@/styles/notes.module.css';
type Format='bold'|'italic'|'list'|'quote'|'inlineMath'|'displayMath';
export function insertNoteFormat(value:string,start:number,end:number,format:Format){
 const selected=value.slice(start,end);let replacement:string;
 if(format==='list'||format==='quote')replacement=selected.split('\n').map(line=>(format==='list'?'- ':'> ')+line).join('\n');
 else {const marks=format==='bold'?['**','**']:format==='italic'?['_','_']:format==='inlineMath'?['$','$']:['\n$$\n','\n$$\n'];replacement=marks[0]+selected+marks[1]}
 return value.slice(0,start)+replacement+value.slice(end);
}
export function NoteToolbar({value,onChange,selection,disabled}:{value:string;onChange:(value:string)=>void;selection:{start:number;end:number};disabled?:boolean}){
 const{t}=useUiI18n();return <div className={styles.toolbar} role="toolbar">{(['bold','italic','list','quote','inlineMath','displayMath'] as const).map(format=><button key={format} type="button" disabled={disabled} onMouseDown={e=>e.preventDefault()} onClick={()=>onChange(insertNoteFormat(value,selection.start,selection.end,format))}>{t(`notes.${format}`,{})}</button>)}</div>;
}
