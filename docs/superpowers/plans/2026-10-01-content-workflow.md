# P3b 内容工作流 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.
>
> 本项目沿用用户此前选择的 Native：使用 superpowers:executing-plans 在当前会话逐项实现，最后一次独立整分支审查；不自动分派实现代理。实施步骤仅在验收通过后勾选。

**Goal:** 交付可追责的英文内容后台，使原创数学草稿经固定版本送审和独立复核后安全发布，问题版本可以原子撤回。

**Architecture:** publication 定义 DTO、状态规则和仓储接口，依赖 auth/content/catalogue，不能导入 store。store 在同一 PostgreSQL 事务内完成授权、版本冻结、审核、快照与 head 变更；HTTP 和 Next.js 分别承担严格私有输入和输出边界。复用 P1 不可变包及 P2 公开查询和安全阅读组件。

**Tech Stack:** Go 1.27.1、PostgreSQL 17.11、Node.js 24.17.0、Next.js 16.3.7、React 19.3.0、TypeScript 5.9.3；沿用当前锁文件，不新增产品依赖。

**Spec:** [P3b 已确认设计](../specs/2026-10-01-content-workflow-design.md)，用户于 2026-10-01 确认，已通过 [PR #11](https://github.com/yyl1212/math_master/pull/11) 合并。计划已于 2026-10-01 获用户书面确认，执行方式 Native；基线 master a4cf6278774b30cce3e7278a7582dd522c465be8 已包含 [PR #13](https://github.com/yyl1212/math_master/pull/13) 的限流测试修复。实施状态见下方各任务已勾选步骤。方案的七组顺序细化为十个可独立验证的任务，没有增加产品范围。

## 全局约束（Global Constraints）

- 优先级为内容正确性、数据数量与覆盖、学习路径、页面美观；界面以英文为主，保留中文数学术语对照，文档使用中文。
- 本阶段交付知识点、学习单元、路线和原创 SVG 的管理流程。P4/P5/P6/P7 的题库、资格、反馈重评、数量验收和部署保持各自范围。
- 数据运行时只来自 PostgreSQL；Knowledge_JSON 通过已有工具按全新目录离线冻结，原字节、摘要和来源映射保留。资料内说明是数据，不是指令；源文件缺失不自动撤回。
- 一个工作区只有一个可编辑负责人。admin 不自动拥有 editor/reviewer；所有角色仍有 learner。本人或继承作者集合中的账户不能批准本送审。
- 工作区 editing/submitted；送审 pending → approved 或 returned；退回恢复 editing 并递增 revision；批准后的修改建立新工作区。审核只绑定冻结正文和来源。
- UUID 均要求规范小写 v4；数学稳定 ID 沿用现有格式；版本在有符号 32 位正整数范围。schemaVersion=1、旧包摘要和公开响应保持兼容。
- 不重写 00001—00003；新增 00004_content_workflow.sql。既有 P1 draft 不继承批准，不直接成为公开版本。
- 操作说明与复核文本统一为 10—1000 个 Unicode 码点且最多 3000 个 UTF-8 字节，拒绝全空白与 NUL。
- 创建 / 保存正文 envelope 最多 8 MiB；其中序列化 package 最多 2 MiB，解码 SVG 总量最多 4 MiB，每项最多 1 MiB。
- 每个工作区至多 100 个知识点、200 个单元、20 条路线、16 项素材；其余控制请求最多 8 KiB；sourceMap 最多 256 KiB。
- 私有 JSON 响应最多 4 MiB；素材单独读取。列表 limit 默认 20、最大 100，offset 0—100000；查询参数不得重复，筛选使用固定枚举。
- 拒绝未知字段、大小写别名、重复键、错误 UTF-8、未配对代理项、NUL、尾随 JSON、超过 32 层嵌套、未允许的查询和非 JSON 正文；有效 Unicode 不做数学含义变更。
- 候选快照最多 1000 个知识点、4000 个单元、200 条路线、16000 条知识关系和 1000 项素材；规范 JSON 总量最多 32 MiB、唯一 SVG 字节总量最多 10 MiB；任一公开知识或路线响应最多 10 MiB。
- 内容请求总截止 8 秒，Next.js 代理总截止 10 秒，Go 服务器 15 秒读写超时保持；账户接口仍为原来的 4/5 秒、8 KiB 请求和原响应上限。
- 锁等待上限 1 秒。管理锁 1296127049 → 内容锁 1296127048 → UUID 排序用户行 → 操作者会话 → 内容行。CLI 只取内容锁，不反向取得管理锁。
- 有效账户每分钟读取 120、普通写入 30、重操作 10；全服务普通写入 120/分钟、重操作 40/分钟；每进程内容验证与大正文请求最多同时 2 个，不无限排队。
- 准备选择 1—20 个已批准送审，仅合并同一 catalogueVersion；激活核验 expectedHead、expectedManifestSHA、当前审核资格和撤回记录。
- 激活与撤回要求管理员在最近 5 分钟内重新验证；成功变更、审计、幂等结果同事务。重放不能重新切换旧 head。
- 私有 SVG 要求 image/svg+xml、nosniff、CSP sandbox 与 private,no-store，代理校验摘要/白名单，不转发 Set-Cookie。
- 缺少 origin 返回 AUTH_NOT_CONFIGURED；缺少内容迁移返回 CONTENT_NOT_CONFIGURED，既有账户及只读接口按原规则工作，不自动迁移。
- Go 全程 CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1；每次验证经过 tools/verify/run.mjs，最多 540 秒，Go 单批 -timeout 5m。
- 浏览器 workers=1、retries=0、单例 30 秒、globalTimeout=480000，桌面 1280×900、手机 390×844；拆批验证，不突破单次 10 分钟。
- 开发前使用 SSH 拉取最新 master，新建 codex/ 功能分支；Git PR 交付。真实开发库不自动迁移、初始化管理员或发布草稿。
- 不读取/打印 config/server.local.json、真实密码、GitHub 凭据或 .env 内容；需要本机测试连接时仅 set -a; source .env; set +a。随机测试库使用既有 testutil，保留开发库、固定连接库和资料快照。

## 审查重点（Review Focus）

1. 已批准内容的工作区后来退回、复制或修改，阅读/发布必须仍使用冻结 sourceMap、作者集合和字节；任务 3/4 的 TestFrozenSubmissionDoesNotReadMutableWorkspace、TestCopiedAuthorsCannotApprove 验证。
2. 相同 asset ID 换字节、旧单元复用或跨工作区请求摘要，不能混入新图或泄露私有素材；任务 2/5/8 的 TestWorkflowAssetReaderParity、TestReleaseRejectsOldUnitBinding、TestContentSVGScopeAndBoundary 验证。
3. 超时后主动重试，以及 head 已变化后的幂等重放，不能重复发布或覆盖新状态；任务 1/5/9 的 TestWorkflowIdempotencyRechecksPermission、TestActivationReplayDoesNotRestoreOldHead、TestPendingCommandManualRetry 验证。
4. 空 head 的 null、缺失字段、重复/大小写别名、孤立代理项和 chunked 超限不能被两端解码器悄悄改写；任务 7/8 的 TestContentJSONExactBoundary、TestContentProxyRawRequestBoundary 验证。
5. 大正文取消、锁等待时会话到期及验证槽仍在工作，不能早释放容量或用等待前身份写入；任务 1/7/10 的 TestWorkflowSessionExpiresWhileWaiting、TestContentSlotsSurviveCancellation、TestWorkflowCapacityEnvelope 验证。

## 架构、任务依赖与文件责任

~~~mermaid
flowchart TD
    UI[英文编辑、复核、发布与撤回] --> Client[content 客户端与严格响应]
    Client --> Proxy[Next.js 固定内容私有代理]
    Proxy --> HTTP[Go 内容 HTTP 边界]
    HTTP --> Service[publication.Service]
    Service --> Port[publication.Repository]
    Port --> Store[store：统一锁与事务]
    Store --> Check[content：机器校验与素材读取器]
    Store --> DB[(PostgreSQL)]
    Auth[auth：证明解码、角色及会话] --> Service
    Auth --> Store
    DB --> Public[既有公开快照读取]
~~~

~~~mermaid
flowchart LR
    T1[1 契约与事务基础] --> T2[2 校验与导入复用]
    T2 --> T3[3 草稿与固定送审]
    T3 --> T4[4 独立复核]
    T4 --> T5[5 快照与激活]
    T5 --> T6[6 撤回]
    T6 --> T7[7 Go HTTP 与 OpenAPI]
    T7 --> T8[8 Next.js 数据边界]
    T8 --> T9[9 英文管理页面]
    T9 --> T10[10 真实联调与独立审查]
~~~

| 文件责任 | 任务 |
| --- | --- |
| db/migrations/00004_content_workflow.sql；publication/{model,repository,service,policy,rate_limit}.go；auth/content_proof.go；store/workflow_{schema,tx,idempotency}.go 与测试 | 1 |
| content/{validate,assets}.go、asset_reader.go、workflow_validation.go、snapshot_validation.go 与测试；store/import.go 与测试 | 2 |
| publication/draft.go；store/workflow_{draft,submission}.go 与对应测试、workflow_fixture_test.go | 3 |
| publication/review.go；store/workflow_review.go、workflow_review_test.go、workflow_concurrency_test.go | 4 |
| publication/{release,manifest}.go；store/workflow_release.go 与测试；必要的 publication.go 读取辅助提取 | 5 |
| publication/withdrawal.go；store/workflow_withdrawal.go 与测试 | 6 |
| httpapi/content_{routes,json,error,assets}.go 与测试；application.go；cmd/server/main.go；api/openapi.yaml | 7 |
| frontend/src/lib/content/{types,schemas,client,server-client}.ts 与测试；lib/api/content-proxy.ts 与测试；content 两个路由；generated.d.ts | 8 |
| frontend/src/features/content/ 各管理组件与测试、content.module.css、七个 app 页面、site-header.tsx；SafeMarkdown/AssetImage 的可选素材路由参数 | 9 |
| e2etest/workflow_fixture.go 与测试、harness.go；tests/e2e/content-{authoring,review,release,security}.spec.ts；两份 workflow；运维与验收文档 | 10 |

表中的 Go 目录前缀为 backend/internal/，页面和样式前缀为 frontend/src/。不新增数据库外的正式内容源或依赖服务。

## 跨任务接口与数据契约

此节由任务 1 定义；后续任务按同名类型实现，不自行更名。公开 JSON 外层统一 {data:...}，错误沿用 {error:{code,message,requestId}}；所有新增成功均有正文，不使用 204。

### Go 类型

- publication.Access：TokenHash auth.Digest、CSRF auth.Secret、IdempotencyKey string、RequestID string；全部 json:"-"，读取不检查 CSRF，写入检查。actor 从数据库获得，不由 Access 或请求声明。
- publication.Action：listDrafts/readDraft/createDraft/saveDraft/adoptDraft/validateDraft/submitDraft/listSubmissions/readSubmission/reviseSubmission/decideReview/listPublications/readPublication/prepareRelease/activateRelease/previewWithdrawal/withdrawVersion/readDraftAsset/readSubmissionAsset 的固定枚举。
- SourceLink：Knowledge content.VersionRef、BatchSHA256/RelativePath/SHA256/LegacyID/Note string。sourceMap 为 []SourceLink；空数组允许原创输入，路径只能为不含空段、点段、反斜线和绝对前缀的相对路径；两个摘要要求 64 位小写十六进制。
- AssetInput：ID/Base64 string；DraftInput：CatalogueVersion int、Package content.Package、AssetBytes []AssetInput、SourceMap []SourceLink；SaveDraftInput 增加 ExpectedRevision int64。未知作者字段不能进入 DTO。
- AdoptInput：PackageID string、PackageVersion int、Reason string；ValidateInput：ExpectedRevision int64；SubmitInput：ExpectedRevision int64、ExpectedDigest string。
- GateReport：StructuralErrors/CompletenessErrors/HumanReviewRequirements []content.Issue，三项对应 Total int，Truncated/ReadyToSubmit bool、Digest string。总展示问题最多 100；ReadyToSubmit 按未截断的实际总量计算。
- DraftView：ID/OwnerID/CatalogueSHA256 string、CatalogueVersion int、Revision int64、Status string、Package content.Package、SourceMap []SourceLink、AuthorIDs []string、LegacyUnattributed bool、Assets []content.AssetView、Gate GateReport、CreatedAt/UpdatedAt string。
- FrozenBody：CatalogueVersion int、CatalogueSHA256 string、Package content.Package、SourceMap []SourceLink、AuthorIDs []string、LegacyUnattributed bool、Assets []content.AssetView、FrozenDigest string；全部来自冻结记录。
- ReviewChecks：Mathematics/Explanations/Relationships/Sources/Illustrations bool；ReviewInput：Decision string（approve/return）、Checks ReviewChecks、IndependenceNote/Note string。approve 的检查项全 true 且两段说明均合格；return 只强制 Note，Checks 可以 false、IndependenceNote 可为空。
- ReviewDecision：ID/SubmissionID/ReviewerID/FrozenDigest/Decision string、Checks ReviewChecks、IndependenceNote/Note/CreatedAt string；SubmissionView：ID/WorkspaceID/OwnerID/Status string、Revision int64、Frozen FrozenBody、Gate GateReport、Review *ReviewDecision、CreatedAt string。
- MemberIdentity：Kind/ID/PackageID/SHA256 string、Version/PackageVersion int；AssetBinding：Unit content.VersionRef、AssetID/SHA256 string（在 content 包定义）。
- MemberEvidence：SubmissionID/DecisionID/FrozenDigest string、InheritedFrom *string；ManifestMember：Identity MemberIdentity、Evidence MemberEvidence；Manifest：CatalogueVersion int、CatalogueSHA256 string、BaseHead *string、Members []ManifestMember、Bindings []content.AssetBinding。每项追溯原始批准，InheritedFrom 只说明沿用已发布依据。
- Change：Kind/ID/Reason string、Before/After *MemberIdentity；Diff：Added/Replaced/Removed int、Changes []Change。
- PublicationView：ID/Status/ManifestSHA/CreatedAt string、Manifest Manifest、Diff Diff；PrepareInput：SubmissionIDs []string、ExpectedHead *string、Reason string；ActivateInput：ExpectedHead *string、ExpectedManifestSHA/Reason string。
- WithdrawalTarget：Kind string；knowledge/unit/path 分支仅有 ID string 和 Version int，asset 分支仅有 SHA256 string。不同分支的字段不能混入；内部规范目标补充从数据库核验的摘要。
- WithdrawalInput：Target WithdrawalTarget、ExpectedHead *string、Reason string；WithdrawalPreviewInput 只有 Target；WithdrawalPreview：CurrentHead *string、Target WithdrawalTarget、Diff Diff；WithdrawalResult：EventID string、PreviousHead *string、Publication PublicationView。
- ListQuery：Scope/Status string、Limit/Offset int；Page[T]：Items []T、Total/Limit/Offset int；PublicationPage 增加 Head *string。DraftSummary 固定为 ID/OwnerID/PackageID/Status/CreatedAt/UpdatedAt string、PackageVersion/CatalogueVersion/StructuralTotal/CompletenessTotal int、Revision int64；SubmissionSummary 为 ID/WorkspaceID/OwnerID/PackageID/Status/FrozenDigest/CreatedAt string、PackageVersion/CatalogueVersion int、Revision int64。两个列表不含正文、来源映射、作者私有说明或素材字节。
- 字段 JSON 均使用对应 lowerCamelCase；需要首次 null 的 ExpectedHead 必须显式存在，不能把缺失字段当 null。时间固定 RFC3339 UTC。
- publication.Candidate：PublicationID（仅内部 json:"-" 的当前快照 ID）、Manifest、Diff、Snapshot content.Snapshot；Snapshot 为 CatalogueVersion、Knowledge []content.Knowledge、Units []content.Unit、Paths []content.Path、Assets []content.Asset、Bindings []content.AssetBinding。这些类型只用于服务器内部，客户端不能提交候选成员。

### 服务与仓储端口

publication.NewService(repo Repository)*Service。任务 1 的服务测试使用完整 recordingRepository 测试替身，未配置的动作在测试中立即失败；任务 3—6 的数据库测试直接调用已经实现的 Store 方法，不提前把部分实现的 Store 传给 NewService。到任务 6 完成时加入完整 Repository 编译期断言，任务 7 再组装真实 Service，不添加生产占位方法。Service 与 Repository 的下列动作同签名（receiver 不同），错误返回统一 Go sentinel 或受限 ValidationError，不把数据库错误变成客户端详情：

~~~go
Preflight(context.Context, Access, Action) (auth.User, error)
ListDrafts(context.Context, Access, ListQuery) (Page[DraftSummary], error)
CreateDraft(context.Context, Access, DraftInput) (DraftView, error)
ReadDraft(context.Context, Access, string) (DraftView, error)
SaveDraft(context.Context, Access, string, SaveDraftInput) (DraftView, error)
AdoptDraft(context.Context, Access, AdoptInput) (DraftView, error)
ValidateDraft(context.Context, Access, string, ValidateInput) (GateReport, error)
SubmitDraft(context.Context, Access, string, SubmitInput) (SubmissionView, error)
ListSubmissions(context.Context, Access, ListQuery) (Page[SubmissionSummary], error)
ReadSubmission(context.Context, Access, string) (SubmissionView, error)
ReviseSubmission(context.Context, Access, string) (DraftView, error)
DecideReview(context.Context, Access, string, ReviewInput) (SubmissionView, error)
ListPublications(context.Context, Access, ListQuery) (PublicationPage, error)
ReadPublication(context.Context, Access, string) (PublicationView, error)
PrepareRelease(context.Context, Access, PrepareInput) (PublicationView, error)
ActivateRelease(context.Context, Access, string, ActivateInput) (PublicationView, error)
PreviewWithdrawal(context.Context, Access, WithdrawalPreviewInput) (WithdrawalPreview, error)
WithdrawVersion(context.Context, Access, WithdrawalInput) (WithdrawalResult, error)
ReadDraftAsset(context.Context, Access, string, string) ([]byte, error)
ReadSubmissionAsset(context.Context, Access, string, string) ([]byte, error)
~~~

Repository 另消费已有 ReadSession(context.Context,auth.Digest,bool)(auth.SessionRecord,error)、ConsumeRates(context.Context,[]auth.RateKey)error。Service.Preflight 负责动作角色初查、限流和 mustChangePassword，不写内容；store 的 Preflight 做只读当前身份检查，所有具体仓储方法仍自行授权。HTTP 写入在读大正文前检查 CSRF，仓储在事务内再检查。Service.AcquireValidation(context.Context)(func(),error) 提供同一实例两个非排队槽，HTTP 在大正文或需要图校验的动作中持有到实际工作结束。

auth.DecodeContentProof(cookies Cookies, csrf string, write bool)(SessionProof,error) 是纯证明解码，write=false 不要求 CSRF；不创建身份或 preauth。HTTP 创建 Access 并调用 Service，禁止把 Preflight 返回角色缓存为事务授权。

## Task 1：工作流契约、数据库约束与授权基础

**Files:** 新增 db/migrations/00004_content_workflow.sql；content/workflow_model.go（提前定义 Snapshot / AssetBinding 纯数据类型）；publication/model.go、repository.go、service.go、policy.go、rate_limit.go 及 policy_test.go、service_test.go；auth/content_proof.go、content_proof_test.go；store/workflow_schema_test.go、workflow_tx.go、workflow_tx_test.go、workflow_idempotency.go、workflow_idempotency_test.go。

**Interfaces:** 定义上一节全部类型/端口。store 产出内部 workflowTx(ctx context.Context,access publication.Access,action publication.Action,relatedUserIDs []string,fn func(context.Context,*sql.Tx,auth.User,time.Time)error)error、workflowReadTx(ctx context.Context,access publication.Access,action publication.Action,fn func(context.Context,*sql.Tx,auth.User)error)error、workflowReplay(ctx context.Context,tx *sql.Tx,actorID,route,key,digest string)([]byte,bool,error)、workflowRemember(ctx context.Context,tx *sql.Tx,actorID,route,key,digest string,result []byte)error。读取、验证和撤回预览不写成功幂等记录。迁移为 workspace/revision 的送审建立唯一约束，封存状态用延迟约束触发器保证提交时已封存；未封存行仅供同一事务插入关联，不能成为已提交可见状态。

- [x] **Step 1：写失败测试。** 在真实随机数据库验证迁移和事务；以下是测试的精确断言集：

~~~json
{
  "TestWorkflowSchema": {"tables":10,"revisionMinimum":1,"assetByteMaximum":1048576,"orphanOwner":"FK rejection","secondReview":"unique rejection","snapshotMemberAfterManifest":"INSERT/UPDATE/DELETE rejected","frozenSourceAndAuthors":"unchangeable","pendingToApproved":"once","approvedToReturned":"rejected","contentCountsAndHashesAfterUpgrade":"unchanged"},
  "TestWorkflowRoleMatrix": {"learner":"FORBIDDEN","adminCreatesDraft":"FORBIDDEN","editorPublishes":"FORBIDDEN","mustChangePassword":"PASSWORD_CHANGE_REQUIRED","missingSession":"AUTHENTICATION_REQUIRED"},
  "TestWorkflowIdempotencyRechecksPermission": {"sameKeySameRequest":"one committed result","sameKeyDifferentRequest":"IDEMPOTENCY_CONFLICT","revokedActorReplay":"rejected","rollback":"zero result/audit"},
  "TestWorkflowSessionExpiresWhileWaiting": {"afterUserOrContentLockWait":"DB clock check rejects expired session","oldCSRF":"CSRF_FAILED","lockWaitAbove1s":"bounded unavailable"},
  "TestContentRateBudget": {"readUser":120,"writeUser":30,"heavyUser":10,"writeGlobal":120,"heavyGlobal":40,"next":"RATE_LIMITED"}
}
~~~

迁移使用设计中的十个表，外键到 auth_users/catalogue_versions/imported_packages/package_members；工作区 UUID 与 revision、状态枚举、大小 CHECK；送审只允许一次终态更新，冻结字段及子表与 frozenDigest 一致。作者/成员关联禁止在冻结完成后插入新行；事务内先建待封存行和关联，再封存，默认不允许留下未封存的送审。manifest 加入后禁止快照成员插入/改删；P1 未关联 manifest 的草稿不受新冻结触发器约束。content_withdrawals 区分版本与素材摘要分支，数据库唯一身份防重；content_idempotency 固定 actor/route/key 唯一且不可改删。

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/auth ./internal/publication ./internal/store -run 'Test(ContentProof|WorkflowSchema|WorkflowRoleMatrix|WorkflowIdempotencyRechecksPermission|WorkflowSessionExpiresWhileWaiting|ContentRateBudget)$' -timeout 5m -count=1
~~~
预期仅因新契约/迁移/行为不存在而失败，连接或工具链故障先处理。

- [x] **Step 3：实现契约和基础事务。** 使用既有 account/session/dbClock、管理锁和 ConsumeRates。workflowTx 自己从请求剩余截止创建事务，不能使用会固定到 3 秒的 authTx 包装；在获取任何锁之前 SET LOCAL lock_timeout='1s'，按全局锁顺序执行。幂等请求摘要在严格解码后由服务器规范化；成功结果与审计同事务。用户读取限流使用既有全局 auth_read 600/分钟预算及 content_read_user 120/分钟，写入使用本阶段固定预算；不引入无限范围限流键。首版两槽 AcquireValidation 不创建后台计算队列。
- [x] **Step 4：验证 GREEN。** 重跑 Step 2，另运行原 auth/store 账户角色与审计测试，确认新增角色消费不改变旧账户能力。
- [x] **Step 5：提交。** 暂存本任务列出的文件，提交 feat: establish content workflow contracts and transaction guards。

## Task 2：机器完整性、素材读取器与导入事务复用

**Files:** 修改 content/validate.go、assets.go、store/import.go；新增 content/asset_reader.go、workflow_validation.go、snapshot_validation.go、workflow_validation_test.go、snapshot_validation_test.go；增加 store/import_test.go 的兼容断言；新增 content/testdata/workflow-ready.json 和 workflow-ready.svg，均为原创测试素材。

**Interfaces:** content.AssetReader 为 func(context.Context,content.Asset)([]byte,error)；产出 ValidateAndSealWithAssets(ctx,catalogue.Catalogue,Package,AssetReader)(ValidatedPackage,Report)、ValidateWorkflow(ctx,catalogue.Catalogue,Package,AssetReader)(ValidatedPackage,WorkflowReport)、ValidateEditable(ctx,catalogue.Catalogue,Package,AssetReader)(WorkflowReport,error)、ValidateSnapshot(ctx,catalogue.Catalogue,Snapshot,AssetReader)(WorkflowReport,error)、ValidateSVG([]byte)error。WorkflowReport 具有三类完整 Issue 数组与总数，不依赖 publication；publication.GateReport 从它映射。store.importValidatedTx(ctx,*sql.Tx,content.ValidatedPackage)(ImportResult,error) 只做当前已加锁事务中的写入，ImportDraft 原签名保持。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestWorkflowAssetReaderParity": {"fileAndDBBytes":"same package digest/report","pathTraversalOrSymlink":"file reader rejection","DBReader":"never opens asset.path","wrongSHAOrUnsafeSVG":"CONTENT_INVALID","assetAliasBinding":"exact digest"},
  "TestWorkflowCompleteness": {"nineP1Skeletons":"not ready","emptyTheoremProof":"not ready","missingSystemOrObjectivesOrSource":"not ready","twoSameKindAngles":"not ready","blankExample":"not ready","originalURLAndDateEmpty":"allowed with author/license","externalNonHTTPSOrBadDate":"not ready"},
  "TestEditableDraftAndReportTruncation": {"missingPrerequisite":"safe save with structural issue","unsafeMarkup":"reject save","101Issues":"100 displayed,total101,ready=false"},
  "TestSnapshotGraph": {"cycleOrMissingExactPrerequisite":"reject","outOfOrderPath":"reject","relatedMissingTarget":"filter,not prerequisite","publicKnowledgeOrPathAbove10MiB":"reject"},
  "TestImportTransactionReuse": {"oldCLIRepeat":"same digest/alreadyImported","lateSubmissionFailure":"no partial imported package","crossPackageVersionConflict":"whole rollback"}
}
~~~

