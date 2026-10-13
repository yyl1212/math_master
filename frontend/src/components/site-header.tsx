"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import {LanguageSwitch} from "./language-switch";
import { AuthStatus } from "./auth-status";
import {NoteHistoryTracker} from '@/features/notes/note-leave-guard';
import { usePathname } from "next/navigation";
export function SiteHeader() {
 const {t}=useUiI18n();

  const path = usePathname();
  return (
    <header className="site-header">
      <NoteHistoryTracker/>
      <Link
        prefetch={false}
        href="/"
        className="brand"
        aria-label={t("site-header.math.master.home.f407d7",{})}
      >
        <svg viewBox="0 0 40 40" aria-hidden="true">
          <rect
            x="1"
            y="1"
            width="38"
            height="38"
            rx="12"
            fill="currentColor"
          />
          <path
            d="M10 28V13l10 9 10-9v15"
            stroke="#fff"
            fill="none"
            strokeWidth="2.3"
            strokeLinejoin="round"
          />
        </svg>
        <span>
          Math Master<small><UiText notice={uiMessage("site-header.a.world.of.ideas.883eb5",{})}/></small>
        </span>
      </Link>
      <nav aria-label={t("site-header.main.navigation.eb3559",{})} style={{flexWrap:"wrap",minWidth:0}}>
        <Link
          prefetch={false}
          href="/"
          aria-current={path === "/" ? "page" : undefined}
          data-active={path === "/"}
        ><UiText notice={uiMessage("site-header.learning.hub.0af988",{})}/></Link>
        <Link
          prefetch={false}
          href="/knowledge"
          aria-current={path === "/knowledge" ? "page" : undefined}
          data-active={path === "/knowledge" || path.startsWith("/knowledge/")}
        ><UiText notice={uiMessage("nav.knowledgeMap",{})}/></Link>
</nav>
      <LanguageSwitch /><AuthStatus />
    </header>
  );
}
