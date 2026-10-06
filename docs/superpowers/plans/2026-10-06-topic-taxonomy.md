# 主题分类与知识归属执行计划

> **For agentic workers（致执行代理）：** REQUIRED SUB-SKILL：按用户选择使用 superpowers:executing-plans（Native，推荐）或 superpowers:subagent-driven-development。按任务逐项执行，以 - [ ] 跟踪步骤。计划审核及执行方式未确认前，不写产品代码。

**Goal（目标）：** 提供新的三级知识地图、主题详情、准确归属与原知识发布配对能力。

**Architecture（架构）：** 新 taxonomy 模块保存独立分类和归属版本，原知识正文及审核合同保留。分类、归属与知识发布在现有 PostgreSQL 事务中配对激活；前端从新 API 分层读取。

**Tech Stack（技术栈）：** Go 1.27.1、PostgreSQL 17.11、Next.js 16.3.7、React 19.3.0、TypeScript 5.9.3、Node.js 24.17.0；沿用现有 Vitest、Playwright、Zod、goose、pgx，不新增运行依赖。

**Spec（设计）：** [用户已确认的主题学习重构设计](../specs/2026-10-06-topic-learning-refactor-design.md)，2026-10-06 获书面确认。本计划待审。

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

1. 源文件合法但在读取期间被 dot 更新：拒绝混合批次，上一发布保持（A1 TestTopicCaptureChangesDuringRead）。
2. 记录只有导言、软件能力或辅助语义映射：保留来源信息但不虚构具体主题知识（A2 topic-adapter.test.mjs）。
3. 知识移动、同时属于多个具体主题：各祖先计数去重，正文 SHA 保持（A4 TestTaxonomyAncestorCounts）。
4. 审核后撤权、两个管理员竞争激活：不能发布一半分类或继承未经核验的自审（A6 TestTopicPairRevocationRace）。
5. 用户以旧 domain ID、中文多字节搜索或深层叶节点直达：可定位或明确未归类，不能错误重定向（A7 topic-navigation.spec.ts）。

## 文件职责与依赖

| 单元 | 文件责任 | 前置 |
| --- | --- | --- |
| 捕获与适配 | tools/topic-ingest 只读捕获、归一化和草稿文件；不写数据库、不读取资料中的指令 | 无 |
| 纯分类合同 | backend/internal/taxonomy、schemas/topic-*.schema.json 定义分类、归属与规范摘要 | A1、A2 |
| 持久化与只读 API | 00010、store/taxonomy_*、httpapi/taxonomy_* 保存分类并分层查询 | A3 |
| 归属工作流 | 同步现有工作区保存、送审与审核，旁路保存分类归属 | A4 |
| 配对发布与页面 | 同事务切换知识与归属，新地图和主题页读取准确配对 | A5、A6 |

迁移00010_topic_taxonomy.sql 创建 taxonomy_versions、taxonomy_nodes、taxonomy_source_batches、taxonomy_assignment_drafts、taxonomy_submission_assignments、taxonomy_review_bindings、taxonomy_releases、taxonomy_release_assignments、taxonomy_heads、taxonomy_idempotency 和 topic_learning_state。最后一表只有单例配置 experience_mode（legacy 或 topics）、study_enabled、retired_at；初值 legacy/false/null。公开分类读取只要求分类发布，不要求开启个人学习。

taxonomy_heads 首次激活后旧 ActivateRelease 必须拒绝单独切换知识 head。新配对激活内部复用现有事务原语；相同锁顺序为旧管理锁、旧内容发布锁、分类 head、相关工作区/账户，不新增可由请求指定的任意锁。

### Task 1 A1 捕获已接受分类批次

**Files（文件）：**
- Create：tools/topic-ingest/capture.mjs
- Create：tools/topic-ingest/capture.test.mjs
- Create：tools/topic-ingest/test-fixtures.mjs

