import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CredentialsForm } from "./credentials-form";
import { AccountPanel } from "./account-panel";
import { failure } from "@/lib/auth/schemas";
const mocks = vi.hoisted(() => ({ replace: vi.fn(), refresh: vi.fn(), request: vi.fn(), context: vi.fn(), changed: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => mocks }));
vi.mock("@/lib/auth/client", () => ({ authRequest: mocks.request, getAuthContext: mocks.context, notifyAuthChanged: mocks.changed }));
const user = { id: "00000000-0000-4000-8000-000000000001", username: "learner_one", roles: ["learner"] as const, mustChangePassword: false };
const password = "  中文数学密码保留所有空格与字符  ";
beforeEach(() => { Object.values(mocks).forEach(m => m.mockReset()); mocks.context.mockResolvedValue({ ok: true, data: { user: null, csrfToken: "A".repeat(43) } }); });
describe("TestCredentialForms", () => {
  it.each(["register", "login"] as const)("preserves pasted Unicode and blocks duplicate %s writes", async mode => {
    render(<CredentialsForm mode={mode} />);
    await waitFor(() => expect(mocks.context).toHaveBeenCalled());
    expect(screen.getByLabelText("Username")).toHaveAttribute("autocomplete", "username");
    expect(screen.getByLabelText("Password")).toHaveAttribute("autocomplete", mode === "register" ? "new-password" : "current-password");
    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "Learner_One" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: password } });
    let resolve!: (value: unknown) => void;
    mocks.request.mockReturnValue(new Promise(r => { resolve = r; }));
    const form = screen.getByLabelText("Password").closest("form")!;
    fireEvent.submit(form); fireEvent.submit(form);
    expect(mocks.request).toHaveBeenCalledTimes(1);
    expect(mocks.request).toHaveBeenCalledWith({ kind: mode }, { username: "Learner_One", password });
    resolve({ ok: true, data: user });
    await waitFor(() => expect(mocks.replace).toHaveBeenCalledWith(mode === "register" ? "/login" : "/learn"));
    expect(screen.getByLabelText("Password")).toHaveValue("");
    expect(mocks.changed).toHaveBeenCalledTimes(1);
  });
  it("focuses a fixed error and clears the password without navigating", async () => {
    mocks.request.mockResolvedValue(failure("INVALID_CREDENTIALS"));
    render(<CredentialsForm mode="login" />);
    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "learner_one" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: password } });
    fireEvent.submit(screen.getByLabelText("Password").closest("form")!);
    await waitFor(() => expect(screen.getByRole("alert")).toHaveFocus());
    expect(screen.getByRole("alert")).toHaveTextContent("Invalid username or password.");
    expect(screen.getByLabelText("Password")).toHaveValue("");
    expect(document.body.textContent).not.toContain(password);
    expect(mocks.replace).not.toHaveBeenCalled();
  });
});
describe("TestAccountPanel", () => {
  it("shows real identity, preserves sign-in on wrong password, then changes and signs out", async () => {
    render(<AccountPanel user={{ ...user, roles: [...user.roles] }} />);
    expect(screen.getByText("learner_one")).toBeVisible();
    expect(screen.queryByText(/progress|mastered|points/i)).not.toBeInTheDocument();
    mocks.request.mockResolvedValueOnce(failure("INVALID_CREDENTIALS")).mockResolvedValueOnce({ ok: true, data: undefined });
    for (let attempt = 0; attempt < 2; attempt++) {
      fireEvent.change(screen.getByLabelText("Current password"), { target: { value: password } });
      fireEvent.change(screen.getByLabelText("New password"), { target: { value: password + "new" } });
      fireEvent.submit(screen.getByLabelText("New password").closest("form")!);
      await waitFor(() => expect(screen.getByLabelText("Current password")).toHaveValue(""));
      if (!attempt) expect(mocks.replace).not.toHaveBeenCalled();
    }
    expect(mocks.replace).toHaveBeenCalledWith("/login");
    expect(mocks.changed).toHaveBeenCalled();
  });
  it("restricts a temporary session and prevents duplicate sign-out", async () => {
    render(<AccountPanel user={{ ...user, roles: ["learner", "admin"], mustChangePassword: true }} />);
    expect(screen.getByText("Change your password to continue")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Sign out everywhere" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Manage users" })).not.toBeInTheDocument();
    let resolve!: (value: unknown) => void;
    mocks.request.mockReturnValue(new Promise(r => { resolve = r; }));
    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));
    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(mocks.request).toHaveBeenCalledTimes(1);
    resolve({ ok: true, data: undefined });
    await waitFor(() => expect(mocks.replace).toHaveBeenCalledWith("/"));
  });
});

it("temporary password login enters the account password flow",async()=>{mocks.request.mockResolvedValue({ok:true,data:{...user,mustChangePassword:true}});render(<CredentialsForm mode="login"/>);fireEvent.change(screen.getByLabelText("Username"),{target:{value:"learner_one"}});fireEvent.change(screen.getByLabelText("Password"),{target:{value:password}});fireEvent.submit(screen.getByLabelText("Password").closest("form")!);await waitFor(()=>expect(mocks.replace).toHaveBeenCalledWith("/account"))});
