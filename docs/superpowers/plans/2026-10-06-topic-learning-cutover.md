# 旧模块退出与学习迁移执行计划

> **For agentic workers（致执行代理）：** REQUIRED SUB-SKILL：按用户选择使用 superpowers:executing-plans（Native，推荐）或 superpowers:subagent-driven-development。按任务逐项执行，以 - [ ] 跟踪步骤。计划审核及执行方式未确认前，不写产品代码。

**Goal（目标）：** 完成新旧能力分离、旧记录准确迁入、测评写入口退出与受控体验切换。

**Architecture（架构）：** 旧表和冻结结果保留，旧题目历史继续使用原授权/曝光/撤回保护。新内容反馈与study使用独立能力，切换由有限维护命令在确认结构和迁入后记录，回退依赖兼容二进制。

**Tech Stack（技术栈）：** Go 1.27.1、PostgreSQL 17.11、Next.js 16.3.7、React 19.3.0、TypeScript 5.9.3、Node.js 24.17.0；沿用现有 Vitest、Playwright、Zod、goose、pgx，不新增运行依赖。

**Spec（设计）：** [用户已确认的主题学习重构设计](../specs/2026-10-06-topic-learning-refactor-design.md)，2026-10-06 获书面确认。本计划已获用户书面确认。前置：[阶段A](2026-10-06-topic-taxonomy.md)、[阶段B](2026-10-06-topic-study.md)均通过门禁；实现前重新同步已审查依赖。

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

1. 没有题库发布或旧学习结构受限：网站/知识反馈应可用；旧题目讨论仍严格拒绝不完整保护（C1）。
2. 旧记录只有检测通过、诊断或解锁，没有明确完成：不得生成completed或虚构时间（C3）。
3. 旧检测仍active或管理员调用旧写URL：410不改变原答案/结果，历史答案仍按到期与重合保护（C2）。
4. 重跑迁入、已有新笔记、两次并发切换：不覆盖用户新记录、不重复事件、只切换一次（C3、C4）。
5. 新增迁移后旧部署回退：schema不兼容时保持维护隔离，不执行真实down或恢复旧写能力（C6）。

## 文件职责与退出规则

迁移00012_topic_cutover.sql 创建 topic_cutovers、study_migration_batches、study_legacy_event_links。Migration 12 不自动改 experience_mode。旧00001—00009、所有数学冻结字节、答案和成绩不变。

topics模式下拒绝旧路线加入、旧个人开始/完成和解锁写、练习新建/作答/揭示/放弃、检测新建/提交/放弃、题库草稿/校验/送审/审核/激活/撤回新写，以及新判分案件、题目映射、纠错计划写。明确历史读取白名单保持本人/岗位权限；已有旧worker任务可完成历史证据，不改变study。其余现有账户、知识工作区/审核、v2配对发布、知识撤回及内容反馈继续可用。

MODULE_RETIRED 在旧相关错误合同中仅追加具名410响应；它是显式批准的退出兼容变化，不使用503伪装未配置。最终切换前仍为legacy行为，方便原回归验证；最终切换后旧UI显示退出说明，不提供新提交。

### Task 1 C1 反馈按来源拆分能力

**Files（文件）：**
- Modify：backend/internal/store/feedback_tx.go
- Modify：backend/internal/store/feedback_targets.go
- Modify：backend/internal/store/feedback_read.go
- Modify：backend/internal/store/feedback_exposure.go
- Create：backend/internal/store/feedback_topic_mode_test.go
- Modify：frontend/src/features/feedback/new-form.tsx
- Modify：frontend/src/features/feedback/report-link.tsx
- Create：tests/e2e/topic-feedback.spec.ts

**Interfaces（接口）：** feedbackBaseConfigured(ctx,tx) error 只检查账户/内容/反馈；feedbackSourceConfigured(ctx,tx,b feedback.Binding) error 在practice/assessment历史绑定时核验原learning/question/correction守卫。网站和知识来源既有接口、分类网站反馈引用{topicId,taxonomyVersionId}只作为展示定位，不自动决定数学替代发布。