**Interfaces（接口）：** test-fixtures.mjs 提供 stagedFixture()、changingReadFixture() 和 acceptedFixture()，输出同一捕获参数形状；它们只指向测试临时目录。captureTopicBatch({sourceRoot,outDir,selectedPrimaryPaths,previousManifestPath?,signal}): Promise<TopicCaptureManifest>；TopicCaptureManifest={schemaVersion:1,batch:number,sourceFiles:[{path,sizeBytes,sha256}],snapshotId:string,diff:{added,changed,missing},issues:[{code,path}],accepted:boolean}。只在文件摘要、接受状态和同一批次一致时 accepted=true。

- [ ] **步骤1 写行为失败测试。** 写 capture_rejects_staged、capture_rejects_path_escape、capture_changes_during_read、capture_same_batch_is_stable。断言 staged_only=true 或 formal_integration_performed=false 被拒绝；读取中改字节不产生 accepted 清单；同一捕获字节的 snapshotId 相同。fixture 是原创最小元数据，不复制真实资料。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```js
await assert.rejects(() => captureTopicBatch(stagedFixture()), /BATCH_NOT_ACCEPTED/);
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs -- node --test tools/topic-ingest/capture.test.mjs。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 在 capture.mjs 实现上述入口：拒绝符号链接、目录穿越、未知/重复键、NUL；每份分类元数据最多8 MiB，所选主文件每份64 MiB、最多1,000 路径，整条工具经540 秒包装。输出新私有目录0700/文件0600；索引8 项摘要差异列为隔离，批次摘要绑定实际字节。保留旧 snapshot.mjs/source-report.mjs 行为。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；四类拒绝场景和稳定捕获通过，检查 sourceRoot 下无新增、删除或改写；对输出存在返回 OUTPUT_EXISTS，不覆盖上一批。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 增加主题分类批次捕获'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 2 A2 归一化主记录与生成可编辑草稿

**Files（文件）：**
- Create：tools/topic-ingest/adapters.mjs
- Create：tools/topic-ingest/build-drafts.mjs
- Create：tools/topic-ingest/adapters.test.mjs
- Create：tools/topic-ingest/build-drafts.test.mjs

**Interfaces（接口）：** normalizePrimaryFile({bytes,packageId,sourceId,path,sha256}): {records:NormalizedRecord[],auxiliary:AuxiliaryRecord[],issues:Issue[]}；NormalizedRecord={originalId,sourceId,title,originalKind,statement,conditions,proof,proofScope,sourceLocations,rawSHA}。buildDraftInputs({capture,records,resolutions,legacyCatalogue}): TopicDraftEnvelope[]；TopicDraftEnvelope={kind:'topic-draft',schemaVersion:1,draft:publication.DraftInput,assignments:AssignmentInput[],sourceBatchSHA}，每包最多100 知识。resolutions 显式给出网站 ID、合法数学 type 和具体主题，不猜测未知类型。

- [ ] **步骤1 写行为失败测试。** knowledge_points 与 records 各有原记录保留测试；assert auxiliary.length===104 对原创104 条能力夹具成立且它们不进入数学草稿；同作品同 ID 的副本去重，不同作品同标题不合并；缺少明确类型或具体主题解析时输出 issue，不伪造 proof、第二角度或来源批准。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```js
assert.equal(result.auxiliary.length, 104);
assert.equal(result.records.some(r => r.originalId === 'software-capability-1'), false);
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs -- node --test tools/topic-ingest/adapters.test.mjs tools/topic-ingest/build-drafts.test.mjs。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 实现这两个入口，来源映射保留原记录与实际摘要。正文未知字段留在受保护归档，不执行脚本。仅从显式 resolutions 形成现有 DraftInput；可保存的不完整草稿保留完整性缺项。先输出文件，再由正常工作区导入，不直接授予作者、角色或发布 head。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；检查每包知识<=100、原 JSON 主记录摘要可追溯、原索引未改；node 工具测试全通过。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 适配主题知识记录与草稿'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 3 A3 固定分类及归属合同

