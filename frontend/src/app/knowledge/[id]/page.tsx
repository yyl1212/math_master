import {CurrentKnowledgeView} from '@/features/reading/current-knowledge-view';
import {ManagedPersonal} from '@/features/study/managed-pages';
import {readManaged} from '@/lib/knowledge-admin/server-client';
import type {PublicKnowledge} from '@/lib/knowledge-admin/types';
import {ContentState} from '@/components/content-state';
export const dynamic='force-dynamic';
export default async function Page({params,searchParams}:{params:Promise<{id:string}>;searchParams?:Promise<Record<string,string|string[]|undefined>>}){const{id}=await params;const query=searchParams?await searchParams:{};const current=await readManaged<PublicKnowledge>('/api/v3/knowledge/'+id);if(!current.ok)return <><ContentState kind={current.status===404?'not-found':'unavailable'}/><ManagedPersonal id={id}/></>;return <CurrentKnowledgeView initialMode={query.mode==="notes"?"notes":"reading"} knowledge={current.data} personal={<ManagedPersonal id={id}/>}/>}
