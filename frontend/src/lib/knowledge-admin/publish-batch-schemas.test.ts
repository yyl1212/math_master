import {it,expect} from 'vitest';import {publishJobSchema,publishReceiptSchema,publishCreateSchema,publishSchemaForPath} from './publish-batch-schemas';
import {schemaForPath} from './schemas';
const id='11111111-1111-4111-8111-111111111111';
export const job={actorId:id,id,mode:'all',sequence:1,total:41,pending:41,published:0,skipped:0,failed:0,createdAt:'2026-10-19T00:00:00Z'};
it('new batch responses use a separate schema from legacy management',()=>{const path='/api/v3/admin/knowledge/publish-jobs';expect(publishSchemaForPath(path,'POST').parse(job)).toEqual(job);expect(schemaForPath(path,'POST').safeParse(job).success).toBe(false)});
it('publish batch schemas bind actor job sequence and exact counts',()=>{expect(publishJobSchema.parse(job)).toEqual(job);for(const v of [{...job,total:40},{...job,sequence:9007199254740992},{...job,body:'leak'}])expect(publishJobSchema.safeParse(v).success).toBe(false);expect(publishCreateSchema.safeParse({mode:'all',items:[]}).success).toBe(false);const {id:jobId,mode,createdAt,...counts}=job;expect(publishReceiptSchema.safeParse({...counts,jobId,items:Array(21).fill({})}).success).toBe(false)});
