import type { ReactNode } from "react";
import type { Metadata } from "next";
import { SiteHeader } from "@/components/site-header";
import "@/styles/globals.css";
import {readRequestUiLocale} from "@/lib/i18n/server";
import {UiLocaleProvider} from "@/lib/i18n/provider";
export const metadata: Metadata = {
  title: {
    default: "Math Master — A world of ideas",
    template: "%s | Math Master",
  },
  description:
    "Explore a connected map of mathematics, from foundations to new frontiers.",
};
export default async function Layout({ children }: { children: ReactNode }) {
 const locale=await readRequestUiLocale();
  return (
    <html lang={locale}>
      <body><UiLocaleProvider initialLocale={locale}>
        <a className="skip-link" href="#main-content">
          Skip to content
        </a>
        <SiteHeader />
        <main id="main-content" tabIndex={-1}>
          {children}
        </main>
        <footer className="site-footer">
          <span>
            Math Master<span className="footer-dot">·</span>A world of ideas
          </span>
          <span>Built on clear thinking and connected learning.</span>
        </footer>
      </UiLocaleProvider></body>
    </html>
  );
}
