"use client";
import { createContext } from "react";
// The owner comes from the server-rendered private page, never the next login.
export const LearningAccountContext = createContext<{ actorId: string; invalidate: () => void } | null>(null);
