# 个人主题学习执行计划

> **For agentic workers（致执行代理）：** REQUIRED SUB-SKILL：按用户选择使用 superpowers:executing-plans（Native，推荐）或 superpowers:subagent-driven-development。按任务逐项执行，以 - [x] 跟踪步骤。计划审核及执行方式未确认前，不写产品代码。

**Goal（目标）：** 提供知识点状态、主题进度、私人笔记、学习时间线、检索复习和登录后的我的学习入口。

**Architecture（架构）：** 独立 study 模块仅依赖账户、当前公开知识和主题配对，不依赖旧测评/题库就绪。用户命令及时间线事件同事务提交，进度和内容提醒按本人记录查询生成。

**Tech Stack（技术栈）：** Go 1.27.1、PostgreSQL 17.11、Next.js 16.3.7、React 19.3.0、TypeScript 5.9.3、Node.js 24.17.0；沿用现有 Vitest、Playwright、Zod、goose、pgx，不新增运行依赖。

**Spec（设计）：** [用户已确认的主题学习重构设计](../specs/2026-10-06-topic-learning-refactor-design.md)，2026-10-06 获书面确认。本计划已获用户书面确认。前置：[阶段A](2026-10-06-topic-taxonomy.md)通过交付门禁；未合并依赖时只做明确依赖的草稿MR，不自行合并。

## 全局约束

- 分类固定为 63 个一级主题、534 个二级主题、4,969 个具体主题；503 个辅助分类和534 个其他兜底节点单列，完整节点6,603 个。
- 个人状态仅 unlearned、learning、completed、reviewing；实际阅读开始，主动完成，已完成知识主动复习后回到已完成。
- 主题按当前公开可用知识去重；复习不降低完成率，零分母显示暂无内容；分类移动、版本变化不清除笔记或完成事实。
- 旧正文、摘要、来源、批准、原答案和原成绩保持；00001—00009 历史迁移保持原字节。缺失能力拒绝相关操作，不删除永久标记或历史来恢复。
- 控制请求8 KiB；笔记16,000 Unicode 标量且64 KiB、请求128 KiB；私有完整响应2 MiB；普通列表默认20最多100，历史默认20最多50。
- Go 请求8 秒、锁等待1 秒、Next 与客户端10 秒总截止；原账户时限保持。普通读写保留个人120/30、全局600/120 每分钟预算，后台重操作共用原工作槽。
- 每工作区100 知识、当前快照1,000 知识等原内容容量保持；6,603 分类节点不占知识容量。不增加运行依赖、连接池或常驻导入服务。
- Go 验证使用 CGO_ENABLED=0、GOTOOLCHAIN=go1.27.1、-timeout 5m；命令由 tools/verify/run.mjs 限540 秒；浏览器单 worker、零重试、批次480 秒，桌面1280×900、手机390×844。
- 数据库测试只用 TEST_DATABASE_URL 和 testutil.Database 创建的随机 math_master_test_* 库；缺配置导致 skipped 时不报告验证完成，不使用真实开发库或服务器。
- 页面沿用 en、zh-CN 功能词条和受限 Markdown/KaTeX/原创 SVG。GET、SSR、预加载和草稿预览不写学习记录；私人笔记不进入日志或公开材料。
- 未确认命令仅主动同键重试；不得自动重发 POST。每个业务任务先写行为失败测试，确认失败，再最小实现、目标回归和提交。
- 开发前更新 master 并新建 codex/ 分支；三阶段先后交付，每阶段审查与回归后创建 MR，不擅自合并依赖或部署。

## Review Focus 审查重点

1. 普通 GET、预加载或匿名阅读：不得创建个人记录；客户端自动 begin 丢失响应不能重复开始（B3、B6）。
2. 已完成知识正在复习、后来新增或撤回知识：完成事实保持，分母变化明确提示（B4）。
3. 笔记含四字节字符、恶意 HTML、并发保存或删除：精确限额、安全渲染、冲突保留输入，删除后不返回文本副本（B5、B6）。
4. 换账户、强制改密、撤销会话或晚到响应：原账户输入/结果不进入新身份页面（B6、B7）。
5. 多次完成不同材料、同时间戳历史和主题移动：第一次完成及原版本保留，历史可稳定分页（B3、B4、B7）。

## 文件职责与共同类型

