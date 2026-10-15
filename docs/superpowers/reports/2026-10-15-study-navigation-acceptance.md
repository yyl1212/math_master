# 学习工作区顶部导航修复验收

## 问题与修复

用户确认的问题是顶部栏目在切换页面后顺序改变。原LearningHeader按“知识学习、复习工作区、我的笔记、学习历史”排列，ReviewHeader按“知识学习、我的笔记、复习工作区、自评历史”排列，两处样式和选中字重也不同。

本次统一使用StudyNavigation，顺序固定为“知识学习、我的笔记、复习工作区、自评历史、学习历史”。两类历史仍前往各自既有URL。两个页头显式传入当前栏目，组件只渲染五个关闭预取的链接，不读取身份或业务数据。所有链接使用相同字重、尺寸和换行规则，仅当前页背景/颜色变化；导航自身最大宽度1100px并居中，抵消原页面容器宽度差异。

## 可行性与兼容性

根因为展示代码重复，无需改接口或数据结构，实施前审查无阻塞问题。保留两个Header的原props；标题、筛选和业务命令不改。后端、迁移、OpenAPI、业务客户端/hook、路由、依赖、原workflow、Playwright预算和固定旧前端fixture均无差异。

兼容策略仅登记本次准确文件摘要与新增文件。原始files、legacySnapshots、apiSections、apiExceptions及managedOriginalMigrations与master一致；不重置基线，不扩大路径通配豁免。共享组件使用已有六个文案键，并登记原语言场景覆盖。

## 验证记录

环境为仓库锁定的Node/Next/React/Playwright；本机独立PostgreSQL17.11合成测试库，单worker、零重试，保留原480秒浏览器总预算及30秒用例限制。以下命令统一经 `node tools/verify/run.mjs` 限时执行；Go联调构建使用 `CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1`。

| 验证 | 实际结果 |
| --- | --- |
| 旧实现新增Header行为测试 | 10/10失败，实际缺少链接且顺序不一致；不是导入或环境错误 |
| 共享导航接入后同一测试 | 10/10通过 |
| 完整 `npm test` | 148文件、659项通过 |
| `npm run typecheck`、`npm run build`、联调harness构建 | 全部退出0 |
| `TOPIC_COMPATIBILITY_STAGE=managed node --test tools/verify/*.test.mjs` | 103/103通过；首次受沙箱本机监听限制，允许监听后原命令通过，无代码调整 |
| 当前 `page-beauty-learning.spec.ts` | 46/46通过，2.2分钟 |
| 当前 `page-beauty.playwright.config.ts` | Firefox/WebKit桌面与手机共28/28通过，1.1分钟 |
| 原 `study-review-{flow,recall,security}.spec.ts` | 30/30通过，1.9分钟 |
| 原 `note-sharing.spec.ts`、`note-sharing-security.spec.ts`、`site-streamline.spec.ts` | 26/26通过，1.6分钟 |

新增浏览器用例从实际导航点击五个页面，验证中英文的链接顺序与名称、唯一aria-current、绝对水平位置、相对行位置、尺寸、至少44px目标、Tab顺序及页面不横溢。Chromium覆盖1280/390/320；Firefox/WebKit各覆盖桌面/手机及320。几何比较容许1px引擎取整误差，不比较因各页标题说明不同而改变的绝对纵向位置。

截图仅含合成账号和原创测试语料，输入遮挡；本地证据目录 `/private/tmp/math-master-study-navigation-20261015/capture`。`navigation-fixed-{1,2}-zh-CN-{1280,390}.png` 对应“我的笔记”和“复习工作区”；导航顺序相同，背景随当前栏目切换。新截图不覆盖上一阶段基线或验收记录。

## 旧用例环境诊断

额外运行 `ui-language-managed.spec.ts` 时误用了当前前端：该组与上述笔记/导航合跑为31项通过、1项手机用例失败，失败点是在未切换笔记栏目时填写隐藏的Private note。准确master caf2da04的独立源码归档、独立生产构建运行同一用例，也在同一位置复现30秒不可见超时；相关阅读组件、手机折叠CSS和旧测试均与master字节相同。原CI在固定旧前端运行这组用例，本分支保持其配置及断言；当前版输入/语言/折叠由已通过的学习视觉、跨浏览器和复习回归覆盖。共享组件的语言清单仅指向实际运行当前前端的study-review-flow场景。诊断失败不计入上表通过数，也不将其当作本次新增回归。

## 交付边界

本报告记录本地验证，准确HEAD的远端CI及MR地址由当前会话交付。该修复不自行合并或部署生产。
