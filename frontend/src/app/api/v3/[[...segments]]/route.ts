import {getGoOrigin} from '@/lib/api/server-config';import {getAuthConfig} from '@/lib/auth/config';import {createKnowledgeAdminProxy,managedProxyError} from '@/lib/api/knowledge-admin-proxy';
export const dynamic='force-dynamic';
async function handle(request:Request){try{return await createKnowledgeAdminProxy(getGoOrigin(),getAuthConfig())(request)}catch{return managedProxyError()}}
export const GET=handle,POST=handle,PUT=handle,DELETE=handle,PATCH=handle,HEAD=handle,OPTIONS=handle;