迁移00011_topic_study.sql 创建 study_records、study_events、study_notes、study_idempotency、study_content_changes，设置 topic_learning_state.study_enabled=true，并在goose_db_version的version_id=0行新增永久topic_study_enabled标记及不可清除触发器。空库Down可以撤表并将当前study_enabled设为false，但不能清除永久标记；任一新表非空时 Down 整体拒绝。study 状态缺结构时拒绝，不影响已核验的匿名正文。

study.Access{TokenHash auth.Digest,CSRF auth.Secret,IdempotencyKey,RequestID string} 独立定义；不依赖 question.Access。study.KnowledgeRef 复用 taxonomy.KnowledgeRef。StudyRecord={knowledgeId,state,sequence,firstStartedAt,firstCompletedAt,lastCompletedAt,lastReadAt,lastReviewedAt,completedRef,lastReviewRef,lastReviewId,activeReviewId}，时间字段在Go中为*time.Time，在JSON中为ISO8601字符串或null，由数据库产生。StudyDetail={actorId,record,currentKnowledge:taxonomy.KnowledgeSummary或null,pair,available,materialChanged}，不在列表携带完整知识正文。ListQuery={TopicID,State,Q string,ReviewOnly bool,Limit,Offset int}；Page[T]={ActorID string,Items []T,Total,Limit,Offset int}。ReviewOnly=true只查询本人已有开始或完成记录，Q按标题及已有中文对照作字面匹配。CommandInput={knowledge:KnowledgeRef,expectedKnowledgeHead:string,expectedSequence:int64}；对首次未学习的 expectedSequence=0。ReviewInput 在 CommandInput 上增加 reviewId；客户端不得生成 actorId 作为授权。

### Task 1 B1 个人学习合同与状态机

**Files（文件）：**
- Create：backend/internal/study/model.go
- Create：backend/internal/study/state.go
- Create：backend/internal/study/validate.go
- Create：backend/internal/study/state_test.go
- Create：backend/internal/study/validate_test.go
- Modify：api/openapi.yaml
- Modify：frontend/src/lib/api/generated.d.ts
- Create：frontend/src/lib/study/types.ts
- Create：frontend/src/lib/study/schemas.ts
- Create：frontend/src/lib/study/schemas.test.ts

**Interfaces（接口）：** ApplyState(state State,action Action,completedRef,currentRef KnowledgeRef) (State,error)，Action 固定 begin/complete/start-review/finish-review/save-note/delete-note；知识版本升级用 MaterialChanged 独立标记，不把 completed 改回 learning。ValidateCommand(CommandInput) error、ValidateNote(string) error。

- [x] **步骤1 写行为失败测试。** TestStudyStateTransitions 断言 unlearned+begin=learning、learning+complete=completed、completed+start-review=reviewing、reviewing+finish-review=completed；未学习 complete 和未完成 start-review 拒绝；completed+begin 保持 completed；版本升级保持 completed 且 materialChanged=true。TS 严格 schema 拒绝未知状态、字段、错误 actorId 和超限数值。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if got != study.Completed { t.Fatal("ordinary reread reset completion", got) }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/study -run '^TestStudy' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 固定具名 Study* OpenAPI 合同，客户端类型由生成文件导出；完成状态没有 passed、score、qualification、blueprints。状态机只处理行为，不以停留时长、滚动或答题转换状态。
- [x] **步骤4 验证通过。** 重跑步骤2命令；Go/TS 对同一边界样例一致；api:generate 与 typecheck 通过；原学习DTO保持历史读取合同。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 定义简单学习状态合同'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 2 B2 学习结构、身份与事务

**Files（文件）：**
- Create：db/migrations/00011_topic_study.sql
- Create：backend/internal/study/repository.go
- Create：backend/internal/study/service.go
- Create：backend/internal/store/study_tx.go
- Create：backend/internal/store/study_idempotency.go
- Create：backend/internal/store/study_schema_test.go
- Create：backend/internal/store/study_fixture_test.go

**Interfaces（接口）：** study.NewService(repo study.Repository) (*study.Service,error)；Repository方法由B3—B5的具名签名组成；StudyPreflight(ctx,a Access,action Action) (auth.User,error)；store.studyTx(ctx,a,action,fn func(ctx,tx,u,now) error) error；幂等键作用域 user/action/knowledge/key，绑定规范输入摘要。newStudyFixture(t) 消费 A4 fixture，提供 Access(user)、Ref()、Pair()、CountEvents(user,kind)、ResetQuestionTablesForIsolation()。

