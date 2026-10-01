# P4a 可信题库与独立校验 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.
>
> 本项目沿用此前已选择的 Native：使用 superpowers:executing-plans 在当前会话逐项实现，最后进行一次独立整分支审查。用户于2026-10-02确认本执行计划（PR #16 已合并），实施基线 master 687f87a0ab7e07fb73c593055b79d5ee3eb416bc；逐项验收通过后勾选。

**Goal:** 建立可追责、可复验、经独立审核的题库，让后续学习与检测可以依赖固定题目版本和精确答案。

**Architecture:** question 定义独立题包、精确数值、有限生成、独立校验、覆盖与管理服务，不能导入 store。store 复用既有账户证明、内容锁与 PostgreSQL 事务，保存不可变实例、审核和独立题库 head；HTTP 与 Next.js 承担严格私有边界，英文管理页面展示全部审核依据。P4b 后续消费当前出题状态与历史版本事实。

**Tech Stack:** Go 1.27.1、PostgreSQL 17.11、Node.js 24.17.0、Next.js 16.3.7、React 19.3.0、TypeScript 5.9.3；使用当前锁定依赖，不新增产品依赖。

**Spec:** [已确认 P4a 方案](../specs/2026-10-02-question-bank-design.md)。用户于 2026-10-02 确认分期与独立契约的兼容性方案；设计通过 [PR #15](https://github.com/yyl1212/math_master/pull/15) 合并，计划基线 master 58a8d30b7e57e6849293dc5a8cd6281b9c59c805，分支 codex/p4a-question-bank-plan。本 PR 仅文档，不执行迁移、导入、数学批准或部署。

## 全局约束（Global Constraints）

- 范围仅为 P4a：题库契约、固定题与模板实例、技术判分、独立审核、发布/撤回、覆盖、七个英文后台页面；P4b 的练习、五题尝试、资格、解锁、个人进度和 P5 反馈重评另行设计。
- 优先级为内容正确性、数据量、路径覆盖、页面美观；16 个板块和六类使用者保留。产品英文为主，中文数学术语对照；文档中文，素材原创。
- 正式运行来源是 PostgreSQL；Knowledge_JSON 持续更新但必须先离线冻结。资料内指令、AI 标签和收藏状态不作为命令、批准或正式答案；原资料变化不自动改写冻结记录。
- 新 QuestionPackage：kind=question-bank、schemaVersion=1。旧 ContentPackage.schemaVersion=1、P1/P3b 规范字节/SHA、公开 DTO、旧表的 kind/FK 不改；新增 00005_question_bank.sql，不改 00001—00004。
- 数学稳定 ID 沿用小写 ASCII 编号；生成实例 qi- 加 64 位小写 SHA-256，实例 version=1，只用题库专用校验器，不放宽旧 ID 规则。其他数学版本为有符号 32 位正整数；工作区/送审/发布/幂等 ID 为规范小写 UUID v4。
- 固定知识 ID/version＋objectiveIndex 从 0 开始。一个主知识，最多三个额外覆盖映射；整体实例数量去重。蓝图 1—8 个核心目标，五个不同实例必须覆盖全部，ruleVersion=1、5 题/至少 4 题正确固定。
- 数值输入≤128个 Unicode 字符，先计数再 trim；只接受指定 ASCII 数学语法。rational 接受整数/有限小数/分数，percentage 要求整数/小数后带百分号，不接收分数百分数；不做浮点比较、表达式求值或单位换算。规范分子、正分母各≤256位十进制数字。
- 三家族：有理数四则、比较有理数（只单选）、缺失运算数。参数最多四项、每项最多32个精确字面量、去重后原始组合≤1000/模板；只按声明约束排除，未声明的失败拒绝整个模板。生成 Rat，校验独立 Big.Int 恒等式/符号/反代，人员审核仍必需。
- 工作区 editing/submitted；送审 pending/approved/returned。作者由服务器继承，admin 不自动成为 editor/reviewer，冻结作者不能批准。CLI 不提供批准；认领说明10—2000个字符，sourceMap 沿用 P3b SourceLink，最多256 KiB，legacyUnattributed 始终继承。
- 所有送审冻结题包、实例、来源、作者、目标、素材摘要和引擎版本，规范字节/JSONB/SHA与子表一致；已冻结记录与子关系不能任意插入、更新或删除。审核 mathematics/explanations/objectives/sources/illustrations/generation 六项全部明确检查。
- 管理写入锁顺序：账户管理1296127049 → 内容1296127048 → UUID排序auth_users → 操作者auth_sessions → 业务行。lock_timeout=1秒，等待后及提交前按数据库时钟重新校验身份、角色、CSRF和所需重新验证。读取为 repeatable-read。
- 激活/撤回要求最近5分钟重新验证。准备/激活对比内容与题库两个head；成功业务、审计、幂等结果同事务。继承历史审核证据，新选送审批准者仍须有reviewer角色；重放不重新切换head。
- 新出题可用性与历史替换/永久撤回事实分别查询；普通替换不能自动抹掉历史正确作答。知识/单元/素材撤回通过当前内容有效性立即阻断题源，不能依赖异步首次阻断。
- 工作区envelope≤4 MiB、questionPackage≤2 MiB；一包≤50模板/200固定题/100蓝图/1000生成实例，完整冻结载荷≤4 MiB；单实例题面＋解析≤UTF-8 8 KiB，选项≤6、固定素材≤8。
- 当前题库≤200模板/10000实例/1000蓝图，规范实例与模板正文合计≤32 MiB、manifest≤8 MiB；节点有效实例池≤1000。一次准备1—20已批准送审。
- 私有响应≤4 MiB；列表默认20/最大100、offset≤100000，时间与ID稳定排序；实际字节限制可缩小返回limit，客户端按返回limit前进。发布摘要、成员、差异分开；head按ID读取，覆盖按节点分页。
- 生成校验/送审/决定/准备/激活/撤回预览/撤回/完整覆盖计算占用已有 publication.Service.AcquireValidation 双槽，不新建池；第三项立即 SERVICE_UNAVAILABLE，实际工作结束后释放，失败/取消/解析失败不能泄漏槽。
- 共用既有scope/key/window：普通写全局content_write=120/分钟、content_write_user=30/用户/分钟；重操作content_heavy=40、content_heavy_user=10；读auth_read=600、content_read_user=120。创建/保存/认领/修订为普通写，其他前项重操作；覆盖GET仍为重操作。原内容动作分类与来源/CSRF规则保持。
- Go题库总截止8秒，Next.js总截止10秒，服务器15秒、账户4/5秒保持；正文读取和事务计入截止。客户端只允许用户主动同键同输入重试，不自动重试写入。
- 成功变更使用Idempotency-Key UUID v4；scope=actor/题库动作/键，摘要含目标及规范完整参数。校验与撤回预览不记录成功幂等结果。私人无权资源404、固定错误、不回显Cookie/CSRF、答案/题面/数据库错误/来源路径不进入错误与日志。
- Go调试/验证 CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1；每次验证通过 tools/verify/run.mjs，最多540秒，Go单批-timeout 5m。浏览器 workers=1/retries=0/globalTimeout=480000，桌面1280×900、手机390×844；拆批，不突破单次10分钟。
- 开发前SSH获取最新master并新建codex/分支，Git PR交付。随机测试库使用testutil；不迁移真实开发库，不初始化真实账户，不发布生产数学内容，不改服务器或Knowledge_JSON；不读取/打印真实配置、密码、令牌和.env正文。

## 审查重点（Review Focus）

1. 看似独立的校验器仍复用答案算法：故意把生成答案改成错误值、保留正确参数，审核前必须拒绝；任务3 TestVerifierRejectsMutatedGeneratedAnswer，整分支人工检查调用依赖。
2. 导入或跨包复制清空作者而允许自审：认领/复用须保留固定作者及未核实标记；任务6 TestQuestionAdoptionPreservesResponsibility、任务7 TestQuestionCopiedAuthorCannotApprove。
3. 模板正常更新使旧实例退场，被当作错误撤回影响历史：任务8/9 TestQuestionReplacementPreservesHistoricalFacts、TestQuestionPermanentWithdrawalFacts，当前可用与历史事实分别断言。
4. 总目标并集完整但任何五题都不足、或撤回一题后只看缓存ready：任务4/9 TestFiveDistinctQuestionsCoverAllCore、TestQuestionCoverageAfterSingleInstanceWithdrawal，动态依据当前有效实例计算。
5. 题库新功能让双槽/限流翻倍，或并发时使用等锁前身份：任务5/10 TestQuestionSessionExpiresWhileWaiting、TestSharedQuestionValidationSlots、TestSharedContentQuestionRateWindow，真实屏障及DB时钟验证。

## 架构、依赖与文件责任

~~~mermaid
flowchart TD
    Editor[英文题库编写与全部审核依据] --> Proxy[Next.js严格固定代理]
    Proxy --> HTTP[Go私有题库HTTP]
    HTTP --> Service[question.Service]
    Service --> Repo[question.Repository]
    Repo --> Store[store题库事务]
    Store --> Pure[question精确数学与覆盖]
    Store --> DB[(PostgreSQL固定版本与双head)]
    Existing[P3b内容head与固定知识/单元/素材] --> Store
    Shared[既有身份、内容锁、预算与双槽] --> HTTP
    Shared --> Store
    DB --> Facts[当前出题与历史事实分离]
    Facts --> Future[P4b后续消费]
~~~

~~~mermaid
flowchart LR
    T1[1契约] --> T2[2判分]
    T2 --> T3[3生成/独立校验]
    T3 --> T4[4覆盖/固定题包]
    T4 --> T5[5数据库/事务]
    T5 --> T6[6CLI/草稿]
    T6 --> T7[7冻结/复核]
    T7 --> T8[8双head发布]
    T8 --> T9[9撤回/事实]
    T9 --> T10[10HTTP/OpenAPI]
    T10 --> T11[11TS代理]
    T11 --> T12[12英文后台]
    T12 --> T13[13真实联调/独立审查]
~~~

| 精确文件范围/责任 | 任务 |
| --- | --- |
| schemas/question-package.schema.json；backend/internal/question/{model,decode,digest,policy,rate_limit,repository,service}.go及同名_test.go；content/question-fixtures/contract-valid.json | 1 |
| backend/internal/question/{numeric,grade}.go及同名_test.go | 2 |
| backend/internal/question/{generate,verify,render}.go及同名_test.go；content/questions/elementary-rationals.v1.json | 3 |
| backend/internal/question/{coverage,validate}.go及同名_test.go；content/question-fixtures/{bad-answer,coverage-trap}.json | 4 |
| db/migrations/00005_question_bank.sql；backend/internal/store/{workflow_tx,managed_identity,question_tx,question_idempotency}.go及测试；question_schema_test.go | 5 |
| backend/internal/store/{question_import,question_draft,question_export}.go及测试；backend/internal/cli/question.go及测试；backend/cmd/question-{check,import,export}/main.go | 6 |
| backend/internal/store/{question_submission,question_review,question_fixture_test}.go及测试；question/review.go及测试 | 7 |
| backend/internal/question/{manifest,release}.go及测试；backend/internal/store/{question_release,question_read}.go及测试 | 8 |
| backend/internal/question/{withdrawal,facts}.go及测试；backend/internal/store/{question_withdrawal,question_coverage,question_facts}.go及测试 | 9 |
| backend/internal/httpapi/question_{routes,json,error,dispatch}.go及测试；application.go；backend/cmd/server/main.go；api/openapi.yaml；frontend/src/lib/api/generated.d.ts；api/question-boundary-cases.json | 10 |
| frontend/src/lib/question/{types,schemas,client,server-client}.ts及测试；frontend/src/lib/api/question-proxy.ts及测试；frontend/src/app/api/v1/question-bank/{route.ts,[...segments]/route.ts} | 11 |
| frontend/src/features/question/{draft-list,draft-editor,template-fields,fixed-fields,blueprint-fields,generation-panel,submission-list,review-panel,publication-panel,diff-panel,withdrawal-panel,coverage-panel,pending-command,page-access}.tsx/ts；frontend/src/features/question/{authoring,review,publication,withdrawal,coverage}.test.tsx、pending-command.test.ts；question.module.css；下述七个app页面；frontend/src/components/auth-status.tsx及测试 | 12 |
| backend/internal/e2etest/{question_fixture,question_control}.go及测试、harness.go；tests/e2e/question-{helpers,authoring,review,release,security}.ts/spec.ts；两份CI；docs/operations/{question-bank,2026-10-02-p4a-acceptance}.md；README.md、路线图 | 13 |

任务中简写 question/、store/、httpapi/ 均为 backend/internal/ 下对应目录；全部生产函数明确归属下方接口。每个任务须实现可独立测试的结果，不用生产占位方法假装实现完整Repository。素材只引用已有P3b公开SHA，P4a不新增参数图片生成器。

## 跨任务契约与固定接口

### 数据模型与规范化

下列名称在question包定义；JSON使用lowerCamelCase，必需字段显式存在，不适用字段明确null/空数组。时间RFC3339 UTC；int/int64在Go与Node同时验证原始整数词法，不能先用浮点转换；JSON深度沿用32。类型的字符串枚举逐项进入schema/OpenAPI/Zod，禁止客户端作者、批准或候选head字段。

| 类型 | 完整字段/判别约束 |
| --- | --- |
| Ref | ID string、Version int，复用content.VersionRef；Identity增加SHA256 string；KnownObject含Ref、SHA256、Kind(knowledge/unit/asset)，asset身份另保留ID和SHA，不接收文件路径 |
| ObjectiveCoverage | Knowledge Ref、ObjectiveIndices []int；主知识必有一个非空映射，最多三个额外知识映射；知识/索引各去重并解析固定版本 |
| Rational | Numerator/Denominator string，约分、正分母、规范整数，无负零/前导零；Numerator的负号不计数字位数 |
| Parameter/ParameterValue | Name string、Values []string；实例值改为Value string，规范为Rational的n/d字面量；参数按Name、值按精确数值和规范字面量确定排序 |
| EngineSpec | Family(rational_arithmetic/rational_comparison/missing_operand)、Operation(add/subtract/multiply/divide/compare)、UnknownSide(left/right或null)、GeneratorVersion int、VerifierVersion int，首版均1；家族限定参数名：left/right、left/right、known/result；四则UnknownSide=null且不接受compare，比较Operation=compare/UnknownSide=null，缺失运算数非compare且UnknownSide必须left/right |
| Constraint/Distractor | 固定字符串枚举；约束nonzero_divisor/nonnegative_result/distinct_operands，只按家族适用性接受；数值干扰negate/plus_one/minus_one/reciprocal（最多4种、不重复），比较固定lt/eq/gt，不接受自定义脚本 |
| Choice/AssetRef | Choice含ID/Text string；AssetRef含ID/SHA256 string，绑定当前公开固定素材。选项ID为稳定数学ID，不由显示顺序决定 |
| VerificationWitness | Engine EngineSpec、Parameters []ParameterValue；独立核验数值答案和定义域；概念固定单选可为null，其他固定题必须有支持家族见证 |
| QuestionBody | Type(single_choice/numeric)、Knowledge Ref、Coverage []ObjectiveCoverage、Units []Ref、Prompt/Explanation string、AnswerFormat(rational/percentage或null)、Choices []Choice、CorrectChoiceID *string、CorrectNumeric *Rational、Witness *VerificationWitness、Assets []AssetRef、Sources []content.Source；单选只有CorrectChoiceID，数值只有CorrectNumeric且无Choices |
| Template | ID string、Version int、Knowledge Ref、Coverage []ObjectiveCoverage、Units []Ref、Type、AnswerFormat、PromptTemplate/ExplanationTemplate string、Engine EngineSpec、Parameters []Parameter、Constraints []Constraint、Distractors []Distractor、Assets []AssetRef、Sources []content.Source；比较固定single_choice/null且无自定义干扰 |
| FixedQuestion | ID string、Version int、Body QuestionBody；不接收模板批准或客户端SHA |
| BlueprintSource | Kind(template/instance)、Ref；instance只能引用同包fixedQuestions，模板引用同包templates；冻结后补齐准确SHA，禁止临时客户端题源 |
| Blueprint | ID string、Version int、Knowledge Ref、CoreObjectiveIndices []int、Sources []BlueprintSource、CoverageNote string、RuleVersion int=1、QuestionCount int=5、PassCount int=4 |
| QuestionPackage | Kind string=question-bank、SchemaVersion int=1、ID string、Version int、Templates []Template、FixedQuestions []FixedQuestion、Blueprints []Blueprint |
| Instance | Identity Identity、Origin(fixed/template)、Template *Identity、Parameters []ParameterValue、GeneratorVersion/VerifierVersion *int、Body QuestionBody；固定题相应字段null/[]，生成身份由服务器计算 |
| ResolvedObjective | Knowledge Identity、ObjectiveIndex int、Text string，冻结准确目标文本与原知识SHA，不在审核时改取当前目标 |
| FixedKnowledge/FixedUnit | Knowledge含Identity、Title/TitleZh string、Objectives []string；Unit含Identity、Knowledge Ref、AssetIDs []string，批量读取所需固定字段，不复制全部讲解正文；ReferenceSnapshot含KnowledgeHead *string、CatalogueVersion int、CatalogueSHA256 string、Knowledge []FixedKnowledge、Units []FixedUnit、Assets []content.AssetView；仅内部可信固定读取或离线引用快照，不作为用户批准 |
| Issue/GenerationReport | Issue复用content.Issue但提示不放原题面/路径；GenerationReport含Template Identity、RawCombinations/ExcludedCombinations/ValidInstances int、GeneratorVersion/VerifierVersion int、ConstraintCounts [{constraint,count}]，完整计数不截断 |
| CoverageNode | Knowledge Identity、Blueprint *Identity、EffectiveInstances/FixedInstances/GeneratedInstances/AssessmentInstances/DuplicateInstances int、CoreObjectiveIndices []int、CoveredObjectiveIndices []int、FiveQuestionFeasible bool、Ready bool、Reasons []Issue；SupplementaryObjectiveIndices []int显式列出非核心目标 |
| CoverageReport | KnowledgeHead/QuestionHead *string、PublishedKnowledge/ApprovedTemplates/FixedQuestions/EffectiveInstances/DuplicateInstances int、Nodes Page[CoverageNode]；全局数量按实例身份去重 |
| ValidationReport/SealedPackage | Report含StructuralErrors/CompletenessErrors/HumanReviewRequirements []Issue及三项Total int、Truncated/ReadyToSubmit bool、Digest string、Generation []GenerationReport、Coverage []CoverageNode、PackageBytes/FrozenBytes int；最多展示100问题但总数和ready按完整结果。SealedPackage含Package、PackageSHA、Instances []Instance、Generation、Resolved []KnownObject、Objectives []ResolvedObjective、ReferenceSnapshot；仅内部，不接受客户端输入 |

渲染只认识该家族的参数占位符和解析中的answer：四则left/right，比较left/right，缺失known/result；prompt禁止answer占位符。插值产生受控数值/数学文本，不生成HTML/链接/代码；其他占位符拒绝。nonzero_divisor仅用于四则divide的right或缺失左运算数divide的known；nonnegative_result用于四则结果或缺失家族result参数；distinct_operands仅用于四则/比较left-right。其他家族/操作组合拒绝，未知右除数的定义域/唯一性不靠约束偷偷排除。带数学见证的单选Choice.Text必须是可严格解析的ASCII数值或固定lt/eq/gt关系；受控界面可据此渲染公式，不把任意题面文本当数值。全部选择项做数学等价去重，不静默删除碰撞干扰项；题意与解释仍经人员审核。

规范摘要字节为Go json.Marshal的固定字段结构{purpose,body}，无BOM/换行、UTF-8，保留Go默认HTML转义；body按上表类型顺序，SHA-256计算完整字节。数据库保存同一完整结构为JSONB并校验字节解析相等。摘要按用途分别 question-package-v1/question-instance-identity-v1/question-instance-body-v1/question-submission-v1/question-manifest-v1，使用Go固定字段顺序、规范参数/关系排序和明确null/[]；数组的知识目标/选项顺序保持有语义的原顺序。摘要输入不含自身SHA（CanonicalInstance保留ID/version但排除Identity.SHA256）。ValidationReport.Digest绑定输入、固定引用与全量生成结果；expectedDigest核验此结果，另一个FrozenDigest还绑定服务器固定作者与来源，二者不要求相等。同模板、参数及引擎版本在不同题包生成同一实例身份；成员来源另存，不把题包包装ID加入实例身份。规范化规则以golden bytes锁定，不修改旧digest函数。Canonical函数仅对已规范输入固定编码/摘要，不解析数学；参数归一排序由任务3/4完成，生产入库只能通过SealedPackage，不能直接把未校验客户端正文当规范结果。

### 管理请求与响应

Access直接别名publication.Access，SourceLink直接别名publication.SourceLink，Page[T]字段Items/Total/Limit/Offset；数据库actor而非客户端author。SourceMap路径规则与256 KiB上限沿用现有实现。一般原因、审核Note/IndependenceNote/GenerationNote沿用P3b说明规则（10—1000 Unicode码点、≤3000 UTF-8字节、非空/NUL拒绝）；AdoptInput.Reason使用设计的10—2000码点（最多6000 UTF-8字节），不混用旧1000上限。

| 类型 | 完整字段 |
| --- | --- |
| DraftInput/SaveDraftInput | CatalogueVersion int、QuestionPackage QuestionPackage、SourceMap []SourceLink；Save再含ExpectedRevision int64。无assetBytes或客户端作者 |
| AdoptInput/ValidateInput/SubmitInput | Adopt：PackageID string、PackageVersion int、Reason string；Validate：ExpectedRevision int64；Submit：ExpectedRevision int64、ExpectedDigest string |
| DraftSummary | ID/OwnerID/PackageID/Status/CreatedAt/UpdatedAt string、PackageVersion/CatalogueVersion/StructuralTotal/CompletenessTotal int、Revision int64 |
| DraftView | ID/OwnerID/CatalogueSHA256/Status/CreatedAt/UpdatedAt string、CatalogueVersion int、Revision int64、QuestionPackage、SourceMap、AuthorIDs []string、LegacyUnattributed bool、Gate ValidationReport |
| FrozenBody | CatalogueVersion int、CatalogueSHA256 string、QuestionPackage、SourceMap、AuthorIDs、LegacyUnattributed、Resolved []KnownObject、Objectives []ResolvedObjective、Generation []GenerationReport、InstanceIdentities []Identity、Coverage []CoverageNode、GeneratorVersions/VerifierVersions []int、FrozenDigest string；摘要同时覆盖单独存放的全部Instance字节，不仅覆盖身份 |
| FrozenPayload | Body FrozenBody、Instances []Instance，内部数据库冻结JSONB和规范字节的完整载荷；外部SubmissionView只给Body和实例分页，SHA从包含全部实例正文的Payload计算 |
| SubmissionSummary/SubmissionView | Summary：ID/WorkspaceID/OwnerID/PackageID/Status/FrozenDigest/CreatedAt string、Revision int64、PackageVersion/CatalogueVersion int；View：ID/WorkspaceID/OwnerID/Status/CreatedAt、Revision、Frozen FrozenBody、Gate ValidationReport、Review *ReviewDecision |
| ReviewChecks/ReviewInput/ReviewDecision | Checks六项bool：Mathematics/Explanations/Objectives/Sources/Illustrations/Generation。Input：Decision(approve/return)、Checks、IndependenceNote/GenerationNote/Note string；approve全部true且三说明合格（无模板时GenerationNote明确不适用理由），return只强制Note。Decision加ID/SubmissionID/ReviewerID/FrozenDigest/CreatedAt string |
| MemberIdentity/MemberEvidence/ManifestMember | Identity：Kind(template/instance/blueprint)、ID/PackageID/SHA256 string、Version/PackageVersion int；Evidence复用publication.MemberEvidence固定SubmissionID/DecisionID/FrozenDigest/InheritedFrom；Member含Identity/Evidence，复制题目作者不来自Evidence可编辑字段 |
| Manifest/Change | Manifest：CatalogueVersion int、CatalogueSHA256 string、BaseKnowledgeHead/BaseQuestionHead *string、Members []ManifestMember、Resolved []KnownObject；Change：Kind/ID/Reason string、Before/After *MemberIdentity；元数据与全量实例正文分存 |
| DiffSummary/PublicationSummary | Diff：Added/Replaced/Removed int；Summary：ID/Status/ManifestSHA/CreatedAt string、BaseKnowledgeHead/BaseQuestionHead *string、CatalogueVersion int、CatalogueSHA256 string、TemplateCount/InstanceCount/BlueprintCount int、Diff DiffSummary；不含全量manifest/body/changes |
| PublicationPage/MemberPage/ChangePage | PublicationPage含Page[PublicationSummary]、Head *string；MemberPage含Page[ManifestMember]、PublicationID/ManifestSHA string；ChangePage含Page[Change]、PublicationID/ManifestSHA string |
| PrepareInput/ActivateInput | Prepare：SubmissionIDs []string、ExpectedKnowledgeHead/ExpectedQuestionHead *string、Reason string；Activate：ExpectedKnowledgeHead/ExpectedQuestionHead *string、ExpectedManifestSha/Reason string；head字段必须出现，空head显式null |
| WithdrawalTarget/PreviewInput/WithdrawalInput | Target：Kind(template/instance/blueprint)、ID string、Version int（不允许asset分支）；Preview：Target；Withdrawal：Target、ExpectedKnowledgeHead/ExpectedQuestionHead *string、Reason string |
| WithdrawalPreview/WithdrawalResult | Preview：CurrentKnowledgeHead/CurrentQuestionHead *string、Target、AffectedTemplates/AffectedInstances/AffectedBlueprints int、Diff DiffSummary、Changes Page[Change]、ImpactDigest string；Result：EventID string、PreviousKnowledgeHead/PreviousQuestionHead *string、Publication PublicationSummary。完整影响通过新发布的changes分页核对，预览不截断计数 |
| ListQuery/CoverageQuery | List：Scope(mine/review/all)、Status string、Limit/Offset int；每端口只接受适用枚举；实例/成员/差异只有limit/offset。Coverage只有limit/offset/knowledgeId（可省略，固定数学ID）；撤回预览允许limit/offset稳定分页，POST参数只含Target |
| BaseManifest/ApprovedSubmission/Candidate | BaseManifest含Head *string、Manifest *Manifest、Instances []Instance（空head为null/[]）；ApprovedSubmission含SubmissionID string、Frozen FrozenBody、Instances []Instance、Decision ReviewDecision；Candidate含Manifest、Diff DiffSummary、Changes []Change、Instances []Instance，均内部可信数据，不接收客户端候选 |
| HistoricalFacts | Identity、Approval *MemberEvidence、Replacements []{from Identity,to *Identity,publicationId,createdAt}、Withdrawals []{eventId,target,reason,createdAt}，仅内部按请求身份取有限事实；不设置单个valid布尔值 |

WithdrawalPreview的完整影响≤题库上限，分页通过同一两个head＋ImpactDigest识别变化；激活/撤回前重新核验。客户端分页时保留相同目标并检查摘要，变化提示重新预览；这不新增批准或自动重试。

### 错误、列表及浏览器结果约束

question新sentinel及HTTP映射：ErrDraftConflict→409 QUESTION_DRAFT_CONFLICT；ErrPublicationStale→409 QUESTION_PUBLICATION_STALE；ErrReviewConflict/ErrIdempotencyConflict/ErrImmutableConflict/ErrVersionConflict分别映射同名REVIEW_CONFLICT/IDEMPOTENCY_CONFLICT/IMMUTABLE_CONFLICT/VERSION_CONFLICT；ErrInvalid/ErrNotReady/ErrLimitExceeded/ErrReviewRequired→422 QUESTION_INVALID/QUESTION_NOT_READY/QUESTION_LIMIT_EXCEEDED/REVIEW_REQUIRED；ErrNotConfigured→503 QUESTION_BANK_NOT_CONFIGURED。身份/CSRF/权限/重新验证/限流/notFound沿用auth的sentinel及状态；HTTP正文实际超限用413 PAYLOAD_TOO_LARGE，未知错误/取消/截止/槽不足只用503 SERVICE_UNAVAILABLE。创建、认领、修订、送审、prepare、withdraw为201，其他成功200，重放保留原状态。

列表Scope/Status空字符串表示缺省：drafts为admin默认all、其余editor默认mine，只接受mine/all；submissions为admin→all、reviewer→review、其余editor→mine（按此优先级），review队列排除作者且只显示可审pending；publications仅all。请求范围仍按实际角色授权（显式all只给admin，review只给reviewer，mine须editor）；状态分别限定editing/submitted、pending/approved/returned、prepared/published，未知组合400。实例/成员/差异/preview只接受limit/offset，不接受scope/status；coverage另允许knowledgeId。稳定排序createdAt＋ID，冻结实例按ID，成员/差异按kind＋ID，跨页检查绑定摘要/两个head，不拼接不同快照。

浏览器QuestionAction逐项等于Go Action字符串；QuestionRoute为封闭判别联合：kind加该动作所需id/query，草稿/送审/发布详情及子动作必须id，list动作带相应query、coverage用CoverageQuery、preview允许分页；无任意URL、目标path或额外id。QuestionResult<T>为{ok:true,data:T}或{ok:false,status:number,code:QuestionErrorCode,message:string,requestId:string,retryAfter?:number}；QuestionErrorCode覆盖上述固定代码与既有AUTH_NOT_CONFIGURED（配置未就绪沿用现有账户语义）。所有OpenAPI DTO均完整lowerCamelCase，新增字段不混入旧schema。

### 纯逻辑、服务与仓储

~~~go
ParseNumeric(input, mode string) (Rational, error)
GradeNumeric(correct Rational, mode, input string) (GradeResult, error)
GradeChoice(choices []Choice, correctID, inputID string) (GradeResult, error)
CanEnterAnswer(correct Rational, mode string) bool
Generate(ctx context.Context, t Template) ([]Instance, GenerationReport, error)
VerifyInstance(i Instance) error
FiveQuestionCover(core []int, candidates []CandidateCoverage) ([]string, bool, error)
ValidateEditable(ctx context.Context, input DraftInput, refs ReferenceSnapshot) (ValidationReport, error)
ValidateAndSeal(ctx context.Context, input DraftInput, refs ReferenceSnapshot) (SealedPackage, ValidationReport, error)
CanonicalPackage(p QuestionPackage) ([]byte, string, error)
CanonicalInstance(i Instance) ([]byte, string, error)
CanonicalValidation(input DraftInput, sealed SealedPackage) ([]byte, string, error)
CanonicalFrozen(body FrozenBody, instances []Instance) ([]byte, string, error)
NewService(repo Repository, acquire func(context.Context) (func(), error)) (*Service, error)
~~~

GradeResult只有Correct bool；解析错误使用独立受限NumericFormatError(Code string)，格式错误不能当成Correct=false提交。CandidateCoverage含InstanceID string、ObjectiveIndices []int，函数要求不同身份，最多1000候选/8核心、返回确定性五个ID见证。SHA/参数生成内部辅助不对浏览器提供答案服务。SealedPackage的旧引擎版本实现按登记版本保留；不存在版本拒绝，不默认换成最新版。

Repository方法全部加Question前缀避免与现有Store的同名publication方法冲突；Service对应方法去掉Question前缀（例如ListQuestionDrafts→ListDrafts）。ReadSession/ConsumeRates沿用Store已有方法，其他端口完整定义如下：

~~~go
QuestionPreflight(context.Context, Access, Action) (auth.User, error)
ListQuestionDrafts(context.Context, Access, ListQuery) (Page[DraftSummary], error)
CreateQuestionDraft(context.Context, Access, DraftInput) (DraftView, error)
ReadQuestionDraft(context.Context, Access, string) (DraftView, error)
SaveQuestionDraft(context.Context, Access, string, SaveDraftInput) (DraftView, error)
AdoptQuestionDraft(context.Context, Access, AdoptInput) (DraftView, error)
ValidateQuestionDraft(context.Context, Access, string, ValidateInput) (ValidationReport, error)
SubmitQuestionDraft(context.Context, Access, string, SubmitInput) (SubmissionView, error)
ListQuestionSubmissions(context.Context, Access, ListQuery) (Page[SubmissionSummary], error)
ReadQuestionSubmission(context.Context, Access, string) (SubmissionView, error)
ListQuestionInstances(context.Context, Access, string, ListQuery) (Page[Instance], error)
ReviseQuestionSubmission(context.Context, Access, string) (DraftView, error)
DecideQuestionReview(context.Context, Access, string, ReviewInput) (SubmissionView, error)
ListQuestionPublications(context.Context, Access, ListQuery) (PublicationPage, error)
ReadQuestionPublication(context.Context, Access, string) (PublicationSummary, error)
ListQuestionMembers(context.Context, Access, string, ListQuery) (MemberPage, error)
ListQuestionChanges(context.Context, Access, string, ListQuery) (ChangePage, error)
PrepareQuestionRelease(context.Context, Access, PrepareInput) (PublicationSummary, error)
ActivateQuestionRelease(context.Context, Access, string, ActivateInput) (PublicationSummary, error)
PreviewQuestionWithdrawal(context.Context, Access, WithdrawalPreviewInput, ListQuery) (WithdrawalPreview, error)
WithdrawQuestionVersion(context.Context, Access, WithdrawalInput) (WithdrawalResult, error)
ReadQuestionCoverage(context.Context, Access, CoverageQuery) (CoverageReport, error)
~~~

Service.Preflight从QuestionPreflight读取实际身份并调用共用Rates；每个store方法事务内再授权。Action固定枚举对应上述管理动作，另有listInstances/listMembers/listChanges/readCoverage。Authorize(user auth.User,action Action)error、IsRead/IsHeavy/IsIdempotent(Action)bool、Rates(actor string,action Action)([]auth.RateKey,error)由question/policy.go与rate_limit.go定义；readCoverage既只读又heavy，Rates先判断heavy再判断read。Repository另有ReadSession(context.Context,auth.Digest,bool)(auth.SessionRecord,error)、ConsumeRates(context.Context,[]auth.RateKey)error。Service.AcquireValidation直接调用注入的既有函数，不创建channel；Task1测试用完整recordingRepository，Store至任务9才添加完整编译期断言。

store内部：questionTx(ctx,Access,Action,related []string,fn func(context.Context,*sql.Tx,auth.User,time.Time)error)error、questionReadTx(ctx,Access,Action,fn func(context.Context,*sql.Tx,auth.User)error)error；配置函数questionConfigured(ctx context.Context,tx *sql.Tx)error；公共证明函数managedIdentity(ctx,*sql.Tx,publication.Access,write,lock bool,related []string)(auth.User,auth.SessionRecord,time.Time,error)仅校验会话/凭据/CSRF，动作授权与reauth仍由workflow/question wrappers分别决定，原workflow函数签名不变。

store内部批量referenceSnapshot(ctx,*sql.Tx,refs []Ref,units []Ref,assets []AssetRef)(ReferenceSnapshot,error)只读取当前内容head的固定对象，不能任意打开草稿；实例/成员/作者/证据批量读取。questionRequestDigest(action question.Action,target string,input any)(string,error)只计算严格解码原输入的规范JSON摘要，包含全部原因/head/revision/目标，不能替换为数学归一后的题包SHA；questionReplay/questionRemember与P3b形状相同但新表、题库动作scope，重放前仍调用当前身份授权。当前状态与历史事实函数为questionOfferable(ctx,*sql.Tx,identities []Identity)([]Instance,error)、questionHistoricalFacts(ctx,*sql.Tx,identities []Identity)([]HistoricalFacts,error)，仅供可信服务器使用；不新增公共路由。

CLI接口RunQuestion(ctx context.Context,command string,args []string,stdout,stderr io.Writer)int；check/import使用--input题包envelope、--references离线固定引用快照；export使用--id/--version/--out（新目录）。Archive完整字段Envelope DraftInput、PackageSHA string、Instances []Instance、GeneratorVersions/VerifierVersions []int、SourceResponsibility {AuthorIDs []string,LegacyUnattributed bool}；这些是来源证据，不是可导入批准。导入重算实例/摘要、用DB核验既有固定作者；外来作者ID保留为历史来源并标未核实，不作为自审资格证明或可由CLI指定平台作者。原包、实例与来源可复验，认领后的新送审摘要因新增实际作者而不同。

## Task 1：独立契约、规范摘要与服务策略

**Files:** 文件责任表任务1全部文件；不修改旧schema/model/digest。
**Interfaces:** 产出上述所有question数据声明、Repository、Action、NewService、CanonicalPackage/CanonicalInstance，严格DecodePackage(io.Reader)(QuestionPackage,error)、DecodeDraft(io.Reader)(DraftInput,error)。模板参数/枚举的家族细节按契约表锁定。

- [x] **Step 1：写失败测试。** TestQuestionContractBoundary断言未知kind/schema、重复字段/大小写别名/尾随JSON/未配对代理项/32层＋1拒绝，4 MiB envelope/2 MiB package边界；TestQuestionCanonicalIdentity断言固定golden bytes/SHA、关系排序稳定但选项与目标顺序有语义、qi-ID长度67；TestQuestionRoleAndRates断言角色矩阵、coverage GET heavy、30/10/120原预算与未知action拒绝；TestQuestionServiceInjection断言nil repo/acquire拒绝、注入函数实际调用、不存在新槽池。旧内容mathID对67字符仍拒绝。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionContractBoundary":{"unknownField":"reject","duplicateKey":"reject","envelopeMax":4194304,"packageMax":2097152}},
  {"TestQuestionCanonicalIdentity":{"instanceIDLength":67,"oldMathIDAcceptsGeneratedID":false,"repeatDigest":"equal","emptyPackageSHA":"67a0f28c3f9b225341c40caba289789df64efefcee4523aefadc029306081498"}},
  {"TestQuestionRoleAndRates":{"normalUser":30,"heavyUser":10,"readUser":120,"coverage":"heavy"}},
  {"TestQuestionServiceInjection":{"thirdSlot":"SERVICE_UNAVAILABLE","newSlotPool":false}}
]
~~~

