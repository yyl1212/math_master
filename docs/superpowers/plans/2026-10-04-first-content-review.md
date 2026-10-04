# P6b：首批数学内容独立复核、发布与正式验收实施计划

> **供实施者使用：** 使用 superpowers:executing-plans，在当前会话沿用用户已选择的 Native 方式按任务执行。准备段实现结束进行一次独立整分支代码审查及完整回归；正式段按真实人员和环境门槛执行，不将技术测试计作数学批准。

日期：2026-10-04。状态：用户已确认[P6b 书面方案及第十节兼容边界](../specs/2026-10-04-first-content-review-design.md)；本实施计划待审阅，产品实现尚未开始。方案分支为 codex/p6b-content-review-design，已获取的 master 为 b78108d3dd363eae237dae59f76a7be774b9de85。计划获确认后重新通过 SSH 获取最新 master，在干净隔离工作树新建 codex/p6b-content-review-preparation，记录实际基线。如果文档 PR #26 尚未合并，只带入本次获确认的文档提交，不自行合并文档 PR。

**Goal（目标）：** 先交付受保护、全量可定位的复核材料、现有工作区导入文件、登记表和离线证据校验工具；再由真实独立审阅者完成正常审核、双 head 发布及八类学习检查，取得既有 content-audit 的真实 accepted 报告。准备段完成时整体 P6b 仍为 awaiting_review。

**Architecture（架构）：** 新 Go content-review 入口从显式清单和固定来源捕获输入；复用现有内容校验、题目生成/封存及 canonical 摘要，导出私有材料。正式作者和审阅者使用既有应用、权限和工作流。离线校验负责文件、登记和最终上下文一致性；现有一致只读数据库验收负责当前审核、资格、撤回、双 head 和数量事实。

**Tech Stack（技术栈）：** Go 1.27.1 / CGO_ENABLED=0、Node 24.17.0、PostgreSQL 17.11、已有 Next.js/TypeScript、Go test、Node test、Playwright；不增加依赖。

**Spec（已批准方案）：** [P6b 设计](../specs/2026-10-04-first-content-review-design.md)；数学基线与知识 ID 沿用[P6a 已批准计划](2026-10-04-first-content-preparation.md)的固定知识版本、目标及前置图；正式门槛沿用[P6 总体设计](../specs/2026-10-04-first-content-acceptance-design.md)。

## Global Constraints（全局约束）

- Native 顺序执行准备段 Task 1—10，每任务五步，共五十步。真实操作 R1—R4 是后续条件任务，未落实实际人员、非生产环境与操作范围时保持未执行，不伪造通过或以等待资源代替准备段交付。
- 保留 30 个首批知识 ID 与完整前置路线；初始材料为 30 知识、30 单元、1 路线、9 SVG、24 模板、450 固定及 384 生成实例、30 blueprint。真实修订使用显式新版本与映射，复核新对象全部内容，旧批准不传播。正式门槛仍为 30/20/300、每节点十有效实例/五检测及恰好五题覆盖全部核心目标、一次练习曝光后五题见证。
- 映射的 574 个原创对象 = 70 个正文/路线/素材 + 24 模板 + 450 固定实例 + 30 blueprint。384 个生成实例继承准确模板来源，另列逐实例复核登记；对象登记共 958 条，另有 205 条来源登记，共 1163 个对象/来源条目，再展开各自必需检查项。不能把 834 实例误写成 834 条原始映射。
- 固定 snapshotId=eebca0e9f8cbe8ce9afbed5f1f872f382f54e42b8004fcb57d23d05b99ff879a、source-report SHA=7bd7c12118b915bc0a0ae4398b8018f1ba76e305736703a0617802677c7874ad；初始 source-map 原字节 SHA=26da1e09d9249d7642e6cd26d3ee4014304184acf53569ef24702cdda20d64b0。本计划不重新采集持续更新的 Knowledge_JSON，八项高级资料差异继续隔离。修订映射保留旧文件并按实际新 SHA 登记。
- 数学正文和界面保持英文为主；说明、操作手册及证据摘要使用中文。图形为既有原创 SVG；机器校验、AI_checked、CC-BY-SA/CC0 标签均不等于真实数学或许可复核。全部 834 个实例、5 项证明、9 图及所有陈述/条件逐项检查，不能抽样。
- 不改变 HTTP/OpenAPI/生成类型、DraftInput、AcceptanceEvidence/schemaVersion=1、原 content-audit 参数/JSON、00001—00008、题型、生成器、权限、判分、曝光/冷却、资格、永久撤回和双 head 事务规则。旧 LoadDraft 默认 v1 布局不变。只增加离线入口、私有 schemaVersion=1 文件及内部有界适配；发现必须改变既有契约时先提交精确兼容审查。
- content-review 不读取数据库配置、不连接数据库/网络、不执行资料内指令、不初始化人员、不迁移或批准发布。导入文件不含 authorIds、reviewer、approval、qualified、head、submissionId、decisionId 或 frozenDigest；真实作者和固定送审信息来自服务器。
- 输出全新目录 0700、文件 0600，不覆盖旧包。拒绝不安全路径、符号链接、设备/FIFO、读取中变动与超预算。原始 corpus、含答案材料、人员材料、账户、私有路径和自由案件正文留在受保护目录，公共仓库只归档脱敏数量/身份/摘要/结论及测试日志。
- 最终八检查全部绑定实际运行代码 G、准确路线及稳定的两 head；构建证明由负责人保存。后续文档提交可有另一个 SHA，不重标旧报告。反馈纠错在明确非生产环境由知情参与者做真实案件/标记演练；永久撤回后用新版本/必要新路线恢复，黑名单不清除。
- 单批 Go 5m、浏览器 8m、包装 9m、CI job 30m，任何单次测试小于十分钟，已有单操作 8s 和容量门槛保留。保留旧 72 Node、350 前端、22 批/162 浏览器、全部旧 Go/数据库及六项容量测试；新契约测试追加，不用新测替换旧测。
- 当前 master 的两次 push workflow 已成功。文档 head bea6f17047ecc968a1801f23eeba48a70051af37 的 PR Go verify 在既有学习容量准备查询超时，同 head push 与 master 同测试通过；[诊断](../../operations/evidence/p6b-design/ci-diagnosis.md)记录已知事实。具体慢查询原因尚未证实，不放宽四分钟准备、五分钟测试、八秒运行验证或减少 1000 条对照。后续准确实现 head 四次 workflow/六 job 必须实际成功；失败按根因流程处理，不能靠旧绿色宣布通过。

## Review Focus（审查重点）

