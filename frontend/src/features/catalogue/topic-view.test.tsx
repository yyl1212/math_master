import {render,screen,fireEvent} from "@testing-library/react";
import {it,expect} from "vitest";
import {TopicMap,TopicView} from "./topic-view";
import {UiLocaleProvider} from "@/lib/i18n/provider";
const pair={knowledgeHead:null,taxonomyHead:"11111111-1111-4111-8111-111111111111",taxonomyVersionId:"a".repeat(64)},root={id:"msc-13",code:"13-XX",name:"Original fixture theme",nameZh:"原创主题",kind:"primary" as const,level:1 as const,parentId:null};
const leaf={id:"msc-13c60",code:"13C60",name:"Original fixture specific",nameZh:"具体主题示例",kind:"primary" as const,level:3 as const,parentId:"msc-13c",ancestors:[root,{...root,id:"msc-13c",code:"13Cxx",nameZh:"原创二级主题",level:2 as const,parentId:"msc-13"}],hasChildren:false,publishedKnowledgeCount:0};
it("shows current classification totals and paged topic navigation without learning paths",()=>{
 render(<TopicMap result={{ok:true,data:{items:[{...root,ancestors:[],hasChildren:true,publishedKnowledgeCount:0}],total:63,limit:20,offset:0,pair}}} q=""/>);
 expect(screen.queryByText(/4,969/)).toBeVisible();expect(screen.getByRole("link",{name:"Original fixture theme"})).toHaveAttribute("href","/topics/msc-13");expect(screen.queryByRole("link",{name:/path/i})).not.toBeInTheDocument();expect(screen.getByRole("link",{name:"Next topics"})).toHaveAttribute("href",expect.stringContaining("offset=20"));
});
it("shows a deep link ancestry and empty reviewed content in Chinese",()=>{
 render(<UiLocaleProvider initialLocale="zh-CN"><TopicView detail={{summary:leaf,pair}} childrenPage={{items:[],total:0,limit:20,offset:0,pair}} knowledge={{items:[],total:0,limit:20,offset:0,pair}}/></UiLocaleProvider>);
 expect(screen.getByRole("link",{name:"原创主题"})).toHaveAttribute("href","/topics/msc-13");expect(screen.queryByText("此主题暂无已发布知识点。")).toBeVisible();expect(screen.getAllByText("13C60",{exact:true})).toHaveLength(1);
});
