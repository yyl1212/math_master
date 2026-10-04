import {proxyNotification} from '@/lib/api/notification-proxy';
export const dynamic='force-dynamic';
function handle(request:Request){return proxyNotification(request,[])}
export {handle as GET,handle as POST,handle as PUT,handle as DELETE,handle as PATCH,handle as OPTIONS,handle as HEAD};
