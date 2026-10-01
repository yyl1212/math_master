import { headers } from "next/headers";
import { readServerSession } from "@/lib/auth/server-client";
import { AccountPanel } from "@/features/auth/account-panel";
import { AuthState } from "@/features/auth/auth-state";
export const dynamic = "force-dynamic";
export const metadata = { title: "Your account" };
export default async function Page() {
  const session = await readServerSession((await headers()).get("cookie") ?? "");
  if (!session.ok) return <AuthState kind={session.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable"} />;
  if (!session.data) return <AuthState kind="anonymous" />;
  return <AccountPanel user={session.data} />;
}
