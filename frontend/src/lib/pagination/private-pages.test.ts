import {it,expect} from 'vitest';
import {resolveFeedbackRoute,readFeedbackResponse} from '@/lib/feedback/schemas';
import {resolveNotificationRoute,readNotificationResponse} from '@/lib/notification/schemas';
import {resolveCorrectionRoute,readCorrectionResponse} from '@/lib/correction/schemas';
const actorId='11111111-1111-4111-8111-111111111111';
const response=()=>new Response(JSON.stringify({actorId,data:{items:[],total:0,limit:20,offset:0}}),{headers:{'Content-Type':'application/json','Cache-Control':'private, no-store','X-Content-Type-Options':'nosniff','X-Request-ID':'unavailable'}});
it('keeps private page headers and envelope validation for opt-in reads',async()=>{
 for(const[path,resolve,read]of [['/api/v1/feedback/tickets',resolveFeedbackRoute,readFeedbackResponse],['/api/v1/notifications',resolveNotificationRoute,readNotificationResponse],['/api/v1/corrections/cases',resolveCorrectionRoute,readCorrectionResponse]] as const){expect(()=>resolve(path+'?pagination=page&page=1')).not.toThrow();expect(()=>resolve(path+'?page=1')).toThrow();await expect(read(response(),path+'?pagination=page&page=1',new AbortController().signal)).resolves.toMatchObject({actorId,data:{total:0}})}
});