- [x] **Step 2：验证 RED。** Run 以下两个独立批次，预期新行为缺失失败：
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/content -run 'Test(Workflow|Editable|Snapshot)' -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run 'Test(ImportTransactionReuse|ImportIsIdempotent|VersionCannotBeOverwritten|ImportRollsBackLateFailure)' -timeout 5m -count=1
~~~

- [x] **Step 3：实现校验与复用。** 共同结构检查接收读取器，旧 ValidateAndSeal 调用文件适配器并保留原 Report 和 REVIEW_REQUIRED 语义。Editable 模式允许缺失数学引用和正文，但拒绝危险文本、错误素材字节与 Schema。Workflow 模式执行设计第 6 节最低完整性要求；素材字节或数学含义改变须创建相应知识/单元新版本；机器校验检查摘要和绑定，是否属于数学含义修改由复核核对。Snapshot 模式独立检查整图及公开 view 大小，不把一个随意 Package 拼装成可信审核证明。引用遍历采用有界迭代算法并检查 ctx，避免大图递归爆栈；按计数和字节上限预检后再读取正文或 SVG。
- [x] **Step 4：验证 GREEN 与 P1 兼容。** 重跑 Step 2，再跑完整 content/cli/import 旧测试；重新导出旧包摘要必须相同，无新增发布 head。
- [x] **Step 5：提交。** 提交 feat: separate content readiness from independent review。

