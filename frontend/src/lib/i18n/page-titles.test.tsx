// @vitest-environment jsdom
import {test,expect,vi} from "vitest";
import {render,screen,fireEvent,waitFor} from "@testing-library/react";
const route=vi.hoisted(()=>({path:"/knowledge"}));
vi.mock("next/navigation",()=>({usePathname:()=>route.path}));
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

test("a reused page title follows client navigation with an unsaved locale preference",()=>{
 route.path="/knowledge";const view=render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><UiPageTitle messageKey="page.knowledge"/></UiLocaleProvider>);fireEvent.click(screen.getByRole("button",{name:"中文"}));route.path="/knowledge/another";document.title="Knowledge | Math Master";
 view.rerender(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><UiPageTitle messageKey="page.knowledge.id"/></UiLocaleProvider>);expect(document.title).toBe("知识阅读 | Math Master");
});
