import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthStatus } from "./auth-status";
import { SiteHeader } from "./site-header";
import { CredentialsForm } from "@/features/auth/credentials-form";
import { failure } from "@/lib/auth/schemas";
vi.mock("next/navigation", () => ({ usePathname: () => "/login", useRouter: () => ({ replace: vi.fn(), refresh: vi.fn() }) }));
afterEach(() => vi.unstubAllGlobals());
describe("TestAuthStatus", () => {
  it("shares the initial context with the form and refreshes on auth change", async () => {
    let resolve!: (value: Response) => void;
    const fetcher = vi.fn().mockReturnValue(new Promise(r => { resolve = r; }));
    vi.stubGlobal("fetch", fetcher);
    render(<><AuthStatus /><CredentialsForm mode="login" /></>);
    expect(fetcher).toHaveBeenCalledTimes(1);
    resolve(Response.json({ data: { user: null, csrfToken: "A".repeat(43) } }));
    await screen.findByRole("link", { name: "Sign in" });
    fetcher.mockResolvedValue(Response.json({ error: { code: "SERVICE_UNAVAILABLE", message: failure().message, requestId: "a".repeat(32) } }, { status: 503 }));
    window.dispatchEvent(new Event("math-master:auth-change"));
    await screen.findByText("Accounts unavailable");
    expect(screen.queryByRole("link", { name: "Sign in" })).not.toBeInTheDocument();
  });
  it("does not select Knowledge Map on account pages", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ data: { user: null, csrfToken: "A".repeat(43) } })));
    render(<SiteHeader />);
    expect(screen.getByRole("link", { name: "Knowledge Map" })).toHaveAttribute("data-active", "false");
    await waitFor(() => expect(screen.getByRole("link", { name: "Sign in" })).toBeVisible());
  });
});
