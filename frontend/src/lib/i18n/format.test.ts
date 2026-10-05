import {it,expect} from "vitest";
import {formatUiNotice,uiMessage} from "./format";
import {enMessages} from "./messages/en";
import {zhMessages} from "./messages/zh-CN";
it("system labels translate and data never becomes a lookup key",()=>{expect(formatUiNotice("zh-CN",uiMessage("nav.knowledgeMap",{}))).toBe("知识地图");expect(formatUiNotice("zh-CN",{kind:"literal",text:"Save draft"})).toBe("Save draft");expect(formatUiNotice("zh-CN",uiMessage("common.itemNumber",{number:2}))).toBe("第 2 项");});
it("untrusted parameter text stays literal",()=>expect(formatUiNotice("zh-CN",uiMessage("common.namedItem",{name:"<script>Save draft</script>"}))).toBe("项目：<script>Save draft</script>"));
it("Chinese keys and parameters match English",()=>{expect(Object.keys(zhMessages).sort()).toEqual(Object.keys(enMessages).sort());for(const[k,v]of Object.entries(enMessages)){expect([...zhMessages[k as keyof typeof enMessages].matchAll(/\{([a-zA-Z0-9_]+)\}/g)].map(m=>m[1]).sort()).toEqual([...v.params].sort());}});

// These assertions are compiled by typecheck; no runtime fabrication is allowed.
function messageTypeContract(){
 // @ts-expect-error unknown UI key
 uiMessage("not.a.real.key",{});
 // @ts-expect-error required parameter is missing
 uiMessage("common.itemNumber",{});
 // @ts-expect-error parameter must be text or a number
 uiMessage("common.itemNumber",{number:{}});
}
void messageTypeContract;
it("compact notices retain their serialized shape",()=>expect(JSON.parse(JSON.stringify(uiMessage("common.itemNumber",{number:2})))).toEqual({kind:"system",key:"common.itemNumber",values:{number:2}}));
