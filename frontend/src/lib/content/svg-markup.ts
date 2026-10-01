const elements = new Set("svg g rect line path circle ellipse polygon polyline text tspan title desc".split(" "));
const attributes = new Set("xmlns width height viewBox x y x1 y1 x2 y2 cx cy r rx ry d points transform fill stroke stroke-width font-size font-family text-anchor role aria-label".split(" "));
const color = /^(?:none|black|white|red|green|blue|gray|grey|orange|purple|yellow|transparent|#[a-fA-F0-9]{3}|#[a-fA-F0-9]{6}|#[a-fA-F0-9]{8})$/;
const xmlChar = (n: number) => n === 9 || n === 10 || n === 13 || n >= 0x20 && n <= 0xd7ff || n >= 0xe000 && n <= 0xfffd || n >= 0x10000 && n <= 0x10ffff;
const validCharacters = (s: string) => [...s].every(c => xmlChar(c.codePointAt(0)!));
const blank = (s: string) => /^[\u0009-\u000d\u0020\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]*$/.test(s);
function entities(raw: string): string | null {
    let out = "";
    for (let i = 0; i < raw.length; i++) {
        if (raw[i] !== "&") {
            out += raw[i];
            continue;
        }
        ;
        const end = raw.indexOf(";", i + 1);
        if (end < 0)
            return null;
        const name = raw.slice(i + 1, end);
        const predefined: Record<string, string> = { amp: "&", lt: "<", gt: ">", apos: "'", quot: '"' };
        let decoded = predefined[name];
        if (decoded === undefined) {
            let n: number;
            if (/^#[0-9]+$/.test(name))
                n = Number(name.slice(1));
            else if (/^#x[0-9a-fA-F]+$/.test(name))
                n = parseInt(name.slice(2), 16);
            else
                return null;
            if (!Number.isSafeInteger(n) || !xmlChar(n))
                return null;
            decoded = String.fromCodePoint(n);
        }
        ;
        out += decoded;
        i = end;
    }
    return validCharacters(out) ? out : null;
}
export function validSVGMarkup(bytes: Uint8Array): boolean {
    try {
        if (bytes.byteLength > 1 << 20)
            return false;
        const raw = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(bytes);
        if (!validCharacters(raw))
            return false;
        let at = 0, count = 0, roots = 0;
        const stack: string[] = [];
        const space = () => {
            const start = at;
            while (at < raw.length && /[ \t\r\n]/.test(raw[at]))
                at++;
            return at > start;
        };
        const name = () => {
            const start = at;
            while (at < raw.length && /[A-Za-z0-9_.:-]/.test(raw[at]))
                at++;
            return raw.slice(start, at);
        };
        while (at < raw.length) {
            if (raw[at] !== "<") {
                let end = raw.indexOf("<", at);
                if (end < 0)
                    end = raw.length;
                const text = raw.slice(at, end);
                if (text.includes("]]>"))
                    return false;
                const decoded = entities(text);
                if (decoded === null || stack.length === 0 && !blank(decoded))
                    return false;
                at = end;
                continue;
            }
            if (raw.startsWith("<!--", at)) {
                const end = raw.indexOf("-->", at + 4);
                if (end < 0 || raw.slice(at + 4, end).includes("--") || raw[end - 1] === "-")
                    return false;
                at = end + 3;
                continue;
            }
            if (raw.startsWith("<![CDATA[", at)) {
                const end = raw.indexOf("]]>", at + 9);
                if (end < 0)
                    return false;
                const text = raw.slice(at + 9, end);
                if (stack.length === 0 && !blank(text))
                    return false;
                at = end + 3;
                continue;
            }
            if (raw.startsWith("</", at)) {
                at += 2;
                const closing = name();
                space();
                if (raw[at++] !== ">" || stack.pop() !== closing)
                    return false;
                continue;
            }
            if (raw.startsWith("<!", at) || raw.startsWith("<?", at))
                return false;
            at++;
            const tag = name();
            if (!elements.has(tag) || stack.length > 64 || ++count > 10000)
                return false;
            const root = stack.length === 0;
            if (root && (tag !== "svg" || ++roots > 1))
                return false;
            const seen = new Set<string>();
            let xmlns = false, selfClosing = false;
            for (;;) {
                const spaced = space();
                if (raw.startsWith("/>", at)) {
                    at += 2;
                    selfClosing = true;
                    break;
                }
                ;
                if (raw[at] === ">") {
                    at++;
                    break;
                }
                ;
                if (!spaced)
                    return false;
                const key = name();
                if (!attributes.has(key) || seen.has(key))
                    return false;
                seen.add(key);
                space();
                if (raw[at++] !== "=")
                    return false;
                space();
                const quote = raw[at++];
                if (quote !== "'" && quote !== '"')
                    return false;
                const end = raw.indexOf(quote, at);
                if (end < 0)
                    return false;
                const text = raw.slice(at, end);
                if (text.includes("<"))
                    return false;
                const decoded = entities(text);
                if (decoded === null)
                    return false;
                at = end + 1;
                if (key === "xmlns") {
                    if (decoded !== "http://www.w3.org/2000/svg")
                        return false;
                    xmlns = true;
                }
                ;
                if ((key === "fill" || key === "stroke") && !color.test(decoded) || decoded.toLowerCase().includes("url("))
                    return false;
            }
            if (root && !xmlns)
                return false;
            if (!selfClosing)
                stack.push(tag);
        }
        return roots === 1 && stack.length === 0;
    }
    catch {
        return false;
    }
}
