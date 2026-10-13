import {getGoOrigin} from '@/lib/api/server-config';import {getAuthConfig} from '@/lib/auth/config';import {createKnowledgeAdminProxy,managedProxyError} from '@/lib/api/knowledge-admin-proxy';
import {createNoteSharingProxy,isNoteSharingPath} from '@/lib/api/note-sharing-proxy';
export const dynamic='force-dynamic';
async function handle(request:Request){try{const factory=isNoteSharingPath(new URL(request.url).pathname)?createNoteSharingProxy:createKnowledgeAdminProxy;return await factory(getGoOrigin(),getAuthConfig())(request)}catch{return managedProxyError()}}
export const GET=handle,POST=handle,PUT=handle,DELETE=handle,PATCH=handle,HEAD=handle,OPTIONS=handle;
