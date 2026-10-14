# 第三阶段：页面美感优化实施计划

> **执行要求：** 用户确认方案和本计划后，沿用此前Native方式，由当前实施者按任务执行 `superpowers:executing-plans`，完成后做一次新鲜的全分支代码与兼容审查。以下复选项是未来开发工作，不代表本轮已完成。

**目标：** 采用已选的安静学术阅读风格，优化全部当前页面，重点改善阅读、笔记和复习的文字、层级及手机体验。

**架构：** 保留现有路由、SSR、身份边界及命令状态机，以CSS令牌、无状态PageHeading和可选study正文外观统一呈现。通过独立新workflow验证视觉，不改原workflow或固定旧前端fixture。

**技术栈：** 仓库锁定的Next16/React19、TypeScript、CSS Module、KaTeX及Playwright；不新增产品依赖。

**方案：** [完整方案](../specs/2026-10-14-page-beauty-design.md)。[逐页表](../reports/2026-10-14-page-beauty-page-audit.md)的A01—A25及26个旧入口是本计划的覆盖清单。

**路径约定：** `styles/`、`components/`、`features/` 简写指 `frontend/src/` 下的路径；`tests/`、`docs/`、`tools/`、`.github/` 指仓库根路径。

## 全局约束

- 当前数据库15保持；backend、迁移、OpenAPI、生成类型、业务客户端/Schema/命令hook、生产运维和依赖锁文件不变。
- 现有全部workflow、`tests/e2e/playwright.config.ts`、固定旧前端fixture及其manifest保持字节不变；新增 `.github/workflows/page-beauty.yml`。
- 当前正文桌面17px/1.9、手机16px/1.85；普通功能文字至少13px；文本对比≥4.5:1，交互目标44px、表单主控件48px。
- 390宽标准夹具中英文页头≤184px；320/长用户名/管理员展开态不裁内容、不横溢。读取/比对断点分别保持1100/800。
- 原笔记审批/公开最新10条/撤回再修改、原四学习状态、原排期、揭示前无参考、空答仅忘记、命令确认及身份清理全部保持。
- 单次命令≤540秒；Go `CGO_ENABLED=0`、测试 `-timeout 5m`。原Playwright480秒、单worker、零重试、原两个视口均不变。
- CSS gzip总量增量≤10KiB；代表路由初始Client JS增量≤8KiB。无远程字体、新资源域或第三方插图；新增小图形只用原创SVG/CSS。
- 视觉断言对实际渲染的尺寸/对比/焦点和业务行为负责，不写逐个CSS声明的镜像单元测试，不用旧像素完全一致阻止有意改版。

## 审查重点

1. 长中文/英文、320窄屏、200%文字放大及真实缩放：文字不裁剪、焦点/错误仍可见，公式只在局部滚动（T2/T3/T7）。
2. 未保存笔记/默写、切语言/切布局/浏览器返回：输入和焦点保留，不能因新标题或双份DOM卸载状态（T3/T4）。
3. 揭示前、知识更新/下架、换账户及迟到响应：私人内容不被视觉重排预取或残留（T3/T4/T7）。
4. 管理员长表单、审批意见、对话框、未知请求：原禁用/确认/幂等及焦点恢复保持（T5/T6）。
5. 旧兼容前端继承全局CSS、原CI末尾job限制：不修改固定fixture/原workflow；新workflow单独有验证且不能漏跑（T1/T7/T8）。

## 执行顺序与开发准入

- [ ] 用户确认四份书面成果。当前仅已确认风格，不能把风格选择当作开发批准。
- [ ] `git fetch origin master`，复核与方案基线差异；复用现有隔离worktree，从届时最新master创建 `codex/page-beauty` 并带入文档，不在master开发。
- [ ] 按Native流程建本计划专属ledger；依次T1→T2→T3→T4→T5→T6→T7→T8。建议三个验收批次为基础/阅读、其余页面、完整回归；最终一份MR。
- [ ] 每任务提交能审查的增量，并同步该次实际改动的兼容精确路径/摘要登记；出现需要改API、业务hook、原CI或固定fixture的情况，先修订设计与兼容审查，不能以美化名义顺手重构。