## Task 3：本人草稿、认领修订与固定版本送审

**Files:** 新增 publication/draft.go、draft_test.go；store/workflow_draft.go、workflow_submission.go、workflow_draft_test.go、workflow_submission_test.go、workflow_fixture_test.go。

**Interfaces:** 实现 Create/Read/Save/Adopt/Validate/SubmitDraft、ListDrafts、ReadSubmission、ListSubmissions、ReviseSubmission，以及受作用域限制的两类 Asset 读取。产出 publication.FrozenDigest(FrozenBody)(string,error)、DraftDigest(DraftView)(string,error)。从输入到归一化字段只做确定序列化，不改变有效 Unicode 数学文本。摘要只序列化明确的内容字段：目录 version/SHA、包、素材摘要、来源、排序作者和 legacy 标记；DraftDigest 额外包含工作区 ID/revision，均排除 Gate、时间与摘要字段自身，避免循环定义。测试 helper newWorkflowFixture(t *testing.T) 复用既有 setup/newAuthFixture；Access(username string,recent bool)publication.Access、Input()publication.DraftInput、Submitted(owner string)publication.SubmissionView 在本任务定义，后续复用；初始化测试账户/角色只在随机库。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestWorkflowDraftOwnership": {"otherEditorReadOrWrite":"NOT_FOUND","adminRead":"allowed","adminOnlyWrite":"FORBIDDEN","saveExpectedRevision1":"revision2","staleSave":"DRAFT_CONFLICT,unchanged"},
  "TestWorkflowAdoptAndRevision": {"legacyDraft":"legacyUnattributed=true,no approval","revisionOfOwnApproved":"new editing workspace,inherited authors","otherOwnerRevision":"NOT_FOUND","oversizedLegacy":"CONTENT_LIMIT_EXCEEDED,no workspace"},
  "TestWorkflowSubmitAtomic": {"missingProofOrBody":"CONTENT_NOT_READY","staleDigest":"DRAFT_CONFLICT","valid":"pending,workspace submitted,fixed member hashes","lateInsertFailure":"zero partial submission/import/audit","sameKey":"same submission"},
  "TestFrozenSubmissionDoesNotReadMutableWorkspace": {"afterReturnThenSourceMapAndBodyEdit":"old frozen body/authors/source unchanged","SQLFrozenRelationInsertAfterSeal":"rejected","oldPrivateAsset":"fixed exact bytes"},
  "TestWorkflowAuthorInheritance": {"sameVersionAdoption":"inherits known authors plus adopter","clientAuthorIDs":"invalid input","newMaterialRevision":"inherits base responsibility"}
}
~~~

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/publication ./internal/store -run 'Test(WorkflowDraft|WorkflowAdopt|WorkflowSubmit|FrozenSubmission|WorkflowAuthor)' -timeout 5m -count=1
~~~

