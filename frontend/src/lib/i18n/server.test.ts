import {it,expect,vi} from "vitest";
import {readRequestUiLocale} from "./server";
const state=vi.hoisted(()=>({value:""}));
vi.mock("next/headers",()=>({headers:async()=>new Headers({cookie:state.value})}));
it("separate-request-locales-do-not-share-state",async()=>{state.value="math_master_ui_locale=zh-CN";expect(await readRequestUiLocale()).toBe("zh-CN");state.value="math_master_ui_locale=en";expect(await readRequestUiLocale()).toBe("en");state.value="math_master_ui_locale=en; math_master_ui_locale=zh-CN";expect(await readRequestUiLocale()).toBe("en");});
