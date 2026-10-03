import { it, expect } from 'vitest';
import { createPendingNotificationCommand } from './pending-command';
import { id, otherId } from '@/lib/correction/test-fixtures';
it('freezes an owned read command with a reusable key and empty input', () => { const p = createPendingNotificationCommand(id, otherId); expect(p).toMatchObject({ actorId: id, notificationId: otherId, input: {} }); expect(Object.isFrozen(p)).toBe(true); expect(Object.isFrozen(p.input)).toBe(true); expect(p.key).toMatch(/^[0-9a-f-]{36}$/); });
it('rejects unowned or invalid identity references', () => { expect(() => createPendingNotificationCommand('', otherId)).toThrow(); expect(() => createPendingNotificationCommand(id, 'foreign')).toThrow(); });
