# 查询分页与文字操作实施计划

> 执行者必须使用 `superpowers:executing-plans` 或用户选择后的 `superpowers:subagent-driven-development`，按任务复选框实施；具体执行方式由总计划审阅确定。

**Goal:** 全部纳入范围的查询准确显示页数，支持图标导航及跳转，表格和列表行内动作使用文字按钮。

**Architecture:** 共用数字模型和分页控件，链接模式传递可序列化导航数据，回调模式保留原客户端身份边界。游标查询通过显式模式增加真实总量和任意页访问，原协议保持。

**Tech Stack:** 仓库现有 Go、Next.js / TypeScript、PostgreSQL、Zod、Vitest 和 Playwright；不升级依赖。

**Spec:** [已确认设计](../specs/2026-10-19-query-pagination-bulk-publish-design.md)，并遵循[总计划](2026-10-19-query-pagination-bulk-publish.md)的全部 Global Constraints。

## Global Constraints

- 总计划约束全部适用。新协议必须显式 `pagination=page`，旧模式响应字节含义、游标检查及原分页上限保持。
- 空结果为“第 0 / 0 页，共 0 条”；正常 `Y=ceil(total/limit)`、`X=floor(offset/limit)+1`。
- 跳转只接受 `1..Y` 的安全正整数；图标自制，点击区域至少 44px；320px 宽允许换行。
- 单命令最多 540 秒；Go `CGO_ENABLED=0`、`-timeout 5m`；真实数据库必须配置隔离测试 URL。

## Review Focus

总计划第 1、2、5 项分别落实到 P2/P3、P1/P4、P6；此外检查单页控件、旧 offset 极限后的尾页、匿名笔记和已退场页面。

## Task 1: P1：共用数字模型、导航与分页控件

**Files:** 新增 `backend/internal/pagination/{model,query,model_test,query_test}.go`；新增 `frontend/src/lib/pagination/{model,query,model.test,query.test}.ts`；新增 `frontend/src/components/{pagination.tsx,pagination.test.tsx}`、`frontend/src/styles/pagination.module.css`；修改 `frontend/src/lib/i18n/messages/{zh-CN,en}.ts`。

**Interfaces:** Go：`Request{Enabled bool, Page int64, Limit int, Cursor string}`，`Meta{Total int64,Limit int,Offset int64}`，`Page[T]{Items []T; Meta}`、`ActorPage[T]{ActorID string; Page[T]}` 均使用对应小写 JSON 名；`Parse(values url.Values,defaultLimit,maxLimit int)(Request,error)` 只解析分页键，业务层继续检查完整白名单；`Window(total,page int64,limit int)(Meta,error)` 将非空越界页归到末页、空集 offset=0。新页码值和计算限制为 JavaScript 安全整数，旧接口更窄限制只约束旧模式。

前端：`PageMeta={total:number;limit:number;offset:number}`；`pageModel(meta:PageMeta):{currentPage:number;totalPages:number;lastOffset:number;needsCorrection:boolean}`；`pageHref(pathname:string,query:Record<string,string>,offset:number,options:{offsetKey:string;pageMode:boolean;limit:number;pageKey?:string;cursorKey?:string}):string`，pageMode 时写所属 pageKey 并移除该区块旧 cursorKey（默认分别为 page/cursor）；早期历史使用 legacyPage/legacyCursor，其他区块参数保留。

`Pagination` 接收 `meta:PageMeta,label:ReactNode,disabled?:boolean`；navigation 为互斥联合：`{kind:'link',pathname,query,offsetKey,pageMode,pageKey?,cursorKey?}` 或 `{kind:'callback',onPageChange:(offset:number)=>Promise<void>|void}`。链接模式跨 RSC 只传普通数据，组件内部生成链接；`aria-labelledby` 指向组件自己生成的隐藏标签，不把 ReactNode 强转字符串。导航错误交给所属页面展示，重复回调由控件及页面请求 guard 防止。