**Files（文件）：**
- Create：schemas/topic-catalogue.schema.json
- Create：schemas/topic-assignment.schema.json
- Create：backend/internal/taxonomy/model.go
- Create：backend/internal/taxonomy/validate.go
- Create：backend/internal/taxonomy/digest.go
- Create：backend/internal/taxonomy/validate_test.go
- Create：backend/internal/taxonomy/digest_test.go
- Modify：api/openapi.yaml
- Modify：frontend/src/lib/api/generated.d.ts
- Create：frontend/src/lib/taxonomy/types.ts
- Create：frontend/src/lib/taxonomy/schemas.ts
- Create：frontend/src/lib/taxonomy/schemas.test.ts

**Interfaces（接口）：** taxonomy.KnowledgeRef{ID string,Version int,SHA256 string}；TopicNode{ID,Code,Name,NameZh,Kind string,Level int,ParentID *string}；PairRef{KnowledgeHead,TaxonomyHead *string,TaxonomyVersionID string}；Query{Q,ParentID,Kind string,Level,Limit,Offset int}；Page[T]{Items []T,Total,Limit,Offset int,Pair PairRef}。CapturedBatch={manifest:TopicCaptureManifest,nodes:[]TopicNode,sourceRecordIndex:[]SourceRecordRef}；Version={ID,SnapshotID,TaxonomySHA string,Batch int}。ExperienceMode为legacy/topics；Kind 固定 primary/auxiliary/other；ValidateCatalogue([]TopicNode) error、CanonicalAssignment(AssignmentInput) ([]byte,string,error)。AssignmentInput={knowledge:content.VersionRef,topicIds:[]string,sourceRefs:[{sourceId,workFamilyId,recordId,path,sha256}],sourceBatchSHA:string}。

- [ ] **步骤1 写行为失败测试。** TestTaxonomyCountsAndParents 校验 63/534/4969、辅助503、兜底534、共6603；TestTaxonomyDuplicateCode、MissingParent、Cycle、WrongLevel 均失败；TestTaxonomyIDKeepsOfficialCode 断言 13C60 对应 msc-13c60；TestTaxonomyCanonicalDigest 断言顺序等价输入同摘要、来源或主题变化异摘要。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if counts.Level1 != 63 || counts.Level2 != 534 || counts.Specific != 4969 { t.Fatal(counts) }
if node.Code != "13C60" || node.ID != "msc-13c60" { t.Fatal(node) }
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/taxonomy -run '^TestTaxonomy' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 实现纯验证与摘要，二级父节点必须一级，具体主题父节点必须二级，辅助类型不混入正常统计。新 OpenAPI 定义具名 Topic* 合同，保留旧 schemaVersion=1 合同；响应禁止公开 sourceRefs 私有路径。同步定义 TS 具名类型/严格 schema，A5/A6 的纯组件以这些合同和回调接入；网络接线在A7完成。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；JSON Schema 与 Go 拒绝同一批错误输入；运行 npm run api:generate 更新生成类型，再 typecheck；旧合同差异只允许追加具名合同。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 定义三级主题与归属合同'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 4 A4 分类持久化与分层查询

**Files（文件）：**
- Create：db/migrations/00010_topic_taxonomy.sql
- Create：backend/internal/taxonomy/repository.go
- Create：backend/internal/taxonomy/service.go
- Create：backend/internal/store/taxonomy_schema.go
- Create：backend/internal/store/taxonomy_read.go
- Create：backend/internal/store/taxonomy_import.go
- Create：backend/internal/store/taxonomy_fixture_test.go
- Create：backend/internal/store/taxonomy_read_test.go
- Create：backend/internal/store/taxonomy_schema_test.go
- Create：backend/internal/cli/topic_catalogue.go
- Create：backend/internal/cli/topic_catalogue_test.go
- Create：backend/cmd/topic-catalogue-import/main.go

