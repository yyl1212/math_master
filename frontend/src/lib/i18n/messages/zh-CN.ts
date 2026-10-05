import type {MessageKey} from "../types";
export const zhMessages = {
  "nav.knowledgeMap": "知识地图",
  "common.itemNumber": "第 {number} 项",
  "common.namedItem": "项目：{name}",
  "common.unavailable": "服务暂时不可用。",
  "auth.error.forbidden": "你没有此操作权限。",
  "auth.error.notFound": "未找到资源。",
  "feedback.error.notFound": "反馈不存在或不再可用。",
  "locale.label": "界面语言",
  "locale.english": "English",
  "locale.chinese": "中文",
  "locale.selection": "当前语言：{language}"
} satisfies Record<MessageKey,string>;