- [x] **步骤1 写行为失败测试。** TestFeedbackTopicModeWithoutQuestionBank：网站/知识工单与纯文本讨论正常；TestFeedbackTopicModeLegacyOverlap：活动旧检测重合仍409、不交付文本；TestFeedbackTopicModeOwnerCannotHandle、自有主题定位及准确知识版本保留。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if status != 409 || responseContainsDiscussion { t.Fatal("legacy overlap bypassed") }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedbackTopicMode' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 拆掉无条件feedbackConfigured旧题库依赖，在真实Binding解析后按来源守卫；站点工单不因退休模块失败。原题目曝光事务保持，失败不交付；回复和状态变更保持五状态、序号及同键回执。
- [x] **步骤4 验证通过。** 重跑步骤2命令；旧 TestFeedbackExposureFailureClosed、OriginalTemplateExposure、ExpiredActive 回归通过；topic-feedback.spec.ts真实双视口通过，不能靠不调用曝光函数替代安全检查。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'refactor: 按来源解耦内容反馈'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 2 C2 显式退出旧写接口和题库界面

**Files（文件）：**
- Create：backend/internal/httpapi/topic-retirement.go
- Create：backend/internal/httpapi/topic_retirement_test.go
- Modify：backend/internal/httpapi/application.go
- Modify：backend/internal/httpapi/learning_dispatch.go
- Modify：backend/internal/httpapi/question_dispatch.go
- Modify：backend/internal/httpapi/correction_dispatch.go
- Modify：backend/internal/store/learning_tx.go
- Modify：backend/internal/store/question_tx.go
- Modify：backend/internal/store/correction_tx.go
- Modify：backend/internal/study/model.go
- Modify：backend/internal/store/learning_paths.go
- Modify：backend/internal/store/assessment_commands.go
- Modify：backend/internal/store/practice_commands.go
- Create：backend/internal/store/topic_retirement_test.go
- Modify：api/openapi.yaml
- Modify：frontend/src/lib/api/generated.d.ts
- Modify：backend/internal/httpapi/learning_error.go
- Modify：backend/internal/httpapi/question_error.go
- Modify：backend/internal/httpapi/correction_error.go
- Modify：frontend/src/lib/learning/schemas.ts
- Modify：frontend/src/lib/question/schemas.ts
- Modify：frontend/src/lib/correction/schemas.ts
- Modify：frontend/src/lib/i18n/error-keys.ts
- Modify：frontend/src/components/site-header.tsx
- Modify：frontend/src/components/auth-status.tsx
- Modify：frontend/src/app/paths/[id]/page.tsx
- Modify：frontend/src/app/practice/[id]/page.tsx
- Modify：frontend/src/app/assessments/[id]/page.tsx
- Modify：frontend/src/app/editor/questions/page.tsx
- Modify：frontend/src/app/review/questions/page.tsx
- Modify：frontend/src/app/admin/question-publications/page.tsx
- Create：frontend/src/features/study/retired-module.tsx
- Create：tests/e2e/topic-retirement.spec.ts

**Interfaces（接口）：** retirementPolicy(mode taxonomy.ExperienceMode,route RetirementRoute) ('allow'|'retired'|'historical-read') 根据精确方法与路由判断，不按前缀一刀切；RetirementRoute={Method,Path,Module,Action string}仅由服务端路由解析生成，taxonomy.ExperienceMode='legacy'|'topics'在A3定义。study.ErrModuleRetired是服务层共同哨兵错误，HTTP退休响应={error:{code:'MODULE_RETIRED',message:'This module has been retired.',requestId}}，HTTP410、no-store。Store.ReadExperienceMode(ctx)返回模式，不写DB。数据库事务内写服务还需核验模式，不能只在HTTP上挡请求。

