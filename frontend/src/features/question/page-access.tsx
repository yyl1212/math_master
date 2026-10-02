import "server-only";
import { headers } from "next/headers";
import { readServerSession } from "@/lib/auth/server-client";
import type { Role, User } from "@/lib/auth/types";
import { QuestionState } from "./question-state";
export async function questionPageAccess(roles: Role[]): Promise<{
    user: User;
    cookie: string;
} | {
    error: React.ReactNode;
}> { const cookie = (await headers()).get("cookie") ?? "", session = await readServerSession(cookie); if (!session.ok)
    return { error: <QuestionState status={session.status} code={session.code}/> }; if (!session.data)
    return { error: <QuestionState status={401}/> }; if (session.data.mustChangePassword || !roles.some(r => session.data!.roles.includes(r)))
    return { error: <QuestionState status={403}/> }; return { user: session.data, cookie }; }