**Interfaces（接口）：** Store.InstallTaxonomyBatch(ctx,capture:taxonomy.CapturedBatch) (taxonomy.Version,error)、ListTopics(ctx,q:taxonomy.Query) (taxonomy.Page[taxonomy.TopicSummary],error)、ReadTopic(ctx,id string) (taxonomy.TopicDetail,error)、ListTopicKnowledge(ctx,id string,q taxonomy.Query) (taxonomy.Page[taxonomy.KnowledgeSummary],error)。TopicDetail={summary:TopicSummary,pair:PairRef}；TopicSummary 包含 TopicNode、Ancestors []TopicNode、PublishedKnowledgeCount、HasChildren；KnowledgeSummary 包含 KnowledgeRef、Title、TitleZh、TopicIDs。新增 newTaxonomyFixture(t) 返回{Repo,Ctx,SourceBatch,Access(name,recent),Ref(),CountSQL(query,args...)}，提供真实隔离库、原创目录与公开知识，不读真实资料。topic-catalogue-import --archive 只安装状态为draft的分类/sourceRecordIndex，读取受保护DATABASE_URL并记录system-import事实，不伪造人员批准，不写任何head；首个公开目录仍由管理员配对激活。

- [ ] **步骤1 写行为失败测试。** TestTaxonomySchemaEmptyRoundTrip 与 NonemptyDownDenied；TestTaxonomyAncestorCounts 令一个知识归属两个兄弟叶节点，祖先数仍为1；TestTaxonomyQueriesIgnoreUnpublished；TestTaxonomyBodySHAUnchanged；限20默认、100上限和512 UTF-8 字节搜索，百分号/下划线按字面查询。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if root.PublishedKnowledgeCount != 1 { t.Fatal("shared knowledge counted twice", root) }
if beforeSHA != afterSHA { t.Fatal("classification rewrote mathematics") }
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestTaxonomy' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 在00010 建立唯一代码、父节点外键、版本/发布封存不可修改、归属去重和必要查询索引。查询同一只读快照内的知识与分类配对，只计当前有效知识；源安装是内部存储入口，CLI 仅作为有数据库凭据的受控运维导入，管理员公开发布仍使用正常会话与近期认证，不能通过公开 JSON 写源版本。archive最多32 MiB且每个元数据单体8 MiB，原始数学正文不入此archive；安装批次不表示已批准。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；旧00001—00009 SHA 不变、迁移只在随机库运行；删除非空历史被拒绝；查询不包含草稿标题、来源私有路径或原始资料。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 存储主题分类并提供分层查询'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 5 A5 冻结知识主题归属并参加审核

**Files（文件）：**
- Create：backend/internal/store/taxonomy_workspace.go
- Create：backend/internal/store/taxonomy_review.go
- Create：backend/internal/store/taxonomy_workflow_test.go
- Modify：backend/internal/store/workflow_draft.go
- Modify：backend/internal/store/workflow_submission.go
- Modify：backend/internal/store/workflow_review.go
- Create：frontend/src/features/content/topic-fields.tsx
- Create：frontend/src/features/content/topic-fields.test.tsx
- Modify：frontend/src/features/content/draft-editor.tsx
- Modify：frontend/src/features/content/review-panel.tsx

**Interfaces（接口）：** SaveDraftTopics(ctx,a publication.Access,draftID string,in taxonomy.DraftTopicInput) (taxonomy.DraftTopicView,error)；DraftTopicInput={expectedDraftRevision:int64,expectedAssignmentRevision:int64,taxonomyVersionId:string,member:AssignmentInput}，一次保存一个知识点的归属，严格遵守8 KiB控制请求；DraftTopicView={draftId,draftRevision,assignmentRevision,taxonomyVersionId,members:[]AssignmentInput,digest,readyToSubmit}。freezeDraftTopicsTx(ctx,tx,draftID,submissionID string,revision int64) error、bindTopicReviewTx(ctx,tx,submissionID,reviewID string) error 仅为 store 内部事务函数。TopicFields({draftId,draftRevision,value,onSaved}) 展示准确主题及来源；保存内容后再保存对应 revision 的当前知识归属，未确认或其他知识归属缺项均不送审。

