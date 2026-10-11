import {CurrentTopicPage} from '@/features/catalogue/current-catalogue';
export const dynamic='force-dynamic';
export default async function Page({params,searchParams}:{params:Promise<{id:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){return <CurrentTopicPage topicKey={(await params).id} searchParams={await searchParams}/>}