## T1：补齐基线、状态清单与有界视觉工具

**文件：** 新 `tests/e2e/page-beauty-helpers.ts`、`docs/operations/evidence/page-beauty/baseline.json`；在三个新spec文件中先建立场景/基线采集。现有fixtures/scene-control只复用，不修改harness。

**接口：** `measurePage(page: Page): Promise<{viewportWidth:number; pageWidth:number; headerHeight:number}>`；`captureAudit(page:Page, info:TestInfo, label:string):Promise<void>`。后续任务直接使用这些函数，不另起浏览器或数据库。基线JSON固定 `sourceCommit`、运行环境、画面caseId/路由/身份/语言/宽度、测量值、CSS gzip总量及代表路由初始JS大小。

- [ ] 为A01—A25定义caseId及25条独立展示行；补有讨论的A18/A20、审核理由/记录、笔记各状态、阅读/陈述/证明模式。使用真实UI在专属测试库构造反馈，不伪造生产Cookie。
- [ ] 保存本轮可重现基线：中英文、1280/390/320；额外准备768/1440、长名称、空/少/多项及长公式语料。截图只含合成数据，失败诊断沿用输入遮挡，不保存令牌或真实私人材料。
- [ ] 先验证helpers在故意制造页面横溢、遮挡焦点时能报告失败，再回到真实基线确认工具工作。需求改进断言在所属T2/T3加入RED，本任务不假装基线已达到新设计。
- [ ] 记录基础资源计量口径：全部构建CSS静态资源去重后，以同一gzip等级计算；各代表路由收集初始HTML引用的Client脚本去重后计算同样gzip大小。记录首页、阅读、笔记、复习、管理五路由。
- [ ] 输出逐页状态矩阵，A18/A20补图后关闭本轮证据缺口；不向Git写150张重复大图，选择性基线/改后图和完整CI artifact均注明环境。

**验证：** 分别用 `node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR -u MATH_MASTER_E2E_FRONTEND npm run e2e -- page-beauty-public.spec.ts`、learning、admin三文件。每文件≤480秒，单用例≤30秒，超出按场景拆分，不增加预算。

**完成标准：** 当前流程可真实打开；页表每行有可追踪caseId/源码结论；所有数据均为专属合成夹具。提交：`test: 建立全站视觉与状态基线`。

## T2：令牌、页面标题、页头与通用控件

**文件：** 新 `styles/tokens.css`、`styles/page-ui.module.css`、`components/page-heading.tsx`及测试；修改 `styles/globals.css`、`components/site-header.tsx`、`styles/auth.module.css`。需要新可读标签时精确更新i18n文案及语言覆盖登记；纯ReactNode槽组件按既有noOwnText机制登记，不能伪造文案覆盖。

**接口：** `PageHeading({title,description?,meta?,actions?}: {title:ReactNode;description?:ReactNode;meta?:ReactNode;actions?:ReactNode})`。无hook、无身份/路由读取、无Client指令；旧UiPageTitle职责保持。旧CSS变量名保留，语义令牌按方案映射。

- [ ] 先写 `PageHeading` 语义测试：正好一个一级标题；可选说明/元数据/动作不出现空容器或重复标题；传入LocalizedName与原链接不改可访问名称。
- [ ] 增加真实浏览器RED：390标准夹具中英文headerHeight≤184；实际次要文字/背景对比≥4.5；控件边界≥3；44px目标及48px主表单控件；文本焦点不被裁。
- [ ] 按方案拆出令牌并应用；手机语言/账户同一utility区域；保留品牌→主导航→utility顺序、NoteHistoryTracker唯一实例及原AuthStatus/AdminNavigation逻辑。
- [ ] 长用户名、管理员菜单展开、320与768屏宽不能裁入口；键盘顺序与可见DOM一致。保留forced-colors原生select与reduced-motion处理。
- [ ] 不删除旧基础选择器；检查旧快照页头也继承部分全局CSS这一事实，以原浏览器回归检查影响。

**验证：** PageHeading单元测试、`form-controls.spec.ts`、`site-streamline.spec.ts`及新public视觉文件；类型检查。分别执行，不拼成超过预算的一条命令。

