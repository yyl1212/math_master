import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthStatus } from "./auth-status";
import { SiteHeader } from "./site-header";
import { CredentialsForm } from "@/features/auth/credentials-form";
import { failure } from "@/lib/auth/schemas";
const modeFixture=vi.hoisted(()=>({managed:false as boolean|null,mode:"legacy" as "legacy"|"topics"}));
vi.mock("@/lib/knowledge-admin/mode-client",()=>({useContentMode:()=>modeFixture.managed}));
vi.mock("@/lib/taxonomy/client",()=>({taxonomyRequest:vi.fn(async()=>({ok:true,data:{mode:modeFixture.mode}}))}));
vi.mock("next/navigation", () => ({ usePathname: () => "/login", useRouter: () => ({ replace: vi.fn(), refresh: vi.fn() }) }));
afterEach(() => {vi.unstubAllGlobals();modeFixture.mode="legacy";modeFixture.managed=false});
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
it("QuestionRoleNavigationDoesNotGrantEditorOrReviewerToAdmins",async()=>{
 vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({data:{user:{id:"11111111-1111-4111-8111-111111111111",username:"question_admin",roles:["learner","admin"],mustChangePassword:false},csrfToken:"A".repeat(43)}})));
 render(<AuthStatus/>);expect(await screen.findByRole("link",{name:"Publish question bank"})).toHaveAttribute("href","/admin/question-publications");expect(screen.queryByRole("link",{name:"Write questions"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"Review questions"})).not.toBeInTheDocument()
});

it("topics hides old question routes while retaining account and content management",async()=>{modeFixture.mode="topics";vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({data:{user:{id:"11111111-1111-4111-8111-111111111111",username:"topic_manager",roles:["learner","editor","reviewer","admin"],mustChangePassword:false},csrfToken:"A".repeat(43)}})));render(<AuthStatus/>);await screen.findByRole("link",{name:"Your account"});expect(screen.queryByRole("link",{name:"Publish question bank"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"Write questions"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"Review questions"})).not.toBeInTheDocument();expect(screen.getByRole("link",{name:"Edit content"})).toBeVisible()});

it("managed hides retired review and question menus for an administrator",async()=>{modeFixture.managed=true;vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({data:{user:{id:"11111111-1111-4111-8111-111111111111",username:"manager",roles:["learner","editor","reviewer","admin"],mustChangePassword:false},csrfToken:"A".repeat(43)}})));render(<AuthStatus/>);expect(await screen.findByRole("link",{name:"Knowledge management"})).toHaveAttribute("href","/admin/knowledge");for(const name of ["Edit content","Review content","Publish question bank","Write questions","Review questions"])expect(screen.queryByRole("link",{name})).toBeNull()});