- [x] **步骤1 写行为失败测试。** TestStudyWithoutQuestionBank：随机库中未启用题库或无题库发布仍可开始学习；TestStudyMissingSchemaFailClosed、NonemptyDownDenied、CrossActorKeyIsolation、ReplayCommitIdentity；同键不同输入409，两用户同键互不串回执。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if started.State != study.Learning { t.Fatal("new study depends on question bank", started) }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestStudy(Schema|Without|Missing|Cross|Replay|Nonempty)' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 建立 user+knowledge 唯一记录、只追加事件、版本引用、序号、笔记唯一、事件幂等索引。身份使用 managedIdentity，提交前再次核验同一会话及强制改密，8 秒事务/1 秒锁等待，旧账户预算保持。行锁先账户再当前知识配对再个人记录；GET只读，笔记正文不进幂等回执或日志。
- [x] **步骤4 验证通过。** 重跑步骤2命令；空Up/Down仅随机库、非空拒绝；关闭题库能力不影响新study；原账户与匿名正文回归通过。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 建立独立个人学习事务'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 3 B3 开始完成与一轮复习

**Files（文件）：**
- Create：backend/internal/store/study_actions.go
- Create：backend/internal/store/study_actions_test.go
- Create：backend/internal/study/digest.go
- Create：backend/internal/study/digest_test.go

**Interfaces（接口）：** BeginStudy(ctx,a study.Access,id string,in study.CommandInput) (study.StudyDetail,error)、CompleteStudy(...)、StartStudyReview(...)；FinishStudyReview(ctx,a,id,in study.ReviewInput)。事件类型 started/completed/review-started/review-finished；begin 更新最近阅读，但已学习知识不重复 started；每知识唯一 activeReviewId。单次返回字段包含当前sequence供下一命令使用。

- [x] **步骤1 写行为失败测试。** TestStudyBeginReplay：同键及新键重复 begin 只一个 started；TestStudyCompletionKeepsFirstTime：新版明确完成增加事件、第一次时间不变；TestStudyReviewSingleRound：两个标签页竞争最多一轮；TestStudyChangedHeadRejectsCompletion 与会话撤销阻止事件提交。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if got.FirstCompletedAt == nil || !got.FirstCompletedAt.Equal(first) || eventCount != 3 { t.Fatal("completion history lost or duplicated") }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestStudy(Begin|Completion|Review|Changed|Commit)' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 四种操作用 B2 事务；先核验当前知识/head，再处理同键及语义重复：begin对已开始、complete对已完成同一材料、start-review对同一现有轮次、finish-review对刚结束的同一lastReviewId返回当前事实，不制造事件或因旧sequence误报冲突。只有实际状态变化核验expectedSequence并递增；普通lastReadAt更新不递增状态sequence。当前公开知识不可用/准确head不符时不生成完成或复习事件。相同材料重复 complete 返回原完成，不增加事件；复习结束绑定准确reviewId和当前知识，内容已变化时给出明确版本冲突，保留可恢复的复习记录。身份晚到不驱动导航。
- [x] **步骤4 验证通过。** 重跑步骤2命令；实际 PostgreSQL 竞争和失败回滚通过；触发事件数符合断言，不伪造旧资格；单次请求超时可同键确认而不重复完成。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 记录知识学习与复习动作'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 4 B4 主题进度、时间线和内容变化投影

**Files（文件）：**
- Create：backend/internal/store/study_read.go
- Create：backend/internal/store/study_history.go
- Create：backend/internal/store/study_content_changes.go
- Create：backend/internal/store/study_read_test.go
- Create：backend/internal/store/study_history_test.go
- Modify：backend/internal/store/taxonomy_release.go
- Modify：backend/internal/store/workflow_withdrawal.go