- [x] **步骤1 写行为失败测试。** TestTopicRetirementExactRouteMatrix 对C范围逐路径/方法断言，legacy允许原成功、topics写410、历史GET受保护、账户和知识发布仍可用；TestTopicRetirementConcurrentCutover 在服务事务barrier后切换，旧写不能提交。浏览器旧active显示停用、原score/answers字节不变、导航无题库/路线。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if mode == "topics" && status != 410 { t.Fatal("retired write accepted", status) }
if !bytes.Equal(beforeAnswers, afterAnswers) { t.Fatal("legacy facts changed") }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/httpapi ./internal/store -run '^TestTopicRetirement' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 加入请求及事务内当前模式核验，模式改变后幂等重放仍410，不复活旧写。受退出影响的交互写事务先对experience配置行FOR SHARE，再沿用旧锁顺序，持有至提交；仅从未启用新层且无配置的旧库默认legacy，永久标记已启用但配置或结构缺失时503，不退回旧写。旧历史新路径 /learning-history?archive=legacy&attempt=... 明确标档案、本人访问、终态结果走原保护；旧API列表保留安全元数据。新纠错页仅显示知识修订/撤回与回顾提醒，已有旧案件提供受保护历史入口，旧后台重试仅现有任务。
- [x] **步骤4 验证通过。** 重跑步骤2命令；新MODULE_RETIRED有中英文词条且生成类型通过；topic-retirement.spec.ts和旧答案/撤回/曝光安全用例分别执行；省略客户端权限不能绕过服务检查。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'refactor: 退出路线测评与题库新写流程'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 3 C3 迁入真实学习事件

**Files（文件）：**
- Create：db/migrations/00012_topic_cutover.sql
- Create：backend/internal/study/migration.go
- Create：backend/internal/store/study_migration.go
- Create：backend/internal/store/study_migration_test.go
- Create：backend/internal/cli/topic_learning.go
- Create：backend/internal/cli/topic_learning_test.go
- Create：backend/cmd/topic-learning-maintenance/main.go

**Interfaces（接口）：** MigrateLegacyStudyBatch(ctx,limit int,cursor *study.LegacyCursor) (study.MigrationReport,error)；cursor=(recordedAt,eventID)，limit1..50，每事务8秒，全部命令540秒；report={processed,createdEvents,linkedEvents,conflicts,cursor,done}。CLI topic-learning-maintenance migrate --batches=1..10 --limit=1..50、inspect、verify；DATABASE_URL仅从受保护环境读取，不支持命令行口令。

- [x] **步骤1 写行为失败测试。** TestStudyMigrationOnlyExplicitFacts：只迁learning_events started/completed；仅passed/diagnostic/unlocked不生成completed；TestStudyMigrationRetainsTimeAndVersion、RerunIdempotent、ExistingStudyNotOverwritten、WithdrawnKnowledgeHistory。断言原事件SHA/时间/计数不变，新事件带originEventId及sourceKind='legacy'。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if migrated.CompletedCount != explicitCompletionCount { t.Fatal("assessment pass invented completion") }
if firstRun.CreatedEvents != firstCount || secondRun.CreatedEvents != 0 { t.Fatal("migration not idempotent") }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -run '^TestStudyMigration' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 在00012建立user+legacyEvent唯一映射、迁入批次断点和只追加事件。迁入写由系统维护记录标识，不伪造用户会话；与用户新命令使用相同账户和record锁。存在更新的新记录时不覆盖state、sequence、笔记或最近字段，只补历史来源；按时间选择第一次开始/完成，保留首次和最近事实。
- [x] **步骤4 验证通过。** 重跑步骤2命令；每批终止/重跑从原游标继续；同微秒事件不漏；原答案/结果表对比完全相同。CLI无工作返回processed=0、done实际值，不宣称有限批次等于全库完成。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 迁入真实学习时间线'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 4 C4 体验切换检查与有限命令

**Files（文件）：**
- Create：backend/internal/store/topic_cutover.go
- Create：backend/internal/store/topic_cutover_test.go
- Modify：backend/internal/cli/topic_learning.go
- Modify：backend/internal/cli/topic_learning_test.go
- Modify：backend/internal/config/config.go
- Modify：backend/cmd/server/main.go

**Interfaces（接口）：** InspectTopicCutover(ctx) (study.CutoverReport,error)、ActivateTopicExperience(ctx,in study.CutoverInput) (study.CutoverReport,error)。CutoverInput={expectedPair:taxonomy.PairRef,codeSHA,expectedMigrationBatchId,reason}，允许空知识发布的KnowledgeHead=null，分类head必须有效。报告必须确认完整00010—00012结构、taxonomy配对、迁入done且conflicts=0、历史保护和兼容版本；CutoverReport={mode,pair,schemaReady,migrationDone,unmappedLegacyEvents,conflicts,cutoverId?,recordedAt?}。CLI activate 要求已有备份记录和显式参数，真实运行另按运维授权，测试只在随机库。