- [x] **Step 3：实现工作区与冻结。** revision 初值 1，更新必须使用条件 UPDATE，素材替换同事务。包/SVG/sourceMap 先有界校验，标准 Base64 重新编码必须等于原输入。作者集合从负责人、base_submission 和相同固定成员的既有冻结作者并集得到，排序去重后参与摘要。送审在统一锁下再次检查 revision/digest、正式版本冲突和机器完整性，调用 importValidatedTx，再保存冻结 sourceMap/作者/成员与 frozenDigest；提交前无可变工作区读取进入冻结 view。列表 scope 仅 mine/all，all 仅 admin；reviewer 可读取冻结送审但不能读取任意编辑工作区。
- [x] **Step 4：验证 GREEN。** 重跑 Step 2；重复导入、冻结失败、私有素材作用域均必须通过；确认测试创建和清理只操作随机库。
- [x] **Step 5：提交。** 提交 feat: add authoring workspaces and immutable submissions。

## Task 4：独立复核、终态竞争与角色撤销

**Files:** 新增 publication/review.go、review_test.go；store/workflow_review.go、workflow_review_test.go、workflow_concurrency_test.go；扩充 workflow_fixture_test.go 的 Approved(owner,reviewer string)SubmissionView helper。

**Interfaces:** 实现 DecideReview 和送审筛选；产出 ValidateReviewInput(ReviewInput)error，冻结 ReviewerID、FrozenDigest 和最终决定，不允许重新编辑决定。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestCopiedAuthorsCannotApprove": {"authorWithReviewer":"FORBIDDEN","copiedOrAdoptedKnownAuthor":"FORBIDDEN","independentReviewer":"approved with matching frozenDigest"},
  "TestReviewChecksAndReturn": {"oneUncheckedOrBlankIndependence":"CONTENT_NOT_READY","returnWithNote":"returned,workspace editing,revision+1","secondDifferentDecision":"REVIEW_CONFLICT","sameKey":"original decision"},
  "TestReviewRoleRevocationRacesDecision": {"revocationCommitsFirst":"rejected","decisionCommitsFirst":"record retained,old session revoked afterward"},
  "TestConcurrentReviewFinalState": {"twoReviewersAtDBBarrier":"one decision,one REVIEW_CONFLICT,one event"}
}
~~~

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/publication ./internal/store -run 'Test(CopiedAuthors|ReviewChecks|ReviewRoleRevocation|ConcurrentReview)' -timeout 5m -count=1
~~~