| 易遗漏类别 | 具名测试及负责任务 |
| --- | --- |
| 修订清单仍读 v1、摘要/校验/导出读取不同字节，来源与数学版本混淆 | Task 1 TestSelectedDraftMixedVersions、TestSelectedDraftCapturedAssets、TestCapturedSourcesSameBytes；Task 2 TestReviewScopeIdentityMismatch |
| 共同出处重复 SourceLink、来源遗漏、参数实例未登记、导入偷偷携带批准 | Task 2 TestReviewScopeFullBaseline；Task 3 TestReviewImportSourceCoalescing、TestReviewImportNoWorkflowIdentity；Task 4 TestReviewRegisterRequiredChecks |
| 输入/输出路径逃逸、FIFO、符号链接、变动、大输入、取消或半成品冒充完整 | Task 1 TestSelectedDraftUnsafeInputs；Task 5 TestReviewOutputNoOverwrite、TestReviewOutputFailureCleanup、TestReviewOutputLimits |
| 手填 passed、旧 G/双 head、重复/未知检查、错误 frozen 或未复核对象成为证据 | Task 6 TestReviewFrozenBinding、TestReviewIndependencePending；Task 7 TestReviewEvidenceContext、TestReviewEvidenceFiles、TestReviewEvidenceNoReadyFile |
| 测试身份转为正式、旧工作流/数据库被改变、CI 跳过旧检查 | Task 8 TestContentReviewOfflineOnly；Task 9 TestContentReviewWorkflowRoundTrip；Task 10 新兼容及 CI 正反保护、完整旧矩阵；R1—R4 真实门槛 |

## 文件职责及架构图

| 新增/修改文件 | 职责 |
| --- | --- |
| backend/internal/contentaudit/draft_selection.go、draft_selection_test.go（新增）；draft.go、decode.go（内部复用） | 显式选择、捕获同一份输入与 SVG，来源原字节适配；提取私有共用函数，旧公开函数行为保留 |
| backend/internal/contentreview/model.go、prepare.go、imports.go、render.go、output.go、evidence.go 及对应 *_test.go（新增） | 私有契约、完整范围、现有导入转换、材料和登记、原子输出、证据校验 |
| backend/internal/contentreview/test_fixture_test.go（新增） | 小型原创/脱敏测试辅助，不能存书籍原文、真实身份或密码 |
| backend/internal/cli/content_review.go、content_review_test.go；backend/cmd/content-review/main.go（新增） | 两个离线子命令、显式参数、错误码和无副作用保障 |
| backend/internal/store/content_review_import_test.go（新增） | 原正常服务/Store 工作流的技术导入、送审、批准、导出和身份对照，随机测试库 |
| content/review/elementary-foundations.input.v1.json（新增） | 首批七个元数据文件及素材根的明确安全相对路径；不是运行时内容 |
| tools/verify/content-review.test.mjs、content-review-ci.test.mjs（新增）；.github/workflows/backend.yml（追加） | 既有契约/新入口保护、新纯层批次和原集成入口包含性；frontend 工作流不改 |
| docs/content/p6b-review-checklist.md、docs/operations/content-review.md、docs/operations/evidence/p6b/（新增）；content-acceptance.md、路线图（更新） | 全量人工检查、真实操作前置、技术证据与正式门槛交接 |
| content/packages/、questions/、assets/、source-maps/（条件修改） | 只在真实复核发现问题时追加准确版本与映射，按 R2 重新复核；不预设数学修改 |

~~~mermaid
flowchart TD
    Inputs[显式清单 + 固定快照/来源 + 草稿/SVG] --> Capture[有界捕获同一份字节]
    Capture --> Go[既有 Go 校验/生成/封存/摘要]
    Go --> Private[全量私有复核包 + 未检查登记 + 现有导入文件]
    Private --> Humans[真实独立全量数学/出处复核]
    Humans --> Workflow[正常工作区导入/固定送审/审核]
    Workflow --> Heads[知识 head 后五题库双 head 发布]
    Heads --> Learning[正常学员与最终八类实际检查]
    Workflow --> Frozen[实际冻结身份/批准和独立性材料]
    Learning --> Evidence[绑定 G/路线/两 head 的实际文件]
    Frozen --> Verify[离线 verify-evidence]
    Evidence --> Verify
    Verify --> Audit[既有一致只读 content-audit]
    Heads --> Audit
    Audit --> Formal[实际 accepted 或明确待复核原因]
~~~

## 固定接口、文件契约与预算

以下为计划中的新增接口，不表示当前已经实现；JSON 字段使用表中 camelCase，未知字段、重复键、非法 UTF-8/NUL、超范围整数均复用 question.DecodeStrictJSON 拒绝。数学版本与素材 null 规则沿用 contentaudit.ObjectIdentity，不另造摘要算法。

**输入适配（contentaudit 包）。** DraftSelection 的字段 CataloguePath/ContentPath/QuestionPaths/AssetsRoot 分别为 string/string/[]string/string；SelectedDraft 含 Input DraftInput、Files []CapturedFile、Assets map[string][]byte，CapturedFile={Path string, Bytes []byte}，Path 为根内相对路径。Files 包含七份元数据与实际 SVG 捕获字节；Assets 按素材 ID 索引同一份 SVG。新增：

```go
LoadSelectedDraft(ctx context.Context, root string, s DraftSelection) (SelectedDraft, error)
CheckSelectedDraft(ctx context.Context, selected SelectedDraft) (DraftFacts, error)
ReadFileUnder(ctx context.Context, root, relative string, limit int) ([]byte, error)
LoadSourcesFromBytes(ctx context.Context, snapshot string, reportRaw, mapRaw []byte) (SourceBundle, error)
```

私有 checkDraftWithReader(ctx, DraftInput, content.AssetReader) 提取现有 CheckDraft 的完整校验/封存流程；旧 CheckDraft 仍使用原文件读取器，新 CheckSelectedDraft 使用捕获字节。LoadSources 的校验主体复用于 LoadSourcesFromBytes；原 LoadSources 签名及失败语义保留。不在 contentreview 包直接调用另一个包的未导出函数。ReadFileUnder 验证规范根和每层安全路径，再沿用 readRegular；素材根另验证目录，不错误调用只接受普通文件的 fixedPath。

**测试辅助约定。** sha([]byte) string 只在 *_test.go 使用标准 SHA256/hex 对原字节求摘要；不得拿它代替 content.Digest 或题库 canonical 身份。countRows/countStatus 聚合所有登记分片，不能只看首页。示例中的 release 为 ReleaseContext，scope/bundle/links 为相应测试实际返回值。