**完成标准：** 两种语言/三种身份的页头与控件可用，尺寸/对比RED转GREEN；不改身份逻辑。提交：`style: 统一安静学术风格与导航控件`。

## T3：阅读、私人笔记与正文排版

**文件：** 新 `styles/current-prose.module.css`；修改 `features/reading/{safe-markdown,current-knowledge-view}.tsx`、`features/notes/{note-preview,shared-note-view,admin-share-view}.tsx`、`features/study/{managed-note-editor,managed-controls}.tsx`、`features/study-review/self-rating.tsx`、`styles/notes.module.css`及相关测试。旧 `features/study/note-editor.tsx` 与固定fixture不改。

**接口：** SafeMarkdown新增可选 `presentation?:'default'|'study'`，既有props/default保持。当前阅读、NotePreview及ReviewSections传study；CSS作用于同一个安全渲染根，不建立第二份正文或公式DOM。

- [ ] 先加RED：当前正文1280为17px/1.9、390为16px/1.85；窄屏长公式仅局部滚动；没有内容的可选分节不产生空标题。
- [ ] 加保护断言：默认模式仍按原语义渲染；study模式的HTML、危险URL、公式宏、缺素材退化文本与原安全测试相同。不得改trust、展开上限、URL/素材白名单。
- [ ] 接入study排版与当前分节间距、引用/来源/例子样式。局部滚动只对确实溢出的公式/表格给键盘入口，不让每个短公式多占Tab。
- [ ] 原1100阅读/笔记切换保持；在1099/1100两侧、编辑/预览、保存成功/未确认、冻结分享、改版/下架状态检查同一笔记控件始终保留输入。
- [ ] 真实测试：写草稿→换语言→换宽度→取消离开→保持文字和焦点；保存/刷新/撤回再修改仍使用原命令。A04/A06/A10/A24补前后对照。

**验证：** Vitest的 `safe-markdown.test.tsx`、`note-preview.test.tsx`、`current-reading.test.tsx`、`note-editor.test.tsx`；既有 `reading.spec.ts`、`note-editor-experience.spec.ts`、`note-sharing.spec.ts`及security文件按其原前端/场景运行；新learning视觉文件使用当前前端。

**完成标准：** 当前阅读字号提升、原安全与笔记流程不变；新/旧模式分别有证据。提交：`style: 优化数学正文与笔记阅读编辑`。

## T4：我的学习、复习、默写和两类历史

**文件：** 修改 `features/study/{managed-learning-view,managed-history}.tsx`、`features/notes/own-note-list.tsx`、`features/study-review/{queue,history,session,recall-editor,comparison,self-rating}.tsx`、`styles/{managed-learning,study-review,notes}.module.css`及已有相关测试。

**接口：** 消费原DTO和原ReviewCommandFlow；不修改 `use-review-command.ts`、leave-guard、身份Provider、所有state/事件/排期类型；仅替换标题宿主与展示类。

- [ ] 先加视觉/行为RED：常规工作区h1桌面≤40px、手机≤32px；五入口/四状态在320均可辨识；列表标题、时间、状态和操作明确分层。
- [ ] 统一学习/复习导航的视觉规格，保留各自原链接和aria-current；保留mode=review旧筛选。due计数仍为due+changed，其他桶、筛选字段和分页原样。
- [ ] 默写作答→比对→自评按原条件呈现；比对800/801两侧只有原可见栏，内容仍存在于合法揭示后的会话；不增加参考预取、不用CSS隐藏提前获取的参考。
- [ ] 保存草稿→切语言/宽度/返回→取消离开保留内容/焦点；冻结后无编辑输入；空答只可忘记；迟到响应/改版/下架/注销仍由原状态机清理。
- [ ] 两个来源学习历史分别翻页；自评历史把当时和当前安排分开排版，原编号继续在details，不改日期语义或事件排序。
- [ ] A09—A14全部状态有截图和行为证据；对比需显式等待自评按钮出现后采集，不能把异步过程中的旧编辑态标成比对态。

**验证：** 相关Vitest；既有 `study-review-flow.spec.ts`、recall、security与 `site-streamline.spec.ts`；新learning文件。命令逐个有界执行。

