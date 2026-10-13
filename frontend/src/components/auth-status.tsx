"use client";
import {UiText} from '@/lib/i18n/ui-text';
import {uiMessage} from '@/lib/i18n/format';
import Link from 'next/link';
import {useEffect,useState,type ReactNode} from 'react';
import {usePathname} from 'next/navigation';
import {getAuthContext} from '@/lib/auth/client';
import type {AuthContext,AuthResult} from '@/lib/auth/types';
import styles from '@/styles/auth.module.css';
export function AuthStatus(){
 const path=usePathname(),[state,setState]=useState<AuthResult<AuthContext>|null>(null);
 useEffect(()=>{let live=true,generation=0;const refresh=()=>{const revision=++generation;void getAuthContext().then(result=>{if(live&&revision===generation)setState(result)})};refresh();window.addEventListener('math-master:auth-change',refresh);return()=>{live=false;generation++;window.removeEventListener('math-master:auth-change',refresh)}},[path]);
 const text=(key:'auth-status.my.learning.c60fdf'|'site-header.my.reports.cc6e3f'|'managed.admin'|'auth-status.manage.users.58606e')=><UiText notice={uiMessage(key,{})}/>;
 return <div className={styles.status} aria-live="polite">{!state?<span><UiText notice={uiMessage('auth-status.checking.account.d18f41',{})}/></span>:!state.ok?<Link prefetch={false} href="/account"><UiText notice={uiMessage('auth-status.accounts.unavailable.5b2487',{})}/></Link>:state.data.user?<>
  <Link prefetch={false} href="/account">{state.data.user.username}</Link>
  {!state.data.user.mustChangePassword&&<><Link prefetch={false} href="/learn">{text('auth-status.my.learning.c60fdf')}</Link><Link prefetch={false} href="/feedback">{text('site-header.my.reports.cc6e3f')}</Link>
   {state.data.user.roles.includes('admin')&&<AdminNavigation><Link prefetch={false} href="/admin/knowledge">{text('managed.admin')}</Link><Link prefetch={false} href="/admin/note-shares"><UiText notice={uiMessage('notes.approval',{})}/></Link><Link prefetch={false} href="/review/feedback"><UiText notice={uiMessage('streamline.feedback.manage',{})}/></Link><Link prefetch={false} href="/admin/users">{text('auth-status.manage.users.58606e')}</Link></AdminNavigation>}
  </>}
 </>:<Link prefetch={false} href="/login"><UiText notice={uiMessage('auth-status.sign.in.bfd402',{})}/></Link>}</div>;
}
function AdminNavigation({children}:{children:ReactNode}){const[mobile,setMobile]=useState(false);useEffect(()=>{if(typeof window.matchMedia!=='function')return;const media=window.matchMedia('(max-width: 600px)'),update=()=>setMobile(media.matches);update();media.addEventListener('change',update);return()=>media.removeEventListener('change',update)},[]);return mobile?<details className={styles.adminNavigation}><summary><UiText notice={uiMessage('notes.management',{})}/></summary><div>{children}</div></details>:<span className={styles.adminNavigation}>{children}</span>}