- [x] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/question -run 'TestQuestion(ContractBoundary|CanonicalIdentity|RoleAndRates|ServiceInjection)$' -timeout 5m -count=1；预期新契约不存在或行为失败，工具故障不算RED。
- [x] **Step 3：实现声明与纯边界。** schema为封闭判别结构；生成器/完整仓储暂不接入生产，不添加假成功占位。包和实例SHA分别计算，稳定ID/版本规则不修改旧代码。服务测试通过测试专用recordingRepository验证分发与限流。
- [x] **Step 4：验证GREEN。** 重跑Step2；追加同入口go test ./internal/content ./internal/publication -timeout 5m -count=1，确认旧digest/角色/ID用例通过。
- [x] **Step 5：提交。** 暂存任务1精确文件，提交 feat: 建立独立题库契约与策略。


Task1空数组结构夹具的规范字节（不含代码围栏换行）；完整性检查仍可判其未就绪，不影响结构摘要测试：

~~~json
{"purpose":"question-package-v1","body":{"kind":"question-bank","schemaVersion":1,"id":"contract-fixture","version":1,"templates":[],"fixedQuestions":[],"blueprints":[]}}
~~~

## Task 2：精确数值格式与纯判分

**Files:** question/{numeric,grade}.go及测试。
**Interfaces:** 消费Rational/QuestionBody；产出ParseNumeric、GradeNumeric、GradeChoice、CanEnterAnswer、GradeResult/NumericFormatError，不增加HTTP判分端口。