- [x] **步骤1 写行为失败测试。** TestTopicCutoverRefusesPendingMigration、RefusesPartialSchema、RefusesPairMismatch、SingleWinner、RetainsActiveAttemptBytes；触发截止竞争时mode保持legacy，正常切换仅一条审计记录，旧active行与冻结结果字节不变。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if successes != 1 || cutoverEvents != 1 { t.Fatal("cutover duplicated") }
if missingSchema && modeAfter != "legacy" { t.Fatal("unsafe cutover") }
```

- [x] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -run '^TestTopicCutover' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [x] **步骤3 实现交付单元。** 同一事务先对配置行FOR UPDATE，再按旧管理/内容锁顺序锁定预期pair和完成迁入批次；旧交互写已提交或等待该配置行，不存在切换后遗漏提交。用NOT EXISTS再查所有旧started/completed事件是否已有迁入链接，不能仅相信早先done；出现新事件则返回需继续迁入，保持legacy。codeSHA核对可信运行版本元数据，不能仅信请求声明。检查通过再改mode=topics、记录retired_at与不可变cutover事实；schema/个人写入冲突时明确失败。服务启动不自动迁移或切换。
- [x] **步骤4 验证通过。** 重跑步骤2命令；基于topics模式再次执行返回原切换记录，不更新日期或让旧mode恢复；实际旧写在事务内拒绝，新study与反馈独立可用。
- [x] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 增加受控主题学习切换'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 5 C5 内容回顾提醒与保留后台整合

**Files（文件）：**
- Create：frontend/src/features/study/content-reminders.tsx
- Create：frontend/src/features/study/content-reminders.test.tsx
- Modify：frontend/src/features/study/overview.tsx
- Modify：frontend/src/features/correction/case-list.tsx
- Modify：frontend/src/features/correction/case-panel.tsx
- Modify：frontend/src/features/correction/result-panel.tsx
- Modify：frontend/src/features/notification/inbox.tsx
- Modify：frontend/src/app/review/corrections/page.tsx
- Modify：frontend/src/app/notifications/page.tsx
- Modify：frontend/src/features/content/package-fields.tsx
- Modify：frontend/src/lib/i18n/messages/en.ts
- Modify：frontend/src/lib/i18n/messages/zh-CN.ts
- Modify：docs/operations/ui-language-coverage.json
- Create：tests/e2e/topic-content-corrections.spec.ts

**Interfaces（接口）：** ContentReminders({items:StudyContentReminder[]})；StudyContentReminder={changeId,knowledgeId,kind:'updated'|'withdrawn',recordedAt,currentRef?,reviewed:boolean}。查看提醒只读，完成复习由既有FinishStudyReview产生；topics模式不展示score/retake资格操作。内容工作区保留原数学字段、来源及主题选择，旧path字段仅历史读取，不要求新稿生成路线。

- [ ] **步骤1 写行为失败测试。** 提醒中文/英文切换不mark-read、不complete；已完成知识材料升级保留completed但显示更新；撤回正文不可读、私人笔记仍可读；知识修订须正常审核与发布，resolved工单不能直接发布。旧纠错Original/Corrected历史只对本人且遵守原重合保护。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```ts
expect(afterReview.Completed).toBe(beforeReview.Completed);
expect(markReadRequests).toHaveLength(0);
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/study/content-reminders.test.tsx。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 把当前学习回顾提醒放在我的学习，旧通知页在topics模式导向该入口并保留历史通知只读链接。纠错后台围绕知识工单、现有撤回和替代发布提供定位，停用新判分方案与题目映射编辑；既有worker关闭/开启都不影响新study判断。原审核角色与管理员自审例外保持。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；topic-content-corrections.spec.ts真实双视口通过；旧内容编写/审核/撤回回归，私人正文不进入提醒列表；新草稿paths=[]仍能完成正常送审。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'refactor: 整合内容回顾与知识纠错后台'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 6 C6 部署兼容与阶段门禁