**输入清单。** InputManifest={schemaVersion:1,cataloguePath,contentPath,questionPaths,assetsRoot}；初始路径为 content/catalogue/domains.json、content/packages/elementary-foundations.v1.json、五个 content/questions/elementary-foundations-{numbers,operations,fractions,decimals,ratios}.v1.json、content/assets。五题包路径/逻辑 ID 都必须不同，后续混合版本明确写入新清单。不按文件名排序推断最高版本。准备参数的 route/version 必须与内容包唯一完整路线一致，三十知识 ID 与 P6a 固定表集合一致。

**纯层入口（contentreview 包）。** PrepareInput 含 CodeSHA string、Route content.VersionRef、Manifest InputManifest、ManifestRaw []byte、Selected contentaudit.SelectedDraft、Sources contentaudit.SourceBundle、SourceReportRaw/SourceMapRaw []byte、FixtureOnly bool。BuildScope(ctx, PrepareInput) (ReviewScope,error) 生成 Objects（全量）、MappedObjects（原始映射子集）、DerivedInstances（派生子集）、Sources 及 RequiredChecks；Prepare(ctx, PrepareInput) (Bundle,error) 负责完整导入与材料。Bundle={Manifest ReviewManifest, Files []ExportFile}，ExportFile={Path string,Bytes []byte}。所有原字节与已解码值先交叉核对，再复用 CheckSelectedDraft/EvaluateDraft/ValidateAndSeal；不信任调用方给出的 ready 标记。

ReviewScope.Objects/MappedObjects 为 []contentaudit.ObjectIdentity，DerivedInstances 为准确身份/模板/[]question.ParameterValue 的索引，Sources 为 []SourceRecord（path/fileSHA256/datasetId/recordId 与已知出处/条件），RequiredChecks 为 map[string][]string（对象/来源 key 到必需检查名）。ReviewManifest/schemaVersion=1 记录 codeSHA、fixtureOnly、catalogueVersion/catalogueSHA256、route（ID/版本/SHA）、snapshotId、sourceReportSHA256、sourceMapSHA256、inputManifestSHA256、inputs[]、objects[]、derivedInstances[]、sources[]、files[]。inputs/files 均为安全相对路径、bytes、sha256。对象身份 key 为 kind/ID/版本/SHA；生成实例额外记录准确模板身份/参数（使用 question.Identity/[]question.ParameterValue），来源记录 key 为 path/文件 SHA/datasetId/recordId。manifest.files 不含 manifest 自己，避免自摘要循环；manifest 原字节 SHA 由输出摘要/后续 sidecar 保存。

**登记与材料。** ReviewRegister/schemaVersion=1 含 manifestSHA256、parts[] FileRef；每个 register/<序号>.json 分片含同版本/manifestSHA256 与 rows[]，最多 100 行且不超过 8MiB。每行 key、object 或 source（二选一）、checks[]，每检查 name/status/basis/issue/reviewerRef。status 仅 unreviewed/passed/returned，初始全部 unreviewed，basis/issue/reviewerRef 初始空。knowledge 检查 statement/conditions/prerequisites/proof/objectives；unit 检查 explanations/examples/counterexamples；path 检查 closure/order；asset 检查 mathematics/accessibility/viewports/rights；template 检查 body/domain/generation/answers/explanation/objectives；instance 检查 body/answers/explanation/objectives；blueprint 检查 exact_binding/core_semantics/five_cover/exposure_cover；source 检查 provenance/version_conditions/use/rights。无 proof 的节点登记明确不要求 proof，不能省略实际五项证明。原始未检查登记及 manifest.files 的 SHA 保持不变；实际审阅在新 review-register.final.json 与 register-final/ 分片副本完成，EvidenceInput 指向该最终文件并复算分片 SHA，按同一 manifestSHA256/完整行集合核对，不要求最终审阅状态字节等于原始未检查字节。每个 passed 须有独立核对依据和 reviewerRef，returned 须有问题位置；空字符串不能作为已复核。

输出为 review-manifest.json、review-register.json、sources.json、imports/knowledge.json 与 imports/questions/<逻辑包ID>.json、五主题 materials/<主题>/ 下的正文/模板/实例/blueprint Markdown，以及 images/<安全素材ID>.svg。实例按稳定身份排序，每页最多 100 个、字节预算不足时继续拆页，保证末尾条目也可定位。sources.json 只含已有出处/条件/定位及使用状态，未引用记录的未知出处/条件留空待实际核对，不凭记录 ID 编造引用，不复制原 corpus 正文；正文/模板/答案只来自原创应用草稿。所有材料文字转义 Markdown/HTML，来源内容仅作数据，不执行链接或代码。

**导入接口。** BuildImports(ctx, PrepareInput, ReviewScope) (Imports,error)，Imports={Knowledge publication.DraftInput,Questions []question.DraftInput}。SourceLink 合并键为 knowledge ID/版本 + snapshotId(batchSHA256) + RelativePath + FileSHA256，LegacyID 不在去重键中；同键涉及的 sourceId/recordId/legacyId/use/conditions 排序写入 Note/LegacyID，完整关系保存在私有 sources.json，不丢出处条件。knowledge 导入包含全包及 Base64 SVG；题包只含实际引用知识的 links。复用 publication.DraftAssets、content.ValidateWorkflow、question.ValidateAndSeal 检查最终导入字节预算和身份。

**证据文件。** ReleaseContext/schemaVersion=1 含 codeSHA、catalogueVersion/catalogueSHA256、route、knowledgeHead/questionHead（均 ID/SHA）、manifestSHA256、fixtureOnly、buildRecord FileRef、attestedBy、attestation FileRef。FileRef={path,sha256}，路径只在 evidence-root 内。FrozenBinding={space,submission FileRef,archive *FileRef,independenceVerified,attestation FileRef}：space=knowledge/questions；submission 按既有 publication.SubmissionView 或 question.SubmissionView 解码；题库 archive 为既有 question.Archive，使用实际发布后的正常导出实例复算 CanonicalFrozen；知识使用 publication.FrozenDigest。审批 ID、作者集合、reviewer 和 frozenDigest 从真实导出中读取，不在空模板预填。

EvidenceInput/schemaVersion=1 含 releaseContext FileRef、reviewRegister FileRef、bindings[]、learningChecks[]。每 LearningFile={name,file FileRef}；文件内容 LearningRecord/schemaVersion=1 含 name/result、完整 context（与 ReleaseContext 相同的代码/目录/路线/两 head/manifest/fixtureOnly）、executedAt、steps[]（action/expected/observed）、attachments[] FileRef、attestedBy、attestation FileRef。八个名称精确为 reading/pass/fail/prerequisites/review/practice_exposure/retake/feedback_correction，result 仅 passed/failed/not_run。实际人员/构建的真实性由负责人核验并保留材料，程序只检查一致性；最后数据库验收重新核对当前资格与审批事实。

