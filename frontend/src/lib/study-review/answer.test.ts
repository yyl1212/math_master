import {it,expect} from 'vitest';
import {isReviewAnswerBlank} from './answer';
it('uses the Unicode White_Space set used by Go and the database',()=>{
 const whitespace=[...Array.from({length:5},(_,i)=>i+9),32,133,160,5760,...Array.from({length:11},(_,i)=>8192+i),8232,8233,8239,8287,12288].map(n=>String.fromCodePoint(n));
 expect(isReviewAnswerBlank('')).toBe(true);
 for(const value of whitespace){expect(isReviewAnswerBlank(value)).toBe(true);expect(isReviewAnswerBlank('x'+value)).toBe(false)}
 for(const value of ['\ufeff','\u180e','\u200b','\u0000','x'])expect(isReviewAnswerBlank(value)).toBe(false);
});
