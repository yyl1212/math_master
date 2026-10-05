import {it,expect} from "vitest";
import {parseUiLocaleCookie,buildUiLocaleCookie} from "./config";
it.each([["","en"],["math_master_ui_locale=en","en"],["other=x; math_master_ui_locale=zh-CN","zh-CN"],["math_master_ui_locale=fr","en"],["math_master_ui_locale=en; math_master_ui_locale=zh-CN","en"]])("strict browser preference %s",(raw,want)=>expect(parseUiLocaleCookie(raw)).toBe(want));
it("cookie is a preference independent of authentication",()=>{expect(buildUiLocaleCookie("zh-CN",true)).toBe("math_master_ui_locale=zh-CN; Path=/; Max-Age=31536000; SameSite=Lax; Secure");expect(buildUiLocaleCookie("en",false)).toBe("math_master_ui_locale=en; Path=/; Max-Age=31536000; SameSite=Lax");});