VerifyEvidence(ctx context.Context, root string, manifest ReviewManifest, input EvidenceInput) (Verification,error)；Verification={Conclusion string,Evidence *contentaudit.AcceptanceEvidence,Files []ExportFile}。conclusion=evidence_ready/not_ready/awaiting_review。只有真实复核登记齐全、冻结关联一致、八份实际检查文件都已执行并校验后，才可能写既有 acceptance-evidence.json；存在 failed 时写明确 not_ready 证据，保留失败而不变成通过。未检查、未落实独立性或 not_run 时仅写校验报告，无 acceptance-evidence.json。fixtureOnly 保留到既有证据，不能升级为 accepted。

**CLI 与预算。** RunContentReview(ctx context.Context,args []string,stdout,stderr io.Writer) int；命令全程 MaxCommandDuration=120s，仍由既有九分钟包装器约束。prepare 显式 --root --input-manifest --snapshot --source-report --source-map --route --version --code-sha --out，可附 --fixture-only。verify-evidence 显式 --evidence-root --review-manifest --input --out；所有文件参数必须是规范绝对路径、根内相对引用安全。未知/重复 flag、额外位置参数、任何 --database-env/--publish 等均拒绝。退出码 0=prepared/evidence_ready，3=awaiting_review，2=非法契约/身份或 not_ready，1=IO/预算/取消；stderr 只输出稳定原因码，stdout 只含状态、数量、fixtureOnly 与 SHA，不输出私有材料。

沿用来源元数据 4MiB、map 256KiB、单 corpus 64MiB、SVG 1MiB及原数学/封存/请求预算；新 InputManifest 64KiB，单登记/材料文件 8MiB，最多 256 输出文件、总输出 256MiB。登记/材料超过单文件预算按稳定身份分片，索引合并时仍受总预算，不能截断条目。binding 最多 20（既有候选上限），学习附件最多 64 个、每个 8MiB、整个证据读取总计 256MiB；输出超预算失败，清理仅本次新目录。最大合法五题包/知识导入与素材用实测校验，已有 DraftInput 请求上限不扩大。

## 准备段：十任务、五十步骤

以下命令从仓库根执行；各任务 RED/GREEN 都保存真实命令、退出码、完整代码 SHA、原始日志及 SHA。测试辅助函数在所属测试文件先实现，不留下虚构的测试占位符；缺少生产接口造成编译 RED 时单独记录，之后必须补充真实错误输入 RED。任务提交只 stage 列出的文件。

### Task 1：显式输入选择和同字节验证

文件：新增 contentaudit/draft_selection.go、draft_selection_test.go；内部提取 draft.go/decode.go；新增 content/review/elementary-foundations.input.v1.json。

Consumes：已批准 P6a 草稿、固定源、清单契约。Produces：SelectedDraft、同一份 SVG/元数据及可复用来源校验，旧入口行为一致。

- [ ] **1.1 写测试。** 实现 selectedFixture(t) 返回临时根与 DraftSelection；从当前原创文件复制夹具，修订一个题包到 v2 并明确选择。TestSelectedDraftMixedVersions 断言读取 v2、其余明确 v1；TestSelectedDraftCapturedAssets 在捕获后改盘上 SVG，校验/导出仍使用已捕获原字节或发现输入不一致，不能重新悄悄读新字节。UnsafeInputs 覆盖绝对/../路径、重复题包逻辑 ID、根/中间/末端 symlink、FIFO/设备、缺文件、读取中换文件、目录误作文件及预算+1。CapturedSourcesSameBytes 验证报告/map 原字节与解码/SHA 一致。保留旧 LoadDraft 的真实 v1 回归。
- [ ] **1.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -run 'Test(SelectedDraft|CapturedSources)' -timeout 5m -count=1`。记录缺少新接口及至少一个真实非法输入失败，不能靠损坏夹具造 RED。
- [ ] **1.3 最小实现。** 实现上节四接口及私有共用函数；逐个文件捕获一次，所有验证从捕获字节读取，strict 解码、根/目录验证与普通文件读取沿用旧规则。旧 LoadDraft/LoadSources/CheckDraft 的签名、默认选择、8s 单读及错误语义保留。
- [ ] **1.4 GREEN。** 同一 RED 命令通过，再运行 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -timeout 5m -count=1`。核心断言 `if got.Input.Questions[i].Version != 2 || sha(got.Files[j].Bytes) != wantRawSHA { t.Fatal("选择或字节不一致") }`；测得全部旧测无丢失。
- [ ] **1.5 提交。** `git add backend/internal/contentaudit/draft_selection.go backend/internal/contentaudit/draft_selection_test.go backend/internal/contentaudit/draft.go backend/internal/contentaudit/decode.go content/review/elementary-foundations.input.v1.json`；`git commit -m 'feat: 增加明确版本的复核输入适配'`。

### Task 2：全量准确身份与必需登记集合

文件：contentreview/model.go、prepare.go、prepare_test.go、test_fixture_test.go。

Consumes：SelectedDraft/SourceBundle、原字节和固定首批 ID。Produces：ReviewScope、稳定身份与必需检查集合，纯层无 IO/写入。

