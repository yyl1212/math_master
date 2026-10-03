import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {correctionPage,correctionPageError} from '@/features/correction/page-data';
import {CorrectionAccountProvider} from '@/features/correction/correction-account';
import {ResultPanel} from '@/features/correction/result-panel';
import {CorrectionState} from '@/features/correction/status';
import {correctionServer} from '@/lib/correction/server-client';
export const dynamic='force-dynamic';
export const metadata={title:'Your learning correction'};
export default async function Page({params}:{params:Promise<{id:string}>}){try{return await withCorrectionDeadline(undefined,async signal=>{const {id}=await correctionAwait(params,signal),out=await correctionPage(a=>correctionServer.readOwn(id,a),false,signal);return <CorrectionAccountProvider actorId={out.actorId}><ResultPanel key={out.data.result.id} initial={out.data}/></CorrectionAccountProvider>})}catch(e){return <CorrectionState error={correctionPageError(e)}/>}}
