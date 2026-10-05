// @vitest-environment jsdom
import {test,expect,vi} from "vitest";
import {render,screen,fireEvent,waitFor} from "@testing-library/react";
vi.mock("next/navigation",()=>({usePathname:()=>"/knowledge"}));
import {UiPageTitle} from "@/components/ui-page-title";
import {UiLocaleProvider} from "./provider";
import {LanguageSwitch} from "@/components/language-switch";
test("functional title follows locale and remains stable after streamed metadata",async()=>{
 render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><UiPageTitle messageKey="page.knowledge"/></UiLocaleProvider>);
 expect(document.title).toBe("Knowledge Map | Math Master");
 fireEvent.click(screen.getByRole("button",{name:"中文"}));
 expect(document.title).toBe("知识地图 | Math Master");
 document.title="Knowledge Map | Math Master";
 await waitFor(()=>expect(document.title).toBe("知识地图 | Math Master"));
});