- [x] **Step 1：写失败测试。** TestNumericModesAndBoundary：1/2=2/4=0.50、1/-2=-0.5、.5/1./+002/负零合法归一、分数只在slash两侧空格/制表符、裸50不能当50%、50%=1/2；128/129原始字符边界、256/257位结果、零分母、指数/表达式/NaN/Unicode数字/数字内部空白/分数百分数拒绝。TestExactGrade：未知choiceID返回格式错误；合法错误答案Correct=false；非法格式不产生判分结果。TestNumericAnswerRepresentable：rational的1/3可输入、percentage的1/3无有限百分数表达不可入库，1/2可50%，整数/分数/小数最短合法形式均超过128字符时false。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestNumericModesAndBoundary":{"rational":["1/2","2/4","0.50"],"allEqual":true,"percentageBare50":"format_error","zeroDenominator":"format_error","maxInputCharacters":128,"maxResultDigits":256}},
  {"TestExactGrade":{"unknownChoice":"format_error","wrongValidAnswer":false}},
  {"TestNumericAnswerRepresentable":{"rationalThird":true,"percentageThird":false,"percentageHalf":true}}
]
~~~

- [x] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/question -run 'Test(NumericModesAndBoundary|ExactGrade|NumericAnswerRepresentable)$' -timeout 5m -count=1。
- [x] **Step 3：实现判分及可输入性函数。** 用big.Int构造十进制/分数，big.Rat归一比较；原始rune数先限制，trim只作用首尾。256位答案检测为内部规范Rational检查，不经128字符的用户输入解析器，二者上限不可混用。规范答案校验包括约分和正分母，不使用ParseFloat或表达式引擎。CanEnterAnswer枚举整数/规范分数/终止小数候选最短字符数，percentage先乘100再要求分母只含2/5且含百分号；数值题答案至少一种合法表示≤128字符，否则不能送审。
- [x] **Step 4：验证GREEN。** 重跑Step2，再单独通过入口运行question全套，预期0失败；输入判分不修改任何数据库。
- [x] **Step 5：提交。** 仅任务2文件，提交 feat: 增加精确数值与稳定选项判分。

