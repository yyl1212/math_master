import {it,expect} from "vitest";
import {uiError} from "./errors";
import {formatUiNotice} from "./format";
it("validated wire error remains English while display follows locale",()=>{const f={code:"FORBIDDEN",message:"You do not have permission.",requestId:"unavailable"};expect(formatUiNotice("zh-CN",uiError("auth",f))).toBe("你没有此操作权限。");expect(f.message).toBe("You do not have permission.");});
it("namespaces keep different meanings and unknown codes are safe",()=>{expect(formatUiNotice("zh-CN",uiError("auth",{code:"NOT_FOUND"}))).toBe("未找到资源。");expect(formatUiNotice("zh-CN",uiError("feedback",{code:"NOT_FOUND"}))).toBe("反馈不存在或不再可用。");expect(formatUiNotice("zh-CN",uiError("content",{code:"unknown-private-detail"}))).toBe("服务暂时不可用。");});
