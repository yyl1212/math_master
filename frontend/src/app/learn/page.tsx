import {ManagedLearnPage,ManagedHistoryPage} from '@/features/study/managed-pages';
import {historyHref} from '@/lib/knowledge-admin/study-navigation';
import {getUiMetadata} from '@/lib/i18n/server';
export const dynamic='force-dynamic';
export async function generateMetadata(){return getUiMetadata('page.learn')}
export default async function Page(props:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const query=await props.searchParams;return query.view==='history'?<ManagedHistoryPage key={historyHref(query)} searchParams={Promise.resolve(query)}/>:<ManagedLearnPage searchParams={Promise.resolve(query)}/>;}
