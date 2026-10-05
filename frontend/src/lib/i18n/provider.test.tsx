import {it,expect,vi} from "vitest";
import {render,screen,fireEvent} from "@testing-library/react";
import {useEffect,useState} from "react";
import {UiLocaleProvider,useUiI18n} from "./provider";
import {LanguageSwitch} from "@/components/language-switch";
import {UiText} from "./ui-text";
import {uiMessage} from "./format";
function Dirty({mounted}:{mounted:()=>void}){const[v,set]=useState("Original Save draft");useEffect(mounted,[]);return <><label>raw<input value={v} onChange={e=>set(e.target.value)}/></label><UiText notice={uiMessage("nav.knowledgeMap",{})}/></>}
it("locale-switch-keeps-dirty-input-and-component-instance",()=>{const mounted=vi.fn();render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><Dirty mounted={mounted}/></UiLocaleProvider>);fireEvent.change(screen.getByLabelText("raw"),{target:{value:"Unsubmitted 中文 data"}});fireEvent.click(screen.getByRole("button",{name:"中文",exact:true}));expect(screen.getByText("知识地图")).toBeVisible();expect(screen.getByLabelText("raw")).toHaveValue("Unsubmitted 中文 data");expect(mounted).toHaveBeenCalledTimes(1);expect(document.documentElement.lang).toBe("zh-CN");});
it("cookie-write-failure-keeps-current-page-selection",()=>{vi.spyOn(document,"cookie","set").mockImplementation(()=>{throw new Error("blocked")});render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><UiText notice={uiMessage("nav.knowledgeMap",{})}/></UiLocaleProvider>);fireEvent.click(screen.getByRole("button",{name:"中文",exact:true}));expect(screen.getByText("知识地图")).toBeVisible();});