**Interfaces（接口）：** ReadStudyOverview(ctx,a) (study.Overview,error)、ListStudyTopics(ctx,a,q study.ListQuery) (study.Page[study.TopicProgress],error)、ListStudyKnowledge(ctx,a,q study.ListQuery) (study.Page[study.StudyDetail],error)、ReadStudyKnowledge(ctx,a,id string) (study.StudyDetail,error)、ListStudyHistory(ctx,a,q study.HistoryQuery) (study.HistoryPage,error)。TopicProgress={topicId,total,completed,learning,reviewing,added,removed,completedRatio:*float64}；HistoryQuery={knowledgeId,topicId,from,to,kind,cursor,limit}；游标是版本1的 (occurredAt,id) 边界，最长512 ASCII字节。appendStudyContentChangesTx(ctx,tx,before,after taxonomy.KnowledgeSet,publicationId string) error 只写全站差异；taxonomy.KnowledgeSet=map[string]taxonomy.KnowledgeRef，以稳定知识ID比较新增、升级和移除。

- [x] **步骤1 写行为失败测试。** TestStudyReviewPreservesProgress 用 total=2/completed=1，复习后仍1/2；TestStudyZeroDenominatorRatioNull、AncestorDedup、TopicMoveKeepsRecord；TestStudyHistoryEqualTimestampCursor 不漏或重复；TestStudyReminderNoFanout 令用户数增长，发布变化事件数量不变，GET前后个人事件数不变；TestStudyReviewSearchOwnerOnly排除别人记录和本人未学习内容，并核验中文、多字节边界、百分号与下划线字面匹配。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if progress.Completed != 1 || progress.Total != 2 || *progress.CompletedRatio != 0.5 { t.Fatal(progress) }
if userEventCountBefore != userEventCountAfter { t.Fatal("GET appended events") }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestStudy(Progress|Zero|Ancestor|TopicMove|History|Reminder)' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 在同一只读快照内连接当前主题知识与本人记录，不依赖旧qualified字段。主题分母变化显示增减；正文变化与分类移动分开。内容更新、撤回提醒从全站变化事件与本人版本投影；时间线归属过滤使用当前主题，事件保留当时知识/分类版本。删除正文不恢复撤回内容。
- [x] **步骤4 验证通过。** 重跑步骤2命令；分页所有权、UTC排序和数据库时间通过；待归类、不可用、升级仍可查看私人摘要；数据量500用户/1000知识时单页查询<=8秒、响应<=2MiB。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 汇总主题进度与知识时间线'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 5 B5 私人笔记保存冲突与删除

**Files（文件）：**
- Create：backend/internal/store/study_notes.go
- Create：backend/internal/store/study_notes_test.go
- Create：backend/internal/study/note.go
- Create：backend/internal/study/note_test.go

**Interfaces（接口）：** ReadStudyNote(ctx,a,id string) (study.NoteView,error)、SaveStudyNote(ctx,a,id,in study.NoteInput) (study.NoteReceipt,error)、DeleteStudyNote(ctx,a,id,in study.NoteDeleteInput) (study.NoteReceipt,error)。NoteInput={expectedRevision:int64,knowledge:KnowledgeRef,body:string}；NoteDeleteInput={expectedRevision:int64}；NoteView={knowledgeId,revision,body,knowledge,updatedAt}；NoteReceipt仅knowledgeId/revision/deleted/updatedAt，正文不留回执。删除后保留不含正文的修订墓碑，新保存必须使用墓碑revision，防止ABA覆盖。

- [x] **步骤1 写行为失败测试。** TestStudyNoteOwnerOnly 包含管理员读他人404；TestStudyNoteUnicodeLimit 对16,000与16,001码点、64KiB边界逐项断言；TestStudyNoteConflictingTabs 一次成功一次409；TestStudyNoteDeleteNoRecoverableBody 验证表与幂等回执均无删除正文；含原始HTML/危险数学宏被拒绝，允许安全公式。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if !errors.Is(err, auth.ErrNotFound) { t.Fatal("administrator read another person's note", err) }
if storedBodyAfterDelete != "" { t.Fatal("deleted text retained") }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/study -run '^TestStudyNote' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 复用受限 Markdown/数学检查但不允许图片/附件。保存/删除按同键、准确修订和原归属核验；笔记可在正文撤回后继续本人读取和编辑，编辑不恢复旧正文。事件只写动作、revision及知识身份；删除清当前body，重放receipt不返回文本。
- [x] **步骤4 验证通过。** 重跑步骤2命令；存储、幂等表、事件和诊断检查不泄露正文；角色变更/强制改密阻止写；并发ABA测试通过。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 保存私人知识笔记'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 6 B6 私有学习 API、账户边界与阅读控件

