import {it,expect} from 'vitest';
import {schemaForNotePath} from '@/lib/note-sharing/schemas';
import {schemaForReviewPath} from '@/lib/study-review/schemas';
import {schemaForPath} from '@/lib/knowledge-admin/schemas';
const actorId='11111111-1111-4111-8111-111111111111',page={actorId,items:[],total:0,limit:20,offset:0};
it('explicit numbered mode accepts its own strict page while old cursor schemas remain strict',()=>{
 for(const[path,schema]of [['/api/v3/study/notes',schemaForNotePath],['/api/v3/shared-notes',schemaForNotePath],['/api/v3/admin/note-shares',schemaForNotePath],['/api/v3/study/history',schemaForPath]] as const){expect(schema(path+'?pagination=page').safeParse(page).success).toBe(true);expect(schema(path).safeParse(page).success).toBe(false);expect(schema(path).safeParse({actorId,items:[],nextCursor:null}).success).toBe(true)}
 expect(schemaForReviewPath('/api/v3/study/review/history?pagination=page').safeParse({...page,asOf:'2026-10-19T00:00:00Z'}).success).toBe(true);
});
