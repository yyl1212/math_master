import Link from "next/link";
import { headers } from "next/headers";
import { readServerSession } from "@/lib/auth/server-client";
import { CredentialsForm } from "@/features/auth/credentials-form";
import { AuthState } from "@/features/auth/auth-state";
export const dynamic = "force-dynamic";
export const metadata = { title: "Sign in" };
export default async function Page() {
  const session = await readServerSession((await headers()).get("cookie") ?? "");
  if (!session.ok) return <AuthState kind={session.code === "INVALID_COOKIE" ? "invalid-cookie" : "unavailable"} />;
  if (session.data) return <section className="content-state"><h1>You are signed in</h1><p>{session.data.username}</p><Link className="button" prefetch={false} href="/account">View account</Link></section>;
  return <CredentialsForm mode="login" />;
}
