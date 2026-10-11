import {CurrentKnowledgeView} from '@/features/reading/current-knowledge-view';
import {ManagedPersonal} from '@/features/study/managed-pages';
import {readManaged} from '@/lib/knowledge-admin/server-client';
import type {PublicKnowledge} from '@/lib/knowledge-admin/types';
import {ContentState} from '@/components/content-state';
export const dynamic='force-dynamic';
export default async function Page({params}:{params:Promise<{id:string}>}){const{id}=await params;const current=await readManaged<PublicKnowledge>('/api/v3/knowledge/'+id);if(!current.ok)return <><ContentState kind={current.status===404?'not-found':'unavailable'}/><ManagedPersonal id={id}/></>;return <CurrentKnowledgeView knowledge={current.data} personal={<ManagedPersonal id={id}/>}/>}