**完成标准：** 原学习/分享/排期事实保持，视觉门禁通过。提交：`style: 统一学习复习工作区与默写比对`。

## T5：首页、目录、分享页、账户与本人反馈

**文件：** 修改 `features/catalogue/{current-home,current-catalogue,catalogue-search}.tsx`、`features/notes/shared-note-list.tsx`、`features/auth/{credentials-form,account-panel}.tsx`、`features/feedback/{ticket-list,new-form,discussion-panel,status}.tsx`及对应模块样式；按需调整现有page壳，不改SSR取数和参数构造。

**接口：** PageHeading及原LocalizedName/UiText；原分页、来源参数、表单字段、回执/状态组件保持。该任务不新增数据请求或设置。

- [ ] 先建立A01—A08/A15—A18视觉及焦点用例：少量主题、长双语标题、空搜索、匿名登录提示、错误表单、有讨论及待确认反馈。
- [ ] 首页压缩hero和分区留白；地图/主题的查询、两个分页和署名区保留；分享卡片主次分明，公开最新10条及登录全部的边界不变。
- [ ] 登录/注册/账户按统一表单规格分组，保留自动填充、强制改密、注销、撤销会话与原焦点处理，不增删字段。
- [ ] 本人反馈按来源、状态、讨论、补充信息分区；原讨论显式加载与分页不改，不自动获取额外私人数据。
- [ ] 补充样式后的错误/无权/不可用/空结果区别，global loading/error/not-found及RetiredEntry随共同壳验收；不改变任何旧入口目标。

**验证：** 现有首页/目录/账户/反馈单元与真实浏览器（沿用原场景和配置）；新public视觉文件。A18本轮源码分析须补真实讨论画面。

**完成标准：** 所有当前公开与本人非复习页面有对应caseId；功能与URL兼容。提交：`style: 完善发现分享账户与反馈页面层级`。

## T6：知识管理、笔记审批、人员与管理员反馈

**文件：** 修改 `features/knowledge-admin/{list,edit,upload}.tsx`、`features/notes/{admin-share-queue,admin-share-view}.tsx`、`features/auth/admin-users.tsx`、`features/feedback/{ticket-list,discussion-panel,review-panel}.tsx`及其样式；新admin视觉文件扩充A19—A25。

**接口：** 原表单/命令处理函数、editToken、sequence、上传预检token、原role权限保持；展示只消费已有数据。

- [ ] 先加RED：管理员长表格无页面级横溢且局部可键盘横滚；所有原列可获取；长标题/错误/未确认提示可见；原对话框关闭恢复焦点。
- [ ] 上传、筛选、结果三段规范；保留单份table DOM。禁止为手机渲染第二套有相同id/按钮的卡片，不隐藏原列。
- [ ] 编辑组仅调整标题/间距，字段保持挂载；删除与保存区分视觉，保留原确认；不添加会卸载字段的标签页/折叠组或固定底栏。
- [ ] 审批正文、意见、处理按钮与事件记录分区；不删理由、不提前确认请求成功、不让重排按钮越过busy/unconfirmed保护。
- [ ] 普通学习者、旧editor/reviewer和强制改密均保持原权限；人员列表与对话框、管理员反馈详情补所有核心状态画面。

**验证：** 现有 `admin-knowledge-direct.spec.ts`、`note-sharing.spec.ts`及security、`feedback-review.spec.ts`和账户安全文件；新admin视觉文件。分别使用各自要求的原前端模式，不将旧测试偷偷改跑当前前端。

**完成标准：** A19—A25全部闭环，无管理员能力或写入语义变化。提交：`style: 优化知识管理与审批工作台`。

## T7：独立视觉CI、资源预算与兼容回归

**文件：** 新 `.github/workflows/page-beauty.yml`、`tests/e2e/page-beauty.playwright.config.ts`、cross-browser文件；新 `tools/verify/page-beauty-ci.mjs`、`page-beauty-assets.mjs`及对应测试；精确更新语言覆盖和兼容政策/摘要。原workflow/Playwright配置不改。