- [ ] **2.1 写测试。** 实现 reviewFixture(t) PrepareInput：原创草稿加脱敏合成来源/报告，fixtureOnly=true；来源文件/记录 ID 与映射一致，不复制收藏原文。FullBaseline 断言 574 mapped、384 derived、958 对象、205 来源、全部五 proof 与九图；每个 derived 绑定真实准确模板，固定实例 SHA 使用既有封存身份。IdentityMismatch 覆盖错版本/SHA、重复对象/记录、漏末尾参数实例、缺 blueprint/素材、未知源、非首批三十 ID、错路线及 map 超限；修订版本清单成功且旧 SHA 不复用。
- [ ] **2.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -run '^TestReviewScope' -timeout 5m -count=1`，断言缺漏不能返回可准备范围。
- [ ] **2.3 最小实现。** 实现 BuildScope，检查 captured 输入与 Raw/Manifest 一致，复用 CheckSelectedDraft、EvaluateDraft 的真实草稿口径；固定对象按现有 canonical 类型求 SHA，生成实例用 SealedPackage.Instances 身份。来源登记覆盖所有所选记录，标记 unused，不把 unused 丢弃；30 首批 ID 常量来自已批准表，版本从明确草稿取得。
- [ ] **2.4 GREEN。** 同一命令及全 contentreview 五分钟单测；`if len(scope.Objects)!=958 || len(scope.DerivedInstances)!=384 || len(scope.Sources)!=205 { t.Fatal("全量范围遗漏") }`，重复/错身份/非法版本全部拒绝。
- [ ] **2.5 提交。** `git add backend/internal/contentreview/model.go backend/internal/contentreview/prepare.go backend/internal/contentreview/prepare_test.go backend/internal/contentreview/test_fixture_test.go`；`git commit -m 'feat: 固定全量数学复核范围与身份'`。

### Task 3：现有导入格式与共同来源合并

文件：contentreview/imports.go、imports_test.go；只补充 model.go 必需类型。

Consumes：PrepareInput 与 ReviewScope。Produces：publication.DraftInput 和五个 question.DraftInput，含安全 SVG 和来源追溯，不含审批信息。

- [ ] **3.1 写测试。** SourceCoalescing 为同一知识/批次/path/SHA 放入两个不同 record/legacyId，断言只一个 SourceLink 且两记录/条件均在登记及 Note；不同文件 SHA/知识版本不得合并。NoWorkflowIdentity 严格解码现有 DraftInput 并断言 JSON 无 authorIds/reviewer/approval/head/frozenDigest。ImportBudgets 校验实际知识 Base64+来源数组和五包逻辑/封存预算；错误 SVG、缺资产、外部来源路径、未引用知识链接、源用途遗漏、临界预算+1 均失败。
- [ ] **3.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -run '^TestReviewImport' -timeout 5m -count=1`，原始逐记录转换应在重复 SourceLink 案例失败。
- [ ] **3.3 最小实现。** 实现 BuildImports，用已捕获的 SVG 原字节编码；按准确去重键合并完整来源说明、稳定排序，仅输出实际使用知识的链接。复用 DraftAssets/ValidateWorkflow/ValidateAndSeal；对照每个实例、模板、知识/单元/素材/路线 identity，无新的数学重算器。
- [ ] **3.4 GREEN。** 同一命令通过；`if len(links)!=1 || !strings.Contains(links[0].Note,"记录A") || !strings.Contains(links[0].Note,"记录B") { t.Fatal("共同出处信息丢失") }`；全部六导入经原接口校验且无额外字段。实测字节数保存到技术证据。
- [ ] **3.5 提交。** `git add backend/internal/contentreview/imports.go backend/internal/contentreview/imports_test.go backend/internal/contentreview/model.go`；`git commit -m 'feat: 生成兼容现有工作流的复核导入文件'`。

### Task 4：完整材料、未检查登记和确定性 manifest

文件：contentreview/render.go、render_test.go；完成 prepare.go/model.go。

Consumes：范围和六个已验证导入。Produces：Bundle，各对象/参数实例均可定位，全部登记初始未检查。

- [ ] **4.1 写测试。** RequiredChecks 断言全部 1163 行分片、各类必需检查、五 proof、九图双视口/权利项、末尾生成实例/来源均出现且无 passed；MaterialsComplete 对照原英文陈述、全部题面/选项/答案/解析/参数域/目标和 blueprint。测试 Markdown/HTML 特殊字符、伪代码指令、安全图片引用、分页边界及未引用来源。相同输入两次 Prepare 的文件原字节和 SHA 相同。
- [ ] **4.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -run '^TestReview(Register|Materials|Manifest)' -timeout 5m -count=1`。
- [ ] **4.3 最小实现。** 完成 Prepare；按五主题、稳定身份及预算拆材料页/登记分片，复用 sealed 实例，不自行求答案。检查所有必需项恰好一次；导入、登记、材料、图片完成后生成 Manifest.Files。Bundle.Files 不包含 manifest，输出层最后序列化 manifest，避免循环 SHA。来源原文不复制，既有许可标签保留待核验状态。
- [ ] **4.4 GREEN。** 同一命令通过，再全 contentreview；核心断言 `if countRows(bundle)!=1163 || countStatus(bundle,"passed")!=0 { t.Fatal("范围或初始状态错误") }`；全部文件摘要可复算，非法输入拒绝且无答案/来源算法分叉。
- [ ] **4.5 提交。** stage 本任务四文件，`git commit -m 'feat: 导出全量数学复核材料与未检查登记'`。

### Task 5：有界普通文件与原子私有输出

文件：contentreview/output.go、output_test.go；补充预算常量。

Consumes：Bundle/Verification.Files 和全新规范绝对输出路径。Produces：完整受保护新目录或安全失败，无旧文件改动。

- [ ] **5.1 写测试。** NoOverwrite 验证已有目录/文件/同名 symlink 完全不变；FailureCleanup 在第 N 次写/关闭失败及 context 取消时仅清理本次新目录。Limits 验证 256 文件、8MiB 文件、256MiB 总字节各临界值及 +1、重复路径、../、绝对路径、symlink 父目录、FIFO、文件读取中变更；模式断言目录 0700/文件 0600。写入失败不得留下可识别完整 manifest。
- [ ] **5.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -run '^TestReviewOutput' -timeout 5m -count=1`。
- [ ] **5.3 最小实现。** WriteBundle(ctx context.Context,out string,bundle Bundle) error 与 WriteVerification(ctx,out,Verification) error；复用一个有界私有 writer。先校验所有安全路径/摘要/总预算，以 Mkdir 原子保留新输出，O_EXCL 创建文件/子目录，manifest 最后写；失败只删除自己新建目录。通过 test-only writer 注入失败，不改变生产功能。输入仍用 Task 1 的有界普通文件读取。
- [ ] **5.4 GREEN。** 同一命令及纯层全测；`if oldSHA!=sha(readOld()) || outputExistsAfterFailure { t.Fatal("覆盖或残留") }`。在可取消读写及最大合法材料测试中记录耗时/字节；超限不截断，无法完整输出即失败。
- [ ] **5.5 提交。** stage output.go/output_test.go/model.go，`git commit -m 'feat: 保护复核包的有界原子输出'`。

### Task 6：真实固定送审、全量复核及独立性绑定

文件：contentreview/evidence.go、evidence_test.go；补充 model.go。

Consumes：manifest、复核分片、真实 SubmissionView/Archive 文件和负责人独立性材料。Produces：经过一致性检查的冻结对象与 decision 集合，不能自行证明自然人。

