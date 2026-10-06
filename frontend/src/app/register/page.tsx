import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.register");}
import Link from "next/link";
import { headers } from "next/headers";
import { readServerSession } from "@/lib/auth/server-client";
import { CredentialsForm } from "@/features/auth/credentials-form";
import { AuthState } from "@/features/auth/auth-state";
export const dynamic = "force-dynamic";

export default async function Page() {
  const session = await readServerSession((await headers()).get("cookie") ?? "");
  if (!session.ok) return <><UiPageTitle messageKey="page.register"/>{<AuthState kind={session.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable"} />}</>;
  if (session.data) return <><UiPageTitle messageKey="page.register"/>{<section className="content-state"><h1><UiText notice={uiMessage("page.you.are.signed.in.2d55cb",{})}/></h1><p>{session.data.username}</p><Link className="button" prefetch={false} href="/account"><UiText notice={uiMessage("auth-state.view.account.407143",{})}/></Link></section>}</>;
  return <><UiPageTitle messageKey="page.register"/>{<CredentialsForm mode="register" />}</>;
}
