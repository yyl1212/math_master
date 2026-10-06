import {it,expect} from "vitest";
import {render,screen,fireEvent} from "@testing-library/react";
import {readFileSync} from "node:fs";
import {KnowledgeView} from "@/features/reading/knowledge-view";
import {UiLocaleProvider} from "./provider";
import {LanguageSwitch} from "@/components/language-switch";
it("public-ui-changes-with-identical-knowledge-body",()=>{const p=JSON.parse(readFileSync("../backend/internal/content/testdata/workflow-ready.json","utf8"));const k={...p.knowledge[0],title:"Save draft",proof:"A preserved original proof."};const data={knowledge:k,units:p.units,assets:[]};const before=JSON.stringify(data);render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><KnowledgeView result={{ok:true,data}}/></UiLocaleProvider>);fireEvent.click(screen.getByRole("button",{name:"中文"}));expect(screen.queryByRole("heading",{name:"证明"})).toBeInTheDocument();expect(screen.getByRole("heading",{name:"Save draft"})).toBeVisible();expect(screen.getByText("A preserved original proof.")).toBeVisible();expect(JSON.stringify(data)).toBe(before);});
