# P4b 学习、检测与知识解锁 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.
>
> 沿用用户此前选定的 Native：superpowers:executing-plans 在当前会话逐项实现，最后一次独立整分支审查。用户已书面确认本计划，文档 PR #18 已合并；从最新 master 1b002b8608aa87f0dfb73a442a75d14f24302400 新建 codex/p4b-learning-assessment。Task1—12 已逐项完成，Task13 验收进行中；独立整分支审查、实现 PR 和 CI 交付尚未完成。

**Goal:** 交付真实的阅读动作、安全练习、五题检测、诊断、资格解锁及固定路线进度，让用户准确看到已学知识并可回顾。

**Architecture:** assessment 定义安全题面、固定尝试、覆盖选择及精确判分，依赖 question，不能导入 learning 或 store。learning 依赖 assessment 的事实类型，定义状态、资格、权限和统一学习服务；store 实现二者的事务与有界查询，HTTP/Next.js 只传递动作和展示安全结果。所有答案交付先记录曝光，资格及新解锁在同一数据库事务内核验准确版本与永久撤回事实。

**Tech Stack:** Go 1.27.1、PostgreSQL 17.11、Node.js 24.17.0、Next.js 16.3.7、React 19.3.0、TypeScript 5.9.3；复用 Zod 4.6.5、Vitest 5.0.3、Playwright 1.63.0 与当前锁定依赖，不新增产品依赖。