## Task 3：有界生成、受控插值与独立校验

**Files:** question/{generate,verify,render}.go及测试；原创content/questions/elementary-rationals.v1.json（技术草稿，不计已审核数量）。
**Interfaces:** 消费Task1/2，产出Generate、VerifyInstance；每家族分离生成答案和Big.Int验证，不共享答案算法。

- [ ] **Step 1：写失败测试。** TestGeneratorFiniteSpace：4/5参数、32/33值、1000/1001组合、精确等价参数先去重、零有效实例、未声明除零、显式排除完整计数、包级总生成量由任务4验收；TestMissingOperandUniqueSolution覆盖0*x=0多解、0*x=1无解和除法定义域；TestTemplateChoiceEquivalence拒绝1/2与2/4、干扰项等于答案；TestVerifierRejectsMutatedGeneratedAnswer在正确参数保留时把数值/choice答案变错，独立拒绝三个家族；TestControlledQuestionRender拒绝未知占位符、prompt的answer、HTML/链接注入。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestGeneratorFiniteSpace":{"parameterMax":4,"valueMax":32,"rawMax":1000,"undeclaredDivisionByZero":"reject","silentSkip":false}},
  {"TestMissingOperandUniqueSolution":{"zeroTimesXEqualsZero":"reject","zeroTimesXEqualsOne":"reject"}},
  {"TestTemplateChoiceEquivalence":{"halfVsTwoFourths":"reject"}},
  {"TestVerifierRejectsMutatedGeneratedAnswer":{"arithmetic":"reject","comparison":"reject","missing":"reject"}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/question -run 'Test(GeneratorFiniteSpace|MissingOperandUniqueSolution|TemplateChoiceEquivalence|VerifierRejectsMutatedGeneratedAnswer|ControlledQuestionRender)$' -timeout 5m -count=1。
- [ ] **Step 3：实现Generate/VerifyInstance。** 先检查完整组合/字节预算再分配；按固定枚举约束检查每组合，计数完整且不吞失败；生成Rat、验证独立交叉乘积或反代/唯一性，定时检查context。三家族版本1单独登记，未知/不存在旧版本拒绝；重复身份不增加题量。
- [ ] **Step 4：验证GREEN。** 重跑Step2；对技术题包全部参数完整校验，验证篡改fixture确实失败，同输入重复生成字节与身份一致。不得把校验成功写成数学批准。
- [ ] **Step 5：提交。** 仅任务3文件，提交 feat: 生成可复验实例并独立检查数学答案。

## Task 4：固定引用、五题覆盖与题包封存

**Files:** question/{coverage,validate}.go及测试；两个负向原创fixture。
**Interfaces:** 消费Task3和ReferenceSnapshot；产出FiveQuestionCover、ValidateEditable、ValidateAndSeal、SealedPackage/ValidationReport。草稿结构合法可保存，送审必须完整且绑定当前公开固定引用。

- [ ] **Step 1：写失败测试。** TestFiveDistinctQuestionsCoverAllCore：8/9核心、1000/1001池、重复ID不能凑5题、恰好5覆盖、并集完整但需6题才覆盖→false；最多12候选的小池以固定种子20261002生成100组，用独立穷举五组合对照可行性和返回五ID见证、不得复用可变父状态；TestQuestionFixedReferences：objectiveIndex越界、知识版本不匹配、未公开K/单元/素材、跨知识错映射、第四额外映射、蓝图题源跨包、核心缺失拒绝；TestQuestionSealLimits：50/51模板、200/201固定、100/101蓝图、1000/+1实例、4 MiB frozen、8 KiB题面解析、6/7选项、8/9素材；TestFixedNumericWitness：无适用见证或答案没有128字符内合法表示拒绝，概念单选必须列人工复核；101问题只显示100但ready=false。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestFiveDistinctQuestionsCoverAllCore":{"coreMax":8,"poolMax":1000,"duplicateQuestionCountsTwice":false,"unionNeedsSix":false}},
  {"TestQuestionFixedReferences":{"unpublishedVersion":"reject","objectiveIndexOutOfRange":"reject","extraCoverageMax":3}},
  {"TestQuestionSealLimits":{"templates":50,"fixed":200,"blueprints":100,"instances":1000,"frozenBytes":4194304}},
  {"TestFixedNumericWitness":{"missingWitness":"reject","unenterableAnswer":"not_ready"}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/question -run 'Test(FiveDistinctQuestionsCoverAllCore|QuestionFixedReferences|QuestionSealLimits|FixedNumericWitness)$' -timeout 5m -count=1。
- [ ] **Step 3：实现完整校验与封存。** 覆盖用按候选迭代的(count≤5,mask≤255)状态，倒序数量更新防重复取同一题并记录五ID见证；非核心目标显式补充。验证题源闭包、固定引用SHA、受限Markdown和来源，再冻结完整实例字节；不因显示报告截断而跳过数学检查。
- [ ] **Step 4：验证GREEN。** 重跑Step2；记录最大单模板1000组合/节点1000候选在8秒内是否完成，未达标定位优化，不改限额。草稿缺公开引用时safe gate提示，不能读取其他人员知识草稿。
- [ ] **Step 5：提交。** 仅任务4文件，提交 feat: 验证固定题包与完整五题目标覆盖。

## Task 5：数据库不可变约束与共享授权事务

**Files:** 00005迁移；store/{managed_identity,question_tx,question_idempotency}.go及测试、question_schema_test.go；小范围修改workflow_tx.go及原测试。
**Interfaces:** 产出questionTx/questionReadTx/questionConfigured/managedIdentity/questionReplay/questionRemember；保留workflowIdentity/workflowTx/workflowReadTx签名、动作授权、原错误映射。

- [ ] **Step 1：写失败测试。** TestQuestionSchema：设计18张领域表、FK/唯一身份/参数摘要、固定字节JSONB/SHA一致、冻结后父/子UPDATE/DELETE/额外关联拒绝、只插review或半送审提交拒绝、workspace状态窄例外、00001—00004字节和原数据SHA不变；TestQuestionSessionExpiresWhileWaiting真实屏障覆盖账户行/内容行等待后过期、提交前过期、撤权/密码版本/CSRF/5分钟边界；TestQuestionIdempotencyAtomicity：同键同输入一次结果、异输入409、撤权重放拒绝、审计失败全部回滚。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionSchema":{"tables":18,"partialSubmissionCommit":"reject","postFreezeChildInsert":"reject","oldMigrationBytes":"unchanged"}},
  {"TestQuestionSessionExpiresWhileWaiting":{"sessionExpires":"reject","roleRevoked":"reject","lockMaxSeconds":1,"reauthMaxSeconds":300}},
  {"TestQuestionIdempotencyAtomicity":{"sameRequest":"one_result","differentRequest":"IDEMPOTENCY_CONFLICT","auditFailure":"rollback"}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run 'TestQuestion(Schema|SessionExpiresWhileWaiting|IdempotencyAtomicity)$' -timeout 5m -count=1，仅全新随机测试库。
- [ ] **Step 3：追加迁移与共享证明。** 建设计列出的workspaces/authors、packages/templates/instances/blueprints、coverage/sources、submissions/authors/members、decisions、publications/members/heads、withdrawals/events/idempotency；CHECK、准确FK、唯一固定身份、延迟一致性与不可变触发器。managedIdentity返回会话和DB时钟，CSRF是否必需由wrapper指定，不允许读取动作因为共享提取变成要求CSRF。wrapper各自Authorize/reauth，先锁后取当前proof、提交前再核验；error不暴露SQL。
- [ ] **Step 4：验证GREEN。** 重跑Step2；独立单批运行原workflow_tx/roles/account并发和幂等全部回归。迁移只运行在随机库，questionConfigured检查缺表返回固定QUESTION_BANK_NOT_CONFIGURED，服务器不自动up。
- [ ] **Step 5：提交。** 任务5文件，提交 feat: 增加题库固定记录与授权事务约束。

## Task 6：离线检查、不可变导入导出与本人草稿

**Files:** store/question_{import,draft,export}.go及测试；cli/question.go/question_test.go；三个cmd/main.go。
**Interfaces:** 产出RunQuestion、ImportQuestionDraft(ctx,Archive)(QuestionImportResult,error)、ExportQuestionArchive(ctx,id string,version int)(Archive,error)，以及Repository的draft六个方法和QuestionPreflight；QuestionImportResult含PackageID/PackageSHA/Status string、PackageVersion/ImportedInstances/DuplicateInstances int，Status固定draft。

- [ ] **Step 1：写失败测试。** TestQuestionCLIArchiveRoundTrip保持包/实例SHA、引擎版本、sourceMap和可验证作者来源；同库既有作者由DB继承、外来记录不继承批准、输出只能新目录且无路径泄漏；TestQuestionDraftOwnership：本人编辑、admin只读、他人404、expectedRevision冲突、结构合法不完整可保存、private知识草稿不能读取；TestQuestionAdoptionPreservesResponsibility：10/2000及±1边界、复制原作者/legacy标记不可清空、CLI不能指定可信平台作者或review，当前认领者新增整理责任。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionCLIArchiveRoundTrip":{"packageSHA":"equal","instanceSHA":"equal","sourceResponsibility":"preserved","approvalImported":false}},
  {"TestQuestionDraftOwnership":{"otherEditor":"NOT_FOUND","adminSave":"FORBIDDEN","staleRevision":"QUESTION_DRAFT_CONFLICT"}},
  {"TestQuestionAdoptionPreservesResponsibility":{"reasonMin":10,"reasonMax":2000,"authorsCleared":false,"legacyCleared":false}}
]
~~~

- [ ] **Step 2：验证RED。** 分两批：node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/cli -run TestQuestionCLI -timeout 5m -count=1；同入口go test ./internal/store -run 'TestQuestion(DraftOwnership|AdoptionPreservesResponsibility)$' -timeout 5m -count=1。
- [ ] **Step 3：实现工具与草稿。** 导入只取既有内容锁、再核验数据库引用，不反向取admin锁；检查/导入用输入上限和8秒数学截止，不把离线references视为公开批准。保存只固定输入和revision，validate针对已保存revision实时读取公开引用/完整生成；summary不含答案/来源。不存在snapshot或已有同版本异字节时整次失败，原CLI调用接口不改。
- [ ] **Step 4：验证GREEN。** 重跑Step2；用临时新目录和随机库执行check→import→export→check，技术草稿始终零批准零公开；原content check/import/export回归通过。导出失败仅清理本次创建目录。
- [ ] **Step 5：提交。** 任务6文件，提交 feat: 增加未批准题包工具与安全草稿编写。

## Task 7：冻结送审、作者继承与独立复核

**Files:** store/question_{submission,review}.go及测试、question_fixture_test.go；question/review.go及测试。
**Interfaces:** 产出Submit/List/Read/ReviseQuestionSubmission、ListQuestionInstances、DecideQuestionReview；FrozenBody/Instance关联保证同一冻结摘要，分页不读取后来工作区。

- [ ] **Step 1：写失败测试。** TestQuestionFrozenSubmission：expectedRevision/digest双对比、重算后来源变化、冻结完整实例/作者/来源/目标/assetSHA、原工作区/文件变化后逐字节相同；TestQuestionCopiedAuthorCannotApprove：editor+reviewer、adopt/跨题包复用同ID/version相同字节时继承作者且均不可自审，异字节VERSION_CONFLICT；TestQuestionReviewFinality：六项check和无模板说明、竞争审核仅一终态、returned原工作区revision+1、approved修订新workspace且不继承批准；TestQuestionInstancePages：全部分页可见、别送审同ID不能借页越权、实际4 MiB缩页、冻结后不得补成员。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionFrozenSubmission":{"mutableWorkspaceChangesFrozenBytes":false,"mutableSourceChangesFrozenBytes":false,"payloadIncludesAllInstances":true}},
  {"TestQuestionCopiedAuthorCannotApprove":{"adopter":"reject","copiedAuthor":"reject"}},
  {"TestQuestionReviewFinality":{"finalDecisions":1,"checks":6,"returnedRevisionIncrement":1}},
  {"TestQuestionInstancePages":{"allInstancesReadable":true,"unboundSubmission":"NOT_FOUND","maxBytes":4194304}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run 'TestQuestion(FrozenSubmission|CopiedAuthorCannotApprove|ReviewFinality|InstancePages)$' -timeout 5m -count=1。
- [ ] **Step 3：实现送审与审核事务。** 精确同版本查出原作者，锁其账户行；以全量冻结载荷规范摘要保存字节/JSONB/子表，实例页面仅读取冻结绑定。审核队列排除作者，reviewer只批准当前pending且Checks全true；author、sourceMap、engine信息不能来自当前workspace回查。
- [ ] **Step 4：验证GREEN。** 重跑Step2；批量实例及审核依据读取不得N+1展开每个历史送审，审计/终态错误使全事务回滚；原P3b独立复核回归保持。
- [ ] **Step 5：提交。** 任务7文件，提交 feat: 冻结题库并要求真实独立复核。

## Task 8：固定manifest、双head准备与激活

**Files:** question/{manifest,release}.go及测试；store/question_{release,read}.go及测试。
**Interfaces:** 产出BuildCandidate(ctx,BaseManifest,[]ApprovedSubmission,ReferenceSnapshot)(Candidate,error)（内部Candidate含Manifest/DiffSummary/Changes/Instances），Repository发布五个方法＋member/change分页；BaseManifest只加载一次验证，ApprovedSubmission含冻结body/instances/decision。

- [ ] **Step 1：写失败测试。** TestQuestionDualHeadActivation：任何一head准备后变化409、manifestSha不符、撤权/rehash/reauth过期拒绝、审计失败head不变、重放不恢复旧head；TestQuestionEvidenceEligibility：新选reviewer已撤权拒绝、已发布继承证据可沿用、作者/原冻结digest冲突拒绝；TestQuestionReplacementPreservesHistoricalFacts：同模板新version退出旧generated实例、引用旧模板蓝图须同批批准更新、旧正文不删除、不写永久撤回；TestQuestionPublicationPagination：101历史/head不在第一页、每项summary≤4 MiB、未来published状态增字节仍可读、members/changes精确分页；TestQuestionCandidateCapacity验证200/10000/1000、32/8 MiB以及1—20送审边界。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionDualHeadActivation":{"changedKnowledgeHead":"QUESTION_PUBLICATION_STALE","changedQuestionHead":"QUESTION_PUBLICATION_STALE","replaySwitchesHead":false}},
  {"TestQuestionEvidenceEligibility":{"newRevokedReviewer":"reject","inheritedEvidence":"preserve"}},
  {"TestQuestionReplacementPreservesHistoricalFacts":{"oldInstanceBody":"preserve","oldBlueprintUnchanged":"reject","permanentWithdrawalCreated":false}},
  {"TestQuestionPublicationPagination":{"historyRows":101,"headFoundByID":true}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run 'TestQuestion(DualHeadActivation|EvidenceEligibility|ReplacementPreservesHistoricalFacts|PublicationPagination|CandidateCapacity)$' -timeout 5m -count=1。
- [ ] **Step 3：实现固定候选与激活。** 批量校验当前准确K/unit/asset与catalogue身份，整批五题可覆盖，更新template须对应准确蓝图题源。prepare固定两个head和manifest/diff但不公开；activate同事务重新授权、5分钟验证、两head比较、黑名单/审核/容量核验，再变更head/状态/审计/幂等。PublicationSummary不携带大正文，head单独读取。
- [ ] **Step 4：验证GREEN。** 重跑Step2；同步屏障覆盖知识发布与题库激活竞争，只有符合实际两个head的一方可提交；原P3b发布/公开读取回归保持。
- [ ] **Step 5：提交。** 任务8文件，提交 feat: 通过固定双head证据激活可信题库。

## Task 9：永久撤回、当前覆盖与历史事实

**Files:** question/{withdrawal,facts}.go及测试；store/question_{withdrawal,coverage,facts}.go及测试。
**Interfaces:** 产出PreviewQuestionWithdrawal、WithdrawQuestionVersion、ReadQuestionCoverage、questionOfferable/questionHistoricalFacts；到此var _ question.Repository=(*Store)(nil)才加入生产编译断言。

- [ ] **Step 1：写失败测试。** TestQuestionPermanentWithdrawalFacts：template移除全部关联实例/蓝图、fixed实例移除显式蓝图、单generated实例只移该实例并重新计算模板蓝图ready、blueprint不删正确讲解；TestQuestionCoverageAfterSingleInstanceWithdrawal：5→4不足、不用重复凑数/旧ready、允许撤成空库；TestQuestionKnowledgeWithdrawalStopsOffer：K/unit/asset当前不可用立即停止、无关K更新不整体停止；TestQuestionWithdrawalReplayRace：两head变化/不同键重复目标409，激活或旧送审不能带回黑名单；TestQuestionHistoricalFactSeparation：普通替换仅退出new offer，永久撤回另列固定事实，历史字节可追溯；TestQuestionCoverageCounts：distinct总量与多节点覆盖区别、固定题/模板/实例不混算。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionPermanentWithdrawalFacts":{"template":"instances_and_dependent_blueprints","generatedInstance":"only_instance_and_recomputed_readiness"}},
  {"TestQuestionCoverageAfterSingleInstanceWithdrawal":{"effectiveInstances":4,"ready":false,"emptyHeadAllowed":true}},
  {"TestQuestionWithdrawalReplayRace":{"blacklistRestored":false,"differentKeySameTarget":"IMMUTABLE_CONFLICT"}},
  {"TestQuestionHistoricalFactSeparation":{"replacementIsWithdrawal":false}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run 'TestQuestion(PermanentWithdrawalFacts|CoverageAfterSingleInstanceWithdrawal|KnowledgeWithdrawalStopsOffer|WithdrawalReplayRace|HistoricalFactSeparation|CoverageCounts)$' -timeout 5m -count=1。
- [ ] **Step 3：实现撤回闭包与分离读取。** 从当前head固定影响/ImpactDigest，按Target规则生成新manifest与黑名单/审计/幂等同事务；不把非当前成员当不存在或永久撤回。coverage repeatable-read读取两个head及固定有效性，每节点动态FiveQuestionCover，报告分页面向reviewer/admin；历史读取不套当前head存在性过滤。
- [ ] **Step 4：验证GREEN。** 重跑Step2；原知识/单元/素材撤回仍使用原闭包/锁且不增旧表kind；无用户assessment/learning写入。随机库记录全部新表/原表兼容回归结果。
- [ ] **Step 5：提交。** 任务9文件，提交 feat: 永久撤回题目并区分当前可用与历史证据。

## Task 10：Go私有HTTP、共享资源与OpenAPI

**Files:** httpapi/question_{routes,json,error,dispatch}.go及测试；application.go；cmd/server/main.go；api/openapi.yaml；generated.d.ts；api/question-boundary-cases.json。
**Interfaces:** QuestionOptions{Service *question.Service,PublicOrigin string,Production/Configured bool}，AuthOptions增加Question *QuestionOptions；QuestionReady(ctx context.Context,db *sql.DB)(bool,error)只读检查18表/迁移是否齐备。NewService注入同一个publicationService.AcquireValidation，serveQuestion固定路由，不提供学习者答案/自由判分端口。

- [ ] **Step 1：写失败测试。** TestQuestionHTTPContract所有设计端口、根/额外段404、方法405/Allow、状态201/200及幂等重放、未知/重复JSON/query/原始整数词法/UTF-8/NUL/终止边界；TestQuestionPrivateScopeAndErrors：learner不可读答案、private越权404、缺迁移503与旧业务正常、error/log无题面/路径/SQL/secret；TestSharedQuestionValidationSlots：P3b+题库共用两槽、第三项立即503、取消/失败/慢body释放时机；TestSharedContentQuestionRateWindow跨后台共享30/10配额、coverage计heavy且有Retry-After；TestQuestionBodyDeadline包含body/SQL8s、超大chunked413、取消/断连无残留事务，原账户4s不改。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionHTTPContract":{"unknownPath":404,"wrongMethod":405,"create":201,"save":200,"maxResponseBytes":4194304}},
  {"TestSharedQuestionValidationSlots":{"sharedTotalSlots":2,"third":503,"cancellationLeaksSlot":false}},
  {"TestSharedContentQuestionRateWindow":{"normalUserCombined":30,"heavyUserCombined":10,"coverage":"heavy"}},
  {"TestQuestionBodyDeadline":{"goSeconds":8,"oversizeChunked":413,"accountSeconds":4}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/httpapi -run 'Test(QuestionHTTPContract|QuestionPrivateScopeAndErrors|SharedQuestionValidationSlots|SharedContentQuestionRateWindow|QuestionBodyDeadline)$' -timeout 5m -count=1。
- [ ] **Step 3：实现固定路由与错误映射。** 严格解码前做实际账户证明和写Origin/CSRF；大正文及重操作持已有槽至实际结束。错误状态/code逐项实现设计第11节，QUESTION_*只映射新sentinel；完整业务响应实际序列化≤4 MiB、no-store，秘密证明不序列化。控制请求≤8 KiB，创建/保存4 MiB；撤回预览允许limit/offset并验证全部影响计数摘要。共享资源装配只有一个publication.Service，不再NewService两次制造独立槽。
- [ ] **Step 4：验证GREEN。** 重跑Step2；单批原httpapi完整回归。api:generate两次均零差异；比较master旧paths/schema结构逐项不变，仅新增题库端口/DTO。共享边界用例文件含请求原字节、路径/方法、期待status/code，两端读取同一用例。
- [ ] **Step 5：提交。** 任务10文件，提交 feat: 接入严格私有题库API与共享资源边界。

## Task 11：TypeScript私有契约、原始代理与客户端

**Files:** lib/question/{types,schemas,client,server-client}.ts及测试；lib/api/question-proxy.ts及测试；两个固定Next.js路由。
**Interfaces:** questionRouteRequest(route QuestionRoute):{method:string,path:string}|null；readQuestionResponse(response:Response,action:QuestionAction,signal?:AbortSignal):Promise<QuestionResult<unknown>>；createQuestionProxy(rawGoOrigin:string,options:{publicOrigin:string,production:boolean},fetcher?:typeof fetch)返回(request:Request,segments:string[])=>Promise<Response>；readServerQuestion<T>(route,cookieHeader):Promise<QuestionResult<T>>；requestQuestion<T>(route,input?,pendingKey?,signal?):Promise<QuestionResult<T>>。

- [ ] **Step 1：写失败测试。** TestQuestionProxyBoundary读取api共享raw用例，重复/未知键、1e0/1.0版本、unsafe int、null/遗漏字段、未知status拒绝，允许的原字节不重新stringify；TestQuestionResponseIntegrity核验status/DTO/headers、实际4 MiB、null review/head、分页缩limit、来源答案只授权角色；TestQuestionProxyCancellation body到fetch到response全程10s/信号、redirect拒绝、仅选账户cookie/CSRF、Set-Cookie不转发、SSR server-only/no-store；TestQuestionClientNoAutomaticRetry成功键生成一次、超时不猜失败/自动重写。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"TestQuestionProxyBoundary":{"version1e0":"reject","duplicateKey":"reject","acceptedBodyBytes":"identical"}},
  {"TestQuestionResponseIntegrity":{"maxBytes":4194304,"missingHead":"reject","headNull":"accept"}},
  {"TestQuestionProxyCancellation":{"totalSeconds":10,"redirect":"reject","forwardSetCookie":false}},
  {"TestQuestionClientNoAutomaticRetry":{"automaticWrites":0,"manualSameInputKey":"unchanged"}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd frontend -- npm test -- src/lib/question src/lib/api/question-proxy.test.ts，预期新行为失败。
- [ ] **Step 3：实现封闭Zod与固定代理。** 复用现有raw-json/bytes纯读取工具但保持旧接口不变；按新的具名DTO判别输入输出，原body bytes直接转发。普通SSR不获取CSRF，写入从现有context受控取得；只允许白名单GoOrigin与明确路由，拒绝通用URL参数。客户端错误文字英文，数值算法只在Go。
- [ ] **Step 4：验证GREEN。** 重跑Step2；单独入口npm run typecheck和原content/auth/lib客户端单元回归。api:generate产物与Task10契约一致，所有私有fetch不进入公共缓存。
- [ ] **Step 5：提交。** 任务11文件，提交 feat: 增加严格题库私有代理与类型客户端。

## Task 12：英文编写、审核、发布与撤回页面

**Files:** 文件表任务12组件/样式/测试；frontend/src/app/editor/questions/page.tsx、editor/questions/drafts/[id]/page.tsx、review/questions/page.tsx、review/questions/[id]/page.tsx、admin/question-publications/page.tsx、admin/question-publications/[id]/page.tsx、admin/question-withdrawals/page.tsx；auth-status导航。
**Interfaces:** Task11数据层；draft-editor只编辑DraftInput，generation-panel只显示保存revision的ValidationReport，review-panel冻结正文＋Instance分页＋六项检查，publication-panel两个head＋差异＋显式reauth/activate，coverage-panel节点分页（发布后台与复核页分别按admin/reviewer权限调用）。pending-command保留{key,route,input,status}并仅手动同输入重试。

- [ ] **Step 1：写失败测试。** DraftEditor显示三家族/精确参数/主知识目标索引及中英文目标、保存冲突保留输入、旧校验摘要不能送新revision；ReviewPanel展示全部source/authors/legacy/engine/约束排除＋所有实例页，不以首屏代替整批，六项检查与generation说明必需；PublicationPanel head跨页单取、差异分页/失效准备/重新验证不丢键；WithdrawalPanel解释影响/不足五题/分页摘要变化重预览；QuestionPendingCommand只主动同键重试，取消/未确认结果保留输入、不自动假成功；角色导航admin无editor/reviewer入口。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"DraftEditor":{"staleDigestSubmit":"blocked","conflictDropsInput":false}},
  {"ReviewPanel":{"checks":6,"allInstancePages":"available","mutableWorkspaceUsed":false}},
  {"PublicationPanel":{"bothHeadsRequired":true,"headCrossPage":"read_by_id"}},
  {"QuestionPendingCommand":{"retry":"manual_same_key_same_input","cancelMeansFailure":false}}
]
~~~

- [ ] **Step 2：验证RED。** node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/question src/components/auth-status.test.tsx，预期新页面行为失败。
- [ ] **Step 3：实现七页及组件。** 复用SafeMarkdown/公式/现有SHA素材接口与键盘对话框，按真实permission决定入口，加载/空状态/未配置/未就绪/来源未核实英文明确。生成结果与正式批准数量分开；可按页完整阅读，不提供“全部已看”虚假标记。取消仅结束等待，明确写入结果可能未确认，主动同键重试固定输入。
- [ ] **Step 4：验证GREEN。** 重跑Step2；单独npm run typecheck/build和现有reading/content/auth组件回归。1280×900与390×844检查长ID、公式、键盘焦点/对话框、无横向页面溢出；不新增产品依赖或复制外部图片。
- [ ] **Step 5：提交。** 任务12文件，提交 feat: 增加可信题库英文管理后台。

## Task 13：真实联调、性能预算、兼容验收与独立审查

**Files:** e2etest/question_fixture.go/question_control.go及测试、harness.go；tests/e2e/question-helpers.ts与四个question-*.spec.ts；现有backend.yml/frontend.yml；question-bank.md/2026-10-02-p4a-acceptance.md、README/路线图。
**Interfaces:** Fixture只在随机库构造原创公开K/模板和真实不同账户；控制端口只有环回、随机能力令牌，生产server不能导入e2etest。浏览器走真实Next.js→Go→PostgreSQL，不伪造成功业务API。沿用harness安全启动/清理协议，不返回可供生产使用的批准。

- [ ] **Step 1：写失败联调。** question-authoring：真实创建→保存→生成校验→送审、错答案/范围/冲突、CLI草稿认领；question-review：两个实际账户、作者自审拒绝、冻结完整分页、六项批准/退回/修订；question-release：真实prepare→reauth→activate→coverage→withdraw、两个head陈旧/模板替换/单实例撤回；question-security：learner/private404、缺迁移、取消/超时/同键手动重试、role撤回与共享槽/预算；分别运行两viewports，预期缺少题库装配行为失败。

断言数据（测试须逐项断言；下列对象不作为产品配置）：

~~~json
[
  {"question-authoring":{"realStack":true,"sourceRole":"editor"}},
  {"question-review":{"distinctReviewer":true,"selfApproval":"reject"}},
  {"question-release":{"realReauth":true,"staleEitherHead":"reject","withdrawal":"atomic"}},
  {"question-security":{"fakeSuccessAPI":false,"viewports":["1280x900","390x844"],"workers":1,"retries":0,"singleBatchMaxSeconds":540}}
]
~~~

- [ ] **Step 2：运行RED。** 构建隔离harness和frontend后，每场景各一次：node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-authoring.spec.ts（review/release/security分别替换）；基础环境故障先定位，不能当业务RED。
- [ ] **Step 3：补齐真实装配与CI。** harness同一个publication.Service注入题库双槽，用testutil每批新随机库；API错误/审计/manifest从真实DB断言。backend CI加入question纯逻辑与store题库测试，frontend CI新增四个独立限时场景批次；保留原所有测试、540s入口与retries=0。容量测试记录最大合法生成、送审和200/10000/1000候选的真实耗时、SQL查询计划、累计分配/最大驻留，必须在原8秒/字节预算内；失败先优化，不延长时限或缩小案例假通过。
- [ ] **Step 4：回归GREEN并形成证据。** 各自入口go vet、go test question/auth/content/publication/httpapi/e2etest/testutil、store/cli分批、go build ./cmd/...；npm typecheck/test/build/audit、两次api:generate零差异；全部原公开/账户/内容浏览器批次＋四题库批次。留存最大容量数据、原schema/导出SHA/旧迁移零差异、随机库清理、原开发库无迁移。验收文件逐项对照设计15节12组；代码/技术fixture通过不计为20批准模板或300生产实例。
- [ ] **Step 5：独立整分支审查与交付。** 按Native约定调用requesting-code-review，使用新上下文的独立审查覆盖本计划Review Focus、真实独立校验器、冻结作者/关联、两head/锁、替换与永久事实、错误/分页泄漏、最大预算和旧兼容。必要修复先写真实RED再GREEN；保存所有裁定/修复及验证证据，不以旧CI代替最新提交。通过SSH推送、创建并附加实现PR，等待最新SHA四项CI全部完成，按用户授权处理合并，不自动部署。

## 计划自查与可行性门槛

| 已确认设计章节 | 对应任务与验收 |
| --- | --- |
| 1—3范围/分期/架构 | 全局约束、文件/依赖图；任务13不把P4a技术完成算P4b或P6生产数量 |
| 4身份/旧契约/知识目标 | 任务1/4/5/6；golden SHA、qi-ID、固定引用、旧版本/FK/导出回归 |
| 5判分 | 任务2数值格式/精确比较/错误分类，10/11禁止自由判分端口 |
| 6完整生成/独立校验 | 任务3/4所有组合、错答案注入、固定题见证、包预算 |
| 7冻结/独立复核 | 任务5/6/7：作者继承、CLI无批准、冻结子表、全部实例分页、六检查 |
| 8发布/可用/历史/撤回 | 任务8/9：两head、批准资格、替换蓝图、黑名单、事实分离 |
| 9覆盖/P4b边界 | 任务4/9/13：五题位集合、不同身份、动态有效池；未来曝光/尝试/资格端口未实现 |
| 10事务/数据库 | 任务5及6—9：18表、FK/延迟约束、锁/DB时间/幂等/审计 |
| 11端口/具名DTO/英文页 | 完整契约表、任务10/11/12；私有404、严格原始JSON、分页与错误 |
| 12资源/截止 | 任务3/4/8/10/13；共享槽和scope、所有字节/数量限额、最大合法实测 |
| 13—16文件/兼容/验收/书面门槛 | 文件责任表、任务13、自查；仅方案已确认，执行计划待审阅 |

自查已覆盖契约名/类型/参数/返回值、任务消费产出、规范与冻结摘要、分页资源和具体失败测试；没有产品占位方法或开放式“稍后决定”。提交前再次检查全部引用文件、接口名称一致、旧端口/schema差异及文档链接。静态无阻塞；最大合法载荷8秒性能、真实并发、独立校验及全回归是实施中的硬验收，不能用计划自查替代。

执行前在干净隔离工作区SSH获取最新master、新建codex/p4a-question-bank分支（名称已存在时核实用途，不覆盖）；先复核基线CI、执行方案和本计划。保留Native执行方式，用户确认本计划覆盖需求后再开始Task1。中途需要修改已批准的兼容性/家族/限额/资格边界时先提交可审阅设计，不静默扩大范围。
