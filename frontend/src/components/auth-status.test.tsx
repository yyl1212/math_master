import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthStatus } from "./auth-status";
import { SiteHeader } from "./site-header";
import { CredentialsForm } from "@/features/auth/credentials-form";
import { failure } from "@/lib/auth/schemas";
import {UiLocaleProvider,useUiI18n} from '@/lib/i18n/provider';
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
 render(<AuthStatus/>);expect(await screen.findByRole("link",{name:"Knowledge management"})).toHaveAttribute("href","/admin/knowledge");expect(screen.queryByRole("link",{name:"Publish question bank"})).toBeNull();expect(screen.queryByRole("link",{name:"Write questions"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"Review questions"})).not.toBeInTheDocument()
});

it("topics hides old question routes while retaining account and content management",async()=>{modeFixture.mode="topics";vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({data:{user:{id:"11111111-1111-4111-8111-111111111111",username:"topic_manager",roles:["learner","editor","reviewer","admin"],mustChangePassword:false},csrfToken:"A".repeat(43)}})));render(<AuthStatus/>);expect(await screen.findByRole("link",{name:"topic_manager"})).toHaveAttribute('href','/account');expect(screen.queryByRole("link",{name:"Publish question bank"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"Write questions"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"Review questions"})).not.toBeInTheDocument();expect(screen.queryByRole("link",{name:"Edit content"})).toBeNull();expect(screen.getByRole("link",{name:"Knowledge management"})).toBeVisible()});

it('opens account settings through the username while keeping one learning entry in both languages',async()=>{
 vi.stubGlobal('fetch',vi.fn().mockResolvedValue(Response.json({data:{user:{id:'11111111-1111-4111-8111-111111111111',username:'student',roles:['learner'],mustChangePassword:false},csrfToken:'A'.repeat(43)}})));
 function Language(){const{setLocale}=useUiI18n();return <button onClick={()=>setLocale('zh-CN')}>中文</button>}
 render(<UiLocaleProvider initialLocale="en"><Language/><AuthStatus/></UiLocaleProvider>);
 expect(await screen.findByRole('link',{name:'student'})).toHaveAttribute('href','/account');
 expect(screen.getByRole('link',{name:'My learning'})).toHaveAttribute('href','/learn');
 expect(screen.queryByRole('link',{name:'Your account'})).toBeNull();expect(screen.queryByRole('link',{name:'Learning history'})).toBeNull();
 screen.getByRole('button',{name:'中文'}).click();await screen.findByRole('link',{name:'我的学习'});
 expect(screen.queryByRole('link',{name:'账户'})).toBeNull();expect(screen.queryByRole('link',{name:'学习历史'})).toBeNull();
});

it("managed hides retired review and question menus for an administrator",async()=>{modeFixture.managed=true;vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({data:{user:{id:"11111111-1111-4111-8111-111111111111",username:"manager",roles:["learner","editor","reviewer","admin"],mustChangePassword:false},csrfToken:"A".repeat(43)}})));render(<AuthStatus/>);expect(await screen.findByRole("link",{name:"Knowledge management"})).toHaveAttribute("href","/admin/knowledge");for(const name of ["Edit content","Review content","Publish question bank","Write questions","Review questions"])expect(screen.queryByRole("link",{name})).toBeNull()});

it('ordinary production navigation hides administrator and retired entry points',async()=>{modeFixture.managed=true;vi.stubGlobal('fetch',vi.fn().mockResolvedValue(Response.json({data:{user:{id:'11111111-1111-4111-8111-111111111111',username:'student',roles:['learner','reviewer'],mustChangePassword:false},csrfToken:'A'.repeat(43)}})));render(<SiteHeader/>);await screen.findByRole('link',{name:'student'});for(const name of ['Feedback review','Notifications','Corrections review','Website feedback'])expect(screen.queryByRole('link',{name})).toBeNull();expect(screen.getAllByRole('link',{name:'My learning'})).toHaveLength(1);});
