"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Markdown from "react-markdown";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import type { AssetView } from "@/lib/api/types";
import type {AssetScope} from "@/lib/content/types";
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
function limitMath(label:string) {
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
              label +
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
  assetScope,
}: {
  source: string;
  assets: AssetView[];
  assetScope?: AssetScope;
}) {
  const {t,locale}=useUiI18n();
  const available = new Map(assets.map((a) => [a.id, a]));
  return (
    <div className={styles.markdown} lang="en">
      <Markdown
        skipHtml
        remarkPlugins={[remarkMath, ()=>limitMath(t("safe-markdown.formula.could.not.be.displayed.1952de",{}))]}
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
              <AssetImage asset={asset} alt={alt ?? ""} assetScope={assetScope} />
            ) : (
              <span className={styles.figureError} lang={locale}><UiText notice={uiMessage("safe-markdown.illustration.is.not.available.ef0518",{})}/></span>
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
              <span className={styles.formulaError} lang={locale}><UiText notice={uiMessage("safe-markdown.formula.could.not.be.displayed.1952de",{})}/><code>{children}</code>
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