- [ ] **步骤1 写行为失败测试。** TestTopicAssignmentOwnerOnly、TestTopicAssignmentStaleRevision、TestTopicFrozenAssignmentIgnoresWorkspace、TestTopicAuthorReviewBoundary；断言 save 后改变该知识正文或来源使其旧归属不可送审，普通作者不能批准，admin+reviewer 可按既有例外自审。组件断言内容已保存但归属失败时保留选择并显示未完成，不假报全部保存。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if !errors.Is(err, publication.ErrDraftConflict) { t.Fatal("stale assignment accepted", err) }
if frozenDigestBefore != frozenDigestAfter { t.Fatal("frozen topics changed") }
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestTopic(Assignment|Frozen|Author)' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 实现工作区独立归属修订并在现有 SubmitDraft/DecideReview 事务内冻结、绑定同一决定；SaveDraft同步比较每知识正文与来源摘要，不变的归属可随当前工作区revision重新绑定，发生变化的归属变为待核对，避免无关编辑要求全包重复保存。不修改原 FrozenBody 字节。送审时有明确归属则强制冻结；首次兼容的旧已发布知识可待归类，最终 topics 切换后新送审必须有具体主题。审核页加载对应冻结归属，确认 Relationships 时覆盖其核对。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；再运行 topic-fields.test.tsx 与既有内容作者/审核用例；归属失败整次送审回滚，旧数学和作者继承记录不被洗掉。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 让知识主题归属进入冻结审核'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 6 A6 知识与归属配对发布及撤回

**Files（文件）：**
- Create：backend/internal/taxonomy/release.go
- Create：backend/internal/store/taxonomy_release.go
- Create：backend/internal/store/taxonomy_release_test.go
- Modify：backend/internal/store/workflow_release.go
- Modify：backend/internal/store/workflow_withdrawal.go
- Modify：frontend/src/features/content/publication-panel.tsx
- Modify：frontend/src/features/content/diff-panel.tsx

**Interfaces（接口）：** PrepareTopicRelease(ctx,a publication.Access,in taxonomy.PrepareInput) (taxonomy.ReleaseView,error)、ActivateTopicRelease(ctx,a publication.Access,id string,in taxonomy.ActivateInput) (taxonomy.ReleaseView,error)。PrepareInput={submissionIds:[]string,expectedPair:PairRef,reason:string}；ActivateInput={expectedPair:PairRef,manifestSHA:string,reason:string}。applyTopicWithdrawalTx(ctx,tx,knowledgeHead string,removed []taxonomy.KnowledgeRef) error；ReleaseView={id,status,pair,manifestSHA,assignmentsSHA,knowledgePublicationId,diff,createdAt}，diff={added:[]KnowledgeRef,removed:[]KnowledgeRef,changedTopicMemberships:[{knowledge,oldTopicIds,newTopicIds}]}。旧 ActivateRelease 的核心拆为内部 activateWorkflowReleaseTx，保留角色、5 分钟认证、重放和原审核证据核验。

- [ ] **步骤1 写行为失败测试。** TestTopicPairAtomicFailure 断言归属写入故障两 head 均不变；TestTopicPairRevocationRace 断言审核后撤权不能激活；TestTopicPairStaleHead、TestTopicLegacyActivationCannotBypassPair、TestTopicWithdrawalKeepsHistory。分类调整断言知识 SHA 原值、界面显示准确新增/移出差异。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if !sameWorkflowHead(beforeKnowledge, afterKnowledge) || beforeTaxonomy != afterTaxonomy { t.Fatal("partial publication") }
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestTopic(Pair|Legacy|Withdrawal)' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 在旧一致锁顺序中准备和激活完整配对，首次可发布空知识分类目录；任何 head 改变重新准备。taxonomy_heads 存在后，旧单独激活以既有 PUBLICATION_STALE 拒绝，v2 配对入口采用明确预期 head；撤回事务生成过滤后的归属发布，不能留下指向受撤回当前正文的可用成员。当前审核者资质与自审角色重新核验。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；并发和故障用实际 SQL barrier 验证，不靠 sleep；保留 TestActivationReplayDoesNotRestoreOldHead、TestActivationInheritedApprovalSurvivesReviewerRoleLoss；密码验证不自动激活。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 配对发布知识与主题归属'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 7 A7 公开主题 API 和三级浏览页面

