import "server-only";
import { headers } from "next/headers";
import { readServerSession } from "@/lib/auth/server-client";
import type { Role, User } from "@/lib/auth/types";
import { ContentState } from "./content-state";
export async function contentPageAccess(roles: Role[]): Promise<{
    user: User;
    cookie: string;
} | {
    error: React.ReactNode;
}> { const cookie = (await headers()).get("cookie") ?? "", session = await readServerSession(cookie); if (!session.ok)
    return { error: <ContentState status={session.status} code={session.code}/> }; if (!session.data)
    return { error: <ContentState status={401}/> }; if (session.data.mustChangePassword || !roles.some(r => session.data!.roles.includes(r)))
    return { error: <ContentState status={403}/> }; return { user: session.data, cookie }; }
