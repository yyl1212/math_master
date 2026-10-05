import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.login");}
import Link from "next/link";
import { headers } from "next/headers";
import { readServerSession } from "@/lib/auth/server-client";
import { CredentialsForm } from "@/features/auth/credentials-form";
import { AuthState } from "@/features/auth/auth-state";
export const dynamic = "force-dynamic";

export default async function Page() {
  const session = await readServerSession((await headers()).get("cookie") ?? "");
  if (!session.ok) return <><UiPageTitle messageKey="page.login"/>{<AuthState kind={session.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable"} />}</>;
  if (session.data) return <><UiPageTitle messageKey="page.login"/>{<section className="content-state"><h1>You are signed in</h1><p>{session.data.username}</p><Link className="button" prefetch={false} href="/account">View account</Link></section>}</>;
  return <><UiPageTitle messageKey="page.login"/>{<CredentialsForm mode="login" />}</>;
}