- [x] **Step 3：实现决定与独立性。** 在管理锁、内容锁、排序用户行及 session 之后读取冻结作者和 pending 状态；批准复核者不在作者集合，检查项和文本完整；退回与恢复 editing 同事务。竞争测试使用事务锁、通道和数据库可观察等待状态同步，不用 Sleep 或先后 HTTP 调用冒充并发。
- [x] **Step 4：验证 GREEN。** 重跑 Step 2，Task 1/3 权限、幂等和冻结回归全部通过。
- [x] **Step 5：提交。** 提交 feat: enforce independent fixed-version content review。

## Task 5：候选合并、不可变 manifest 与原子激活

**Files:** 新增 publication/release.go、manifest.go、release_test.go、manifest_test.go；store/workflow_release.go、workflow_release_test.go；扩充 workflow_concurrency_test.go；按需修改 store/publication.go 的事务读取辅助，既有 Reader 签名不变。

**Interfaces:** publication.BuildCandidate(base Candidate,batches []ReviewedBatch)(Candidate,error)、ManifestDigest(Manifest)(string,error)；ReviewedBatch 包含 SubmissionView、MemberIdentity 和固定 AssetBinding，从可信数据库加载。实现 PrepareRelease、ActivateRelease、List/ReadPublication。store.loadWorkflowCandidate(ctx,*sql.Tx,head *string)(Candidate,error) 先计数/计字节，再读取，不能调用另开事务的 withPublication 来校验激活。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestReleaseMerge": {"firstExpectedHeadNull":"prepared,public head absent","differentCatalogue":"VERSION_CONFLICT","sameMemberDifferentVersionAcrossBatches":"VERSION_CONFLICT","knowledgeReplacement":"old owned units/assets listed removed","unchangedDependentOldPrerequisiteOrPath":"CONTENT_INVALID"},
  "TestReleaseRejectsOldUnitBinding": {"sameAssetAliasNewBytesWithOldUnit":"IMMUTABLE_CONFLICT or CONTENT_INVALID","newKnowledgeAndUnitVersions":"review required before prepare"},
  "TestManifestFrozen": {"arrayInputOrderChanges":"same canonical manifestSHA","memberChange":"different SHA","SQLMemberMutationAfterPrepare":"rejected","CLIUnreviewedMembers":"REVIEW_REQUIRED"},
  "TestActivationReplayDoesNotRestoreOldHead": {"sameKeyAfterLaterPublish":"returns historical result,current head stays later","sameKeyChangedPayload":"IDEMPOTENCY_CONFLICT"},
  "TestActivationRacesRevocationAndHead": {"reviewerRevokedFirst":"REVIEW_REQUIRED","adminRevokedFirst":"rejected","twoReleasesSameBase":"one active,other PUBLICATION_STALE","failedAudit":"no head change"}
}
~~~

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/publication ./internal/store -run 'Test(Release|Manifest|Activation)' -timeout 5m -count=1
~~~

- [x] **Step 3：实现候选及激活。** Manifest 按 kind/ID/version/package/sha 固定排序，Bindings 按 unit ID/version/asset ID 排序，作者去重排序；Digest 使用带用途标识的 Go JSON 结构，不采用客户端 hash。对同 ID 新版本替换与旧归属成员清理给出完整 Diff。既有已发布依据可以继承，新增部分重新核验 reviewer 当前角色；无 manifest 的旧公开测试快照仍可读取，但新流程不能继承没有独立批准的成员。准备前验证 PublicationView 序列化不超过 4 MiB，避免创建无法读取的候选。激活统一事务中重新核验 baseHead、manifest、机器整图、当前资格、黑名单，再一次切换 head；published 历史继续保留。
- [x] **Step 4：验证 GREEN。** 重跑 Step 2，真实 PostgreSQL 竞争与旧公开 domain/knowledge/path/asset 测试通过。新增成员同一批次整体发布，公共请求不能混合新旧版本。
- [x] **Step 5：提交。** 提交 feat: prepare and atomically activate reviewed snapshots。

## Task 6：撤回目标、依赖闭包与派生快照

**Files:** 新增 publication/withdrawal.go、withdrawal_test.go；store/workflow_withdrawal.go、workflow_withdrawal_test.go；扩充 workflow_concurrency_test.go。

**Interfaces:** publication.WithdrawCandidate(base Candidate,target WithdrawalTarget)(Candidate,error) 为纯计算；实现 PreviewWithdrawal、WithdrawVersion。撤回生成独立 published 快照、事件和 head，不修改旧 snapshot/members；复用 Task 5 canonical manifest 与读取助手。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestWithdrawalClosure": {"rootKnowledge":"root,prerequisite dependents,their units/assets and containing paths unavailable","relatedOnly":"other knowledge retained,edge filtered","unit":"its knowledge paused","assetSHA":"all knowledge using exact bytes paused","path":"only path removed"},
  "TestWithdrawalVersionLedger": {"problemTarget":"permanent unique blacklist","dependentEviction":"diff only,not blacklist","unpublishedApprovedTarget":"new head prevents stale activation","noPriorHead":"empty published head"},
  "TestPublicationWithdrawalRace": {"withdrawalCommitsFirst":"old activation PUBLICATION_STALE","activationCommitsFirst":"withdraw derives from new head","sameKeyReplay":"never restores previous head"},
  "TestWithdrawalRollbackAndHistory": {"lateFailure":"old head and no new event","oldVersionsAndReview":"retained","currentPublicAssetAndDetails":"404 for impacted content"}
}
~~~

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/publication ./internal/store -run 'Test(Withdrawal|PublicationWithdrawal)' -timeout 5m -count=1
~~~

- [x] **Step 3：实现撤回与预览。** Target 查库获得可信版本/字节摘要，不接受客户端摘要替代版本；在固定上限内计算知识前置反向闭包，再移除单元/素材及涉及路线。非前置边沿用读取过滤。新快照只包含原来已发布且仍可用的成员，并保留其原批准证据；空快照仍保留目录。预览是只读业务动作，不写事件或幂等成功；真实撤回在管理员最近验证、CSRF、expectedHead 及同一发布锁下重新计算。P4/P5 扩展点写入注释和运维契约，不创建学习表。
- [x] **Step 4：验证 GREEN。** 重跑 Step 2，以及 Task 5 激活竞争和旧公开读取，确认撤回提交后新请求不再取得问题版本。
- [x] **Step 5：提交。** 提交 feat: withdraw fixed content versions without losing history。

## Task 7：Go 私有 HTTP、严格 JSON、SVG 与 OpenAPI

**Files:** 新增 httpapi/content_routes.go、content_json.go、content_error.go、content_assets.go、content_test.go、content_json_test.go、content_assets_test.go、content_integration_test.go；修改 application.go 和 cmd/server/main.go；修改 api/openapi.yaml。

**Interfaces:** 新增 ContentOptions{Service *publication.Service,PublicOrigin string,Production bool,Configured bool}；在 AuthOptions 增加 Content *ContentOptions，保留 NewApplicationHandler 旧参数形式。产出 routeContent(path,method string)(contentRoute,error)、decodeContentJSON(reader io.Reader,limit int64,dst any)error、serveContent(...options ContentOptions)、contentReady(ctx,*sql.DB)(bool,error)。Configured 只由受限表存在性检查获得；不自动执行迁移。复用 auth.DecodeContentProof、Cookie 选择与 requestID，固定接口如下：

