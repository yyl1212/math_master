import { parseContentJSON } from '../content/raw-json';
import { notificationReadInputSchema } from './schemas';
import { CorrectionRequestError } from '../correction/types';
export { correctionAwait as notificationAwait, readCorrectionBytes as readNotificationBytes, withCorrectionDeadline as withNotificationDeadline } from '../correction/bytes';
export function validateNotificationBytes(raw: Uint8Array): Record<string, never> { try {
    if (raw.byteLength > 65536)
        throw new Error();
    return notificationReadInputSchema.parse(parseContentJSON(raw));
}
catch {
    throw new CorrectionRequestError('INVALID_REQUEST');
} }
