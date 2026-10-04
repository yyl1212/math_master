import 'server-only';
import { notificationServer } from '@/lib/notification/server-client';
import { correctionPage } from '../correction/page-data';
import { CorrectionRequestError, type PageQuery } from '@/lib/correction/types';
export { correctionPageQuery as notificationPageQuery, correctionPageError as notificationPageError } from '../correction/page-data';
export function notificationPage(query: PageQuery, signal?: AbortSignal) {
    return correctionPage(async (access) => {
        const [page, count] = await Promise.all([notificationServer.list(query, access), notificationServer.count(access)]);
        if (page.actorId !== count.actorId)
            throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
        return { actorId: page.actorId, data: { page: page.data, count: count.data.count } };
    },false,signal);
}