**Spec:** [已确认 P4b 方案](../specs/2026-10-02-learning-assessment-design.md)。用户于2026-10-02确认功能设计及第14节兼容性变化，并授权合并 P4a [PR #17](https://github.com/yyl1212/math_master/pull/17)；合并基线 master 46206fc31e05651dd97c49390e4fb3bee76988ed。本文与设计共用文档 [PR #18](https://github.com/yyl1212/math_master/pull/18)，产品实现必须另建分支。

## Global Constraints

- 文档中文，产品英文为主并保留中文术语；六类学习者、16学习板块、数据正确性与路线优先沿用总体方案；不批准正式数学内容，不改写 Knowledge_JSON。
- 已发布讲解可浏览；主动开始、明确完成、检测、资格、历史解锁是不同事实。检测不制造阅读完成，滚动与停留不制造学习时长。
- 五个不同有效实例、全部核心目标、至少4/5正确；ruleVersion=1；最多1000候选、8核心目标；只用实际发布的有限实例，禁止动态生成未审核题。
- 三十分钟曝光按服务器数据库时间；最近一次账户全局已提交检测的五题永久排除到另一检测提交，含 node/diagnostic/review 及 affected。
- 每账户一个 active 检测和一个 active 练习；24小时有效区间为 [createdAt,expiresAt)；练习只接受一次有效作答，revealed 无分数、无资格。
- 普通推进须所有准确前置版本“明确完成＋有效通过”，有效诊断可替代；只计算当前直接后继，不遍历全部历史路线或伪造祖先资格。
- 普通题库替换保留正确历史证据；当前知识新版要求新证据；永久撤回实时限制。历史稳定ID解锁永久保留，路线版本、顺序与分母固定。
- 新迁移00006显式启用，旧00001—00005及数学契约/摘要用途不变；未配置时原P4a可用，全部学习接口503 LEARNING_NOT_CONFIGURED，启动不自动迁移。
- 新Go总期限8秒、Next总期限10秒、锁等待1秒；原账户4/5秒和服务器15秒不变；JSON响应完整包装≤4194304 bytes，请求≤8192 bytes。
- 数值输入复用P4a：UTF-8有效、trim前≤128 Unicode字符、ASCII整数/小数/分数或百分比语法、规范分子/分母≤256位；不使用浮点数。
- 分页默认20、最多100、offset最多100000；素材复用旧MIME/SVG/字节限制和响应头。公开响应不加个人字段，所有个人读取 private/no-store。
- 重选题共用原2个实际validation槽及 content_heavy 40/10每分钟；写共用 content_write 120/30；读共用 auth_read 600/content_read_user 120；不增加并发池或第二配额。
- 锁顺序：共享1296127049→共享1296127048→UUID排序用户行→操作者会话→曝光状态→尝试→学习行；原管理写保留排他锁。
- 写入要求当前learner角色、会话、CSRF及UUID v4键；拥有者来自服务端。editor/reviewer/admin不自动越权，同键重试检查当前权限。
- 草稿答案仅页面内存；无服务器自动保存、无localStorage答案；取消不保证回滚，无自动写重试，同键重试保留确切输入。
- 测试仅 testutil 随机库和合成原创内容；不用真实DATABASE_URL、服务器、人员账户或持续更新的源资料。CGO_ENABLED=0、GOTOOLCHAIN=go1.27.1。
- 每批命令用 tools/verify/run.mjs 最多540秒；Go -timeout 5m；浏览器 workers=1、retries=0、globalTimeout=480000、1280×900及390×844。
- P5反馈/批量重评/通知、P6正式内容批准和P7生产部署保持原分期；任务1—13完成后才进入独立审查、SSH推送、实现PR和最新SHA CI门槛。

## Review Focus

1. 同用户先创建检测再通过其他角色或同键重放取答案：曝光先提交、提交受影响且不授予资格；任务4/7测试 LearningExposureActiveAttempt、AssessmentExposureAfterCreation。
2. 请求在锁等待中跨过到期、撤权、密码变化或取消：提交前重新证明，完整回滚且不泄漏局部分数；任务2/7测试 LearningCommitProof、AssessmentExpiryAndRace。
3. 最新通过证据撤回但另有有效通过，或旧题库被普通替换：使用替代证据、保留历史，准确新版不继承；任务5/8测试 LearningAlternativeEvidence、LearningHistoricalProjection。
4. 当前稀疏候选和模板曝光使静态ready失真：排除后重新覆盖，优先未见，retryAt只能表示真实恢复；任务1/3/7测试 LearningSelection、LearningSourcePool、AssessmentNotReady。
5. 断网后命令已提交、用户换账号或撤回答案后重放：业务身份/状态幂等，展示按当前限制，旧内存不串用户；任务8/10/12测试 LearningReplayProjection、LearningClientPending、AssessmentUnconfirmed。

---

## 文件责任与任务依赖

| 任务 | 新增/修改的主要文件 | 唯一责任 |
| --- | --- | --- |
| 1 | backend/internal/assessment/{model,projection,selection,grade,digest}.go；backend/internal/learning/{model,state,policy,digest}.go | 纯类型、状态、判分、白名单与覆盖选择 |
| 2 | db/migrations/00006_learning_assessment.sql；backend/internal/store/learning_{tx,idempotency}.go | 16实体、不变量、共享锁、配置与事务 |
| 3 | backend/internal/store/assessment_{sources,evidence}.go | 定向选题、真实发布依据与固定历史正文 |
| 4 | backend/internal/store/exposure_{record,question}.go；既有question事务/命令/读取文件 | 全部管理答案交付及曝光序号 |
| 5 | backend/internal/store/learning_{actions,qualification,paths}.go | 阅读、路线加入、资格与历史授予 |
| 6—7 | backend/internal/store/practice_{commands,read}.go、assessment_{commands,read}.go | 练习和正式检测状态机 |
| 8 | backend/internal/store/learning_{read,history,assets}.go | 个人投影、固定进度、安全回顾与素材 |
| 9 | backend/internal/learning/{repository,service}.go；backend/internal/httpapi/learning_{dispatch,routes,json,error}.go；api/openapi.yaml | 服务编排及严格Go私有接口 |
| 10 | frontend/src/lib/learning/；frontend/src/lib/api/learning-proxy.ts；新learning API route | 闭合TS契约、原始字节代理和私有SSR |
| 11—12 | frontend/src/features/{learning,practice,assessment}/；五个新页面及既有地图/阅读/导航 | 真实英文学习、作答、进度与回顾 |
| 13 | backend/internal/e2etest/；tests/e2e/learning-*.spec.ts；.github/workflows/；docs/operations/ | 真实全链路、容量、兼容性与验收 |

~~~mermaid
flowchart LR
    T1[1 纯契约] --> T2[2 数据与事务]
    T2 --> T3[3 定向题源]
    T3 --> T4[4 全入口曝光]
    T4 --> T5[5 学习与资格]
    T5 --> T6[6 练习]
    T6 --> T7[7 正式检测]
    T7 --> T8[8 私有读取]
    T8 --> T9[9 Go接口与夹具]
    T9 --> T10[10 TS与代理]
    T10 --> T11[11 学习页面]
    T11 --> T12[12 作答页面]
    T12 --> T13[13 全回归与容量]
    T13 --> Review[一次独立整分支审查]
    Review --> PR[SSH推送 实现PR 最新SHA CI]
~~~

### 跨任务契约：以这里的名字为准

assessment 不暴露私有题库DTO；Go JSON均lowerCamelCase，[]不返回null。下列public类型的全部字段在OpenAPI/Zod具名闭合定义，internal类型不进入HTTP。Identity、Choice、AssetRef、Rational和Access分别复用 question 的同名类型。

| 类型/归属 | 精确字段与值 |
| --- | --- |
| assessment.Mode / AttemptState / Outcome | Mode=node/diagnostic/review；State=active/submitted/abandoned/expired；Outcome=passed/failed/affected，未提交为null |
| assessment.PracticeState / Validity | Practice=active/answered/revealed/abandoned/expired；Validity=effective/restricted/stale，不混入固定Outcome |
| assessment.SafeQuestion | Position int；Instance、Knowledge Identity；Type choice/numeric；Prompt string；Choices []Choice；AnswerFormat *string(rational/percentage，仅numeric)；Assets []AssetRef。不含Template、参数、正确答案、解析、见证、作者或sourceMap |
| assessment.Answer / SubmitInput | Answer闭合联合：kind=choice＋choiceId；numeric＋raw；skipped无附加值。SubmitInput={answers:[]PositionAnswer}；PositionAnswer={position:int,instance:Identity,answer:Answer}，必须五个准确位置与身份，原raw保留 |
| assessment.CreateInput / PracticeCreateInput | Create={knowledge:Identity,blueprint:Identity,mode:Mode,expectedKnowledgeHead:string,expectedQuestionHead:string}；PracticeCreate同结构但无blueprint/mode |
| assessment.AttemptSummary | ID string；Kind practice/assessment；Knowledge Identity；Mode *Mode；State string(按Kind判别)；CreatedAt/ExpiresAt time.Time；SubmittedAt *time.Time |
| assessment.AttemptView / PracticeView | Attempt={summary:AttemptSummary,questions:[]SafeQuestion}；Practice={summary:AttemptSummary,question:SafeQuestion,result:*PracticeResult}；active/abandoned/expired无答案结果 |
| assessment.ResultItem | Question SafeQuestion；Answer *Answer；Correct *bool；CorrectChoiceID *string；CorrectNumeric *Rational；Explanation *string；Validity Validity；Reasons []RestrictionReason。restricted项的正确性/答案/解析为null，保留本人原输入 |
| assessment.ResultView / PracticeResult | Result={summary,ruleVersion:int,score:*int,passed:*bool,outcome:Outcome,validity:Validity,reasons:[],items:[]ResultItem,progress:ProgressUpdate}；PracticeResult={outcome:answered/revealed,item:ResultItem}，revealed的Answer/Correct为null；正式结果的Answer非null |
| assessment.ProgressUpdate / AttemptFact (internal) | ProgressUpdate={knowledge:Identity,qualificationGranted:bool,newlyUnlocked:[]Identity}；AttemptFact={ID:string,Knowledge:Identity,Mode:Mode,Outcome:Outcome,Score:*int,Passed:*bool,SubmittedAt:time.Time,Validity:Validity}。learning消费该事实，不反向引用learning类型 |
| assessment.Candidate / SourcePool (internal) | Candidate={Identity,Template:*Identity,Coverage:[]int,Seen:bool,LastSeenAt:*time.Time,ExposedAt:*time.Time,RecentSubmitted:bool}；SourcePool={Knowledge:Identity,KnowledgeHead:string,QuestionHead:string,Blueprint:*question.Blueprint,BlueprintIdentity:*Identity,Candidates:[]Candidate} |
| assessment.Seal / ItemBinding (internal) | Seal={Kind,Knowledge,Mode:*Mode,Blueprint:*Identity,KnowledgePublicationID,QuestionPublicationID:string,RuleVersion:int,Core:[]int,Seed:string,Items:[]ItemBinding}；ItemBinding={Position,Instance,Template:*Identity,Coverage:[]int,Units:[]Identity,Assets:[]AssetRef,Approval:question.MemberEvidence} |
| learning.State / KnowledgeState | State=unlearned/learning/learned/needs-review/mastered；KnowledgeState={knowledge:Identity,title/titleZh:string,state:State,canEnter:bool,everUnlocked:bool,completionValid:bool,startedAt/completedAt:*time.Time,qualification:*QualificationView,prerequisites:[]PrerequisiteState} |
| learning.QualificationView / PrerequisiteState | Qualification={knowledge:Identity,kind:normal/diagnostic,evidenceAttemptId:string,completedEventId:*string,validity:Validity}；Prerequisite={knowledge:Identity,qualified:bool,reasons:[]RestrictionReason} |
| learning.KnowledgeDetail / BlueprintOption | Detail={knowledgeHead:string,questionHead:*string,state:KnowledgeState,objectives:[]string,blueprints:[]BlueprintOption,activeAssessment:*AttemptSummary}；Option={blueprint:Identity,coreObjectiveIndices:[]int,ready:bool,reasons:[]ReadinessReason,retryAt:*time.Time}。目标文本共享objectives，不重复1000份；未知目标索引拒绝 |
| learning.StartInput / CompleteInput / EnrollInput | Start/Complete={knowledge:Identity,expectedKnowledgeHead:string}；Enroll={path:Identity,expectedKnowledgeHead:string}。URI ID须等于正文稳定ID，无client userId |
| learning.PathSummary / PathNode / PathView | Summary={ID:string,path:Identity,title/titleZh:string,knowledgePublicationId:string,totalNodes/completedNodes/passedNodes/unlockedNodes:int,newVersionAvailable:bool,createdAt:time.Time}；Node={position:int,title/titleZh:string,state:KnowledgeState,available:bool,reasons:[]RestrictionReason}；View={summary:PathSummary}，节点另分页 |
| learning.Overview / HistoryEntry / ListQuery | Overview={knowledgeHead/questionHead:*string,availablePaths:[]PublishedPathSummary,startedCount/completedCount/effectivePassedCount/historicalUnlockedCount:int,activePractice/activeAssessment:*AttemptSummary,recent:[]HistoryEntry}，recent≤20；History={ID:string,kind:learning/practice/assessment,path:*Identity,knowledge:Identity,occurredAt:time.Time,state:string(learning为started/completed，practice/assessment用对应State枚举),validity:Validity,attemptId:*string}；ListQuery={Limit/Offset:int} |
| learning.EvidenceFacts / StateFacts (internal) | EvidenceFacts={Knowledge:Identity,Current:bool,CompletionEventID:*string,Passes:[]AttemptFact}；StateFacts={Started/Completed/HadInvalidatedPass:bool,EffectivePass:bool,LatestReviewFailed:bool}；EvidenceView={Qualified:bool,Qualification:*QualificationView} |
| learning.PublishedPathSummary | Path Identity；Title/TitleZh string；TotalNodes int。Overview返回当前发布路线≤200条，供原公开路线页取得准确SHA及加入所需head；空内容head为null |
| learning.EventSeal (internal) | ID/ActorID string（仅服务端）；Knowledge Identity；KnowledgePublicationID string；Units []Identity；Assets []AssetRef；Kind=started/completed；RecordedAt time.Time，首次事件固定同一数学引用与时刻 |
| learning.EvidenceDependency (internal) | Kind=knowledge/unit/asset/template/instance/blueprint；ID string；Version *int（asset为null，其他正整数）；SHA256 string。与ExposureRef的两种身份分开 |
| learning.Receipt / ExposureRef (internal) | Receipt={ResourceKind:knowledge/enrollment/practice/assessment,ResourceID:string,Status:int}，不保存可绕过撤回的答案响应；ExposureRef={Kind:instance/template,Identity:Identity}，每用户/准确数学身份单向更新时刻与序号 |
| learning.Action | readOverview/listKnowledge/readKnowledge/startKnowledge/completeKnowledge/enrollPath/listPaths/readPath/listPathNodes/createPractice/readPractice/answerPractice/revealPractice/abandonPractice/createAssessment/readAssessment/submitAssessment/abandonAssessment/readAssessmentResult/listHistory/readAsset |
| assessment.RestrictionReason / learning.ReadinessReason / learning.ErrorCode | Restriction=knowledge-updated/knowledge-withdrawn/unit-withdrawn/asset-withdrawn/template-withdrawn/instance-withdrawn/blueprint-withdrawn/exposed-after-creation；Readiness=no-blueprint/no-instances/insufficient-coverage/exposure-cooldown/recent-assessment-exclusion；ErrorCode包括设计§12八个新代码及原认证/CSRF/限流/幂等/400/413/404/503公共错误，不能省略枚举 |

所有时间RFC3339、UUID v4小写、SHA小写64hex；knowledgeHead是原publication snapshot ID，questionHead是题库publication UUID，准确发布成员与批准记录均需核验。ResultView.progress是原授予事实，不能替代当前qualification；版本/限制以当前投影为准。completedAt保留首次明确完成事实，completionValid单独表示当前是否有未受限制的同版本完成证据，ResolveState的Completed使用后者。Create/Enroll/Start成功201，Complete/answer/reveal/submit/abandon成功200，重放保存原状态；重复开始保留首次事件但仍返回该动作原状态。空动作请求是严格{}。Practice作答仅允许choice/numeric，具名PracticeAnswerInput闭合oneOf排除skipped；Formal SubmitInput才允许skipped。错误新增独立LearningError DTO：code/message/requestId、可选retryAfter、retryAt、activeAttempt、formatCode；详细信息仅在对应错误分支存在，绝不含正确性。

任务1在learning/model.go定义sentinel：ErrNotConfigured→503 LEARNING_NOT_CONFIGURED；ErrVersionStale→409 LEARNING_VERSION_STALE；ErrPrerequisitesUnmet→409 LEARNING_PREREQUISITES_UNMET；ErrAssessmentNotReady→409 ASSESSMENT_NOT_READY；ErrAssessmentActive→409 ASSESSMENT_ACTIVE（practice也用此码，activeAttempt.Kind区分）；ErrAssessmentExpired→409 ASSESSMENT_EXPIRED；ErrStateConflict→409 ASSESSMENT_STATE_CONFLICT；ErrAnswerFormatInvalid→400 ANSWER_FORMAT_INVALID。同键冲突复用question.ErrIdempotencyConflict。

Repository采用统一learning.Service外观，函数在各任务定义；下面Go签名中的ctx为context.Context、tx为*sql.Tx、a为question.Access、q为learning.ListQuery，带明示类型的参数以明示类型为准；分页返回 question.Page[T]。只读GET不创建学习/解锁事实；答案结果GET进入可写曝光事务。Seal只存不可变数学引用、审核依据和规则，不复制整份私有题包；载入时逐SHA核验原不可变行。CanonicalSeal的用途为 practice-attempt-v1/assessment-attempt-v1，新学习事件为 learning-event-v1，整个用途包装≤4194304 bytes；不修改P4a规范摘要。

### Task 1: 纯契约、白名单、精确判分与覆盖选择

**Files:** Create assessment/model.go、projection.go、selection.go、grade.go、digest.go；learning/model.go、state.go、policy.go、digest.go（均在backend/internal/）；Test 对应model_test.go、projection_test.go、selection_test.go、grade_test.go、digest_test.go、state_test.go、policy_test.go。

**Interfaces:** Consumes question.GradeChoice([]Choice,string,string)、GradeNumeric(Rational,string,string)→(GradeResult,error)。Produces ProjectQuestion(position int,i question.Instance) (SafeQuestion,error)；ValidateAnswers(items []question.Instance,in SubmitInput) error；GradeFive(items []question.Instance,in SubmitInput) (int,bool,error)；SelectFive(ctx context.Context,core []int,candidates []Candidate,seed [32]byte) ([]question.Identity,bool,error)；EarliestReady(ctx context.Context,core []int,candidates []Candidate,now time.Time,seed [32]byte) (*time.Time,error)；CanonicalSeal(Seal) ([]byte,string,error)；learning.CanonicalLearningEvent(EventSeal) ([]byte,string,error)；learning.ResolveState(StateFacts) State；EvaluateEvidence(EvidenceFacts) EvidenceView；Authorize(auth.User,Action) error；IsRead(Action) bool；IsHeavy(Action) bool；IsIdempotent(Action) bool；Rates(actor string,action Action) ([]auth.RateKey,error)。

- [x] **Step 1: 写纯函数失败测试。** 测试夹具本任务定义为五个不同choice实例，每题选项a/b、正确a；另用1/2数值题。固定断言如下：
~~~go
func TestLearningGradeFourOfFive(t *testing.T) { // fiveChoiceItems/allChoiceAnswers在此测试文件定义
  score, passed, err := GradeFive(fiveChoiceItems(), allChoiceAnswers("a","a","a","a","b"))
  if err != nil || score != 4 || !passed { t.Fatal(score, passed, err) }
}
func TestLearningStateReviewPrecedence(t *testing.T) {
  if ResolveState(StateFacts{Completed:true, EffectivePass:true, LatestReviewFailed:true}) != "needs-review" { t.Fatal("state") }
}
~~~
增加具名表测：LearningProjection禁止全部私有字段；LearningGrade三题false、五题true、skipped计错、.5/2/4/50%精确、128/129字符、未知choice整体报错；LearningSelection以6候选证明优先未见且五题覆盖、同seed同序、重复身份拒绝、1000/1001候选与8/9核心边界。LearningRetryAt在曝光恰30分钟时可选、最近检测排除无法恢复时nil；LearningSeal用途/SHA/完整包装±1 byte；LearningPolicy完整21动作与角色/旧配额。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/assessment ./internal/learning -timeout 5m -count=1。Expected: 新接口缺失或上述特定断言失败。
- [x] **Step 3: 实现上述纯接口。** projection逐字段构造；先ValidateAnswers全部五项再GradeFive。selection以6×256状态DP保存最佳五项：先最大化未见数，再按见题历史和SHA256(seed＋准确身份)的固定排序决胜，不调用会重新按ID排序的P4a FiveQuestionCover。EarliestReady排除最近检测后，按曝光到期时刻二分可行性，同一覆盖算法复验，不逐毫秒/逐题展开整库。policy所有动作仅learner；createPractice/createAssessment为heavy，其余读/写按全局约束。
- [x] **Step 4: 确认GREEN。** 同Step2命令，Expected: 两包PASS；安全题面JSON无私有字段，所有精确边界PASS。
- [x] **Step 5: 提交。** Run: git add backend/internal/assessment backend/internal/learning；git commit -m "feat: define learning contracts and safe assessment rules"。Expected: 仅本任务文件提交。

### Task 2: 迁移、共享事务、配置与业务幂等

**Files:** Create db/migrations/00006_learning_assessment.sql；backend/internal/store/learning_tx.go、learning_idempotency.go；Test learning_schema_test.go、learning_tx_test.go、learning_idempotency_test.go、learning_fixture_test.go（store目录）。

**Interfaces:** Consumes task1 learning.Authorize/Rates/Receipt/Seal。Produces Store.LearningPreflight(ctx context.Context,a question.Access,action learning.Action) (auth.User,error)；Store.learningTx(ctx context.Context,a question.Access,action learning.Action,fn func(context.Context,*sql.Tx,auth.User,time.Time) error) error；learningConfigured(ctx context.Context,tx *sql.Tx) (bool,error)；learningReplay(ctx context.Context,tx *sql.Tx,actor,action,target,key,digest string) (learning.Receipt,bool,error)；learningRemember(ctx context.Context,tx *sql.Tx,actor,action,target,key,digest string,receipt learning.Receipt) error；newLearningFixture(t *testing.T) *learningFixture（仅测试）。

- [x] **Step 1: 写真实PG失败测试。** newLearningFixture扩展newQuestionFixture，在发布前将questionInput替换为一个lf-addition模板（left/right各为1、2、3、4，add、rational、无约束，全部关联workflow-fractions@1/core索引0）与一个指向该模板的五题四题蓝图，产生16个不同有限实例；实际QApproved→QPrepare→QActivate；另外两个signup账户learner_a/b只保留learner。提供准确identity/head、原db/repo/ctx/Access；需要多节点时用既有内容/题库送审发布API合成lf-root→lf-middle→lf-target，不能调用尚未实现学习命令预置通过。
~~~go
func TestLearningSchemaActiveUnique(t *testing.T) {
  f := newLearningFixture(t)
  // 插入两个相同owner的active assessment，使用真实已发布引用与合法seal。
  if f.insertTwoActiveAssessments() == nil { t.Fatal("must reject second active") }
}
func TestLearningSchemaTableCount(t *testing.T) {
  f := newLearningFixture(t)
  if f.learningTableCount() != 16 { t.Fatal("expected 16 learning tables") }
}
~~~
上述SQL测试helper在learning_fixture_test.go本任务定义。覆盖五项不足/位置重复/owner外键、seal bytes/JSON/SHA篡改、终态不可逆、结果nullable组合、学习/路线事件不可变、曝光序号倒退。LearningCommitProof覆盖锁后撤权、会话到期、密码变化、取消及最后幂等写失败零残留；LearningSharedLocks证明不同账户并行、管理排他竞争与固定锁序。旧迁移库原P4a成功、学习503；缺任一表不得部分启用。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearning(Schema|CommitProof|SharedLocks|Configured|Idempotency)' -timeout 5m -count=1。Expected: 新表/接口缺失，或SQL不变量未拒绝。
- [x] **Step 3: 实现迁移及上述事务接口。** 新16表严格等于设计§10；learningConfigured只有00006未曾启用且16表全不存在才返回(false,nil)；goose已标记00006或任一学习表存在但不完整时返回ErrNotConfigured，已启用系统不得静默回退为不记录曝光的旧模式；FK使用原(id,version,sha256)可用唯一键；延迟触发器证明active五个items、submitted五个answers＋唯一result、固定publication成员/approval、不可变事件与终态。首次start/completion时刻及原事件不被再次动作覆盖；完成依据更新后的显式动作可以追加证据，当前资格不用首次事件指针替代有效性核验。READ COMMITTED，先两把shared advisory lock，再managedIdentity及用户行；读取clock_timestamp，提交前重证当前会话/角色/CSRF。幂等保存动作/目标/输入摘要及Receipt，不保存答案DTO；同键不同输入冲突，旧expired键仍指向旧资源，必须新键创建。曝光身份允许在对应数学版本发布前存在，只FK拥有者；选题依准确ID/version/SHA匹配，不以同名草稿或不精确FK污染当前题源。
- [x] **Step 4: 确认GREEN。** 同Step2命令；另Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestQuestion(Schema|Tx|Idempotency)' -timeout 5m -count=1。Expected: 新/旧真实PG断言PASS，旧迁移文件零修改。
- [x] **Step 5: 提交。** Run: git add db/migrations/00006_learning_assessment.sql backend/internal/store/learning*；git commit -m "feat: persist learning invariants and shared transactions"。

### Task 3: 定向候选、实际发布依据与固定历史题源

**Files:** Create backend/internal/store/assessment_sources.go、assessment_evidence.go；Test assessment_sources_test.go、assessment_evidence_test.go（store_test），assessment_sources_internal_test.go（store的测试专用adapter）。

**Interfaces:** Consumes task1 SourcePool/Candidate/Seal与task2 tx/fixture。Produces learningSourcePool(ctx context.Context,tx *sql.Tx,actor string,k question.Identity,bp *question.Identity,now time.Time) (assessment.SourcePool,error)；learningLoadItems(ctx context.Context,tx *sql.Tx,seal assessment.Seal) ([]question.Instance,error)；learningEvidenceRestrictions(ctx context.Context,tx *sql.Tx,dependencies []learning.EvidenceDependency) ([]assessment.RestrictionReason,error)。Test-only adapter Store.LearningSourcePoolForTest(ctx context.Context,a question.Access,k question.Identity,bp *question.Identity) (assessment.SourcePool,error) 在assessment_sources_internal_test.go定义并包装合法learningTx，不进入生产构建。

- [x] **Step 1: 写题源失败测试。**
~~~go
func TestLearningSourcePoolExactVersion(t *testing.T) {
  f := newLearningFixture(t)
  pool, err := f.repo.LearningSourcePoolForTest(f.ctx,f.Access("learner_a",false),f.knowledge,&f.blueprint)
  if err != nil || len(pool.Candidates) != 16 || pool.Knowledge != f.knowledge { t.Fatal(pool, err) }
}
~~~
测试专用adapter属package store，夹具与调用测试属store_test；外部测试仅调用该exported test-only方法，避免混用两个测试包。LearningSourcePool同时测错误SHA/版本、无蓝图练习、多个蓝图、准确core索引、最近检测按账户全局排除、template曝光覆盖16个有限实例、同ID不同正文不匹配；LearningHistoricalSource普通替换后固定五题可加载，永久撤回返回限制，不能用prepared或伪造publication。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearning(SourcePool|HistoricalSource)' -timeout 5m -count=1。Expected: 定向接口未实现或错误来源被接受。
- [x] **Step 3: 实现上述题源接口。** 从当前两head、成员/来源/coverage/restriction索引按准确knowledge/blueprint一次批取≤1000身份与覆盖；关联用户曝光/见题/最近全局检测，先不载私有body。选中后仅批载1/5实例及真实review批准，逐SHA复验，禁止questionOfferable逐题调用或读全部manifest。新增覆盖/反向依赖索引只写00006。固定历史与当前可选题源分开，ordinary replacement不作为永久撤回。
- [x] **Step 4: 确认GREEN。** 同Step2命令；Expected: pool定向/历史断言PASS；测试统计候选正文载入为0，选中后恰1/5，不随整个题库增长。
- [x] **Step 5: 提交。** Run: git add backend/internal/store/assessment_sources* backend/internal/store/assessment_evidence* db/migrations/00006_learning_assessment.sql；git commit -m "feat: query bounded published learning question sources"。

### Task 4: 所有原P4a答案交付的事务曝光

**Files:** Create backend/internal/store/exposure_record.go、exposure_question.go；Modify question_tx.go、question_draft.go、question_submission.go、question_review.go、question_idempotency.go；Test exposure_record_test.go、exposure_question_test.go、question_exports_test.go（新）。

**Interfaces:** Consumes task2 learningConfigured及原questionTx/managedIdentity，task3准确数学身份。Produces learningRecordExposure(ctx context.Context,tx *sql.Tx,actor string,refs []learning.ExposureRef,now time.Time) (int64,error)；learningExposureSince(ctx context.Context,tx *sql.Tx,actor string,watermark int64,items []assessment.ItemBinding) (bool,error)；questionExposureRefs(input question.DraftInput,instances []question.Instance) ([]learning.ExposureRef,error)；questionExposeResponse(ctx context.Context,tx *sql.Tx,u auth.User,response any,now time.Time) error；Store.questionAnswerReadTx(ctx context.Context,a question.Access,action question.Action,fn func(context.Context,*sql.Tx,auth.User,time.Time) error) error。

- [x] **Step 1: 写逐入口失败测试。**
~~~go
func TestLearningExposureActiveAttempt(t *testing.T) {
  f := newLearningFixture(t)
  before := f.exposureSequence("author_a")
  _, err := f.repo.ReadQuestionSubmission(f.ctx, f.Access("author_a",false), f.approvedSubmissionID)
  if err != nil || f.exposureSequence("author_a") <= before { t.Fatal("answer delivery must advance sequence", err) }
}
~~~
ExposureQuestion入口表逐项覆盖read/create/save/adopt/revise draft、submit/read/decide submission、instances分页、validate生成预览及同键含答案重放；导出可编辑题包经过ReadQuestionSubmission返回冻结正文也要记录。摘要/list成员/coverage无答案不记录；曝光写失败不能返回正文、响应取消仍保守记录、发布前准确草稿预览在后续同SHA发布后仍排除、同名不同SHA不排除、模板一条映射全部有限实例、旧未配置原行为、启用后缺曝光表时管理答案拒绝交付、管理员撤回后可审准确旧正文。ExportQuestionArchive是受信CLI专用，测试不伪造个人actor。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^Test(LearningExposure|QuestionExports)' -timeout 5m -count=1。Expected: 含答案入口未推进曝光/失败仍交付的特定断言失败。
- [x] **Step 3: 实现上述曝光接口并集成入口。** 原写在同一个questionTx尾部记录，不另开事务；答案GET切换READ COMMITTED可写共享锁事务并重证原角色/作者规则。questionExposureRefs从P4a CanonicalPackage规范正文与原questionCanonical("question-template-v1",template)取得准确模板SHA，固定/生成实例复用CanonicalInstance；无需另跑生成器或等待版本发布才能记曝光。原始未批准草稿以准确SHA记录，仅当选题数学身份完全相同才排除，同名不同正文不匹配；准确模板曝光记录模板而非展开全部正文。每用户序号单调、最后时刻取最大值；所有含答案重放均追加交付事实，既有私有DTO/状态/不可变body不改。
- [x] **Step 4: 确认GREEN。** 同Step2命令；另Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -run 'Question' -timeout 5m -count=1。Expected: 入口矩阵与全部旧question回归PASS，无未覆盖HTTP答案入口。
- [x] **Step 5: 提交。** Run: git add backend/internal/store/exposure* backend/internal/store/question_tx.go backend/internal/store/question_draft.go backend/internal/store/question_submission.go backend/internal/store/question_review.go backend/internal/store/question_idempotency.go backend/internal/store/question_exports_test.go；git commit -m "feat: record question answer delivery before responses"。

### Task 5: 阅读动作、固定路线加入与有效资格

**Files:** Create backend/internal/store/learning_actions.go、learning_qualification.go、learning_paths.go；Test learning_actions_test.go、learning_qualification_test.go、learning_paths_test.go。

**Interfaces:** Consumes task1 ResolveState/EvaluateEvidence、task2 tx/Receipt、task3限制。Produces Store.StartLearning(ctx context.Context,a question.Access,id string,in learning.StartInput) (learning.KnowledgeState,error)；CompleteLearning(ctx context.Context,a question.Access,id string,in learning.CompleteInput) (learning.KnowledgeState,error)；EnrollLearningPath(ctx context.Context,a question.Access,id string,in learning.EnrollInput) (learning.PathView,error)；learningApplyAssessment(ctx context.Context,tx *sql.Tx,u auth.User,now time.Time,fact assessment.AttemptFact) (assessment.ProgressUpdate,error)；learningCurrentEvidence(ctx context.Context,tx *sql.Tx,actor string,k question.Identity) (learning.EvidenceView,error)。

- [x] **Step 1: 写动作与资格失败测试。**
~~~go
func TestLearningCompleteRequiresStart(t *testing.T) {
  f := newLearningFixture(t)
  _, err := f.repo.CompleteLearning(f.ctx,f.Access("learner_a",false),f.knowledge.ID,f.completeInput())
  if !errors.Is(err, learning.ErrStateConflict) { t.Fatal(err) }
}
~~~
LearningActions重复start保留首次时间/事件、complete无start拒绝、旧version拒绝、GET不产生事件；LearningCompletionNewMaterial测知识版本不变而unit/asset引用更新：旧完成被限制、再次明确complete固定新依据、首次completedAt不变、相同依据重复不增加事件；LearningAlternativeEvidence用合法封存SQL夹具测试两种完成/通过顺序、诊断无完成也有资格、普通只有通过无资格、最新复习失败仍有其他通过；LearningPaths根节点、全部准确前置、诊断只被测节点、固定route顺序与分母、显式新版、撤回不删除旧节点。已通过事实夹具必须有真实批准来源、五题/答案/result约束，不用缓存passed替代；Task7再用真实命令复测。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearning(Actions|Complete|AlternativeEvidence|Paths|Qualification)' -timeout 5m -count=1。Expected: 动作/资格接口未实现或违规授予。
- [x] **Step 3: 实现上述接口。** learning_events记录不可变开始/完成及实际publication、准确unit/asset依赖；记录保留首次startedAt/completedAt及原事件；当前完成资格按所有准确版本完成事件的依赖限制择有效证据。若已发布讲解依据改变，用户再次明确complete可追加固定新unit/asset证据；相同依据重复不制造事件、时长或完成计数。显式enroll固定当前path/准确nodes≤1000/分母，根与符合条件授予。learningApplyAssessment只接受已在本事务持久化且SHA/限制重证的AttemptFact；normal/review组合准确完成，diagnostic独立。加入/开始/普通检测创建保存合法首次解锁；完成/通过仅查当前直接后继，稳定ID授予唯一、旧路线不遍历或改写。
- [x] **Step 4: 确认GREEN。** 同Step2命令；Expected: 阅读、两种顺序、多前置、替代证据与固定分母PASS，计数分别对应阅读/有效通过/历史解锁。
- [x] **Step 5: 提交。** Run: git add backend/internal/store/learning_actions* backend/internal/store/learning_qualification* backend/internal/store/learning_paths*；git commit -m "feat: persist explicit learning and versioned qualifications"。

### Task 6: 安全练习的创建、作答、查看与放弃

**Files:** Create backend/internal/store/practice_commands.go、practice_read.go；Test practice_commands_test.go、practice_read_test.go。

**Interfaces:** Consumes task1投影/GradeChoice/GradeNumeric、task2幂等、task3题源、task4曝光。Produces Store.CreatePractice(ctx context.Context,a question.Access,in assessment.PracticeCreateInput) (assessment.PracticeView,error)；ReadPractice(ctx context.Context,a question.Access,id string) (assessment.PracticeView,error)；AnswerPractice(ctx context.Context,a question.Access,id string,in assessment.Answer) (assessment.PracticeView,error)；RevealPractice(ctx context.Context,a question.Access,id string) (assessment.PracticeView,error)；AbandonPractice(ctx context.Context,a question.Access,id string) (assessment.PracticeView,error)。

- [x] **Step 1: 写真实练习失败测试。**
~~~go
func TestLearningPracticeRevealNoQualification(t *testing.T) {
  f := newLearningFixture(t)
  p, err := f.repo.CreatePractice(f.ctx,f.Access("learner_a",false),f.practiceInput())
  if err != nil { t.Fatal(err) }
  got, err := f.repo.RevealPractice(f.ctx,f.Access("learner_a",false),p.Summary.ID)
  if err != nil || got.Summary.State != "revealed" || f.qualificationCount("learner_a") != 0 { t.Fatal(got, err) }
}
~~~
LearningPractice无blueprint/未解锁仍可练习、skipped作答拒绝、优先未见、格式错误active无答案、合法错答一次终结、revealed无score、旧结果不可改、24h边界、单active并发与过期创建、同键复放旧ID、明确放弃、另用户404。若练习先创建再创建重合正式检测，answer/reveal拒绝STATE_CONFLICT；若正式检测先有五题，练习创建排除重合，不能靠旧practice ID取答案。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearningPractice' -timeout 5m -count=1。Expected: 工作流未实现或提前答案/错误资格断言失败。
- [x] **Step 3: 实现上述接口。** 同用户锁内终结expired active后再新建；freeze一题准确来源并用ProjectQuestion返回。Answer仅合法输入一次、规范格式错误不终结；Reveal无评分。终态含答案先曝光，active检测重合先拒绝；记录题面首次/最后见题，不让刷新/同键重放增加“见过题数”；全部练习路径无qualification grant。
- [x] **Step 4: 确认GREEN。** 同Step2命令；Expected: 完整状态转换、无蓝图练习、所有权和曝光PASS。
- [x] **Step 5: 提交。** Run: git add backend/internal/store/practice_*；git commit -m "feat: add safe single-attempt learning practice"。

### Task 7: 五题检测、诊断、提交与并发正确性

**Files:** Create backend/internal/store/assessment_commands.go、assessment_read.go；Test assessment_commands_test.go、assessment_concurrency_test.go。

**Interfaces:** Consumes task1 SelectFive/GradeFive/Seal、task3题源、task4曝光、task5 learningApplyAssessment。Produces Store.CreateAssessment(ctx context.Context,a question.Access,in assessment.CreateInput) (assessment.AttemptView,error)；ReadAssessment(ctx context.Context,a question.Access,id string) (assessment.AttemptView,error)；SubmitAssessment(ctx context.Context,a question.Access,id string,in assessment.SubmitInput) (assessment.ResultView,error)；AbandonAssessment(ctx context.Context,a question.Access,id string) (assessment.AttemptView,error)。

- [x] **Step 1: 写真实检测失败测试。**
~~~go
func TestAssessmentFourOfFive(t *testing.T) {
  f := newLearningFixture(t)
  attempt := f.createDiagnostic("learner_a") // 本任务helper调用真实CreateAssessment
  result, err := f.repo.SubmitAssessment(f.ctx,f.Access("learner_a",false),attempt.Summary.ID,f.answers(attempt,4))
  if err != nil || result.Score == nil || *result.Score != 4 || result.Passed == nil || !*result.Passed { t.Fatal(result,err) }
}
~~~
新增AssessmentNotReady（三十分钟/全局最近五题/模板/8core/1000池/多个blueprint/retryAt）、AssessmentExposureAfterCreation（editor预览及重放后affected、score/pass null；自己提交结果不自伤）、AssessmentExpiryAndRace（同用户create唯一、submit唯一、abandon对submit、锁等待跨24h、撤权/密码/取消）。AssessmentEvidence普通两种顺序、diagnostic不改祖先/阅读、review失败、替换旧题继续提交、知识新版affected、永久撤回与grant两种提交顺序、五项格式全体拒绝无局部分数。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestAssessment' -timeout 5m -count=1。Expected: 正式检测接口缺失或特定资格/竞争断言失败。
- [x] **Step 3: 实现上述接口。** node重证当前canEnter，diagnostic无需前置，review须该知识个人历史。创建固定两publication/准确蓝图/五题/core/rule/crypto-rand32-byte seed/当前曝光序号；active冲突返回已有摘要，过期处理与幂等顺序沿Task2。提交先一次验证五项，读取数据库当前时刻与曝光/准确知识/永久撤回；合法原输入均固定，限制或后曝为affected，无score/pass及资格；否则GradeFive→固定result→ApplyAssessment→自身答案曝光→Receipt，同事务提交。Read只派生expiry，不写学习/资格；原schema result约束不放松。
- [x] **Step 4: 确认GREEN。** 同Step2命令；Expected: 五题3/4/5、全部竞态及准确版本PASS，双提交只有一份原答案/result/grant。
- [x] **Step 5: 提交。** Run: git add backend/internal/store/assessment_commands* backend/internal/store/assessment_read* backend/internal/store/assessment_concurrency_test.go；git commit -m "feat: submit fixed five-question assessments atomically"。

### Task 8: 私有概览、固定进度、历史结果与受控素材

**Files:** Create backend/internal/store/learning_read.go、learning_history.go、learning_assets.go；Test learning_read_test.go、learning_history_test.go、learning_assets_test.go；Modify practice_read.go、assessment_read.go（结果投影与重放）。

**Interfaces:** Produces Store.ReadLearningOverview(ctx context.Context,a question.Access) (learning.Overview,error)；ListLearningKnowledge(ctx context.Context,a question.Access,q learning.ListQuery) (question.Page[learning.KnowledgeState],error)；ReadLearningKnowledge(ctx context.Context,a question.Access,id string,version int) (learning.KnowledgeDetail,error)；ListLearningPaths(ctx context.Context,a question.Access,q learning.ListQuery) (question.Page[learning.PathSummary],error)；ReadLearningPath(ctx context.Context,a question.Access,id string) (learning.PathView,error)；ListLearningPathNodes(ctx context.Context,a question.Access,id string,q learning.ListQuery) (question.Page[learning.PathNode],error)；ReadAssessmentResult(ctx context.Context,a question.Access,id string) (assessment.ResultView,error)；ListLearningHistory(ctx context.Context,a question.Access,q learning.ListQuery) (question.Page[learning.HistoryEntry],error)；ReadLearningAsset(ctx context.Context,a question.Access,attemptID,sha string) ([]byte,error)。Consumes task3限制/task4曝光/task5当前资格。

- [x] **Step 1: 写历史投影失败测试。**
~~~go
func TestLearningReplayProjectionRestricted(t *testing.T) {
  f := newLearningFixture(t)
  saved, key := f.submitPassing("learner_a")
  f.withdrawFirstInstance(saved.Summary.ID) // 调用原真实withdraw API
  replay := f.replaySubmission("learner_a",saved.Summary.ID,key)
  if replay.Items[0].CorrectNumeric != nil || replay.Items[0].Explanation != nil || replay.Validity != "restricted" { t.Fatal("withdrawn answer leaked") }
}
~~~
LearningHistoricalProjection原score不改、affected score/pass null、替代有效证据仍资格、旧版本summary与当前lecture入口分离；LearningRead 101历史分页20/100/offset100000、空目录真实0、无GET授予、三计数不混合、旧分母包含撤回。LearningAsset本人固定引用可读、他人/任意SHA/未批准/永久撤回404，MIME/SVG/bytes/headers旧边界。每次结果GET推进曝光，包括重读；摘要不曝光，不展示固定答案的页不载答案DTO。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearning(Read|HistoricalProjection|ReplayProjection|Asset)' -timeout 5m -count=1。Expected: 私有投影/限制未实现或越权旧素材被读取。
- [x] **Step 3: 实现上述读取接口。** 正常列表只载分页个人state与有界聚合，全部通过证据反连接撤回并选有效替代。旧path固定版分母，newVersionAvailable仅提示；历史非当前知识只summary，不任意载旧lecture。答案result/practice读取与含答案幂等重放在曝光事务内重构当前投影，保存原业务ID/成功状态；管理员固定历史body仍原样。素材返回[]byte，与原ReadSubmissionAsset/GetPublishedAsset一致，HTTP复用contentAsset的SVG校验、image/svg+xml、≤1048576 bytes及安全响应头；按本人practice/assessment实际fixed refs和发布/限制检验，不接任意文件路径。完整JSON包装参与4MiB预算。
- [x] **Step 4: 确认GREEN。** 同Step2命令；Expected: 投影、替代证据、分页和素材权限PASS，原正确结果未因普通替换被失效。
- [x] **Step 5: 提交。** Run: git add backend/internal/store/learning_read* backend/internal/store/learning_history* backend/internal/store/learning_assets* backend/internal/store/practice_read.go backend/internal/store/assessment_read.go；git commit -m "feat: project private progress and restricted learning history"。

### Task 9: 统一服务、严格Go接口、完整OpenAPI与真实联调夹具

**Files:** Create learning/repository.go、service.go、service_test.go（backend/internal/）；httpapi/learning_dispatch.go、learning_routes.go、learning_json.go、learning_error.go、learning_test.go、learning_integration_test.go；api/learning-boundary-cases.json、learning-compatibility-baseline.json；tools/verify/learning-compatibility.test.mjs；e2etest/learning_fixture.go、learning_control.go、learning_fixture_test.go。Modify httpapi/application.go、backend/cmd/server/main.go、e2etest/harness.go、api/openapi.yaml、frontend/src/lib/api/generated.d.ts。

**Interfaces:** Consumes所有前述Store公共方法。Produces learning.Repository（方法逐项等于tasks2/5—8另加ConsumeRates(context.Context,[]auth.RateKey) error）；learning.NewService(repo Repository,acquire func(context.Context)(func(),error)) (*Service,error)；Service.Preflight(ctx context.Context,a question.Access,action Action) (auth.User,error)；Service.AcquireValidation(ctx context.Context) (func(),error)，各业务服务方法与对应Store名/签名一致；httpapi.LearningOptions={Learning:*learning.Service,PublicOrigin:string,Production:bool}，AuthOptions增加Learning *LearningOptions；DecodeLearningInput(raw []byte,action learning.Action) (any,error)。测试control LearningScenario枚举basic/diagnostic/withdrawal/exposure/history/capacity，只在loopback capability受控环境。

- [x] **Step 1: 写HTTP与夹具失败测试。**
~~~go
func TestLearningHTTPFiveAnswersRequired(t *testing.T) {
  h := newLearningHTTPFixture(t) // 本文件用真实fixture+service+application handler
  r := h.submitRaw(h.ownedActiveID(), []byte("{\"answers\":[]}"))
  if r.Code != 400 || bytes.Contains(r.Body.Bytes(), []byte("correctNumeric")) { t.Fatal(r.Code,r.Body.String()) }
}
~~~
LearningHTTP严格逐路径method、未知/重复/多余query、重复JSON键、NUL、整数1.0/1e0、截断/8192±1、完整响应4194304±1、错UUID/SHA/URL编码；所有权/CSRF/Origin/Fetch-Metadata、must-change、21动作rate共享、2槽真实占用/取消释放、8s含读正文。Boundary JSON每case有name/action/raw及accepted/errorCode用于Node共享；至少60个有意义用例含上述五题/numeric分支。兼容snapshot固定当前54旧paths/147schemas/22responses/3securitySchemes逐规范值摘要、00001—00005和旧数学用途；test故意改一个旧字段必须失败；测试snapshot来自46206fc基线，不能由已改OpenAPI生成新“基线”掩盖兼容变化。Harness测试同一服务真实学习流程、reset先清16新表的合法依赖，再清旧数据，不含生产入口或真实库迁移。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/learning ./internal/httpapi ./internal/e2etest -run 'Learning' -timeout 5m -count=1。Expected: route/service不存在或严格边界断言失败。
- [x] **Step 3: 实现上述接口与装配。** HTTP在同一8秒上下文只调用一次Service.Preflight（含Rates），heavy再调用Service.AcquireValidation并defer release，随后decode/dispatch到repo重证；Service业务包装不再次限流或申请槽。槽由既有publication.Service.AcquireValidation注入，不增加池。21个method/path逐项按设计§12（namespace根本身无业务接口）；新增8代码与旧auth/idempotency/413/404/503闭合映射，ErrStateConflict→ASSESSMENT_STATE_CONFLICT。状态/幂等头/Cookie/Retry-After/privateHeaders沿旧模式；asset单独binary受限，不误套JSON预算。OpenAPI全新具名DTO完整enum/oneOf，不增旧字段；JSON格式继续JSON.parse、canonical sort+digest做兼容快照，无新parser依赖。实际服务装配先供harness使用，再供server，harness控制不导入生产。Fixture不同author/reviewer真实批准三节点与有限题源，learner/多角色账户隔离，提前为Task11/12提供真实端到端环境。
- [x] **Step 4: 确认GREEN。** 同Step2命令；另逐条Run: node tools/verify/run.mjs -- node --test tools/verify/learning-compatibility.test.mjs；node tools/verify/run.mjs --cwd frontend -- npm run api:generate。Expected: HTTP/共享boundary/真fixture PASS；旧契约逐值相等，新generated类型稳定；连续第二次api:generate零差异。
- [x] **Step 5: 提交。** Run: git add backend/internal/learning backend/internal/httpapi/learning* backend/internal/httpapi/application.go backend/cmd/server/main.go backend/internal/e2etest api tools/verify/learning-compatibility.test.mjs frontend/src/lib/api/generated.d.ts；git commit -m "feat: expose bounded private learning API and real test harness"。

### Task 10: TypeScript契约、同源原始代理、SSR与主动重试客户端

**Files:** Create frontend/src/lib/learning/types.ts、schemas.ts、client.ts、server-client.ts、bytes.ts、test-fixtures.ts；对应schemas/client/server-client测试；lib/api/learning-proxy.ts、learning-proxy.test.ts；app/api/v1/learning/route.ts、app/api/v1/learning/[...segments]/route.ts。

**Interfaces:** Consumes Task9 OpenAPI及boundary cases。LearningAction逐项等于Go learning.Action；Produces LearningPage<T>={items:T[],total:number,limit:number,offset:number}；LearningRoute封闭联合（kind等于21个Action；id/query/sha只在需要分支存在）；LearningResult<T>={ok:true,data:T}|{ok:false,status:number,code:LearningErrorCode,message:string,requestId:string,retryAfter?:number,retryAt?:string,activeAttempt?:AttemptSummary,formatCode?:string}；requestLearning<T>(route:LearningRoute,input?:unknown,pendingKey?:string,signal?:AbortSignal):Promise<LearningResult<T>>；createLearningProxy(rawGoOrigin:string,config:{publicOrigin:string;production:boolean},fetcher:typeof fetch): (request:Request)=>Promise<Response>；getLearningClient():LearningReadClient；LearningReadClient的11个GET方法名与Go仓储一致但首字母小写（readLearningOverview/listLearningKnowledge/readLearningKnowledge/listLearningPaths/readLearningPath/listLearningPathNodes/readPractice/readAssessment/readAssessmentResult/listLearningHistory/readLearningAsset），去掉ctx/a，保留其余参数，JSON返回Promise<LearningResult<对应DTO>>，素材返回Promise<LearningResult<Uint8Array>>；cookies由原auth读取；validateLearningBytes(raw:Uint8Array,action:LearningAction):unknown。

- [x] **Step 1: 写TS边界失败测试。**
~~~ts
it("active assessment rejects any answer field", () => {
  expect(attemptViewSchema.safeParse({...activeAttemptFixture, correctNumeric:{numerator:"1",denominator:"2"}}).success).toBe(false);
});
it("LearningClientPending reuses exact key and bytes", async () => {
  await retryPending(frozenCommand); // 本文件helper用同route/input/key调用requestLearning两次
  expect(sentKeys).toEqual([frozenCommand.key,frozenCommand.key]);
  expect(sentBodies[0]).toBe(sentBodies[1]);
});
~~~
schemas表测全部Go共享raw JSON结果一致、额外字段/enum/正确性nullable；proxy严格method/URL/query/原始正文转发、双头、8192/4194304±1、cookies、JSON content type、受控binary素材、超时/abort含读响应、错误无raw题库。server-client未登录/不同用户no-store、active页无result。LearningClientPending不自动重试、同键手动重试、编辑新键、用户切换清内存、登录失效保留当前用户未确认状态提示。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd frontend -- npm test -- src/lib/learning src/lib/api/learning-proxy.test.ts。Expected: 新模块缺失或unsafe DTO/raw forward断言失败。
- [x] **Step 3: 实现上述接口。** types复用generated具名DTO，Zod strict闭合oneOf；bytes扩展原raw-json严格解析规则，只新增学习分支而不改原question行为。proxy原Uint8Array直传，不parse/restringify规范化输入；固定白名单路径，无任意URL；总10秒覆盖context/CSRF/请求/正文读取。JSON和asset二分处理，asset保留旧受控MIME/header/bytes。SSR在server-only边界传入本用户Cookie、fetch no-store，不将私有client放全局缓存。
- [x] **Step 4: 确认GREEN。** 同Step2命令；另Run: node tools/verify/run.mjs --cwd frontend -- npm run typecheck。Expected: 共享边界、客户端、代理、SSR PASS，旧auth/content/question测试契约不变。
- [x] **Step 5: 提交。** Run: git add frontend/src/lib/learning frontend/src/lib/api/learning-proxy* frontend/src/app/api/v1/learning；git commit -m "feat: add strict learning clients and same-origin proxy"。

### Task 11: 学习中心、知识状态、固定路线与回顾导航

**Files:** Create frontend/src/features/learning/overview-panel.tsx、knowledge-controls.tsx、path-progress.tsx、history-list.tsx、learning-status.tsx、pending-command.ts；learning.test.tsx、pending-command.test.ts；app/learn/page.tsx、app/learning-history/page.tsx；styles/learning.module.css。Modify app/knowledge/page.tsx、knowledge/[id]/page.tsx、paths/[id]/page.tsx；features/catalogue/knowledge-map.tsx；features/reading/knowledge-view.tsx、path-view.tsx；components/site-header.tsx、auth-status.tsx、learning-hub.tsx。Create tests/e2e/learning-helpers.ts、learning-progress.spec.ts。

**Interfaces:** Consumes requestLearning/getLearningClient及KnowledgeState/PathSummary/HistoryEntry。Produces OverviewPanel({overview:Overview})、KnowledgeControls({detail:KnowledgeDetail})、PathProgress({path:PathSummary,nodes:PathNode[]})、HistoryList({page:LearningPage<HistoryEntry>})、LearningStatus({state:KnowledgeState})，均返回React.JSX.Element；PendingLearningCommand={key:string,route:LearningRoute,input:Readonly<unknown>,actorId:string}；createPendingLearningCommand(route,input,actorId):PendingLearningCommand。UI显示状态由Go给出，无客户端资格计算；learning-helpers固定调用Task9 scenario/reset真实API，不提供假成功responses。

- [x] **Step 1: 写行为失败测试。**
~~~ts
it("route progress keeps withdrawn node in denominator", () => {
  render(<PathProgress path={pathSummary({totalNodes:3,completedNodes:1})} nodes={nodesWithOneWithdrawn}/>);
  expect(screen.getByText("1 / 3 read")).toBeVisible();
  expect(screen.getByText("Unavailable")).toBeVisible();
});
~~~
LearningUI未登录/未配置/无数据/错误、可读未解锁分离、开始与完成两个动作、未完成不显示已学、GET不加入、New route version available显式选择、最新失败needs-review但历史解锁保留、分页及点击当前lecture/旧summary区分。LearningProgress E2E两测试：真实学习＋加入路线的阅读/通过/解锁三计数；新版/撤回后固定分母；两个viewport键盘可达，无横向溢出。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/learning；Expected: 组件不存在或真实数字/动作断言失败。
- [x] **Step 3: 实现上述组件和页面。** 原公开SSR结果保持原DTO，只另取私有状态；匿名保持原阅读。状态英文Unlearned/Learning/Learned/Needs review/Mastered及Locked/Unlocked/Unavailable；Mark as learned依据completionValid显示，不能仅凭历史completedAt隐藏重新明确完成的入口；明确按钮Start learning/Mark as learned/Join route/Review。Learn空状态提示真实起点，无演示进度/时长。使用原创CSS/SVG或已有原创受控素材，图表路径关系沿旧阅读实现；PendingCommand与Task10精确键/输入合同一致，切账户清空。
- [x] **Step 4: 确认GREEN。** 同Step2；另Run: node tools/verify/run.mjs --cwd frontend -- npm run typecheck；生产构建及harness构建后Run: node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-progress.spec.ts。Expected: 单元PASS，真实2场景×2viewport=4 PASS。
- [x] **Step 5: 提交。** Run: git add frontend/src/features/learning frontend/src/app/learn frontend/src/app/learning-history frontend/src/app/knowledge frontend/src/app/paths frontend/src/features/catalogue/knowledge-map.tsx frontend/src/features/reading/knowledge-view.tsx frontend/src/features/reading/path-view.tsx frontend/src/components/site-header.tsx frontend/src/components/auth-status.tsx frontend/src/components/learning-hub.tsx frontend/src/styles/learning.module.css tests/e2e/learning-helpers.ts tests/e2e/learning-progress.spec.ts；git commit -m "feat: display personal learning and fixed route progress"。

### Task 12: 练习、五题表单、终态结果与未确认请求

**Files:** Create frontend/src/features/practice/practice-panel.tsx、practice.test.tsx；features/learning/attempt-asset.tsx、attempt-asset.test.tsx；features/assessment/answer-fields.tsx、assessment-panel.tsx、result-panel.tsx、assessment.test.tsx；app/practice/[id]/page.tsx、app/assessments/[id]/page.tsx、app/assessments/[id]/result/page.tsx；styles/assessment.module.css；tests/e2e/learning-practice.spec.ts、learning-assessment.spec.ts、learning-diagnostic.spec.ts、learning-security.spec.ts。Modify features/learning/knowledge-controls.tsx（blueprint/mode选择入口）。

**Interfaces:** Consumes task10 DTO/client及task11 PendingLearningCommand。Produces AttemptAsset({attemptId:string,asset:AssetRef})；AnswerFields({attemptId:string,question:SafeQuestion,value:Answer,onChange:(answer:Answer)=>void,disabled:boolean})、PracticePanel({view:PracticeView})、AssessmentPanel({view:AttemptView})、ResultPanel({result:ResultView})，均返回React.JSX.Element。客户端只维护原输入，SubmitInput一次五项（skipped显式分支）；SSR active用AttemptView，只有终态结果页ReadAssessmentResult。

- [x] **Step 1: 写作答失败测试。**
~~~ts
it("AssessmentUnconfirmed keeps original five answers for manual retry", async () => {
  render(<AssessmentPanel view={activeFiveQuestionFixture}/>);
  await enterFiveAnswersAndSubmit(); // helper本文件定义，真实requestLearning替身只模拟网络断开
  expect(screen.getByText("Result not confirmed")).toBeVisible();
  expect(sentCommands).toHaveLength(1);
  await userClick("Retry this request");
  expect(sentCommands[1]).toEqual(sentCommands[0]);
});
~~~
AssessmentUI四题通过/三题失败由后端DTO显示、五项format整体错误不露局部分数、数值原raw保留、多个蓝图选择目标文本、no-ready无零分、active冲突继续/明确放弃、到期英文绝对时间、刷新空未提交表单提示、不用持久存储。AttemptAsset测试正常替换后的自有历史素材仍走/api/v1/learning/assets/{attemptId}/{sha}，不回落公开SHA入口；restricted不渲染答案/解析，affected单列不判失败，复习失败不假取消历史解锁；取消/登录失效/换账号无自动写重试。新增E2E八场景：practice无蓝图揭示/非法格式后合法答题；完整五题解锁/五项格式拒绝后3题失败；diagnostic只目标资格/全局最近五题＋曝光不足；他人UUID素材拒绝/断网后同键手动重试及换账号。无route.fulfill假成功。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/practice src/features/assessment。Expected: 新表单缺失或自动重试/提前答案/原始输入丢失断言失败。
- [x] **Step 3: 实现上述表单和页面。** AttemptAsset复用原安全SVG显示规则，URL只由当前自有attemptId及固定AssetRef构造，不接受任意src；choice只选展示ID，numeric格式说明复用P4a并保留输入字符串；没有逐题check按钮。每个mutation先freeze PendingCommand，主动重试原bytes/key，新编辑新key；loading/abort及时可达，disabled不冒充已取消。结果呈现真实4/5规则、validity/补测说明与当前lecture回顾；明确Reveal answer终结练习。SSR及HTML序列化active页无私有答案，草稿只组件内存。
- [x] **Step 4: 确认GREEN。** 同Step2；另typecheck与build分别经verify/run；重建harness后分别Run: node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-practice.spec.ts learning-assessment.spec.ts；node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-diagnostic.spec.ts learning-security.spec.ts。Expected: 八场景×2viewport=16真实PASS，含服务端DB结果验证。
- [x] **Step 5: 提交。** Run: git add frontend/src/features/practice frontend/src/features/assessment frontend/src/features/learning/knowledge-controls.tsx frontend/src/features/learning/attempt-asset.tsx frontend/src/features/learning/attempt-asset.test.tsx frontend/src/app/practice frontend/src/app/assessments frontend/src/styles/assessment.module.css tests/e2e/learning-practice.spec.ts tests/e2e/learning-assessment.spec.ts tests/e2e/learning-diagnostic.spec.ts tests/e2e/learning-security.spec.ts；git commit -m "feat: deliver safe practice and five-question learning UI"。

### Task 13: 最大合法容量、完整回归、CI与中文验收

**Files:** Create backend/internal/store/learning_capacity_test.go、backend/internal/testutil/learning_capacity.go；docs/operations/learning-assessment.md、2026-10-02-p4b-acceptance.md。Modify .github/workflows/backend.yml、frontend.yml；docs/superpowers/plans/2026-09-30-development-roadmap.md、本计划任务勾选；共享capacity构造与场景同步修改 backend/internal/e2etest/learning_fixture.go、learning_control.go、learning_fixture_test.go、harness.go；必要优化只在前述所属文件，新增行为需重新审查。

**Interfaces:** Consumes全部已实现接口与Task9共享真实harness。Produces 可复验性能记录（actual rows、EXPLAIN ANALYZE/BUFFERS、elapsed、allocs/bytes、RSS、超时/锁竞争）及完整验收矩阵；CI继续相同4次push/PR checks，不引入新配额、生产migration或兼容修改。

- [x] **Step 1: 写容量失败测试与CI门槛。**
~~~go
func TestLearningCapacityMaxPool(t *testing.T) {
  f := newLearningCapacityFixture(t) // 本任务用真实发布：200模板/10000实例/1000蓝图/最大合法内容
  started := time.Now()
  got, err := f.repo.CreateAssessment(f.ctx,f.Access("learner_a",false),f.createInput())
  if err != nil || len(got.Questions) != 5 || time.Since(started) >= 8*time.Second { t.Fatal(got,err) }
}
~~~
容量范围按用户已确认的[调整方案](2026-10-02-learning-capacity-adjustment.md)：20条真实批准100节点路线与80条现有短路线验证100条路线分页；保留数据库1000节点防护及其余最大计数。共享capacity场景使用同一数据，初始化单独4分钟，其他场景40秒且所有产品动作8秒不变。容量夹具复用原question/content容量构造器，选题目标1000候选/8核心/稀疏覆盖、≤1000后继、101及大量个人历史分页；1000个蓝图集中同一知识时ReadLearningKnowledge的全部安全选项也≤8秒，复用同事务相同题源/核心的可行性计算，禁止1000次展开整库。每动作8秒内。LearningCapacityExposure批量模板关联、替代通过反连接、历史asset、4MiB完整包装±1、真实2槽与不同用户shared锁/管理exclusive竞争；断言只载五题body、不访问整份bank或遍历全部enrollments。先记录可重现失败与查询计划，性能已满足的新增容量场景应因测试不存在或计量门槛缺失RED，不能伪造功能失败。

- [x] **Step 2: 确认RED。** Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearningCapacity' -timeout 5m -count=1。Expected: 缺容量fixture/统计门槛或真实超限失败，记录原因。
- [x] **Step 3: 完成容量优化、CI覆盖和操作文档。** 若失败仅定向索引/批量查询/流式预算优化，不缩小合法输入或延长截止。backend CI纯包列表加入assessment/learning、兼容snapshot测试；frontend CI新增learning progress、practice/assessment、diagnostic/security三个≤480秒批次，保留全部74既有E2E并加入20新E2E。操作文档明确00006手动迁移、未配置恢复、保留原始作答、曝光失败拒绝答案、无P5自动重评/无生产部署；验收记录不把合成夹具当正式内容。
- [x] **Step 4: 确认GREEN并运行完整矩阵。** 逐条运行下方A—D，Expected: 全部PASS/退出0；old74＋new20=94真实浏览器断言（47场景×2viewports），零跳过/零重试；容量证据记录实际数值，不只写“性能通过”。
- [x] **Step 5: 提交。** Run: git add backend/internal/store/learning_capacity_test.go backend/internal/store/assessment_sources.go backend/internal/store/assessment_evidence.go backend/internal/store/learning_qualification.go backend/internal/store/learning_read.go db/migrations/00006_learning_assessment.sql .github/workflows/backend.yml .github/workflows/frontend.yml docs/operations/learning-assessment.md docs/operations/2026-10-02-p4b-acceptance.md docs/superpowers/plans/2026-09-30-development-roadmap.md docs/superpowers/plans/2026-10-02-learning-assessment.md；git commit -m "test: verify learning capacity compatibility and full regression"。只在完整矩阵通过后勾选Task13；独立审查和PR交付仍未完成。

## 完整验证矩阵与独立交付门槛

所有命令从实施worktree根运行，每行独立进程，禁止把多个批次塞进一个超过540秒的wrapper。新测试名称必须实际被 -run 匹配；保存输出的执行数量，无“no tests to run”替代PASS。

**A：格式、资料保护与兼容性**
~~~sh
git diff --check
node tools/verify/run.mjs --cwd backend -- node -e 'const{spawnSync}=require("node:child_process");const r=spawnSync("gofmt",["-l","."],{encoding:"utf8"});process.stdout.write(r.stdout??"");process.exit(r.status!==0||r.stdout.trim()?1:0)'
node tools/verify/run.mjs -- node --test tools/verify/run.test.mjs tools/verify/learning-compatibility.test.mjs tools/content-ingest/snapshot.test.mjs
~~~
兼容snapshot验证当前基线全部54paths/147schemas/22responses/3securitySchemes，以及原迁移/数学schema与purpose摘要；新名字只新增，不夹带修复P4a暂缓枚举项。

**B：Go静态、纯逻辑、真实数据库与构建**
~~~sh
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go vet ./...
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/assessment ./internal/learning ./internal/question ./internal/auth ./internal/content ./internal/publication ./internal/config ./internal/httpapi ./internal/e2etest ./internal/testutil -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -skip '^Test(Learning|Assessment)' -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^Test(Learning|Assessment)' -skip '^TestLearningCapacity' -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearningCapacitySourceVolume$' -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearningCapacityMaxPool$' -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build ./cmd/...
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/e2e-harness ./cmd/e2e-harness
~~~
数据库前置仅 TEST_DATABASE_URL/testutil随机库；缺测试数据库不能宣称集成验证成功。若包批次逼近5m按确定包/测试类拆分，不重复消耗全部无关测试，也不提高上限。

**C：前端与生成一致性**
~~~sh
node tools/verify/run.mjs --cwd frontend -- npm ci
node tools/verify/run.mjs --cwd frontend -- npm run api:generate
node tools/verify/run.mjs --cwd frontend -- npm run typecheck
node tools/verify/run.mjs --cwd frontend -- npm test
node tools/verify/run.mjs --cwd frontend -- npm run build
node tools/verify/run.mjs --cwd frontend -- npm audit --omit=dev
~~~
保存第一次生成内容，第二次api:generate逐字节一致；提交后CI git diff --exit-code -- frontend/src/lib/api/generated.d.ts。既有SSR隔离、渲染、认证和question测试全保留。

**D：真实浏览器分批**
每行Run均为 node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- 后接一组文件：catalogue.spec.ts reading.spec.ts；auth.spec.ts auth-security.spec.ts；content-authoring.spec.ts；content-review.spec.ts；content-release.spec.ts；content-security.spec.ts；question-authoring.spec.ts；question-review.spec.ts；question-release.spec.ts；question-security.spec.ts；learning-progress.spec.ts；learning-practice.spec.ts learning-assessment.spec.ts；learning-diagnostic.spec.ts learning-security.spec.ts。各批独立启动实际Next生产build→Go harness→随机PostgreSQL，双viewport，不复用真实服务、不返回假成功。针对新增UI保存桌面/手机截图用于验收，截图不含凭据或完整答案库；无需新增素材生成依赖。

Task1—13完成且A—D通过后，Native安排一次不同上下文独立整分支审查，读取本计划/设计及完整diff。重点是全答案入口、锁顺序/曝光/撤回竞争、准确历史与替代证据、最大合法输入、SSR/错误/素材泄漏。重要问题逐项真实RED→修复→GREEN并重跑受影响回归；小项明确延后和理由。审查发现的需求/兼容变更先形成具体可审方案，不能由实施者悄悄改设计。记录审查结论后才通过SSH推送实现分支、git PR、attach_artifact；最新head SHA的四项CI全部通过才宣称实现交付。计划勾选、审查完成、PR创建/合并三种状态分别记录，不自动部署。

## 计划自查与实施前置

| 已确认设计章节 | 任务/验收归属 |
| --- | --- |
| §1—3 用户目标、分期与架构 | 全局约束、依赖图、1/9/11—13 |
| §4—5 学习、准确前置、历史解锁与固定路线 | 1/5/8/11 |
| §6 安全题面与练习 | 1/3/6/9/10/12 |
| §7 五题、诊断、未见优先、到期与模式 | 1/3/7/9/12 |
| §8 全入口曝光与未配置 | 2/4/6—10 |
| §9 历史批准、当前资格、替代证据与撤回 | 3/5/7/8/13 |
| §10—11 16实体、事务、角色、幂等与锁 | 1/2/4—10 |
| §12 HTTP、英文页面、私有素材与取消 | 8—12 |
| §13—15 文件、兼容性、有界资源与可行性 | 文件表、2/3/9/10/13、A—D |
| §16—17 独立审查、测试及书面门槛 | 13、交付门槛、本节 |

作者自查：17节均有任务；65个步骤按失败测试→RED→实现→GREEN→提交；跨任务public DTO/内部fact及仓储方法统一；Review Focus五类均绑定具名测试；代码块只定义必要断言/验证命令，不代写函数体。最大容量、锁竞争与94个真实浏览器回归仍是实施后的运行门槛，本计划没有宣称已测通过。

本计划书面确认并完成文档PR合并后，重新SSH fetch origin master，确认干净工作区，通过using-git-worktrees从最新origin/master另建 codex/p4b-learning-assessment 实施分支；实施前运行基线A—D中已有测试（不调用尚不存在的新包/新用例），确认旧74 E2E基线。不得在文档分支或主工作区开始产品代码。若新master包含影响契约或方案的改动，先核对并更新计划，兼容性变化单独送审；其余按本计划Native逐项执行。
