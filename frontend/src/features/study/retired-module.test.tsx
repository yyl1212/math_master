import {it,expect} from "vitest";
import {render,screen} from "@testing-library/react";
import {RetiredModule} from "./retired-module";
import {readLearningResponse} from "@/lib/learning/schemas";
import {readQuestionResponse} from "@/lib/question/schemas";
import {readPrivateCorrectionJSON} from "@/lib/correction/schemas";
import {UiLocaleProvider} from "@/lib/i18n/provider";
function retired(){const id="a".repeat(32);return new Response(JSON.stringify({error:{code:"MODULE_RETIRED",message:"This module has been retired.",requestId:id}}),{status:410,headers:{"Content-Type":"application/json","Cache-Control":"private, no-store","X-Content-Type-Options":"nosniff","X-Request-ID":id}})}
it("recognizes retired writes across the three original transports",async()=>{const signal=new AbortController().signal;expect(await readLearningResponse(retired(),{kind:"startKnowledge",id:"fractions"},signal)).toMatchObject({ok:false,code:"MODULE_RETIRED"});expect(await readQuestionResponse(retired(),"createDraft",signal)).toMatchObject({ok:false,code:"MODULE_RETIRED"});await expect(readPrivateCorrectionJSON(retired(),signal)).rejects.toMatchObject({code:"MODULE_RETIRED",status:410})});
it("retired UI has learning and protected archive entries but no commands",()=>{render(<UiLocaleProvider initialLocale="zh-CN"><RetiredModule/></UiLocaleProvider>);expect(screen.getByRole("heading",{name:"此模块已停用"})).toBeVisible();expect(screen.getByRole("link",{name:"我的学习"})).toHaveAttribute("href","/learn");expect(screen.getByRole("link",{name:"旧学习档案"})).toHaveAttribute("href","/learning-history?archive=legacy");expect(screen.queryByRole("button")).not.toBeInTheDocument()});
