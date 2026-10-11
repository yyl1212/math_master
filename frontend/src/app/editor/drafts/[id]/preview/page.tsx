import {redirect} from 'next/navigation';
import {redirectLegacyManagement} from '@/lib/knowledge-admin/mode';
export const dynamic='force-dynamic';
export default async function Page(){await redirectLegacyManagement();redirect('/account')}
