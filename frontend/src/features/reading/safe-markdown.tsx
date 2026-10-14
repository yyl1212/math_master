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
import prose from "@/styles/current-prose.module.css";
import {useEffect,useRef} from 'react';
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
  presentation = 'default',
}: {
  source: string;
  assets: AssetView[];
  assetScope?: AssetScope;
  presentation?: 'default' | 'study';
}) {
  const {t,locale}=useUiI18n();
  const root=useRef<HTMLDivElement>(null);
  // Markdown's custom spans can remount even when source is unchanged.
  // Rebind presentation measurements after every render; never fetch content.
  useEffect(() => {
    if (presentation !== 'study' || !root.current) return;
    const element = root.current;
    const scrollers = [...element.querySelectorAll<HTMLElement>(
      '.katex-display,.katex:not(.katex-display .katex),table',
    )];
    let live = true;
    const clear = (item: HTMLElement) => {
      item.removeAttribute('tabindex');
      item.removeAttribute('role');
      item.removeAttribute('aria-label');
    };
    const measure = () => {
      if (!live) return;
      for (const item of scrollers) {
        if (item.scrollWidth > item.clientWidth + 1) {
          item.tabIndex = 0;
          item.setAttribute('role', 'region');
          item.setAttribute('aria-label', t(item.tagName === 'TABLE'
            ? 'reading.tableScroll' : 'reading.formulaScroll', {}));
        } else clear(item);
      }
    };
    measure();
    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(measure);
    observer?.observe(element);
    scrollers.forEach(item => observer?.observe(item));
    void document.fonts?.ready.then(measure);
    return () => { live = false; observer?.disconnect(); scrollers.forEach(clear); };
  });
  const available = new Map(assets.map((a) => [a.id, a]));
  return (
    <div ref={root} className={styles.markdown+(presentation==='study'?' '+prose.study:'')} lang="en" onKeyDown={event => {
      if (presentation !== 'study' || event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return;
      const target = event.target;
      if (!(target instanceof HTMLElement) || target.getAttribute('tabindex') !== '0' || !target.matches('.katex-display,.katex,table') || target.scrollWidth <= target.clientWidth + 1) return;
      if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') { event.preventDefault(); target.scrollLeft += event.key === 'ArrowRight' ? 64 : -64; }
    }}>
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
