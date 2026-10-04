import {correctionAwait,withCorrectionDeadline} from '@/lib/correction/bytes';
import {notificationPage,notificationPageQuery,notificationPageError} from '@/features/notification/page-data';
import {NotificationAccountProvider} from '@/features/notification/notification-account';
import {Inbox} from '@/features/notification/inbox';
import {CorrectionState} from '@/features/correction/status';
export const dynamic='force-dynamic';
export const metadata={title:'Notifications'};
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){try{return await withCorrectionDeadline(undefined,async signal=>{const query=notificationPageQuery(await correctionAwait(searchParams,signal)),out=await notificationPage(query,signal);return <NotificationAccountProvider actorId={out.actorId}><Inbox initial={out.data.page} count={out.data.count}/></NotificationAccountProvider>})}catch(e){return <CorrectionState error={notificationPageError(e)}/>}}
