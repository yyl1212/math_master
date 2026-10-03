import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {correctionPage,correctionPageError} from '@/features/correction/page-data';
import {CorrectionAccountProvider} from '@/features/correction/correction-account';
import {PlanEditor} from '@/features/correction/plan-editor';
import {CorrectionState} from '@/features/correction/status';
import {correctionServer} from '@/lib/correction/server-client';
import {CorrectionRequestError} from '@/lib/correction/types';
export const dynamic='force-dynamic';
export const metadata={title:'Correction plan and review'};
export default async function Page({params}:{params:Promise<{id:string;planId:string;version:string}>}){try{return await withCorrectionDeadline(undefined,async signal=>{const {id,planId,version}=await correctionAwait(params,signal);if(!/^[1-9][0-9]*$/.test(version))throw new CorrectionRequestError('INVALID_REQUEST');const out=await correctionPage(a=>correctionServer.readPlan({id:planId,version:Number(version)},a),true,signal);if(out.data.plan.caseId!==id)throw new CorrectionRequestError('NOT_FOUND');return <CorrectionAccountProvider actorId={out.actorId} management><PlanEditor key={out.data.plan.ref.id+':'+out.data.plan.ref.version} initial={out.data}/></CorrectionAccountProvider>})}catch(e){return <CorrectionState error={correctionPageError(e)}/>}}