**接口：** 新config复用base webServer/安全设置，显式testMatch为cross-browser文件，只在自身projects增加Firefox/WebKit。新CI校验器对新workflow实际run命令、浏览器安装、原配置预算和准确文件做校验。资源工具使用T1的相同路由/资源去重/gzip等级和准确基线提交。

- [ ] 先写篡改负例：删掉任一public/learning/admin/cross-browser命令、漏装某浏览器、增加worker/retry/全局超时、指定旧兼容前端、删除资源测量均应失败。
- [ ] 新workflow用锁定Node/Go/PG/Actions及依赖；Go构建CGO_ENABLED=0；清除MATH_MASTER_E2E_FRONTEND后运行当前前端三文件；独立安装Chromium/Firefox/WebKit，运行cross-browser核心页面冒烟。
- [ ] Chromium覆盖23当前入口+2工作区及26旧入口，包含320/390/768/1280/1440、长公式/长标题/空数据/身份边界；Firefox/WebKit覆盖首页、阅读笔记、复习编辑/比对、管理员表单及焦点。
- [ ] 真浏览器手动检查200%文字放大、400%缩放等效320、键盘/高对比/reduced-motion；有可用iOS真机时补充软键盘输入/滚动/选择情况，无设备则明确未覆盖，不冒充完成，也不新增设备采购或发布硬门禁。
- [ ] 资源预算负例用超限新增CSS/JS资产验证能失败；主文字/次要文字/控件及错误态按实际计算样式测量对比，不对数学下标套字号或颜色断言。
- [ ] 完整跑原全部类型/单元/构建、兼容工具、固定旧前端构建及每条原浏览器命令。原API/迁移/角色/私人边界证据与主题分类63/534/4969保持。
- [ ] 精确登记新文件与已审查变更摘要；原基线、快照、例外不可整体重置。旧study-review末尾job保护继续原样通过。

**验证命令：** 原浏览器命令逐条按workflow执行；新三文件分别 `npm run e2e -- page-beauty-*.spec.ts`；cross-browser使用 `npm exec -- playwright test --config ../tests/e2e/page-beauty.playwright.config.ts`。全部通过 `tools/verify/run.mjs` 有界包装。兼容工具用 `env TOPIC_COMPATIBILITY_STAGE=managed node --test tools/verify/*.test.mjs`。

**完成标准：** 完整矩阵证据、资源预算和原CI全部通过；人工项目如实记录，未完成的发布必要验收不标绿。提交：`test: 验证页面美感跨浏览器与原契约兼容`。

## T8：最终审查、MR与交付

**文件：** 新 `docs/superpowers/reports/2026-10-14-page-beauty-code-review.md`、`page-beauty-acceptance.md`；更新逐页表的实施结果，不改已确认范围。

- [ ] 按Native流程安排一次新鲜全分支代码与兼容审查，覆盖五条审查重点、全部展示改动、scope禁改区和准确回归证据；发现阻塞先修复并跑受影响回归。
- [ ] A01—A25逐行有前后对照/结论；26个旧入口目标/退出语义逐项保留。尚未跑的浏览器或真机验收单独标明，不宣称完整通过。
- [ ] 核对源码差异仅批准文件；backend、迁移、API、业务lib、依赖锁、原workflow/配置和固定fixture字节不变；全部准确HEAD CI通过。
- [ ] 用Git推送分支并创建master MR，描述最终行为、截图与验证、实施风险；创建后附到当前会话。当前授权仅开发后的MR交付，合并与部署另按后续指令。
- [ ] 清理本计划专属合成库、进程及scratch，保留worktree、文档、审查裁定和必要证据；不清理其他项目资源。

**完成标准：** 无代码/兼容阻塞，MR可审查；发布需要的人工验收缺口已闭环或明确列为未满足，不以旧CI代替新改动验证。提交：`docs: 记录页面优化审查与验收`。

## 方案阶段的交付与门禁

本轮只提交四份文档及它们的兼容精确登记。风格已确认，全部产品开发复选项仍未勾选。执行前先读方案、逐页表与[审查报告](../reports/2026-10-14-page-beauty-plan-review.md)，以最终用户确认的书面版本为准。
