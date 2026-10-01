import Markdown from "react-markdown";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import type { AssetView } from "@/lib/api/types";
import { AssetImage } from "./asset-image";
import styles from "@/styles/reading.module.css";
export function safeTextUrl(url: string): string | undefined {
  if (
    !url ||
    url.trim() !== url ||
    url.includes("\\") ||
    url.startsWith("//") ||
    [...url].some((c) => c.codePointAt(0)! < 32 || c.codePointAt(0) === 127)
  )
    return undefined;
  if (/^https:\/\//i.test(url)) {
    try {
      const u = new URL(url);
      return u.protocol === "https:" && u.hostname ? url : undefined;
    } catch {
      return undefined;
    }
  }
  if (/^[a-z][a-z0-9+.-]*:/i.test(url)) return undefined;
  return url;
}
type AstNode = {
  type: string;
  value?: string;
  lang?: string;
  children?: AstNode[];
};
function limitMath() {
  return (tree: AstNode) => {
    const walk = (parent: AstNode) => {
      parent.children?.forEach((node, index) => {
        const math =
          node.type === "math" ||
          node.type === "inlineMath" ||
          (node.type === "code" && node.lang === "math");
        if (math) {
          const value = node.value ?? "";
          const bad =
            /\\(?:def|gdef|edef|xdef|let|futurelet|newcommand|renewcommand|providecommand|includegraphics|href|url|html[A-Za-z]*)\b/.test(
              value,
            );
          if (value.length > 4096 || bad) {
            const text =
              "Formula could not be displayed." +
              (value.length <= 4096 ? " " + value : "");
            parent.children![index] =
              node.type === "inlineMath"
                ? { type: "text", value: text }
                : {
                    type: "paragraph",
                    children: [{ type: "text", value: text }],
                  };
            return;
          }
        }
        walk(node);
      });
    };
    walk(tree);
  };
}
export function SafeMarkdown({
  source,
  assets,
}: {
  source: string;
  assets: AssetView[];
}) {
  const available = new Map(assets.map((a) => [a.id, a]));
  return (
    <div className={styles.markdown}>
      <Markdown
        skipHtml
        remarkPlugins={[remarkMath, limitMath]}
        rehypePlugins={[
          [
            rehypeKatex,
            {
              trust: false,
              maxExpand: 100,
              maxSize: 10,
              strict: "error",
              macros: {},
            },
          ],
        ]}
        urlTransform={(url, key) =>
          key === "src"
            ? /^asset:[a-z][a-z0-9-]{0,63}$/.test(url) &&
              available.has(url.slice(6))
              ? url
              : undefined
            : safeTextUrl(url)
        }
        components={{
          img: ({ src, alt }) => {
            const asset =
              typeof src === "string" ? available.get(src.slice(6)) : undefined;
            return asset ? (
              <AssetImage asset={asset} alt={alt ?? ""} />
            ) : (
              <span className={styles.figureError}>
                Illustration is not available.
              </span>
            );
          },
          a: ({ href, children }) =>
            href ? (
              <a
                href={href}
                rel={
                  href.startsWith("https:") ? "noopener noreferrer" : undefined
                }
              >
                {children}
              </a>
            ) : (
              <span>{children}</span>
            ),
          span: ({ className, children, ...props }) =>
            className?.split(" ").includes("katex-error") ? (
              <span className={styles.formulaError}>
                Formula could not be displayed. <code>{children}</code>
              </span>
            ) : (
              <span {...props} className={className}>
                {children}
              </span>
            ),
        }}
      >
        {source}
      </Markdown>
    </div>
  );
}