- [ ] **6.1 写测试。** FrozenBinding 检查知识 FrozenDigest、题库 CanonicalFrozen、批准决策/同 submission/digest、五项/六项勾选、真实 frozen 作者集合、reviewer 不在作者集合、准确全部对象/素材/模板/实例/blueprint。覆盖导出实例乱序/缺失/重复、用 raw SHA 冒充 frozen、错 catalogue、修改 body/author/reviewer、来源链接不同、漏复核行/必需项及重复 decision。IndependencePending 验证未签署、未核验自然人独立性、unreviewed/returned、空 basis/reviewerRef 不能就绪；所有夹具仅为技术身份。
- [ ] **6.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -run '^TestReview(FrozenBinding|Independence|RegisterEvidence)' -timeout 5m -count=1`。
- [ ] **6.3 最小实现。** 加入 verifyReviewBindings(ctx,root,manifest,register,bindings) 的私有实现。严格读取既有 SubmissionView 和发布 Archive；题库按 frozen.InstanceIdentities 原顺序重组 archive 实例再调用 CanonicalFrozen，不能因 SQL 导出顺序不同造误差。核对 SourceResponsibility 与 frozen 作者集合、全部对象 SHA；review 行关联对应真实 reviewer，来源登记保留负责人核验依据。机器检查签署材料/摘要存在与承诺一致，不把承诺升级为自然人身份证明。知识/题库当前资格最后仍由 content-audit 的数据库事实核验。
- [ ] **6.4 GREEN。** 同一命令及纯层全测；`if got.Conclusion=="evidence_ready" || got.Evidence!=nil { t.Fatal("未复核被当作正式证据") }` 适用于所有未落实案例；完整技术夹具通过文件关联仍保留 fixtureOnly=true。
- [ ] **6.5 提交。** stage evidence.go/evidence_test.go/model.go，`git commit -m 'feat: 校验真实送审与复核登记的准确关联'`。

### Task 7：最终八类实际文件、同上下文及兼容证据

文件：继续 evidence.go/evidence_test.go；必要时仅按同职责拆 evidence_files.go/evidence_files_test.go。

Consumes：Task 6 结果、ReleaseContext、EvidenceInput、八份 LearningRecord 与真实附件。Produces：Verification 和条件生成的原 AcceptanceEvidence。

- [ ] **7.1 写测试。** Context 覆盖同/异知识 head、题库 head、旧 G、旧 route/version/SHA、catalogue/manifest 或 fixture 标记不一致。Files 覆盖改动/不存在附件、错误原字节 SHA、重复键、重复/未知检查、缺步骤/观察/签署、非法 result、路径逃逸、64 附件/8MiB/256MiB 边界。NoReadyFile 验证缺检查/not_run/未复核/未独立时没有 acceptance-evidence；真实 fail 场景的正确拒绝观察可 result=passed，已执行但错误行为则 failed，不能自动推断或改标签。
- [ ] **7.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -run '^TestReviewEvidence' -timeout 5m -count=1`。
- [ ] **7.3 最小实现。** 完成 VerifyEvidence：复算实际文件与附件 SHA、精确八名称集合、共同最终上下文、冻结/登记完整性；缺项 awaiting_review，错误身份拒绝，已执行 failed 保留 not_ready。完整真实文件才构建原 schemaVersion=1 Evidence，不增加外部字段；fixtureOnly 强标记贯穿。写校验 JSON/中文 Markdown 与 SHA 清单，正文和人员材料不进入公共报告。
- [ ] **7.4 GREEN。** 同一命令及 contentreview 全测；生成字节必须经 `contentaudit.DecodeEvidence` 成功，`if evidence.CodeSHA!=release.CodeSHA || evidence.FixtureOnly!=release.FixtureOnly { t.Fatal("绑定丢失") }`。再次读取证据文件不产生任何数据库/网络调用。
- [ ] **7.5 提交。** stage 本任务 evidence 文件及 model.go，`git commit -m 'feat: 校验八类学习文件并生成兼容验收证据'`。

### Task 8：两个有限离线 CLI 与操作说明

文件：cli/content_review.go、content_review_test.go、cmd/content-review/main.go；docs/content/p6b-review-checklist.md、docs/operations/content-review.md。

Consumes：前七任务纯层、明确参数/预算/新输出契约。Produces：可执行 prepare/verify-evidence、中文手册及所有退出码。

- [ ] **8.1 写测试。** TestContentReviewPrepare 成功调用完整小夹具及实际明确混合版本；VerifyPending/Ready/Failed 检查 3/0/2 和证据文件存在条件。OfflineOnly 设置不可用 DATABASE_URL/TEST_DATABASE_URL，两个命令仍按文件完成，日志无私有内容。参数负测含未知/重复 flag、缺参数、非法 G/路线/版本、发布/DB 参数、额外位置项、不同规范根及 FIFO；IO/预算/取消返回 1，无半成品。main 只委派 RunContentReview。
- [ ] **8.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/cli -run '^TestContentReview' -timeout 5m -count=1`。
- [ ] **8.3 最小实现。** 实现 RunContentReview 与 main，普通文件原字节先捕获再严格解码，120s 全程 context、8s 单读/原验证约束不变；不引用 store/sql/http client 或读取连接配置。手册给两个真实参数范例、冻结摘要产生时点、分片/全部检查、许可待核验和正式 R1—R4 前置，不包含凭据或假角色初始化脚本。
- [ ] **8.4 GREEN。** 同一命令及 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build ./cmd/content-review`；退出码、文件模式、无批准字段及 stderr 稳定原因码逐项确认。正常学习/审核 API 不变。
- [ ] **8.5 提交。** stage 本任务六文件，`git commit -m 'feat: 增加离线内容复核命令与中文手册'`。

### Task 9：既有正常工作流的真实数据库技术对照

文件：store/content_review_import_test.go；必要辅助只放该测试文件，不改生产 Store/HTTP。

Consumes：生成的六个 DraftInput、原工作流/随机隔离库。Produces：导入/Export/固定送审/批准/发布身份一致的技术证据，正式数量为零。

