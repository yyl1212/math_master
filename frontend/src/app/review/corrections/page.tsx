import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {correctionPage,correctionPageQuery,correctionPageError} from '@/features/correction/page-data';
import {CorrectionAccountProvider} from '@/features/correction/correction-account';
import {CaseList} from '@/features/correction/case-list';
import {CorrectionState} from '@/features/correction/status';
import {correctionServer} from '@/lib/correction/server-client';
export const dynamic='force-dynamic';
export const metadata={title:'Correction cases'};
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){try{return await withCorrectionDeadline(undefined,async signal=>{const query=correctionPageQuery(await correctionAwait(searchParams,signal)),out=await correctionPage(a=>correctionServer.listCases(query,a),true,signal);return <CorrectionAccountProvider actorId={out.actorId} management><CaseList initial={out.data}/></CorrectionAccountProvider>})}catch(e){return <CorrectionState error={correctionPageError(e)}/>}}
