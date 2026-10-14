import {ReviewHistoryPage} from '@/features/study-review/pages';import {getUiMetadata} from '@/lib/i18n/server';
export const dynamic='force-dynamic';
export async function generateMetadata(){return getUiMetadata('page.learn.review.history')}
export default ReviewHistoryPage;
