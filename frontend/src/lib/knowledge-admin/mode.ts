import 'server-only';import {readManaged} from './server-client';import type {ContentMode} from './types';
export async function getContentMode():Promise<ContentMode|null>{const r=await readManaged<ContentMode>('/api/v3/content-mode');return r.ok?r.data:null}

export async function redirectLegacyManagement(){const mode=await getContentMode();if(mode?.mode!=='managed')return;const {headers}=await import('next/headers');const {readServerSession}=await import('../auth/server-client');const {redirect}=await import('next/navigation');const user=await readServerSession((await headers()).get('cookie')??'');if(!user.ok||!user.data||user.data.mustChangePassword||!user.data.roles.includes('admin'))redirect('/account');redirect('/admin/knowledge')}