| 相对 /api/v1/content 路径 | 方法 / 输入 | 成功 |
| --- | --- | --- |
| /drafts、/drafts/{id} | GET ListQuery / POST DraftInput；GET / PUT SaveDraftInput | 200 Page 或 DraftView；创建 201 |
| /drafts/adopt、/drafts/{id}/validate、/drafts/{id}/submit | POST AdoptInput / ValidateInput / SubmitInput | 201 DraftView / 200 GateReport / 201 SubmissionView |
| /submissions、/submissions/{id} | GET ListQuery / GET | 200 Page / SubmissionView |
| /submissions/{id}/revision、/decision | POST 精确 {} / ReviewInput | 201 DraftView / 200 SubmissionView |
| /publications、/publications/{id} | GET ListQuery / GET | 200 PublicationPage / PublicationView |
| /publications/prepare、/publications/{id}/activate | POST PrepareInput / ActivateInput | 201 PublicationView / 200 PublicationView |
| /withdrawals/preview、/withdrawals | POST WithdrawalPreviewInput / WithdrawalInput | 200 WithdrawalPreview / 201 WithdrawalResult |
| /drafts/{id}/assets/{sha256}、/submissions/{id}/assets/{sha256} | GET，无 query | 200 精确受控 SVG |

列表 query 仅 scope/status/limit/offset，重复拒绝；draft scope=mine/all、status=editing/submitted；submission scope=mine/review/all、status=pending/approved/returned；publication scope=all、status=draft/published，均可省略。review scope 需要 reviewer，并从 pending 队列排除作者本人；status 省略时默认 pending，status=approved/returned 时只列该复核者处理的记录；默认 editor 为 mine、reviewer 为 review、admin 为 all，多角色用户可主动切换合法 scope。路径其他 query、HEAD/OPTIONS、末尾斜线和编码别名均拒绝。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestContentJSONExactBoundary": {"requestLimits":[8388608,8192],"responseMaximum":4194304,"maxDepth":32,"missingExpectedHead":"INVALID_REQUEST","explicitNullFirstHead":"valid","duplicateCaseAliasOrSurrogateOrNUL":"INVALID_REQUEST","chunkedLimitPlus1":"PAYLOAD_TOO_LARGE"},
  "TestContentRoutesAndAuth": {"rootUnknownHEADOPTIONS":"404/405,no public fallthrough","wrongOriginOrDuplicateCSRF":"CSRF_FAILED","bodyNotReadBeforeRoleReject":"true","authUnconfigured":"AUTH_NOT_CONFIGURED","migrationMissing":"CONTENT_NOT_CONFIGURED"},
  "TestContentSlotsSurviveCancellation": {"twoBlockedWorkers":"occupied2","third":"429 immediately","cancelWhileWorkerStillRuns":"slot retained","workerFinishes":"slot released"},
  "TestPrivateContentAsset": {"otherWorkspaceSameSHA":"404","wrongDigest":"404","boundSVG":"exact bytes+CSP+nosniff+private,no-store","SetCookie":"absent"},
  "TestContentHTTPWorkflow": {"actualGoPG":"create,validate,submit,review,prepare,activate,withdraw","oldAuth8KiB4s":"unchanged","badUpstreamInternalError":"fixed503,no credential/body"}
}
~~~

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/httpapi -run 'Test(Content|PrivateContent)' -timeout 5m -count=1
~~~

- [x] **Step 3：实现精确边界与契约。** content JSON Walker 不复用账户 depth=8 的限制值；共享仅纯编码检查，保持账户行为。unknown/重复/case alias 在 Go 与代理一致拒绝；Input 大小按原始流计量，再按组件规范字节计量。生产无认证 origin 或迁移时保持明确故障。Content Handler 自己 8 秒读/计算总截止，SVG 不设置身份 Cookie。OpenAPI info 改为 1.3.0，新增上述路径/类型/固定英文消息，旧 public/auth/admin 组件不改；400/401/403/404/405/428/429/503 复用已有文案，新增错误消息按下面表固定。
- [x] **Step 4：验证 GREEN 与兼容。** 重跑 Step 2 和全套 httpapi/auth/config；契约生成后仅新增 content 类型，账户时限、Cookie 和 SSR 协议不改变。
- [x] **Step 5：提交。** 提交 feat: expose bounded private content workflow APIs。

| HTTP / code | 固定英文 message |
| --- | --- |
| 409 DRAFT_CONFLICT | This draft has changed. Reload it before continuing. |
| 409 REVIEW_CONFLICT | This submission already has a review decision. |
| 409 IMMUTABLE_CONFLICT / VERSION_CONFLICT | This version conflicts with saved content. |
| 409 IDEMPOTENCY_CONFLICT | This request key was already used for different input. |
| 409 PUBLICATION_STALE | Published content has changed. Prepare a new snapshot. |
| 422 CONTENT_NOT_READY | Complete the required content before submitting. |
| 422 CONTENT_INVALID | Content validation failed. |
| 422 REVIEW_REQUIRED | Independent review is required before publication. |
| 422 CONTENT_LIMIT_EXCEEDED | Split this content into smaller reviewed batches. |
| 413 PAYLOAD_TOO_LARGE | This content exceeds the request size limit. |
| 503 CONTENT_NOT_CONFIGURED | Content management is temporarily unavailable. |

## Task 8：Next.js 固定代理、响应 Schema 与客户端

**Files:** 新增 frontend/src/lib/content/types.ts、schemas.ts、client.ts、server-client.ts、svg.ts、svg.test.ts、schemas.test.ts、client.test.ts、server-client.test.ts；lib/api/content-proxy.ts、content-proxy.test.ts；app/api/v1/content/route.ts、content/[...segments]/route.ts；修改 lib/api/generated.d.ts。

**Interfaces:** ContentRoute 为 Task 7 固定方法的判别联合，不能接受自由 URL；ContentResult<T>={ok:true,data:T}|{ok:false,status:number,code:ContentErrorCode,message:string,requestId:string,retryAfter?:number}。产出 contentRouteRequest(route:ContentRoute):{path:string,method:string,kind:ContentEndpoint}|null、createContentProxy(goOrigin:string,options:{publicOrigin:string,production:boolean},fetcher?:typeof fetch)、contentRequest<T>(route:ContentRoute,input?:unknown,key?:string):Promise<ContentResult<T>>、readServerContent<T>(route:ContentRoute,cookieHeader:string):Promise<ContentResult<T>>。AssetScope={kind:"draft"|"submission",id:string} 在 types.ts 定义；readContentAsset(scope:AssetScope,sha:string):Promise<ContentResult<Uint8Array>> 为浏览器的有界素材下载。validateContentSVG(bytes:Uint8Array,expectedSHA:string):boolean 在 server-only svg.ts 定义，使用内置 node:crypto 的 SHA-256；客户端不导入该文件。控制请求使用 auth.getAuthContext，不修改其缓存策略。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestContentProxyRawRequestBoundary": {"rawDuplicateCaseAliasSurrogateNUL":"rejected before JSON reserialization","chunked8MiBPlus1":"413","missingVsNullExpectedHead":"distinct","arbitraryPathOrQueryOrRedirect":"reject"},
  "TestContentProxyResponseBoundary": {"JSON4MiBPlus1OrWrongDTOOrMIME":"fixed503","anySetCookie":"reject,no forwarding","requestIDOrRetryAfterInvalid":"fixed503","errorResponse":"fixed whitelist"},
  "TestContentSVGScopeAndBoundary": {"fixedWorkspaceAndSubmissionPath":"only selected Cookie","SVG1MiBPlus1OrSHAChangeOrUnsafeElements":"fixed503/no bytes","redirectOrSetCookie":"reject"},
  "TestContentClient": {"write":"fresh context+same-origin+one request+UUID key","timeout":"no auto retry","read":"no context mutation","reauthSuccess":"no auto activate"},
  "TestContentSSR": {"selectedSession":"Go read only","noUserOrForbidden":"distinct from503","credentialsOrCSRFInHTML":"absent"}
}
~~~

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd frontend -- npm test -- src/lib/content src/lib/api/content-proxy.test.ts
~~~

- [x] **Step 3：实现严格传输。** Proxy 读原始 request 字节，在重序列化之前完成严格 JSON 检查；每类返回使用与 Go/OpenAPI 相同的完整 Schema。总超时 10 秒，JSON 4 MiB / SVG 1 MiB、无 redirect 和 Set-Cookie。SVG 仅在绑定作用域上调用；server-only svg.ts 使用下述受限 XML 状态解析算法，白名单与 Go assets.go 保持一致，不安装解析依赖，不用正则“去脚本”代替结构检查。客户端写入保持命令输入和 UUID key 的确切副本，失败只显示稳定文案。SSR 只读取、只转发选定 session，不创建 context、CSRF 或身份 Cookie。
SVG 状态解析规则锁定如下：

