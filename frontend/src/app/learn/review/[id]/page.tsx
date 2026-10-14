import {ReviewSessionPage} from '@/features/study-review/pages';import {getUiMetadata} from '@/lib/i18n/server';
export const dynamic='force-dynamic';
export async function generateMetadata(){return getUiMetadata('page.learn.review.id')}
export default ReviewSessionPage;