**Files（文件）：**
- Create：backend/internal/httpapi/study_routes.go
- Create：backend/internal/httpapi/study_json.go
- Create：backend/internal/httpapi/study_dispatch.go
- Create：backend/internal/httpapi/study_test.go
- Modify：backend/internal/httpapi/application.go
- Modify：backend/cmd/server/main.go
- Create：frontend/src/lib/study/client.ts
- Create：frontend/src/lib/study/server-client.ts
- Create：frontend/src/lib/api/study-proxy.ts
- Create：frontend/src/app/api/v2/study/[[...segments]]/route.ts
- Create：frontend/src/features/study/account-boundary.tsx
- Create：frontend/src/features/study/pending-command.ts
- Create：frontend/src/features/study/knowledge-controls.tsx
- Create：frontend/src/features/study/note-editor.tsx
- Create：frontend/src/features/study/knowledge-controls.test.tsx
- Create：frontend/src/features/study/note-editor.test.tsx
- Modify：frontend/src/app/knowledge/[id]/page.tsx

**Interfaces（接口）：** GET overview、topics、knowledge、knowledge/{id}、history；GET knowledge支持topicId、state、q、reviewOnly、limit、offset；POST knowledge/{id}/begin、complete、review/start、review/finish；GET/PUT/DELETE knowledge/{id}/note。useStudyCommand<T>(onSuccess):{run(route,input),retrySameRequest(),pending,busy,error,clear()}；StudyAccountBoundary({actorId,children})；KnowledgeStudyControls({detail})、NoteEditor({knowledgeId,initialNote})。readServerStudy<T>(route,cookieHeader)仅GET；study client返回actorId和具名StudyResult，命令10秒从点击计时包含身份准备。

- [x] **步骤1 写行为失败测试。** TestStudyHTTPBodyBoundaries 对8192/8193控制字节、131072/131073笔记请求、未知/重复键和尾随内容拒绝；组件 actual_read_begins_once、anonymous_and_ssr_do_not_begin、same_key_after_response_loss；note_switch_actor_hides_old_input、late_response_is_ignored、dirty_note_survives_locale_switch。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```ts
expect(beginRequests).toHaveLength(1);
expect(retryRequest.key).toBe(firstRequest.key);
expect(screen.queryByDisplayValue(oldActorNote)).not.toBeInTheDocument();
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/httpapi -run '^TestStudyHTTP' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 接线严格私有同源路由，不接受任意URL；未确认不自动POST，不写localStorage/sessionStorage。阅读成功且当前身份核验完成后客户端触发begin；已完成仍保持completed。笔记明确保存与离开前提示，冲突保留输入；换身份卸载私有子树。语言切换只换文案不产生begin/保存。
- [x] **步骤4 验证通过。** 重跑步骤2命令；npm run api:generate、typecheck及两组件测试通过；使用mock仅验证请求边界，实际身份和提交行为在B8真实浏览器验证；400/401/403/404/409/428/429/503词条按模块校验。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 接入知识学习与笔记界面'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 7 B7 我的学习、复习检索、历史与登录

**Files（文件）：**
- Create：frontend/src/features/study/overview.tsx
- Create：frontend/src/features/study/topic-progress.tsx
- Create：frontend/src/features/study/knowledge-list.tsx
- Create：frontend/src/features/study/history-list.tsx
- Create：frontend/src/features/study/query.ts
- Create：frontend/src/features/study/query.test.ts
- Create：frontend/src/features/study/history-list.test.tsx
- Modify：frontend/src/app/learn/page.tsx
- Modify：frontend/src/app/learning-history/page.tsx
- Modify：frontend/src/features/auth/credentials-form.tsx
- Modify：frontend/src/app/login/page.tsx
- Modify：frontend/src/components/auth-status.tsx
- Modify：frontend/src/lib/i18n/messages/en.ts
- Modify：frontend/src/lib/i18n/messages/zh-CN.ts
- Modify：frontend/src/lib/i18n/page-titles.ts
- Modify：docs/operations/ui-language-coverage.json
- Create：tests/e2e/topic-study-navigation.spec.ts

**Interfaces（接口）：** parseStudyQuery(searchParams):{mode:'learn'|'review',topicId?,state?,q?,cursor?,limit}|null；Overview({data})、TopicProgress({progress})、StudyKnowledgeList({page})、StudyHistoryList({page,filters})。正常登录与已有正常会话访问/login进入/learn，mustChangePassword进入/account；注册仍/login。复习检索仅本人学过的知识，未完成知识进入继续学习，不显示开始复习。

- [x] **步骤1 写行为失败测试。** 登录正常/临时密码/已登录/身份故障四场景不同结果；topic-study-navigation 用真实数据断言完成率不因复习下降、未完成无复习按钮、待归类有入口、等时间戳下一页不漏；字段q含中文按UTF-8字节限制，不把日期截止解释为本机host日期。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```ts
expect(page.url()).toContain('/learn');
expect(await page.getByRole('button', {name: 'Start review'}).count()).toBe(0);
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/study/query.test.ts src/features/study/history-list.test.tsx。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 替换个人页面数据源为v2，不混用旧passed/unlocked统计。保留知识版本和内容更新标记，历史可按知识、主题、日期筛选；新版学习无需Join route。根据topic_learning_state.experience_mode选择旧/新体验，阶段测试用harness模拟topics，不在真实库切换。
- [x] **步骤4 验证通过。** 重跑步骤2命令；运行 topic-study-navigation.spec.ts、原 auth.spec.ts 和 auth-security.spec.ts；各批独立480秒，密码流程和语言未保存输入保持。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 提供主题学习中心与复习历史'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 8 B8 个人学习真实回归与门禁

