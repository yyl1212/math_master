import { it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { validateLearningBytes, LearningInputError } from "./bytes";
import { attemptViewSchema, practiceViewSchema, resultViewSchema } from "./schemas";
import { attempt, practice, result } from "./test-fixtures";
import type { LearningAction } from "./types";
it("all shared Go raw cases agree", () => { const cases = JSON.parse(readFileSync("../api/learning-boundary-cases.json", "utf8")) as {
    name: string;
    action: LearningAction;
    raw: string;
    accepted: boolean;
    errorCode: string | null;
}[]; expect(cases.length).toBeGreaterThanOrEqual(60); for (const c of cases) {
    try {
        validateLearningBytes(new TextEncoder().encode(c.raw), c.action);
        expect(c.accepted, c.name).toBe(true);
    }
    catch (e) {
        expect(c.accepted, c.name).toBe(false);
        expect(e, c.name).toBeInstanceOf(LearningInputError);
        expect((e as LearningInputError).code, c.name).toBe(c.errorCode);
    }
} });
it("active DTOs reject every answer or private field", () => { const a = attempt(); expect(attemptViewSchema.safeParse(a).success).toBe(true); for (const field of ["correctNumeric", "correctChoiceId", "explanation", "parameters", "witness", "sourceMap", "seal"]) {
    expect(attemptViewSchema.safeParse({ ...a, [field]: true }).success).toBe(false);
    expect(attemptViewSchema.safeParse({ ...a, questions: a.questions.map((q, i) => i ? q : { ...q, [field]: true }) }).success).toBe(false);
} ; const p = practice(); expect(practiceViewSchema.safeParse(p).success).toBe(true); expect(practiceViewSchema.safeParse({ ...p, result: result().items[0] }).success).toBe(false); expect(attemptViewSchema.safeParse({ ...a, questions: a.questions.slice(0, 4) }).success).toBe(false); });
it("fixed result score, nullable affected outcome and redaction are consistent", () => { const r = result(); expect(resultViewSchema.safeParse(r).success).toBe(true); expect(resultViewSchema.safeParse({ ...r, score: 3 }).success).toBe(false); expect(resultViewSchema.safeParse({ ...r, outcome: "affected", score: null, passed: null, validity: "restricted", items: r.items.map(i => ({ ...i, correct: null })) }).success).toBe(true); expect(resultViewSchema.safeParse({ ...r, items: r.items.map(i => ({ ...i, validity: "restricted", correct: null, correctNumeric: null, explanation: null, reasons: ["instance-withdrawn"] })) }).success).toBe(true); expect(resultViewSchema.safeParse({ ...r, items: r.items.map(i => ({ ...i, validity: "restricted", reasons: ["instance-withdrawn"] })) }).success).toBe(false); });
