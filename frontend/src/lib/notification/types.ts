import type { components } from '../api/generated';
export type Metadata = components['schemas']['NotificationMetadata'];
export type UnreadCount = components['schemas']['NotificationUnreadCount'];
export type ReadReceipt = components['schemas']['NotificationReadReceipt'];
export type { Envelope, Page, PageQuery, ReadAccess, CommandAccess } from '../correction/types';
export { CorrectionRequestError as NotificationRequestError } from '../correction/types';
export type NotificationRoute = {
    path: string;
    method: 'GET' | 'POST';
    action: 'list' | 'count' | 'read' | 'markRead';
};
