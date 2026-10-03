import type { Metadata } from './types';
import { id, time } from '../correction/test-fixtures';
export { id, otherId, json, context, cursor } from '../correction/test-fixtures';
export const metadata = (): Metadata => ({ id, type: 'checking', evidence: { kind: 'assessment', id }, caseId: id, resultId: null, createdAt: time, readAt: null });