1. fatal UTF-8 解码，校验 XML 1.0 字符范围；禁 NUL、DTD、处理指令和自定义实体。字符仅允许 U+0009/U+000A/U+000D、U+0020—U+D7FF、U+E000—U+FFFD、U+10000—U+10FFFF。
2. 逐字符读取开始/结束/自闭合标签及带单/双引号属性，用栈检查配对；文本和属性只解码 amp/lt/gt/apos/quot 与合法十进制/十六进制数值实体，拒绝未转义的 & 和标签外语法错误。合法注释直接跳过但校验其结束符；CDATA 按原文本处理，禁止作为标签解析，字符范围及根外非空白规则仍生效。
3. 同 Go 仅允许 svg/g/rect/line/path/circle/ellipse/polygon/polyline/text/tspan/title/desc；仅允许 Go svgAttrs 中已有属性，拒绝命名空间前缀、重复属性和不合法 xmlns。
4. 单一根 svg，根 xmlns 必须为 http://www.w3.org/2000/svg；根外只允许空白/合法注释，元素总量最多 10000。深度边界逐字对应 Go validateSVG 的 depth>64 判定，不自行用另一种计数语义。
5. fill/stroke 只接受 Go colorPattern；所有解码后属性拒绝大小写不敏感的 url(。完整解析后不能有未闭合栈或尾随根。
6. 比较实际 SHA-256 与请求摘要后才返回字节；失败返回固定故障，不输出原文。svg.test.ts 与 Go 素材测试共享实体转义 url(、重复属性、注释、65/66 层边界、10000/10001 元素和非法字符的正反例；原 SVG 必须通过。

- [x] **Step 4：验证 GREEN 与类型零漂移。**
~~~bash
node tools/verify/run.mjs --cwd frontend -- npm run api:generate
node tools/verify/run.mjs --cwd frontend -- npm run typecheck
node tools/verify/run.mjs --cwd frontend -- npm test
~~~
生成类型先纳入本任务，再重跑生成，git diff generated.d.ts 必须无新增差异；旧 public/private proxy 测试继续通过。
- [x] **Step 5：提交。** 提交 feat: add strict content proxy and typed clients。

## Task 9：英文编辑、独立复核及发布撤回页面

**Files:** 新增 features/content/draft-list.tsx、draft-editor.tsx、package-fields.tsx、unit-fields.tsx、path-fields.tsx、source-fields.tsx、content-preview.tsx、submission-list.tsx、review-panel.tsx、publication-panel.tsx、withdrawal-panel.tsx、pending-command.ts，以及 authoring.test.tsx、review.test.tsx、publication.test.tsx、pending-command.test.ts；styles/content.module.css；app/editor/page.tsx、editor/drafts/[id]/page.tsx、review/page.tsx、review/[id]/page.tsx、admin/publications/page.tsx、admin/publications/[id]/page.tsx、admin/withdrawals/page.tsx。修改 site-header.tsx、reading/safe-markdown.tsx、asset-image.tsx 及测试。

**Interfaces:** DraftEditor({initial:DraftView})、ReviewPanel({submission:SubmissionView})、PublicationPanel({initial:PublicationPage})、WithdrawalPanel()；其余字段组件接受对应 DTO 与 onChange，不自行调用服务。消费任务 8 的 AssetScope；SafeMarkdown 和 AssetImage 增加可选 assetScope，默认仍使用现有 /api/v1/assets/{sha}。私有路径只能由 UUID 和已绑定 sha 构造，不接受自由 image URL。pending-command.ts 产出 createPendingCommand(route,input):PendingCommand、retryPendingCommand(command):Promise<ContentResult<unknown>>；仅内存保存确切输入和 key，成功后清除，不进入 localStorage/URL。

- [x] **Step 1：写失败测试。**
~~~json
{
  "TestAuthoringForms": {"schemaFields":"knowledge/unit/path/source fully editable","JSONImportExport":"same DTO,no private credential","missingBody":"saved but submit disabled","saveConflict":"unsaved input retained","invalidSVG":"rejected","privatePreview":"bound private endpoint"},
  "TestIndependentReviewUI": {"author":"no approve action","fixedBody":"readonly","uncheckedChecklist":"cannot approve","return":"requires note","503":"not success"},
  "TestPublicationAndWithdrawalUI": {"diff":"add/replace/remove visible","reauth428":"explicit password dialog","reauthSuccess":"does not auto mutate","staleHead":"requires new preview","cancel":"zero mutation","emptyHead":"explicit empty state"},
  "TestPendingCommandManualRetry": {"timeout":"pending exact key/input retained","manualRetry":"same key/input,one additional request","editedInput":"new command/key","doubleClick":"single request","successfulHistoricalReplay":"refresh current head"},
  "TestContentKeyboardAndNavigation": {"role":"real eligible links","dialogClose":"restores trigger focus","forms":"labels/error focus/keyboard","noRealProgress":"unchanged"}
}
~~~

- [x] **Step 2：验证 RED。**
~~~bash
node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/content src/features/reading
~~~

- [x] **Step 3：实现页面。** SSR 读取真实身份并区分未登录、拒绝和故障；结构化表单覆盖设计字段，数组项新增/删除不改他项编号，包/成员版本显式展示并保留用户选择。JSON/SVG 文件导入先检查字节上限和编码；JSON 导入导出固定为 DraftInput envelope，导出通过 readContentAsset 读取绑定字节并标准 Base64 编码，不导出凭据；目录版本由输入明确选择并经 DB 核验，不硬编码资料文件版本。使用既有 Markdown/KaTeX 安全选项预览。sourceMap/manifest 等内部信息仅内容后台展示，不加入学习者公开页面。管理员列表和版本选择来自授权接口。发布/撤回成功再刷新 head；冲突、超时和取消不显示成功。复用既有重新验证接口，密码只在验证对话框使用，不放入内容命令或待重试副本。
- [x] **Step 4：验证 GREEN、构建和旧界面。**
~~~bash
node tools/verify/run.mjs --cwd frontend -- npm test
node tools/verify/run.mjs --cwd frontend -- npm run typecheck
node tools/verify/run.mjs --cwd frontend -- npm run build
~~~
任务内组件测试可用局部网络夹具；任务 10 浏览器必须连接真实 Go/PG。既有知识页、公式安全、认证导航和样式回归通过。
- [x] **Step 5：提交。** 提交 feat: build English content authoring and review workbenches。

## Task 10：真实联调、性能边界、独立审查与 PR 交付

**Files:** 新增 e2etest/workflow_fixture.go、workflow_fixture_test.go；修改 harness.go、harness_test.go；新增 store/workflow_capacity_test.go；修改 store/auth_rate_limit_test.go（仅测试时钟与断言）；新增 tests/e2e/content-authoring.spec.ts、content-review.spec.ts、content-release.spec.ts、content-security.spec.ts；按需扩充 fixtures.ts。修改 .github/workflows/backend.yml、frontend.yml；新增 docs/operations/content-workflow.md、2026-10-01-p3b-acceptance.md；更新 README.md、总体方案、路线图和本计划实测状态。

**Interfaces:** harness 新增 content 场景，在随机库创建 editor、reviewer、admin 和原创测试小路线，接入真正 Service/HTTP。角色授予和登录使用真实账户能力，技术测试的批准不构成独立数学验收。scene/control 继续 loopback、随机控制令牌，生产 server 禁止导入 e2etest。产出整分支验收记录。

- [x] **Step 1：写真实浏览器失败测试。** 两种视口覆盖创建/结构化编写/公式与私有图预览 → 保存 → 送审 → 作者禁止自审 → 另一账户批准/退回 → 管理员准备差异/重新验证/主动激活 → 匿名知识与路线读取 → 预览撤回/重新验证/撤回 → 匿名不可用。另测跨账户素材、无权限写入、复制作者、冲突保留输入、取消对话框、超时手动同 key 重试、故障恢复和旧公开/账户流程。不 route.fulfill 伪造成功；控制请求失败、截图与 runtime 读取沿用已修复的固定诊断，输入和 textarea 遮盖，trace/video 关闭。限流前置已通过 PR #13 提前交付，保留 store/auth_rate_limit_test.go 的 TestAuthRateLimitFixedWindowBoundary：同一固定分钟内前 10 次未知用户名登录返回 ErrInvalidCredentials，第 11 次返回 RateLimitError；数据库测试时钟推进一分钟后再次返回 ErrInvalidCredentials。原 ServicePolicies 的四项分钟预算断言必须使用固定时钟，注册的 10 分钟预算保持原值。
- [x] **Step 2：验证 RED。** 构建 harness 和现有生产前端后运行每个 content spec 独立批次；预期只有新场景或流程缺失失败，不能把错误数据库连接当 RED。
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/ ./cmd/...
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- content-authoring.spec.ts
~~~
其他 content 三个 spec 各自单独运行，使用相同 480 秒上限。

- [x] **Step 3：实现真实夹具与容量测试。** 新增 store/workflow_capacity_test.go（纳入本任务 Files）的 TestWorkflowCapacityEnvelope：合法最大计数/32 MiB JSON/10 MiB SVG、超限计数、4 MiB DTO、10 MiB public view、两个验证槽、锁等待和取消；边界数据同时满足其他限制，不把“只满足一种上限”的输入称为完全合法。记录 8 秒请求下完成情况、Go分配与进程最大驻留、DB查询计划和公开读取既有 3/5 秒预算。新增 BenchmarkWorkflowValidation、BenchmarkWorkflowSnapshot，各跑 -benchtime=3x，记录本机结果不声称生产容量。若最大合法输入不满足时限/内存约束，记录实测并修订方案的相应限制，兼容性修改经审阅后再继续，不静默放宽截止。
账户限流测试复用已交付的 fixedRateFixture(t *testing.T, at time.Time) (*authFixture, func(time.Time))：在严格验证名称的随机隔离库建立一行测试时钟及 public.clock_timestamp() SQL 函数，仅设置该随机数据库的 search_path=public,pg_catalog 并回收旧物理连接，推进函数只更新该行。新连接和仓储并发测试均使用同一固定时钟，不限制最大连接数；不重复编写已经通过 RED→GREEN 的夹具与边界测试。不修改生产 dbClock、固定窗口额度或并发测试的连接池；不通过重试、跳过测试或放宽额度掩盖跨窗问题。
容量/成本独立命令：
~~~bash
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestWorkflowCapacityEnvelope$' -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- /usr/bin/time -l env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/content -run '^$' -bench '^BenchmarkWorkflow(Validation|Snapshot)$' -benchtime=3x -benchmem -timeout 5m
~~~
/usr/bin/time -l 用于本机 macOS 资源记录，Linux CI 使用已有计时能力并注明环境，不将此 macOS 参数原样写入 CI。Benchmark 函数在任务 2 的对应 validation 测试文件中增加。

- [x] **Step 4：完成 GREEN 和整分支回归。** 各命令单独受 540 秒限制；数据库包太慢时按工作流/账户/CLI 分批，不增加单批时限。
~~~bash
node tools/verify/run.mjs -- node --test tools/verify/run.test.mjs tools/content-ingest/snapshot.test.mjs
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go vet ./...
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/auth ./internal/content ./internal/publication ./internal/config ./internal/httpapi ./internal/e2etest ./internal/testutil -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/ ./cmd/...
node tools/verify/run.mjs --cwd frontend -- npm run api:generate
node tools/verify/run.mjs --cwd frontend -- npm run typecheck
node tools/verify/run.mjs --cwd frontend -- npm test
node tools/verify/run.mjs --cwd frontend -- npm run build
node tools/verify/run.mjs --cwd frontend -- npm audit --omit=dev
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- catalogue.spec.ts reading.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- auth.spec.ts auth-security.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- content-authoring.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- content-review.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- content-release.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- content-security.spec.ts
~~~
若必须拆分 store/cli，先用 go test -list . 取得全部测试名，明确分批并检查并集覆盖所有名字；所有原测试均需纳入，不能只挑匹配 Workflow 的测试。gofmt 全目录无未格式化文件，git diff --check 通过，重复 api:generate 零差异。检查新随机库被清理，真实开发库无新增账户/审核/head，源快照保留。CI 增加 publication 包与 content 浏览器批次；每个步骤仍用限时入口。

- [x] **Step 5：独立整分支审查、修复与交付。** 使用 requesting-code-review / verification-before-completion 技能，在技术验收提交后做一次新上下文整分支审查，审查者采用该技能规定的最强可用模型，重点核对五项 Review Focus、真实竞争、触发器、权限、响应边界和旧契约。必要修复先补失败测试，修复后重跑受影响及整分支验证；未解决阻塞问题不宣称完成。记录审查结论和实测验收，勾选步骤只依据已执行证据。提交 test: verify content review publication and withdrawal flows；使用 SSH 推送功能分支，通过 Git 创建面向 master 的 PR 并附到会话，核验确切最新 head 的 CI。不得用真实库里的测试批准或自动部署替代数学/上线验收。

## 可行性、覆盖与执行交接

| 方案内容 | 实施 / 验收落点 |
| --- | --- |
| 目标、职责、旧 Schema/账户兼容 | 全局约束、任务 1/2/7/8/10 |
| 作者责任、工作区与冻结来源 | 任务 1/3/4 |
| 完整性、素材和数学复核 | 任务 2/4 |
| 合并、manifest、权限与原子 head | 任务 1/5 |
| 撤回闭包与旧候选限制 | 任务 5/6 |
| 输入资源边界、幂等和取消 | 任务 1/2/7/8/10 |
| 英文后台、预览和键盘 | 任务 8/9/10 |
| 真实人员、资料持续更新和生产启用 | 运维文档与任务 10；技术测试不会解除这些运营条件 |

已对照当前接口、迁移、CLI 导入锁、私有代理、公开 10 MiB JSON / 1 MiB SVG、SafeMarkdown 的素材路由、随机数据库及浏览器配置完成自查。参数、DTO、返回形态、错误文案、文件责任和测试名均在本计划锁定；没有新增付费服务或产品依赖的静态阻塞。实际资源测量仍是实施验收门槛，不以计划替代实测。

### 计划交付时的基线验证记录

PR #12 的初始提交 612fd9b 仅包含三份文档。前端 push/PR 和后端 push 检查通过，后端 PR 检查在 TestAuthRateLimitsServicePolicies/loginUsernameAndPreauth 失败。CI 将整包输出汇总到末尾，日志没有逐次窗口计数，因此不能逐次还原该次失败。

在临时源码副本和随机隔离 PostgreSQL 库中，主动让一次登录与后续十次登录跨越分钟边界，复现了同一断言失败：login_username 有两个窗口、总次数 11、单窗最多 10，最后返回 ErrInvalidCredentials。未修改的完整 store/cli 本地回归随后通过（store 11.506 秒、cli 1.695 秒）。这证明既有测试隐含的“十一尝试始终同窗”假设不稳定。用户确认计划后，将任务 10 的这项修复提前作为开工前置：PR #13 已通过独立整分支审查、完整本地 Go 回归及四项远程 CI，并合并到 master；生产源码和限额未改。具体 RED→GREEN、固定时钟、重连及保留多连接竞争的实现与证据见 [修复记录](../../operations/2026-10-01-auth-rate-ci-fix.md)。该问题已闭环，其余内容功能仍按十项任务验收。

执行方式沿用已确认的 Native；更新 master、新建功能分支并按 Task 1—10 顺序推进，最后一次独立整分支审查。除上述已交付的 CI 前置修复外，未提前勾选功能步骤；最终技术 PR 和真实数学发布分别验收。

### P3b 实施技术验证记录

Task 1—9 已按 RED→GREEN 逐项提交。Task 10 的真实夹具、容量与回归已通过：Go/store/cli 全部、前端 72 单元测试、46 真实浏览器用例、类型/构建/审计及格式检查；详细环境、性能、边界失败与修复见 [技术验收](../../operations/2026-10-01-p3b-acceptance.md)。最大候选准备约 3.47 秒，生产 8 秒截止及原容量上限保持。Step 5 的独立审查已完成，三项 Important 均经 RED→GREEN 修复并通过整分支回归；已通过 SSH 创建并附上 [PR #14](https://github.com/yyl1212/math_master/pull/14)，精确技术提交 5e6ee0f 的 push/PR 后端与前端四项 CI 全部通过；后续验收文档提交以 PR 最新检查为准。Step 5 依据上述实际证据勾选，工作区保留，未部署或发布真实数学内容。