- [ ] **9.1 写测试。** TestContentReviewWorkflowRoundTrip 使用 testutil 独立 math_master_test_*，普通真实会话/CSRF/幂等键、不同测试作者/审核者和管理员，经既有 service/Store 导入知识→Export 对照→送审/批准→知识发布→五题包固定引用/送审/批准→双 head 发布→正常 Archive 对照。全部对象、SVG、来源数组与 Prepare 身份相同；服务端记录作者，题库最终 frozen 绑定真实知识引用。覆盖同作者拒绝、错 revision/expectedHead、改导入/素材 SHA 和 missing final evidence；技术报告必须 fixtureOnly=true、正式计数0、结论非 accepted。
- [ ] **9.2 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestContentReview' -timeout 5m -count=1`；数据库凭据仅用既有私有 TEST_DATABASE_URL，不写命令或日志。失败必须定位真实导入/身份缺口，不能暂改校验器凑通过。
- [ ] **9.3 最小实现。** 实现测试辅助与证据对照；发现新工具转换缺陷仅修对应 contentreview 文件，生产规则不变。用固定 P6a 私有快照实际执行一次 prepare，实测 205 来源/64 映射/958 对象和六导入请求字节；无人员环境时 verify-evidence 产生 awaiting_review/不产生正式证据。只归档安全 SHA/数量/fixture 状态。
- [ ] **9.4 GREEN。** 同一命令，再运行旧集成入口 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -skip "^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit)" -timeout 5m -count=1`，确认新 TestContentReview 被原入口实际执行、旧组未被排除。实际完整首批与最大小包预算均成立。
- [ ] **9.5 提交。** stage 新测试、必要的准确工具修复及 docs/operations/evidence/p6b/preparation.json，`git commit -m 'test: 对照复核导入与既有真实工作流'`。

### Task 10：兼容保护、完整矩阵、一次整分支审查与交付

文件：tools/verify/content-review.test.mjs、content-review-ci.test.mjs、.github/workflows/backend.yml；docs/operations/evidence/p6b/、content-acceptance.md、路线图。

Consumes：九任务实现及旧 P6a 真实矩阵。Produces：准备段可审查 MR、全部新旧回归与真实远端结果，不宣布正式 P6b 完成。

- [ ] **10.1 写保护测试。** 新 Node 测验证旧 API/类型/迁移/生成器/判分文件的基线摘要、原 CLI/schema 与双 head 参数；CI 正反夹具删除/改名旧批次、取消六容量、加跳过、去 CGO 或放宽截止必须失败。新纯 contentreview 必须有明确批次，原 store/cli 宽入口必须包含 TestContentReview；不得扩充 skip 正则漏掉旧测。
- [ ] **10.2 RED。** `node tools/verify/run.mjs -- node --test tools/verify/content-review.test.mjs tools/verify/content-review-ci.test.mjs`；在内存工作流副本中缺新批次/删除旧批次各有真实 RED，仓库实际旧工作流不删改。
- [ ] **10.3 最小配置与完整验证。** backend 工作流只追加 contentreview 纯层五分钟批次、在原 Node 命令尾追加两个新文件；新 CLI/Store 测按原宽入口执行，前端批次/依赖不变。按下节矩阵完成全部旧测与新测，收集原日志/exit/SHA/测试数，完成手册和交付状态。若再次发生现有容量慢查询，先用日志/查询计划/输入实测定位，再提交符合既有门槛的精确修复审查；不能将重复碰到一次绿色称作修复。
- [ ] **10.4 GREEN 与整分支审查。** 新保护和完整矩阵全部通过；只在实现完成后按 using-superpowers 的 Native 流程调用一次最强模型独立整分支 reviewer，检查本计划五类重点与全部 diff，必要问题逐项裁决/修复并做相关回归，最终提交后核验完整 head 四 workflow/六 job。已有会话偏好保留；本计划编写不启动 reviewer。只有真实全部成功才标记准备段完成；无人员/正式验收时整体仍 awaiting_review。
- [ ] **10.5 提交/MR。** stage 本任务已列文件，`git commit -m 'test: 完成离线复核工具的兼容验证与交接'`；先将具体准备输出、测试及尚未执行的正式门槛写入该临时说明文件，再通过 SSH push codex/p6b-content-review-preparation，`gh pr create --base master --head codex/p6b-content-review-preparation --draft --title 'P6b：全量内容复核准备与证据校验' --body-file /private/tmp/math-master-p6b-preparation-pr-body.md`。创建后调用 attach_artifact 附属 PR；准确最终 head 的远端 CI 状态写入说明。合并按用户后续指令，不由本计划授权。

## 完整回归矩阵与证据规则

准备段合并前新鲜执行下表，原命令取自当前 .github/workflows/{backend,frontend}.yml；另对照 P6a 归档矩阵确认用例与入口没有消失。每条独立运行，不合成超过十分钟的单次测试；数据库仅既有私有 TEST_DATABASE_URL 配置与 testutil 的随机隔离库。旧计数为最低基线，不预写新测数量。

| 验证组 | 精确范围/命令 |
| --- | --- |
| 格式/静态/构建 | 原 gofmt 空输出及 git diff --check；九分钟 wrapper 下 `go vet ./...`、`go build ./cmd/...`，环境 CGO_ENABLED=0/GOTOOLCHAIN=go1.27.1 |
| Node 72 + 新保护 | 原 backend 工作流 Node 命令的全部十一文件，加 tools/verify/content-review.test.mjs、content-review-ci.test.mjs；run.mjs 包装、node --test，不修改原测试 |
| 原 Go 非数据库内容/HTTP | 原工作流“内容与 HTTP 校验”全部十四包；五分钟/不缓存。新增纯层 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1`；原 contentaudit 全组另保留 |
| 原数据库/CLI分组 | 原宽入口 skip 不变；store 学习测评组、反馈组、store/cli 纠错通知组、ContentAudit 组分别沿用原 run/skip；新 TestContentReview 由宽入口实际运行 |
| 六项容量，逐个五分钟 | TestFeedbackCapacity、TestLearningCapacitySourceVolume、TestLearningCapacityMaxPool、TestCorrectionCapacityImpact、TestCorrectionCapacityNotifications、TestContentAuditCapacity；准确原 `-run '^<完整测试名>$' -timeout 5m -count=1 -v`，所有合法数据量、1000 审批对照、8s 运行限制及原准备截止保留 |
| 前端与生成 | 原 wrapper 下 npm run api:generate、npm run typecheck、npm test（原350项）、npm run build、npm audit --omit=dev；`git diff --exit-code -- frontend/src/lib/api/generated.d.ts`，生成类型不得产生变化 |
| 浏览器 22 批/162 项 | 完整逐条执行[原22批命令](../../operations/evidence/p6a/matrix-browser.json)，每批8m/包装9m，正常启动独立 harness。原知识/题库/学习/反馈/纠错/安全及两组 content-acceptance 全保留；本阶段无新页面，不增加复核平台场景 |
| 新工具实际烟测 | 固定 P6a 私有快照 prepare、六个实际导入字节/身份、未复核 verify-evidence（exit3、无 acceptance 文件）、最大小包/分页/取消；公开证据只记 SHA/数量/原因，不存答案或来源正文 |
| 整分支与远端 | 一次 Native 独立整分支审查、问题裁决及必要回归；SSH push 后核验准确完整 head 的 Go/frontend 各 push+pull_request，四 run/六 job 实际 success |

