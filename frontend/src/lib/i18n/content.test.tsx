import {render,screen,fireEvent,waitFor,cleanup} from "@testing-library/react";
import {it,expect,vi} from "vitest";
const mocks=vi.hoisted(()=>({request:vi.fn(),asset:vi.fn(),refresh:vi.fn(),push:vi.fn()}));
vi.mock("@/lib/content/client",()=>({contentRequest:mocks.request,readContentAsset:mocks.asset}));vi.mock("next/navigation",()=>({useRouter:()=>mocks}));
import {DraftEditor} from "@/features/content/draft-editor";import {ReviewPanel} from "@/features/content/review-panel";
import {draftView,submissionView,fixtureSVG,fixtureID} from "@/lib/content/test-fixtures";import {contentFailure} from "@/lib/content/schemas";
import {UiLocaleProvider} from "./provider";import {LanguageSwitch} from "@/components/language-switch";
it("dirty-draft-and-checked-review-preserved-across-toggle",()=>{
 mocks.asset.mockResolvedValue({ok:true,data:fixtureSVG});const d=draftView(),before=JSON.stringify(d);render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><DraftEditor initial={d}/></UiLocaleProvider>);
 const input=screen.getByLabelText("Knowledge 1 statement");fireEvent.change(input,{target:{value:"Save draft — original mathematical statement"}});fireEvent.click(screen.getByRole("button",{name:"中文"}));expect(screen.getByRole("button",{name:"保存草稿"})).toBeVisible();expect(input).toHaveValue("Save draft — original mathematical statement");expect(JSON.stringify(d)).toBe(before);cleanup();
 render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><ReviewPanel submission={submissionView()} user={{id:fixtureID,username:"reviewer",roles:["learner","reviewer"],mustChangePassword:false}}/></UiLocaleProvider>);const check=screen.getByLabelText("Mathematics");fireEvent.click(check);fireEvent.click(screen.getByRole("button",{name:"中文"}));expect(screen.getByLabelText("数学正确性")).toBe(check);expect(check).toBeChecked();
});
it("pending-content-request-keeps-original-body-and-key",async()=>{
 mocks.request.mockReset();mocks.asset.mockResolvedValue({ok:true,data:fixtureSVG});mocks.request.mockResolvedValue(contentFailure());render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><DraftEditor initial={draftView()}/></UiLocaleProvider>);
 fireEvent.click(screen.getByRole("button",{name:"Save draft"}));await waitFor(()=>expect(mocks.request).toHaveBeenCalledOnce());const call=mocks.request.mock.calls[0];await screen.findByRole("button",{name:"Retry previous request"});fireEvent.click(screen.getByRole("button",{name:"中文"}));expect(mocks.request).toHaveBeenCalledOnce();fireEvent.click(screen.getByRole("button",{name:"重试上次请求"}));await waitFor(()=>expect(mocks.request).toHaveBeenCalledTimes(2));expect(mocks.request.mock.calls[1]).toEqual(call);
});
