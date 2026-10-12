import {redirect} from 'next/navigation';
import {historyHref} from '@/lib/knowledge-admin/study-navigation';
import {getUiMetadata} from '@/lib/i18n/server';
export const dynamic='force-dynamic';
export async function generateMetadata(){return getUiMetadata('page.learning-history')}
export default async function Page(props:{searchParams:Promise<Record<string,string|string[]|undefined>>}){redirect(historyHref(await props.searchParams));}