**Files（文件）：**
- Create：backend/internal/httpapi/taxonomy_routes.go
- Create：backend/internal/httpapi/taxonomy_dispatch.go
- Create：backend/internal/httpapi/taxonomy_json.go
- Create：backend/internal/httpapi/taxonomy_test.go
- Modify：backend/internal/httpapi/application.go
- Modify：backend/cmd/server/main.go
- Create：frontend/src/lib/taxonomy/server-client.ts
- Create：frontend/src/lib/taxonomy/client.ts
- Create：frontend/src/lib/api/taxonomy-proxy.ts
- Create：frontend/src/lib/api/topic-management-proxy.ts
- Create：frontend/src/lib/taxonomy/management-client.ts
- Create：frontend/src/lib/taxonomy/management-server-client.ts
- Create：frontend/src/app/api/v2/content/topic-assignments/[[...segments]]/route.ts
- Create：frontend/src/app/api/v2/admin/publications/[[...segments]]/route.ts
- Create：frontend/src/app/api/v2/topics/[[...segments]]/route.ts
- Create：frontend/src/app/topics/[id]/page.tsx
- Create：frontend/src/features/catalogue/topic-view.tsx
- Modify：frontend/src/features/catalogue/knowledge-map.tsx
- Modify：frontend/src/features/catalogue/domain-view.tsx
- Modify：frontend/src/app/page.tsx
- Modify：frontend/src/app/knowledge/page.tsx
- Modify：frontend/src/app/domains/[id]/page.tsx
- Modify：frontend/src/lib/i18n/messages/en.ts
- Modify：frontend/src/lib/i18n/messages/zh-CN.ts
- Modify：frontend/src/lib/i18n/page-titles.ts
- Modify：frontend/src/lib/i18n/page-titles.test.tsx
- Modify：docs/operations/ui-language-coverage.json
- Create：tests/e2e/topic-navigation.spec.ts

**Interfaces（接口）：** GET /api/v2/topics?parentId=&level=&kind=&q=&limit=&offset=、/{id}、/{id}/knowledge；新题型之外的归属管理路由固定为 /api/v2/content/topic-assignments/drafts/{id} 及 /submissions/{id}，配对发布为 /api/v2/admin/publications。taxonomy 服务的管理入口使用 publication.Access，公开入口不读取个人身份。readServerTaxonomy<T>(route):Promise<TaxonomyResult<T>>；readServerTopicManagement<T>(route,cookieHeader)只GET；topicManagementRequest<T>(route,input,key,signal):Promise<TaxonomyResult<T>>通过当前auth context附加CSRF并从点击开始限10秒；TopicView({detail,children,knowledge})；TaxonomyResult={ok:true,data:T}|{ok:false,status,code,requestId}。内部 origin、有限字节读取和严格响应校验复用现有模式，不允许任意代理 URL。

