import {CurrentCatalogue,pageOffset} from '@/features/catalogue/current-catalogue';
import {getUiMetadata} from '@/lib/i18n/server';
export const dynamic='force-dynamic';
export async function generateMetadata(){return getUiMetadata('page.knowledge')}
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const p=await searchParams;return <CurrentCatalogue q={typeof p.q==='string'?p.q:undefined} offset={pageOffset(p.offset)} topicKey={typeof p.topicKey==='string'?p.topicKey:undefined} mode={p.mode==='knowledge'?'knowledge':'topics'}/>}