**Files（文件）：**
- Create：backend/internal/e2etest/study_fixture.go
- Create：backend/internal/e2etest/study_control.go
- Modify：backend/internal/e2etest/harness.go
- Create：tests/e2e/topic-study-progress.spec.ts
- Create：tests/e2e/topic-study-notes.spec.ts
- Create：tests/e2e/topic-study-security.spec.ts
- Create：backend/internal/store/study_capacity_test.go
- Create：docs/operations/topic-study.md
- Modify：tools/verify/topic-learning-compatibility.mjs
- Modify：api/topic-learning-compatibility-baseline.json
- Modify：.github/workflows/backend.yml
- Modify：.github/workflows/frontend.yml

**Interfaces（接口）：** harness scene='topic-study' 通过实际角色、内容发布、分类发布、v2命令形成数据；容量场景实际约束启用，原创1000知识/500用户/100,000时间线事件，私人材料不作正式课程。代码日志只输出固定错误和requestId，不输出笔记。

- [x] **步骤1 写行为失败测试。** 真实双视口测试覆盖打开→开始→笔记→完成→复习→时间线；跨账号、会话撤销、强制改密、timeout同键、分类移动、材料升级/撤回、SSR/预加载。TestStudyCapacityReadPages 验证分页及聚合<=8秒、响应<=2MiB、发布提醒不随用户数写入增长。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if elapsed > 8*time.Second || responseBytes > 2<<20 { t.Fatal(elapsed, responseBytes) }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestStudyCapacityReadPages$' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 分批加入新CI步骤并保持默认legacy旧用例，精确扩展语言覆盖和兼容门禁例外；用户笔记输入截图遮盖，禁trace/video，测试harness不进生产镜像。执行同一最终提交整分支审查，按问题修复必要回归。
- [x] **步骤4 验证通过。** 重跑步骤2命令；总计划矩阵通过并记录SHA/预算；建立阶段B草稿MR，API/页面可在隔离环境使用，正式退出和迁入尚由阶段C完成。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'test: 验收主题学习与私人记录'。提交前 git diff --check 通过，只纳入该任务文件。

## 阶段交付条件

B1—B8 通过，具名新API独立于旧测评，真实账户场景、笔记归属和状态/进度规则通过。真实库 experience_mode 仍保持原值；仅测试控制端可以模拟topics体验。阶段C接管实际迁入、410旧写入口、提醒整合和显式切换。

## 实际阶段验收

B1—B8已按Native逐任务提交和验证，最终整分支独立审查发现的两项输入丢失及删除后比较区残留已通过四项实际RED→GREEN修复。旧模块退出与真实迁入继续阶段C，正式体验未切换。完整裁决和验证见[个人主题学习说明](../../operations/topic-study.md)。
