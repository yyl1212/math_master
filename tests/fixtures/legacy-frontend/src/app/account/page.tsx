import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.account");}
import { headers } from "next/headers";
import { readServerSession } from "@/lib/auth/server-client";
import { AccountPanel } from "@/features/auth/account-panel";
import { AuthState } from "@/features/auth/auth-state";
export const dynamic = "force-dynamic";

export default async function Page() {
  const session = await readServerSession((await headers()).get("cookie") ?? "");
  if (!session.ok) return <><UiPageTitle messageKey="page.account"/>{<AuthState kind={session.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable"} />}</>;
  if (!session.data) return <><UiPageTitle messageKey="page.account"/>{<AuthState kind="anonymous" />}</>;
  return <><UiPageTitle messageKey="page.account"/>{<AccountPanel user={session.data} />}</>;
}