**Files（文件）：**
- Modify：ops/deploy.py
- Modify：ops/verify-deployment.py
- Modify：ops/tests/test_deploy.py
- Modify：ops/tests/test_verify.py
- Modify：tools/verify/topic-learning-compatibility.mjs
- Modify：tools/verify/topic-learning-compatibility.test.mjs
- Modify：api/topic-learning-compatibility-baseline.json
- Create：docs/operations/topic-learning-cutover.md

**Interfaces（接口）：** topicSchemaCompatibility(actualMigration:int,targetCapabilities:{taxonomy,study,retirement}:bool,mode:str)->bool，在ops/deploy.py集中；旧程序无retirement且当前mode=topics时拒绝恢复私人流量。部署状态保留原snapshot/migration核验，不增加自动down或数据库清理。新的/readyz能力项只含安全schema与mode布尔，不含source路径/用户内容。

- [ ] **步骤1 写行为失败测试。** test_topic_mode_old_binary_rejected、test_topic_schema_partial_keeps_maintenance、test_compatible_binary_keeps_database；兼容门禁断言旧迁移、FrozenBody、成绩、原始来源指纹不允许变化。精确列批准修改的旧接口错误合同和旧activation行为，其他文件不新增泛化例外。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```python
self.assertFalse(topicSchemaCompatibility(12, old_capabilities, 'topics'))
self.assertEqual(database_digest_before, database_digest_after)
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs -- python3 -m unittest discover -s ops/tests -p 'test_*.py' -v。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 编写先备份、inspect、有限迁入、verify、activate、健康及恢复核验的中文手册；回退仅兼容新旧schema的应用，真实库不Down。CI将原业务场景保持legacy验证，另加topics退役场景；不用删除旧安全测试解决字节门禁冲突。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；运维全部目标测试、Node兼容门禁通过；用临时镜像/隔离库验证恢复，真实服务器保持原状；原一worker、480秒和数据库保护不放宽。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'test: 核验主题切换与部署兼容'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 7 C7 完整用户流程回归与交付

**Files（文件）：**
- Create：tests/e2e/topic-learning-cutover.spec.ts
- Create：tests/e2e/topic-learning-archive.spec.ts
- Create：backend/internal/store/topic_cutover_capacity_test.go
- Modify：backend/internal/e2etest/study_control.go
- Modify：backend/internal/e2etest/harness.go
- Modify：.github/workflows/backend.yml
- Modify：.github/workflows/frontend.yml
- Modify：.github/workflows/deployment.yml
- Create：docs/operations/topic-learning-acceptance.md
- Modify：README.md

**Interfaces（接口）：** 最终验收消费A、B、C所有合同；harness scene='topic-cutover' 先真实生成旧学习/检测历史，再执行有限迁入和activate，不手工写completed来伪造成功。TestTopicCutoverCapacity 用原创1,000知识/500用户/100,000事件，迁入每批50、单事务8秒。

- [ ] **步骤1 写行为失败测试。** 浏览器完整登录→主题→阅读→笔记→完成→复习→时间线→反馈→内容修订提示；旧学习事实迁入、旧诊断不算完成、历史结果保留、旧写410、多人笔记隔离、语言与手机布局。容量断言单批超时可重跑、断点连续、发布提醒不随用户数逐行插入。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if batchSize > 50 || elapsed > 8*time.Second { t.Fatal("migration batch exceeds budget") }
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestTopicCutoverCapacity$' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 按总计划矩阵完成两种模式所有回归，对同一最终提交开展一次独立代码审查；修复后复跑问题及受影响检查。验收记录代码SHA、每批命令/退出码/耗时、实际覆盖及限制，不使用真实课程或真实账号构造数据。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；最终本机矩阵、远端最新提交CI和独立审查通过后创建实现MR。MR只交付代码与迁移；合并、真实批准、实际切换和部署分别等待对应授权，不伪称网站已上线重构。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'test: 完成主题学习迁移回归'。提交前 git diff --check 通过，只纳入该任务文件。

## 最终交付条件

新分类、状态、笔记、时间线、复习、登录入口及保留后台全部可用；旧写410与受保护历史并存；三迁移与显式切换通过隔离回归；同一最终提交独立审查通过；每批预算保持。阶段实现完毕不自动运行真实迁入或部署。
