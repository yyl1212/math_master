import {it,expect,vi} from "vitest";
import {render,screen,fireEvent,waitFor} from "@testing-library/react";
vi.mock("next/navigation",()=>({useRouter:()=>({replace:vi.fn(),refresh:vi.fn()})}));
vi.mock("@/lib/auth/client",()=>({getAuthContext:vi.fn(async()=>({ok:false,code:"FORBIDDEN",message:"You do not have permission."})),authRequest:vi.fn(),notifyAuthChanged:vi.fn()}));
import {CredentialsForm} from "@/features/auth/credentials-form";
import {FormMessage} from "@/features/auth/auth-state";
import {SelectField} from "@/features/content/field-controls";
import {UiLocaleProvider} from "./provider";import {LanguageSwitch} from "@/components/language-switch";
import {uiMessage} from "./format";
it("auth-error-and-password-dialog-update-without-value-reset",async()=>{
 render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><CredentialsForm mode="login"/></UiLocaleProvider>);
 const input=screen.getByLabelText("Password");fireEvent.change(input,{target:{value:"Unsubmitted test-only password"}});await waitFor(()=>expect(screen.getByRole("alert")).toBeVisible());
 fireEvent.click(screen.getByRole("button",{name:"中文"}));expect(screen.getByLabelText("密码")).toBe(input);expect(input).toHaveValue("Unsubmitted test-only password");expect(screen.getByRole("alert")).toHaveTextContent("你没有此操作权限。");
});
it("role-label-changes-but-submits-original-code and literal notices stay literal",()=>{
 const change=vi.fn();render(<UiLocaleProvider initialLocale="zh-CN"><SelectField label={uiMessage("auth.roles",{})} value="reviewer" options={["reviewer","editor"]} optionLabels={{reviewer:uiMessage("auth.role.reviewer",{}),editor:uiMessage("auth.role.editor",{})}} onChange={change}/><FormMessage message="Save draft"/></UiLocaleProvider>);
 expect(screen.getByRole("option",{name:"审核者"})).toHaveValue("reviewer");fireEvent.change(screen.getByLabelText("角色"),{target:{value:"editor"}});expect(change).toHaveBeenCalledWith("editor");expect(screen.getByRole("status")).toHaveTextContent("Save draft");
});
