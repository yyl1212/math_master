import 'server-only';
import { proxyPrivateCorrection } from './correction-proxy';
import { resolveNotificationRoute, readNotificationResponse } from '../notification/schemas';
import { validateNotificationBytes } from '../notification/bytes';
export function proxyNotification(request: Request, segments: readonly string[] | Promise<readonly string[]>) { return proxyPrivateCorrection(request, segments, '/api/v1/notifications', resolveNotificationRoute, validateNotificationBytes, readNotificationResponse); }
