# KaTeX 安全升级与兼容核验

日期：2026-10-06。用户已确认本小范围兼容方案；合并仍须最新完整 CI 成功。

## 原因与范围

PR #36 的首次学习场景选择器问题修复后，新一轮前端生产构建成功，但 npm 生产依赖审计报告 4 项低危问题：KaTeX 0.16.47 及引用它的数学渲染依赖受到 GHSA-238p-pmpm-9mq7 影响。官方最低修复版为 0.18.2，见[安全公告](https://github.com/advisories/GHSA-238p-pmpm-9mq7)及[修复发布说明](https://github.com/KaTeX/KaTeX/releases/tag/v0.18.2)。

仅将既有 `katex` 固定为 0.18.2，在 `frontend/package.json` 使用 `overrides.katex = "$katex"` 统一直接与间接引用，并更新锁文件。实际依赖树中 `rehype-katex`、`micromark-extension-math` 均去重到同一修复版；其他包版本未改变。没有新增运行依赖。

```mermaid
flowchart LR
  A[知识与题库原文] --> B[现有 Markdown 渲染]
  B --> C[统一 KaTeX 0.18.2]
  C --> D[公式显示]
```

## 兼容边界与审查

跨越 0.17、0.18 的兼容风险已向用户说明并获确认：上游内部函数定义接口与内部 CSS 类有变更。项目没有使用内部函数接口，只使用渲染插件与保留的公共 CSS 入口、`.katex`、`.katex-display`、`katex-error`。发布包已核对公共导出与上述入口存在，实际显示仍通过浏览器核验。

`SafeMarkdown` 的 `trust: false`、`strict: "error"`、`maxExpand: 100`、`maxSize: 10`、独立宏对象、原 HTML/链接/图片过滤及公式长度限制均保持。源码差异审查确认没有修改应用渲染逻辑、CSS、语言词条、Go、OpenAPI、数据库或数学资料。

## 验证

升级前审计实际失败（4 项低危）；升级后同一生产审计成功（各严重度均为 0）。完整前端 403 项 / 88 文件、类型检查、生产构建通过。API 生成声明无字节变化。

本机实际浏览器逐批核对目录与阅读、账户安全、知识与题库作者、固定版本公式以及全部七类语言场景；结果见 `katex-security-upgrade.json`。沿用原双视口、单 worker、零重试、每批时限；所有数据库写入只在本次 55434 临时服务创建的随机隔离库。远端完整矩阵与最终合并结果以 PR 最新 head 的实际记录为准。
