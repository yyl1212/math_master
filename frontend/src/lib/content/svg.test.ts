import { readFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { it, expect } from "vitest";
import { validateContentSVG } from "./svg";
import { fixtureSVG } from "./test-fixtures";
const check = (raw: string) => { const bytes = new TextEncoder().encode(raw); return validateContentSVG(bytes, createHash("sha256").update(bytes).digest("hex")); };
const root = (body: string) => `<svg xmlns="http://www.w3.org/2000/svg">${body}</svg>`;
it("TestContentSVGScopeAndBoundaryParser", () => {
    expect(validateContentSVG(fixtureSVG, createHash("sha256").update(fixtureSVG).digest("hex"))).toBe(true);
    expect(validateContentSVG(fixtureSVG, "a".repeat(64))).toBe(false);
    for (const svg of [root('<script>unsafe()</script>'), root('<rect width="1" width="2"/>'), root('<rect fill="u&#114;l(x)"/>'), root('<g transform="&#x75;rl(x)"/>'), root('<g onclick="run()"/>'), root('<g>&custom;</g>'), root('<g>&#0;</g>'), root('<g>&#xD800;</g>'), root('<g>\u0001</g>'), root('<g></svg>'), root('<!-- illegal -- comment -->'), '<!DOCTYPE svg>' + root(''), '<?xml version="1.0"?>' + root(''), root('') + root(''), '<svg/>', root('') + 'outside'])
        expect(check(svg)).toBe(false);
    for (const svg of [root('<!-- Original comment --><title>One &amp; two &#x1f600;</title>'), root('<g><![CDATA[<script>plain text</script>]]></g>'), root('<rect fill="#aAbBcc" stroke="none"/>')])
        expect(check(svg)).toBe(true);
    expect(check(root('<g>'.repeat(64) + '</g>'.repeat(64)))).toBe(true);
    expect(check(root('<g>'.repeat(65) + '</g>'.repeat(65)))).toBe(false);
    expect(check(root('<g/>'.repeat(9999)))).toBe(true);
    expect(check(root('<g/>'.repeat(10000)))).toBe(false);
    expect(validateContentSVG(new Uint8Array((1 << 20) + 1), "a".repeat(64))).toBe(false);
});
it("TestContentSVGSharedBoundaries", () => {
    const corpus = JSON.parse(readFileSync("../backend/internal/content/testdata/svg-boundaries.json", "utf8")) as {
        name: string;
        valid: boolean;
        svg: string;
    }[];
    for (const item of corpus)
        expect(check(item.svg), item.name).toBe(item.valid);
});
