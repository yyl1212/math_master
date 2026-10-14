import {it,expect,vi} from 'vitest';
import {makePendingReviewCommand,confirmPendingReviewCommand} from './pending-command';
import {actorId,sessionId,receipt,session,date} from './test-fixtures';
const path='/api/v3/study/review/sessions/'+sessionId+'/compare';
it('freezes caller input and validates metadata actor/action/resource/sequence before current read',async()=>{
 const input={expectedSequence:1,body:'本人原作答',unanswered:false};const pending=makePendingReviewCommand(actorId,path,'POST',input);input.body='改成参考';
 const send=vi.fn(async()=>receipt({action:'compare',status:'compared',sequence:2,draftRevision:1}));const read=vi.fn(async()=>session({status:'compared',sequence:2,draftRevision:1,comparedAt:date,answer:'本人原作答'}));
 const result=await confirmPendingReviewCommand(pending,send,read);expect(result.session?.status).toBe('compared');expect(send.mock.calls[0]).toEqual([path,'POST',{expectedSequence:1,body:'本人原作答',unanswered:false},pending.key,actorId]);expect(read).toHaveBeenCalledWith(sessionId,actorId);
 for(const bad of [receipt({action:'draft',sequence:2}),receipt({action:'compare',sessionId:actorId,sequence:2}),receipt({action:'compare',actorId:sessionId,sequence:2}),receipt({action:'compare',status:'compared',sequence:3})]) {const invalid=makePendingReviewCommand(actorId,path,'POST',{expectedSequence:1,body:'本人',unanswered:false});const get=vi.fn();await expect(confirmPendingReviewCommand(invalid,async()=>bad,get)).rejects.toMatchObject({code:'STUDY_REVIEW_NOT_READY'});expect(get).not.toHaveBeenCalled()}
});
it('retries unknown result with original key then GET; confirmed mutation with failed GET only reads again',async()=>{
 const pending=makePendingReviewCommand(actorId,path,'POST',{expectedSequence:1,body:'答案',unanswered:false});const send=vi.fn().mockRejectedValueOnce(new Error('transport interrupted')).mockResolvedValue(receipt({action:'compare',status:'compared',sequence:2,draftRevision:1}));const get=vi.fn().mockRejectedValueOnce(new Error('read failed')).mockResolvedValue(session({status:'compared',sequence:2,draftRevision:1,comparedAt:date,answer:'答案'}));
 await expect(confirmPendingReviewCommand(pending,send,get)).rejects.toThrow();const confirmed=await confirmPendingReviewCommand(pending,send,get);expect(confirmed.receipt.sequence).toBe(2);expect(confirmed.session).toBeNull();const restored=await confirmPendingReviewCommand(pending,send,get);expect(restored.session?.answer).toBe('答案');expect(send).toHaveBeenCalledTimes(2);expect(send.mock.calls[0][3]).toBe(send.mock.calls[1][3]);expect(get).toHaveBeenCalledTimes(2);
});
