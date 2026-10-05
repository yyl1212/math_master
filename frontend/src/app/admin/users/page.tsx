import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.admin.users");}
import { headers } from "next/headers";
import { readServerSession, readServerUsers } from "@/lib/auth/server-client";
import { AdminUsers } from "@/features/auth/admin-users";
import { AuthState } from "@/features/auth/auth-state";
export const dynamic = "force-dynamic";

export default async function Page() {
  const cookie = (await headers()).get("cookie") ?? "";
  const session = await readServerSession(cookie);
  if (!session.ok) return <><UiPageTitle messageKey="page.admin.users"/>{<AuthState kind={session.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable"} />}</>;
  if (!session.data) return <><UiPageTitle messageKey="page.admin.users"/>{<AuthState kind="anonymous" />}</>;
  if (session.data.mustChangePassword || !session.data.roles.includes("admin")) return <><UiPageTitle messageKey="page.admin.users"/>{<AuthState kind="forbidden" />}</>;
  const users = await readServerUsers(cookie, { q: "", limit: 20, offset: 0 });
  if (!users.ok) return <><UiPageTitle messageKey="page.admin.users"/>{<AuthState kind={users.code === "INVALID_COOKIE" ? "invalid-cookie" : users.status === 401 ? "anonymous" : users.status === 403 ? "forbidden" : "unavailable"} />}</>;
  return <><UiPageTitle messageKey="page.admin.users"/>{<AdminUsers initial={users.data} />}</>;
}
