import {headers} from 'next/headers';
import {getContentMode} from '@/lib/knowledge-admin/mode';
import {readServerSession} from '@/lib/auth/server-client';
import {CurrentHome} from '@/features/catalogue/current-home';
import {ContentState} from '@/components/content-state';
import {getUiMetadata} from '@/lib/i18n/server';
export const dynamic='force-dynamic';
export async function generateMetadata(){return getUiMetadata('page.home')}
export default async function Page(){const mode=await getContentMode();if(!mode||mode.mode!=='managed')return <ContentState kind="unavailable"/>;const user=await readServerSession((await headers()).get('cookie')??'');return <CurrentHome signedIn={user.ok&&!!user.data&&!user.data.mustChangePassword}/>}