证据记录实际运行代码 G、开始/结束时间、命令/退出码、日志 SHA、fixtureOnly、实际测试数；不追改旧 P6a。新实现新增测试数量按实际输出汇总，0 skipped、未执行不记 passed。历史失败与诊断原样保留；修复要有真实失败到通过的证据及相同门槛，不能把偶发成功称为根因修复。文档归档 SHA 与 G 不同则明确关系，不给原报告改 G。

## 正式段：条件任务 R1—R4

本计划确认授权准备段技术实现；正式动作另由实际负责人落实人员、非生产数据库用途、正常服务、备份和明确操作范围。当前这些资源未落实，四任务全部待执行。资料/计划批准不代表代理可代真实审阅者批准，不能使用技术 harness 或测试人员/库作为正式证据。

### R1：环境、身份和真实复核资源

文件：私有环境登记/身份核验/备份记录；公共 docs/operations/evidence/p6b/formal-status.json 仅脱敏状态。

- [ ] **R1.1** 登记实际负责人、正常服务入口、明确非生产库及私有连接环境变量；核实用途，不默认生产/旧 test 库或更名夹具。
- [ ] **R1.2** 记录现有 catalogue、00001—00008、构建 G 和备份；未初始化时先落实实际初始化操作范围，再按现有手册执行，不因本计划自动迁移。
- [ ] **R1.3** 真实作者、数学 reviewer、管理员各自本人使用正常账号；按现有流程核验能力/独立自然人和授予角色，保留私有依据，不代人填写批准。
- [ ] **R1.4** 安排全量 834 实例、30 正文/单元、5 proof、9 图及 205 来源；明确出处/许可核验责任、修订处理与非生产纠错参与者。
- [ ] **R1.5** 实际负责人确认资源与范围齐全后登记 R1 完成；缺项只写 awaiting_review，不进入发布操作。

### R2：全量数学/出处复核、修订与真实送审

文件：私有完整登记/问题/六导入/SubmissionView；发现问题时追加准确 content 版本与 source-map，不修改旧证据。

- [ ] **R2.1** 用准备工具固定明确当前清单；按原材料逐项复算题意/选项/唯一答案/解析、全部参数域组合、条件/证明/反例、目标/前置、图示与版权依据。
- [ ] **R2.2** 登记完整 958 对象+205 来源及所有必需检查；同出处许可证据可共享，各数学对象用途/条件结论不得共享成批量通过。
- [ ] **R2.3** returned 项停止送审；真实修订追加版本/相关单元/素材/题库/blueprint/路线和映射，重新 prepare/全量相关复核，旧批准不复用；超出已确认契约时先兼容审查。
- [ ] **R2.4** 正常知识工作区导入、Export 逐身份对照、固定送审，由真实独立 reviewer 完成五项批准；服务器产生作者集合/revision/submissionId/frozenDigest，保存真实导出与本人独立性核验。
- [ ] **R2.5** 确认所有 blocker 处理及知识真实批准，保存数学/许可结论；该状态不是题库批准或正式 accepted。

### R3：正常双 head 发布与知情非生产纠错

文件：私有真实候选/审核/发布/纠错记录；脱敏 head/路线/代码摘要，不提交个人案件正文。

- [ ] **R3.1** 管理员核验知识候选差异、expectedHead/manifest/当前资格，五分钟重认证后激活知识 head；冲突重新准备候选。
- [ ] **R3.2** 五题包固定已发布知识引用、逐包正常送审，真实 reviewer 六项全部核验批准；保存当前实际冻结和正常 Archive 文件。
- [ ] **R3.3** 管理员按当前双 head 核验候选并激活；准确对象闭包/题库不足保持建设中，历史进度不迁移、规则不减弱。
- [ ] **R3.4** 在 R1 明确的非生产库实施真实案件或知情标记演练，检查反馈/独立处理/准确证据/限制/影响/通知/重算或重测；不虚称数学错误或影响生产。
- [ ] **R3.5** 永久撤回后使用新对象版本/必要新路线和重新批准恢复，不清黑名单；发布及纠错稳定后固定最终 catalogue/route/G/两 head，保存前后因果历史。

### R4：最终八检查、文件核验与一致只读正式报告

文件：私有八 LearningRecord/附件/负责人签署；公共脱敏正式报告及交付说明。

- [ ] **R4.1** 用正常学员账号在最终上下文逐项执行 reading/pass/fail/prerequisites/review/practice_exposure/retake，双视口检查整条路线、九图、英文正文和回顾入口，记录实际步骤/预期/观察。
- [ ] **R4.2** feedback_correction 在最终上下文实际读取案件完成、当前资格/正确重测和通知；保存 R3 因果记录，不能只把旧 passed 改 head。任一代码/head/路线变化重新执行受影响检查。
- [ ] **R4.3** 负责人核验真实身份独立性/构建 G/证据真实性，填写 ReleaseContext、分片登记和八类文件/附件；run content-review verify-evidence，实际通过才生成与原 CLI 兼容的 acceptance-evidence。
- [ ] **R4.4** 用既有 content-audit 显式 published、database-env、最终 route/version/G、固定来源/映射/evidence、新输出，执行单个 RepeatableRead/ReadOnly 事务。报告必须实际 accepted、fixtureOnly=false、30/20/300 和逐节点/完整批准/八检查均满足；否则保存真实原因，不能手改结论。
- [ ] **R4.5** 归档实际运行 G 与文档提交的关系及脱敏正式结果，真实负责人确认 P6b 完成；随后才能设计 P7 生产上线、备份/恢复/域名/HTTPS，当前计划不执行部署。

## 计划可行性自审及下一门槛

本计划在已合并代码上审阅接口/预算与流程。已补齐：跨包未导出读取器的复用方式、SVG/来源原字节的一次捕获、共同出处去重、登记分片与全量对象计数、题库 Archive 按冻结身份顺序复算、缺真实信息时不生成验收文件、正常工作流技术测试归入原 CI 宽入口。所需功能无需修改运行时 HTTP/数据库/数学规则；实际导入预算和最大合法输出由 Task 3/5/9 实测验证。未发现必须扩充契约才能开始准备段的阻塞设计问题。

外部执行门槛仍明确：真实人员/许可结论/环境/操作范围未落实，现有 CI 曾有一次准备查询超时且具体原因未证实；这些都不能以设计自审消除。远端准确实现 head 通过是准备段交付门槛，真实 R1—R4/accepted 是整体 P6b 门槛。

请审阅本计划是否准确体现已确认的方案。确认后沿用 Native 执行 Task 1—10，不重复选择执行方式；R1—R4 按实际资源和明确范围推进。计划确认不合并 PR #26、不替代真实数学批准，也不授权生产部署。
