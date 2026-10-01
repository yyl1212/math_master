// Parse raw bytes before reserialization: JSON.parse alone loses duplicate keys
// and silently keeps unpaired Unicode surrogate escapes.
export function validContentString(value: string): boolean { return !value.includes("\0") && [...value].every(c => { const n = c.codePointAt(0)!; return n < 0xd800 || n > 0xdfff; }); }
export function parseContentJSON(bytes: Uint8Array): unknown {
    const raw = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(bytes);
    let at = 0;
    const invalid = (): never => { throw new Error("Invalid content JSON."); };
    const space = () => {
        while (at < raw.length && /[ \t\r\n]/.test(raw[at]))
            at++;
    };
    const string = (): string => {
        const start = at;
        if (raw[at++] !== '"')
            return invalid();
        let closed = false;
        while (at < raw.length) {
            const c = raw[at++];
            if (c === '"') {
                closed = true;
                break;
            }
            ;
            if (c === '\\') {
                if (at >= raw.length)
                    return invalid();
                const escape = raw[at++];
                if (escape === 'u') {
                    if (!/^[0-9a-fA-F]{4}$/.test(raw.slice(at, at + 4)))
                        return invalid();
                    at += 4;
                }
                else if (!'"\\/bfnrt'.includes(escape))
                    return invalid();
            }
            else if (c.charCodeAt(0) < 32)
                return invalid();
        }
        if (!closed)
            return invalid();
        const value: unknown = JSON.parse(raw.slice(start, at));
        if (typeof value !== "string" || !validContentString(value))
            return invalid();
        return value;
    };
    const value = (depth: number): unknown => {
        space();
        const c = raw[at];
        if (c === '"')
            return string();
        if (c === '{' || c === '[') {
            if (depth > 32)
                return invalid();
            at++;
            space();
            if (c === '[') {
                const result: unknown[] = [];
                if (raw[at] === ']') {
                    at++;
                    return result;
                }
                ;
                for (;;) {
                    result.push(value(depth + 1));
                    space();
                    if (raw[at] === ']') {
                        at++;
                        return result;
                    }
                    ;
                    if (raw[at++] !== ',')
                        return invalid();
                }
            }
            const result: Record<string, unknown> = Object.create(null), seen = new Set<string>();
            if (raw[at] === '}') {
                at++;
                return result;
            }
            ;
            for (;;) {
                space();
                const key = string(), fold = key.toLowerCase();
                if (seen.has(fold))
                    return invalid();
                seen.add(fold);
                space();
                if (raw[at++] !== ':')
                    return invalid();
                result[key] = value(depth + 1);
                space();
                if (raw[at] === '}') {
                    at++;
                    return result;
                }
                ;
                if (raw[at++] !== ',')
                    return invalid();
            }
        }
        for (const [text, v] of [["true", true], ["false", false], ["null", null]] as const) {
            if (raw.startsWith(text, at)) {
                at += text.length;
                return v;
            }
        }
        const number = raw.slice(at).match(/^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?/);
        if (!number)
            return invalid();
        at += number[0].length;
        const n = Number(number[0]);
        if (!/^-?(?:0|[1-9][0-9]*)$/.test(number[0]) || !Number.isSafeInteger(n))
            return invalid();
        return n;
    };
    const parsed = value(1);
    space();
    if (at !== raw.length || parsed === null || typeof parsed !== "object" || Array.isArray(parsed))
        return invalid();
    return parsed;
}
