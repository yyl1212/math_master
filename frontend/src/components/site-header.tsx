"use client";
import Link from "next/link";
import { AuthStatus } from "./auth-status";
import { usePathname } from "next/navigation";
export function SiteHeader() {
  const path = usePathname();
  return (
    <header className="site-header">
      <Link
        prefetch={false}
        href="/"
        className="brand"
        aria-label="Math Master home"
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
          Math Master<small>A WORLD OF IDEAS</small>
        </span>
      </Link>
      <nav aria-label="Main navigation" style={{flexWrap:"wrap",minWidth:0}}>
        <Link
          prefetch={false}
          href="/"
          aria-current={path === "/" ? "page" : undefined}
          data-active={path === "/"}
        >
          Learning Hub
        </Link>
        <Link
          prefetch={false}
          href="/knowledge"
          aria-current={path === "/knowledge" ? "page" : undefined}
          data-active={path === "/knowledge" || path.startsWith("/knowledge/")}
        >
          Knowledge Map
        </Link>
        <Link prefetch={false} href="/learn" aria-current={path==="/learn"?"page":undefined} data-active={path==="/learn"}>Learn</Link>
      <Link prefetch={false} href="/feedback" data-active={path.startsWith('/feedback')}>My reports</Link><Link prefetch={false} href="/review/feedback" data-active={path.startsWith('/review/feedback')}>Feedback review</Link><Link prefetch={false} href="/feedback/new?kind=site&area=other">Website feedback</Link></nav>
      <AuthStatus />
    </header>
  );
}