- [ ] **步骤1 写行为失败测试。** TestTaxonomyHTTPQueryBoundaries 断言 q 的512 字节边界、重复键400、任意路径404、辅助类别不混主计数；topic-navigation.spec.ts 用真实 Go/PostgreSQL 断言一级63、二级534、叶4969按页加载、深链接祖先路径、中文搜索、零内容提示、旧 ID 定位或明确待归类；断言初次页面不下载全树。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```ts
expect(firstPage.items).toHaveLength(63);
expect(await page.getByText('13C60', {exact: true}).count()).toBe(1);
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/httpapi -run '^TestTaxonomy' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 接线公开与受保护管理 API，再实现分层地图、板块和主题页。给旧16 domain ID 建立人工可审查别名表，合并/拆分返回目标集合，不随机跳转。浏览知识正文仍使用现有安全 reader；分类显示不修改正文。添加 topicAssignment 管理代理和客户端在 lib/taxonomy 内，表单消费 A5/A6 接口。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；运行 npm run api:generate、typecheck、目标单元和 topic-navigation.spec.ts；双视口与中英文通过；私有管理路由匿名401，公开结果无 sourceRefs。先构建 e2e-harness 和生产 frontend 再运行浏览器，命令见总计划。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'feat: 提供三级主题地图与详情'。提交前 git diff --check 通过，只纳入该任务文件。

### Task 8 A8 分类交付回归与容量门禁

**Files（文件）：**
- Create：tools/verify/topic-learning-compatibility.mjs
- Create：tools/verify/topic-learning-compatibility.test.mjs
- Create：api/topic-learning-compatibility-baseline.json
- Create：backend/internal/store/taxonomy_capacity_test.go
- Modify：backend/internal/e2etest/harness.go
- Create：backend/internal/e2etest/taxonomy_fixture.go
- Create：backend/internal/e2etest/taxonomy_control.go
- Create：docs/operations/topic-taxonomy.md
- Modify：.github/workflows/backend.yml
- Modify：.github/workflows/frontend.yml

**Interfaces（接口）：** verifyTopicCompatibility({root,stage:'taxonomy'|'study'|'cutover'}): Report；原保护文件 SHA 保持，允许变更必须精确列路径、阶段及经审查的目标摘要，不能重算当前文件来放行。e2e 新 scene='topic-catalogue' 使用原创 fixture 安装、实际送审和配对发布；生产 server 不导入 e2etest。

- [ ] **步骤1 写行为失败测试。** 新增 TestTaxonomyCapacity6603With1000Knowledge：6,603 节点+原1,000 知识上限，查询分页/聚合在8 秒内，响应<=2 MiB；兼容测试篡改任一旧迁移、原 schemaVersion=1、未列文件或例外摘要必须失败。
关键断言（放入本任务上列具名测试，局部变量由该用例安排）：

```go
if elapsed > 8*time.Second || responseBytes > 2<<20 { t.Fatal(elapsed, responseBytes) }
```

- [ ] **步骤2 确认失败。** 运行 node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestTaxonomyCapacity6603With1000Knowledge$' -timeout 5m -count=1。预期因本任务行为缺失失败；先排除环境、依赖和数据库未配置，不能把跳过当成红灯。
- [ ] **步骤3 实现交付单元。** 补齐真实 fixture、按新增精确步骤扩展 CI 和旧字节门禁的兼容映射，不删除原回归或放宽超时。独立执行容量场景，完整验收结果记录代码 SHA、环境和原预算；手册区分正式分类与知识发布。A1—A7 的局部单元测试在开发时就使用该兼容检查，A8 汇总交付。
- [ ] **步骤4 验证通过。** 重跑步骤2命令；按总计划矩阵跑 Node/Go/内容账户/公开阅读与分类浏览器；同一最终提交一次独立整分支审查，修复后只复跑受影响检查；创建阶段A MR，不合并、不部署。
- [ ] **步骤5 提交。** git add 本任务上述实际变更文件，再执行 git commit -m 'test: 验收主题分类与发布兼容'。提交前 git diff --check 通过，只纳入该任务文件。

## 阶段交付条件

A1—A8 全部完成、真实数据库及双视口分类流程通过、独立整分支审查无重要未解决项，分类数量及不可变正文检查通过。可用三级目录与主题归属为本阶段成果，不称个人学习或旧模块退出已经交付。阶段B消费本阶段的 KnowledgeRef、PairRef、公开查询和发布事务挂点。
