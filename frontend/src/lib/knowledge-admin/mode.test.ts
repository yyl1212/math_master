import {it,expect,vi} from 'vitest';vi.mock('./server-client',()=>({readManaged:vi.fn(async()=>({ok:false,status:503,code:'CONTENT_NOT_READY'}))}));import {getContentMode} from './mode';
it('does not fall back to legacy after a failed capability read',async()=>{expect(await getContentMode()).toBeNull()});
