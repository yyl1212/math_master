'use client';
import {useEffect,useRef} from 'react';import {useUiI18n} from '@/lib/i18n/provider';
const key='__mmNoteHistoryIndex';let currentIndex=0;
const stateIndex=(state:unknown):number|undefined=>state&&typeof state==='object'&&key in state&&typeof state[key as keyof typeof state]==='number'?state[key as keyof typeof state] as number:undefined;
/** Only route-position metadata is tracked. No note, user, or credential is stored. */
export function NoteHistoryTracker(){useEffect(()=>{
 const previousPush=window.history.pushState,previousReplace=window.history.replaceState;currentIndex=stateIndex(window.history.state)??0;
 previousReplace.call(window.history,{...window.history.state,[key]:currentIndex},'',window.location.href);
 const push:History['pushState']=function(this:History,data,unused,url){currentIndex++;previousPush.call(this,{...data,[key]:currentIndex},unused,url)};
 const replace:History['replaceState']=function(this:History,data,unused,url){previousReplace.call(this,{...data,[key]:currentIndex},unused,url)};
 const pop=(event:PopStateEvent)=>{currentIndex=stateIndex(event.state)??currentIndex};window.history.pushState=push;window.history.replaceState=replace;window.addEventListener('popstate',pop);
 return()=>{if(window.history.pushState===push)window.history.pushState=previousPush;if(window.history.replaceState===replace)window.history.replaceState=previousReplace;window.removeEventListener('popstate',pop)}
},[]);return null}
export function NoteLeaveGuard({dirty,actorId,onDiscard}:{dirty:boolean;actorId:string;onDiscard:()=>void}){
 const{t}=useUiI18n();const discard=useRef(onDiscard);discard.current=onDiscard;
 useEffect(()=>{if(!dirty)return;let accepted=false,restoring=false;let focused:HTMLElement|null=null;
  const focus=(event:FocusEvent)=>{if(event.target instanceof HTMLTextAreaElement)focused=event.target};
  const confirm=()=>window.confirm(t('notes.leave',{}));
  const unload=(event:BeforeUnloadEvent)=>{if(!accepted){event.preventDefault();event.returnValue=''}};
  const click=(event:MouseEvent)=>{if(accepted||event.defaultPrevented||event.button!==0||event.metaKey||event.ctrlKey||event.shiftKey||event.altKey)return;const node=event.target instanceof Element?event.target.closest('a[href]'):null;if(!(node instanceof HTMLAnchorElement)||node.target&&node.target!=='_self'||node.hasAttribute('download'))return;const next=new URL(node.href,window.location.href);if(next.origin===location.origin&&next.pathname===location.pathname&&next.search===location.search)return;if(!confirm()){event.preventDefault();event.stopImmediatePropagation();focused?.focus({preventScroll:true})}else{accepted=true;discard.current()}};
  const pop=(event:PopStateEvent)=>{const target=stateIndex(event.state);if(restoring){restoring=false;event.stopImmediatePropagation();return}if(accepted||target===undefined||target===currentIndex)return;if(confirm()){accepted=true;discard.current()}else{event.stopImmediatePropagation();restoring=true;window.history.go(currentIndex-target);focused?.focus({preventScroll:true})}};
  window.addEventListener('beforeunload',unload);document.addEventListener('click',click,true);document.addEventListener('focusin',focus,true);window.addEventListener('popstate',pop,true);
  return()=>{accepted=true;window.removeEventListener('beforeunload',unload);document.removeEventListener('click',click,true);document.removeEventListener('focusin',focus,true);window.removeEventListener('popstate',pop,true)};
 },[dirty,actorId,t]);return null;
}