- [ ] 写 `pageModel empty/single/boundaries`、`TestWindowSafeIntegerAndClamp`，断言 `pageModel({total:41,limit:20,offset:20}).currentPage===2`、`totalPages===3`；空集 currentPage/totalPages 都为 0，`Window(41,99,20).Offset===40`。
- [ ] 写 `pageHref independent blocks`、`pagination rejects invalid jump`、`pagination callback is single flight`：负数、小数、空值、1e309、超过 9007199254740991、越界不回调；`knowledgeOffset` 翻页保留 `offset`；标签、禁用和 Enter 正常；一页仍显示控件。
- [ ] 运行下列新增测试观察真实 RED，失败原因应是待实现接口或行为；随后实现模型、纯参数转换及组件，不引入产品依赖。
- [ ] 重跑至 GREEN，并通过静态 props 类型保证服务端不能传链接函数；新增中英提示、原创首页/前后页/尾页/跳转 SVG。
- [ ] 提交：`feat: 新增统一查询分页组件与数字边界校验`。

Run：`node tools/verify/run.mjs --cwd frontend -- npm exec -- vitest run src/lib/pagination/model.test.ts src/lib/pagination/query.test.ts src/components/pagination.test.tsx`；另执行 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/pagination -timeout 5m -count=1`。Expected：两组全部 PASS，无跳过。

## Task 2: P2：笔记、当前学习历史及复习历史的页码协议

**Files:** 新增 `backend/internal/{knowledgeadmin,notesharing,studyreview}/numbered.go`、`backend/internal/store/{knowledge_admin_history_page,note_sharing_page,study_review_history_page}.go` 及对应 `*_test.go`、`backend/internal/httpapi/query_pagination_test.go`；修改 `backend/internal/httpapi/{knowledge_admin_routes,knowledge_study_routes,note_sharing_routes,study_review_routes,study_review_json}.go`；修改 `frontend/src/lib/{knowledge-admin,note-sharing,study-review}/{schemas,types,client,server-client}.ts` 和对应已有测试；修改 `frontend/src/lib/api/{knowledge-admin,note-sharing,study-review}-proxy.ts` 及对应测试；更新 `api/openapi.yaml`、生成 `frontend/src/lib/api/generated.d.ts`。

**Interfaces:** 各领域新增 `NumberedRepository`，旧 Repository 不增方法。方法均为 `(ctx context.Context,a Access,q ExistingQuery,p pagination.Request)`：knowledgeadmin 的 `ListManagedHistoryPage` 用 `a knowledgeadmin.Access,q study.HistoryQuery` 返回 `pagination.ActorPage[ManagedHistoryEntry]`；notesharing 的 `ListOwnNotePage/ListSharedNotePage/ListAdminNoteSharePage` 用 `a knowledgeadmin.Access,q notesharing.Query`，分别返回 ActorPage[OwnNoteSummary/SharedNoteSummary/AdminSummary]；studyreview 的 `ListReviewHistoryPage` 用 `a studyreview.Access,q HistoryQuery` 返回 `NumberedHistoryPage{ActorID string;AsOf time.Time;pagination.Page[HistoryItem]}`。以上方法均返回 `(结果,error)`。

Go handler 显式模式时 type assert 对应窄接口；旧模式原分发不变。客户端根据完整路径的 query 选择严格 schema，不能提前 `split('?')[0]` 后丢失模式。新增 `readNumberedManaged<T>`、`readNumberedNote<T>`、`readNumberedReview<T>` 服务端读取入口分别返回所属模块原 `Result<T>` 类型，保留 actor、缓存和截止检查；浏览器读取沿用原函数，内部按路径选择 schema。

READ COMMITTED 模块使用单个 SQL 的 filtered/count/page CTE 计算总量与投影，以同一语句快照保证一致；不把“同一事务”误当成相同语句快照，也不全局改变原身份锁所需隔离级别。有范围绑定的原游标保留原验证；没有携带身份/筛选摘要的旧格式只作为排序锚点，仍以当前用户与过滤条件计算，不能把它当成授权凭据。

- [ ] 写 `TestNumberedNotesFiltersAndAnchor`、`TestNumberedManagedHistory`、`TestNumberedReviewHistory`：41 条/20 页大小分别为 20/20/1；`want.Total==41 && want.Offset==40`；笔记审批保留升序，时间相同有第二键；主题重复关联不重复计数。
- [ ] 写 `TestNumberedOldCursorScopeAndDeletion`、`TestNumberedActorIsolation`：旧锚点边界删除、同时间戳、合法但来自另一账号/筛选的游标，沿用原范围校验；没有范围绑定的格式仍只返回当前 actor 的匹配投影。计数与 items 过滤相同，READ COMMITTED 并发状态修改不能让同一响应的总量与投影错位。
- [ ] 写 HTTP/代理/schema 用例 `numbered mode preserves legacy response`：原模式仍只给原 `nextCursor` 形状；新模式拒绝 `page+cursor`、重复键、无效模式、非整数；旧书签通过新模式锚点对齐。匿名 `/shared-notes/latest` 拒绝升级为完整列表。
- [ ] 先执行 RED；在同一原读取事务内计数及选页，保留 asOf、身份复核、来源可见性和响应大小预算；实现独立响应 schema 及精确代理白名单。
- [ ] `npm run api:generate` 后跑 GREEN；旧模式严格 schema 必须继续成功，不允许 `.passthrough()` 绕过协议。
- [ ] 提交：`feat: 为笔记与学习历史增加兼容页码查询`。

Run：通过 runner 分别执行 `go test ./internal/store -run '^TestNumbered(Notes|Managed|Review|OldCursor|Actor)' -timeout 5m -count=1`、`go test ./internal/httpapi -run '^TestNumbered' -timeout 5m -count=1`；前端分别运行三个模块的 `schemas.test.ts` 与三个代理测试。Expected：新增页码与原游标断言均 PASS。

## Task 3: P3：反馈、通知及可访问归档查询

**Files:** 新增 `backend/internal/{feedback,notification,correction,study}/numbered.go`；新增 `backend/internal/store/{feedback_ticket_page,notification_page,correction_case_page,study_history_page}.go` 及各 `*_test.go`；修改 `backend/internal/httpapi/{feedback,notification,correction,study}_routes.go`；修改 `frontend/src/lib/{feedback,notification,correction,study}/{schemas,types,client,server-client}.ts`、`frontend/src/lib/study/protocol.ts` 和对应测试；修改 `frontend/src/lib/api/{feedback,notification,correction,study}-proxy.ts` 及测试；更新 `api/openapi.yaml` 和生成类型。旧偏移量极限适配修改 `backend/internal/{knowledgeadmin,studyreview}/model.go`、`backend/internal/store/{knowledge_admin_read,knowledge_admin_public,knowledge_admin_study,study_review_read}.go` 及 P2 已列相关路由。

**Interfaces:** 新增四个 `NumberedRepository`，方法 `ListFeedbackTicketPage(ctx,question.Access,review bool,feedback.ListQuery,pagination.Request)(feedback.Envelope[pagination.Page[feedback.Metadata]],error)`、`ListNotificationPage(ctx,question.Access,notification.Query,pagination.Request)(notification.Envelope[pagination.Page[notification.Metadata]],error)`、`ListCorrectionCasePage(ctx,question.Access,correction.Query,pagination.Request)(correction.Envelope[pagination.Page[correction.CaseMetadata]],error)`、`ListStudyHistoryPage(ctx,study.Access,study.HistoryQuery,pagination.Request)(pagination.ActorPage[study.HistoryEntry],error)`。

旧计划/作业/证据等正文“加载更多”接口不改为查询表单。已有 offset 接口在能覆盖全部结果时继续原协议；当前知识/学习/复习结果超过原上限时允许显式页码分支，旧 offset 分支照原上限拒绝。页码分支先 count 再规范化页码，不直接把安全整数 page 乘 limit 形成可能溢出的 OFFSET。

- [ ] 写 `TestNumberedFeedbackFilters`、`TestNumberedNotificationOwner`、`TestNumberedCorrectionArchive`、`TestNumberedLegacyHistory`，断言筛选范围与原权限、archive/退场口径一致，`Total` 等于真实匹配数量，标记已读不改变通知总数。
- [ ] 写 `TestNumberedDeepOffsetKeepsLegacyLimit`：纯窗口向量覆盖 offset=100020 及 managed offset=1000020；HTTP 新模式接受其对应 page，旧模式同 offset 仍拒绝。SQL 使用41条并请求该深页，必须规范化为最后一页 offset=40；无需为数字边界创建百万条正文。
- [ ] HTTP 和客户端用例先 RED；实现参数白名单、计数、稳定排序和独立 schema，不修改旧游标或旧原型的返回类型。
- [ ] 运行 GREEN、更新生成类型，验证原客户端读取新后端仍通过；公共 count 不能携带私人结果。
- [ ] 提交：`feat: 完成反馈通知和归档列表的兼容页码查询`。

Run：runner 下分组执行 `go test ./internal/store -run '^TestNumbered(Feedback|Notification|Correction|Legacy|Deep)' -timeout 5m -count=1`、HTTP Numbered 组，以及四类模块的 schemas/client 和 proxy 测试。Expected：旧/新模式、权限和深页边界全部 PASS。

## Task 4: P4：当前页面、行内文字操作与 URL 状态

**Files:** 修改 `frontend/src/features/catalogue/current-catalogue.tsx`、`frontend/src/features/auth/admin-users.tsx`、`frontend/src/features/knowledge-admin/list.tsx`、`frontend/src/features/study/{managed-learning-view,managed-history,managed-pages}.tsx`、`frontend/src/features/notes/{own-note-list,shared-note-list,admin-share-queue}.tsx`、`frontend/src/features/study-review/{queue,history,pages}.tsx`、`frontend/src/features/feedback/{ticket-list.tsx,page-data.ts}`、`frontend/src/features/notification/{inbox.tsx,page-data.ts}`、`frontend/src/features/correction/{case-list.tsx,page-data.ts}`、`frontend/src/lib/knowledge-admin/study-navigation.ts`；修改 `frontend/src/app/{notes,admin/note-shares,admin/knowledge}/page.tsx`、`frontend/src/styles/globals.css`；新增 `frontend/src/features/pagination/current-pages.test.tsx`，更新上述页面已有测试。

**Interfaces:** 消费 P1 的 `Pagination` 及 P2/P3 页码 DTO；各 href helper保留业务条件，增加所属 `page/legacyPage`，筛选提交清页码。知识管理保留单条状态函数供 B5 接管发布忙状态。`.text-button` 只应用行内动作，主搜索/上传/保存不变。

- [ ] 写 `current pages show accurate pagination`：空/1/41 条准确；`expect(screen.getByText('第 2 / 3 页，共 41 条')).toBeVisible()`；所有当前查询区块都有分页，匿名笔记按 10 条边界展示。
- [ ] 写 `independent topic and history paging`：主题翻页保留知识页参数，当前/早期历史分开；旧 cursor 进入后规范化 URL，浏览器 back/forward 与刷新恢复；重新筛选回第一页。
- [ ] 写 `late response cannot replace newer query`、`row action never submits filter`，验证用户管理用 applied 条件翻页、失败保留原结果、越界最多纠正一次、角色切换取消迟到响应。
- [ ] 先 RED，再接入组件；服务端页只传导航普通数据，客户端页使用 callback。新页数区域移到表格横滚区域之外，320px 不依赖横滚才能找到分页。
- [ ] 跑 GREEN，并逐一检查编辑/发布/下架/查看等文字样式与元素语义，保留原业务 guard 和未保存保护。
- [ ] 提交：`feat: 统一当前查询分页与行内文字操作`。

Run：runner 下执行 `vitest run src/features/pagination/current-pages.test.tsx`、原 auth/admin-users、catalogue/current-catalogue、study/managed-history、study-review/queue/history、notes/own-note-list 测试及 `npm run typecheck`。Expected：原语义和新交互均 PASS。

## Task 5: P5：仍可访问的旧查询覆盖

**Files:** 修改 `frontend/src/features/content/{draft-list,submission-list,publication-panel,topic-workspace}.tsx`、`frontend/src/features/question/{draft-list,submission-list,publication-panel,coverage-panel}.tsx`、`frontend/src/features/learning/{history-list,knowledge-list}.tsx`、`frontend/src/features/study/{knowledge-list,history-list,legacy-archive,pages}.tsx`；新增 `frontend/src/features/pagination/legacy-pages.test.tsx`，修改原 `frontend/src/features/{content,question}/authoring.test.tsx` 与 study 历史测试。固定 `test-fixtures/` 和 `.superpowers/.../legacy/frontend` 不修改。

**Interfaces:** 消费 P1 callback 模式；每个原 load/refresh 保留原输入和错误状态，增加单飞/迟到响应保护。已批准批次与发布列表各自分页，原审核、冻结和发布命令继续使用原 hook。

- [ ] 写 `legacy accessible lists use shared pagination`、`two publication lists keep independent page`，断言 approved total=41 与 publications total=1 各自显示正确页数，操作不交叉。
- [ ] 写 `retired route remains retired`：managed 模式原跳转/410不因 UI 改动恢复写入；legacy 夹具原作者/审核权限及表单输入保持。
- [ ] 运行 RED，精确替换可访问区块分页与行内动作，不重构业务状态机，不全局替换 `.button`；再运行 GREEN。
- [ ] 提交：`feat: 补齐可访问旧查询列表的统一分页`。

Run：runner 下执行新增 legacy-pages、原 content/question authoring、study history 单元组；相关真实 legacy 浏览器测试在 P6 执行。Expected：旧写门禁、独立查询和原语义均 PASS。

## Task 6: P6：协议兼容、覆盖登记及分页验收

**Files:** 新增 `tests/e2e/query-pagination.spec.ts`、`backend/internal/e2etest/query_pagination_fixture.go`、`query_pagination_fixture_test.go`、`query_pagination_control.go`；修改 `backend/internal/e2etest/harness.go`；新增 `tools/verify/query-pagination-ci.mjs`、`query-pagination-ci.test.mjs`、`.github/workflows/query-pagination.yml`；精确修改 `tools/verify/{go-batches.mjs,go-batches.test.mjs,topic-learning-compatibility.mjs,learning-compatibility.test.mjs,feedback-compatibility.test.mjs,correction-compatibility.mjs,admin-review-compatibility.mjs}`、`api/topic-learning-compatibility-baseline.json`、`docs/operations/ui-language-coverage.json`、`docs/operations/evidence/admin-content-review/compatibility-exceptions.json`。原 `api/{learning,feedback,correction}-compatibility-baseline.json` 保持，新增变更通过现有精确例外映射恢复原协议比较，不重写其历史基线。

**Interfaces:** 新 CI 清单列明 P1—P5 新测试及浏览器组；`go-batches.mjs` 为 Numbered 新仓储组登记独立集合，原普通批次不重复执行它，新 workflow 必须精确覆盖该集合；分组测试证明全库存无遗漏、无重复且每批≤原上限。旧门禁继续原断言，只登记准确实际文件和摘要。夹具安装只允许 harness 专用随机库，通过原测试控制接口，不进入生产路由。

- [ ] 写 `query pagination across current pages`、`old cursor bookmarks and guest boundary`、`mobile keyboard jump`：真实 41 条、多区块、筛选、前后退、首页/尾页、Enter、失败和空结果；桌面/390px 两项目，再针对 320px 验证无整页横溢和焦点。
- [ ] `query-pagination-ci.test.mjs` 断言每个新测试被执行、命令均经 runner、浏览器 globalTimeout=480000 且零重试；移除任一测试登记应 RED。
- [ ] 实现合成夹具和独立 workflow，记录新增页码路径精确摘要；原 API 按 legacy/page 两模式做规范化差异审查，原迁移及旧游标响应保持。禁止删除原兼容断言或用通配白名单放行。
- [ ] 运行各组至 GREEN：`go test ./internal/e2etest -run '^TestQueryPagination' -timeout 5m -count=1`；node 新 CI 与原 topic/ui-language 兼容组；`npm exec -- playwright test --config ../tests/e2e/playwright.config.ts query-pagination.spec.ts`。每条经 runner。
- [ ] 审阅 Spec 第 4 节列表逐项有对应验证，再提交：`test: 验证全站分页与旧协议兼容`；将证据并入总计划验收记录。

Expected：所有新 UI、旧协议及匿名/归档边界通过；能力不依赖发布迁移 16，可以独立验收。进入 B 前先保持本组干净、可构建。
