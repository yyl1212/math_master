import { it, expect, vi, afterEach } from 'vitest';
import { createPendingCorrectionCommand } from './pending-command';
import { id, otherId, planInput } from '@/lib/correction/test-fixtures';
afterEach(() => vi.unstubAllGlobals());
it('freezes the original actor, key, route reference and nested input', () => {
    const input = planInput(), ref = { id, version: 1 };
    const p = createPendingCorrectionCommand(id, { kind: 'updatePlan', ref, input: { ...input, expectedSequence: 2 } });
    ref.version = 3;
    input.reason = 'mutated';
    expect(p.actorId).toBe(id);
    expect(p.key).toMatch(/^[0-9a-f-]{36}$/);
    expect(Object.isFrozen(p)).toBe(true);
    expect(Object.isFrozen(p.input)).toBe(true);
    expect(p.kind).toBe('updatePlan');
    if (p.kind === 'updatePlan') {
        expect(p.ref.version).toBe(1);
        expect(p.input.reason).not.toBe('mutated');
    }
});
it('rejects unknown actions, unbound versions and actor input before freezing', () => {
    for (const actor of ['', 'not-an-id'])
        expect(() => createPendingCorrectionCommand(actor, { kind: 'createPlan', caseId: id, input: planInput() })).toThrow();
    expect(() => createPendingCorrectionCommand(id, { kind: 'updatePlan', ref: { id: otherId, version: 0 }, input: { ...planInput(), expectedSequence: 1 } })).toThrow();
});
