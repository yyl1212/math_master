import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import type { ReactNode } from "react";

import { SiteHeader } from "@/components/site-header";
import "@/styles/globals.css";
import {readRequestUiLocale,getUiMetadata} from "@/lib/i18n/server";
import {UiLocaleProvider} from "@/lib/i18n/provider";
export default async function Layout({ children }: { children: ReactNode }) {
 const locale=await readRequestUiLocale();
  return (
    <html lang={locale}>
      <body><UiLocaleProvider initialLocale={locale}>
        <a className="skip-link" href="#main-content"><UiText notice={uiMessage("layout.skip.to.content.ac576a",{})}/></a>
        <SiteHeader />
        <main id="main-content" tabIndex={-1}>
          {children}
        </main>
        <footer className="site-footer">
          <span>
            Math Master<span className="footer-dot">·</span><UiText notice={uiMessage("layout.a.world.of.ideas.84541f",{})}/></span>
          <span><UiText notice={uiMessage("layout.built.on.clear.thinking.and.connected.learning.9c141f",{})}/></span>
        </footer>
      </UiLocaleProvider></body>
    </html>
  );
}

export async function generateMetadata(){return getUiMetadata("page.home");}
