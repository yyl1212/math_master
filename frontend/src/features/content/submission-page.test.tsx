import {it,expect,vi} from 'vitest';
import Page from '@/app/review/[id]/page';
const legacy=vi.hoisted(()=>vi.fn(async()=>{}));
vi.mock('@/lib/knowledge-admin/mode',()=>({redirectLegacyManagement:legacy}));
vi.mock('next/navigation',()=>({redirect:(path:string)=>{throw new Error('Redirect '+path)}}));
it('retired submission URLs never render the legacy approval workbench',async()=>{await expect(Page()).rejects.toThrow('Redirect /account');expect(legacy).toHaveBeenCalledTimes(1)});
