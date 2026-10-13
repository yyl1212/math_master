# 第二阶段：复习功能优化与知识点默写实施计划

> 执行者须使用 superpowers:executing-plans 按任务执行（沿用 Native 偏好），或经用户改选后使用 superpowers:subagent-driven-development。全部步骤用复选框跟踪。当前只完成方案与计划审查，用户确认书面成果后才开发。

**Goal:** 建立个人复习队列、先作答后人工比对的知识点默写和忘记／不熟／熟悉自评排期，保持原学习与笔记分享契约。

**Architecture:** 新studyreview领域与五个独立表保存质量、会话及固定参考；写仓储在原knowledgeTx内衔接原周期，私有读使用模块局部强身份Read Committed事务。所有写返回无正文回执后GET当前态；旧DTO、学习事实、笔记与分享保持。

**Tech Stack:** 仓库锁定Go1.27.1（CGO_ENABLED=0）、PostgreSQL17.11/pgx5.11.0、Next16.3.8/React19.3.0/TypeScript5.9.3、Zod4.6.5、现有Markdown/KaTeX/Vitest/Playwright；不升级依赖。

**Spec:** [详细方案](../specs/2026-10-14-review-recall-design.md)。

## Global Constraints

- 用户自行比对参考并自评，禁止自动判分、文本相似度判分或AI参考生成。
- 保留1—14迁移字节、四学习状态、六旧事件、旧API/DTO、原笔记和审批分享规则。
- grade为forgotten/uncertain/familiar；所有评分先统一计算有效阶段（无安排/Ref变化为0）；阶段0..5；forgotten+600秒，uncertain+86400秒且有效阶段-1；熟悉的阶段1..5间隔86400/259200/604800/1209600/2592000秒；同内容提前熟悉不推进。
- 答案≤16000 Unicode码点且≤65536字节，合法UTF-8在码点上限下最多64000字节，65536是额外保护与保守预算；题干与参考条目原文本UTF-8合计≤524288字节，不包括标题/类型/JSON语法或整个快照，不截断。
- 新请求≤131072字节，单项/列表响应≤1048576字节；HTTP/事务8秒、锁1秒、Next10秒。
- 题干只交付一次；按HTTP同一json.Marshal规则，材料wire≤589824、最大答案wire保守预留393216、最大元数据/envelope≤65536；发起前验证组合≤1048576。单项两标题各≤1024字符/4096字节，列表标题各≤128字符。
- 队列/历史默认20最大50；队列offset≤100000；q≤128字符且512字节。作答/参考不出现在列表、URL、浏览器存储或日志。
- 原readyz形状和原managed/sharing摘要保持；新schema与能力独立，15整组Go/Next/网关发布；未来16拒绝。
- 单命令≤540秒，Go测试-timeout 5m；原浏览器480秒、单worker、零重试；只追加独立新验证作业。
- 所有新命令为本人会话，CSRF/幂等/提交身份复核；同一用户/知识最多一个活动会话、同原reviewId最多一次grade。

## Review Focus

1. 原手动finish或下一轮start与新会话交叉：不能重复结束、结束错误周期或改写已完成事实（Task2/3/5/6）。
2. 知识改版、下架及页面迟到响应：冻结参考不能变成当前正确答案，不能泄漏失效参考或他人作答（Task2/3/5/6）。
3. 相同幂等键在放弃、完成或后续安排后确认：原回执不复活会话、不二次推迟dueAt（Task2/3/4/6）。
4. 长中文、公式、空白答案、窄屏和浏览器历史：输入保留、比对可用、揭示前无目标参考数据（Task1/5/6）。
5. 跨主题重复关系、旧无自评记录、提前复习、极大队列与结构损坏：分类/计数真实，原功能继续守住边界且不放宽预算（Task1/2/4/6/7）。

## 前置步骤与总体顺序

- [ ] 用户确认书面方案与本计划；自评自动排期已确认，具体间隔和模式由整体方案确认覆盖。
- [ ] 重新git fetch origin master，核对新变更与迁移编号；从最新master创建codex/study-review-recall，把已审查文档提交带入；不得在master开发。
- [ ] 核验现有隔离worktree并复用，创建本计划专属Native ledger；逐项核对以下Interfaces，不再读取其他计划scratch。
- [ ] Task1 → Task2 → Task3；Task4/5消费同一契约按Native顺序实施；Task6 → Task7。

## Task 1：领域、固定参考与可解释排期

**Files:** 新backend/internal/studyreview/{model,repository,schedule,reference,validate}.go与{schedule,reference,validate}_test.go。

**Interfaces:** Access复用knowledgeadmin.Access；Ref复用knowledgeadmin.Ref。新增Mode=read|recall、Target=statement|proof、Status=working|compared|completed|abandoned、Rating=forgotten|uncertain|familiar。Schedule{Stage int,DueAt time.Time,LastRef Ref,LastRating Rating,Sequence int64,GradedAt time.Time}；Section{Kind string,Items []string}，Kind仅statement/conditions/scope/system/equations/proof/proof-scope。

输出 NextSchedule(previous *Schedule,current Ref,rating Rating,now time.Time)(Schedule,error)，BuildReference(point knowledgeadmin.SourcePoint,mode Mode,target Target)(FrozenMaterial,error)，ValidateAnswer(body string)error。FrozenMaterial{Prompt []Section,Expected []Section,SHA256 string}仅用于内部快照；响应ExpectedReference{Sections []Section,SHA256 string}不再包含Prompt。Rating常量明确为Forgotten/Uncertain/Familiar，对应上述三个JSON值。

- [ ] 写TestNextScheduleExactIntervals、TestEarlyFamiliarDoesNotAdvance、TestChangedRefResetsEffectiveStage、TestClockOverflow：原阶段0/1/4/5的三评分、提前/到期相等/跨时区、改版、最大日期均断言精确秒数和阶段；补A5→B不熟→B到期熟悉连续断言0/86400→1/86400；纯函数不取客户端时间。
- [ ] 写TestReferenceTargetsAndLimit、TestReferenceWireBudget：陈述五类实际字段、证明题干包含equations/目标仅proof与proof_scope、空证明拒绝；非转义密集材料524288可接受/524289拒绝，wire超589824即使raw未超也拒绝；元数据最大形状及65536答案保守组合≤1MiB，不重复Prompt、不生成/截断。
- [ ] 写TestAnswerUnicodeAndEmptyRecall：16000/16001码点、16000个四字节字符的64000字节合法向量、65536/65537字节仅作超过码点或非法输入拒绝向量、空白、HTML/图片/危险公式宏；复用原文本校验及明确“未写出”标识。
- [ ] 跑新增测试观察真实RED；实现最小函数，定义下游使用的DTO及Repository签名，再跑GREEN。

核心断言向量（time.Second为间隔单位，内容相同且已到期）：

```go
cases := []struct{ stage int; rating Rating; wantStage int; wantSeconds int64 }{
 {0, Forgotten, 0, 600}, {4, Uncertain, 3, 86400},
 {0, Familiar, 1, 86400}, {1, Familiar, 2, 259200},
 {2, Familiar, 3, 604800}, {3, Familiar, 4, 1209600},
 {4, Familiar, 5, 2592000}, {5, Familiar, 5, 2592000},
}
// 对每个case断言Stage及DueAt.Sub(now)；提前熟悉保持原DueAt，
// Ref变化时任何原阶段的familiar均wantStage=1、wantSeconds=86400。
```

Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/studyreview -timeout 5m -count=1

Expected: 全部PASS，固定常量与参考字段对照Spec一致。提交：feat: 定义复习自评排期与默写参考领域。

## Task 2：五表、原周期和原子私人仓储

**Files:** 新db/migrations/00015_study_review.sql、backend/internal/store/study_review_{schema,tx,read,write,cycle,receipt}.go及各*_test.go；仅改knowledge_admin_tx.go的新错误映射；精确改note_sharing_schema_test.go、note_sharing_write_test.go的原迁移目标。

**Interfaces:** Repository输出ReadReviewHealth(ctx)(Health,error)、ReadReviewOverview(ctx,a)(Overview,error)、ListReviewQueue(ctx,a,QueueQuery)(QueuePage,error)、StartReviewSession(ctx,a,StartInput)(CommandReceipt,error)、ReadReviewSession(ctx,a,id)(Session,error)、SaveReviewDraft(ctx,a,id,DraftInput)(CommandReceipt,error)、CompareReviewSession(ctx,a,id,CompareInput)(CommandReceipt,error)、GradeReviewSession(ctx,a,id,GradeInput)(CommandReceipt,error)、AbandonReviewSession(ctx,a,id,AbandonInput)(CommandReceipt,error)、ListReviewHistory(ctx,a,HistoryQuery)(HistoryPage,error)。

输入：StartInput{Knowledge Ref,Mode Mode,Target Target}；DraftInput{ExpectedSequence,ExpectedDraftRevision int64,Body string}；CompareInput{ExpectedSequence int64,Body string,Unanswered bool}；GradeInput{ExpectedSequence int64,Rating Rating}；AbandonInput{ExpectedSequence int64}。Session含ActorID/SessionID/KnowledgeID/ReviewID/Ref/Mode/Target/Status/Sequence/DraftRevision、Title/TitleZH/KnowledgeType、CreatedAt/ComparedAt/GradedAt/AbandonedAt/DraftUpdatedAt、Available/MaterialChanged/Unanswered、Prompt []Section、Answer *string、Reference *ExpectedReference、Schedule *Schedule、Grade *GradeResult。Title/TitleZH/KnowledgeType是发起时从已发布知识冻结的允许元数据，标题各≤1024码点且4096字节、类型为原枚举，计入最大65536元数据预算，不挤入Section；不可用时以状态提示展示。Schedule为知识当前安排，Grade为该会话不可变的当次自评/安排，分别标注，未完成时Grade=null。read答案null，recall答案为本人草稿/冻结答案；Reference只在read或已compared/completed且当前可用Ref一致时交付，其他情况null。当前不可用/Ref不同时Prompt也为空，任何状态都不交付冻结官方正文；仅保留本人答案/允许元数据。初始sequence=1/draftRevision=0；每成功写sequence+1；draft/compare还draftRevision+1，重放不增长；所有超出9007199254740991的序列拒绝。

QueueQuery{Q,TopicKey,Bucket string,Limit,Offset int}；HistoryQuery{KnowledgeID,TopicKey,Mode,Rating,Cursor string,Limit int}；页与Overview均含ActorID/asOf，Queue不含答案和参考。

CommandReceipt{ActorID,Action,SessionID,KnowledgeID,ReviewID string,Knowledge Ref,Sequence,DraftRevision int64,Status Status,AppliedAt time.Time,Grade *GradeResult}；GradeResult{Rating Rating,GradedAt time.Time,Schedule Schedule}，只有grade非null，全部无正文。新私有读helper studyReviewReadTx(ctx,a,fn)使用允许锁的READ COMMITTED事务、原共享身份锁、knowledgeIdentity(false,true,false)，正文需知识行锁及提交前身份/时效复核；不改旧knowledgeTx。新写仍用原knowledgeTx，先身份/资源/摘要确认回执，未命中再做当前Ref、可用性、活动状态与CAS。

- [ ] 写TestReviewSchemaConstraintsAndDigests：五表所有权错绑、非法状态、不可变参考/事件/回执、答案冻结、一个活动会话、一个原周期grade唯一；原两个摘要必须相同。
- [ ] 精确修原sharingSchemaDB为UpTo14；缺分享扩展子场景构造13或DownTo13，保留原version==14、空扩展可Down、已用扩展不可Down与缺14原保存成功断言；新增TestReviewMigration15KeepsSharing，证明15上原分享端到端/摘要不变。先加15观察旧目标错误RED，再修测试夹具，不改旧断言强度。
- [ ] 写TestReviewLifecycleAndOldCycle：completed开新原周期；原reviewing复用；旧finish先完成可补自评不再写旧事件；旧下一轮start使旧会话grade409；放弃只结束新会话；原ApplyManagedStudy仍逐字节不变。
- [ ] 写TestReviewIdentityAndCommitRace、TestReviewReadRevocationOrdering：匿名/他人/管理员他人/强制改密；撤销先提交则拒绝且无正文；读先持账户/会话锁则撤销按真实等待/超时规则处理，提交前身份与新鲜时效再查；公开health不加身份锁。
- [ ] 写TestReviewReceiptsNeverResurrect：同键同输入一次、异输入409；完成、放弃、改版或下架后旧start/draft/compare/grade/abandon先返回无正文原回执再GET当前态，不改安排、正文或原周期；校验actor/action/resource并按原锁序跨标签串行。
- [ ] 写TestReviewCurrentRefAndAvailability、TestReviewProofPromptRedaction：保存/比对/grade期间改版、下架、删除；working/compared/completed时Prompt和Reference同时隐藏；下架后仍可abandon和读本人答案元数据；同Ref恢复仅恢复合规读取。
- [ ] 写TestReviewQueueAndHistoryOwnership：旧已学未安排、原reviewing无新会话仍可继续、无schedule时原materialChanged仍属更新、五桶最高优先级唯一计数、主题多重关系去重、字面搜索、分页边界、过滤输入、下架隐藏正文、同内容提前熟悉与改版第一天。
- [ ] 所有真实SQL测试先RED，再迁移及最小实现；使用原knowledgeTx与TransitionState，不串旧HTTP、不修改原study表或函数。读投影须阶段与当前Ref双重核对。
- [ ] 发起前验证真实序列化预算；新cycle、session、snapshot、event、receipt在422时全无变化；比对时最大合法答案与转义密集参考合并仍≤1MiB。
- [ ] 新schema摘要由真实迁移/规范search_path实测后固定；分别破坏表/约束/触发器/摘要验证503，保留原笔记/分享门禁。

Run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestReview' -timeout 5m -count=1

Expected: 原事件与首次/已完成事实保持、无半提交、两套入口一致；若单条新组超时按领域分批，不改任何原组或预算。提交：feat: 持久化原子复习周期与私人默写记录。

## Task 3：新增契约、Go路由及Next代理

**Files:** 新backend/internal/httpapi/study_review_routes.go、study_review_json.go、study_review_test.go；改application.go、knowledge_admin_routes.go（依赖注入）、backend/cmd/server/main.go；改api/openapi.yaml和generated.d.ts；新frontend/src/lib/study-review/{types,schemas,client,server-client,pending-command}.ts及测试；新lib/api/study-review-proxy.ts及测试；改app/api/v3/[[...segments]]/route.ts。

**Interfaces:** 消费Task2 Repository；追加KnowledgeOptions.Reviews studyreview.Repository。createStudyReviewProxy(rawOrigin,authOptions,fetcher=fetch)及isStudyReviewPath(path)先于通用study派发。readReview<T>(path,cookie)输出严格读DTO；reviewCommand(path,method,body,key)输出CommandReceipt，readSession(id)输出当前Session；pending保存actor/resource/action/key/input并只允许原请求确认，回执须actor/action/resource准确相等后GET当前态。

- [ ] 写TestStudyReviewHTTPBoundsAndPermissions：十个路径、方法/重复参数/额外字段、每个大小边界、8秒取消、CSRF/Origin/Idempotency-Key及安全错误响应；旧各API回归保持。
- [ ] 写TestRecallExpectedAbsentBeforeCompare、TestProofPromptAllSourceRedacted：陈述目标与证明目标在working DTO实际不存在，读模式可见；下架/改版所有状态的题干/目标均消失；缓存及public health不能含用户、答案或正文。
- [ ] 写代理测试：只允许新路径、私有cookie筛选、跨源拒绝、1MiB流式上限、10秒中断、no-store、无Set-Cookie；旧managed与sharing路由仍正确派发。
- [ ] 为全部新对象建立strict Zod及OpenAPI；先失败样例再实现，拒绝服务器错误类型及未知字段；返回序列、actor、resource必须一致。
- [ ] api:generate后原路径/组件规范化对比完全相同；新所有schema引用可解析；未开放新正文公共路径。

Run: 分别用tools/verify/run.mjs执行go test ./internal/httpapi -run '^TestStudyReview' -timeout 5m、frontend npm run api:generate、npm run typecheck及新三类Vitest文件。

Expected: 真实HTTP/代理/严格解析PASS；原OpenAPI、生成类型及路由只追加。提交：feat: 接入受限私人复习与默写接口。

## Task 4：复习工作区、阅读自评及本人历史

**Files:** 新app/learn/review/{page.tsx,history/page.tsx}；features/study-review/{queue,self-rating,history,pages}.tsx及测试；styles/study-review.module.css；改managed-learning-view.tsx、managed-controls.tsx与i18n消息/page-titles/精确覆盖清单。

**Interfaces:** 消费Overview、QueuePage、HistoryPage、Session及CommandReceipt/GradeResult；新增ReviewQueue、SelfRating、ReviewHistory组件及SSR页面，全部使用StudyAccountBoundary。SSR概览/列表顺序获取以避免身份行锁争用，Session阅读宿主供Task5复用。

- [ ] 写TestReviewWorkspaceSections、TestReadingReviewGrade、TestReviewHistoryPrivateMetadata：到期/更新/待安排/未完成/全部、20/50分页、搜索、空队列、标题双语、读模式明确compare再grade、不显示分数；原四状态和进度口径保持。
- [ ] 写TestReviewUncertainCommandsAndAccountSwitch：迟到响应、换账户、强制改密、503恢复、旧回执确认后的新状态刷新；已确认写但GET失败只重读，不重复写或把旧Session/上轮成功当当前状态。
- [ ] 实现三个入口与工作区；原手动start/finish按钮继续可用，与新入口并列；历史分别显示原学习事实和新自评，不伪造旧质量。
- [ ] 手机单栏、键盘焦点、分页后刷新、未来时间与服务器asOf显示清楚；公共导航不展示本人计数。

Run: node tools/verify/run.mjs --cwd frontend -- npm exec -- vitest run src/features/study-review/queue.test.tsx src/features/study-review/self-rating.test.tsx src/features/study-review/history.test.tsx src/features/study/managed-learn-page.test.tsx

Expected: 原导航/学习页面及新阅读复习GREEN。提交：feat: 建立个人复习工作区与自评历史。

## Task 5：先作答后比对的默写会话

**Files:** 新app/learn/review/[id]/page.tsx、features/study-review/{session,recall-editor,comparison}.tsx及对应测试；复用NoteToolbar、NotePreview、NoteLeaveGuard，不改原组件。

**Interfaces:** 消费Task3 Session/DraftInput/CompareInput/GradeInput、Task4 SelfRating；输出受账户保护的SessionPanel、RecallEditor、Comparison。陈述题干仅标题，证明题干与目标明确分离。

- [ ] 写TestRecallSSRHasNoExpected、TestRecallFreezeAndCompare、TestProofPromptAndNoProof：初始HTML/React数据不含目标，自己的预览不包含参考；statement“下式成立”且公式只在equations时证明题干完整；写回执无正文，显式提交并GET当前态后双栏比较；再次编辑不得覆写；空白未写出只能forgotten。
- [ ] 写TestRecallDraftNavigationAndLocale：16k字符/64KiB、长公式、SPA链接/菜单/历史返回/刷新、取消离开、同页切换与双语保留。
- [ ] 写TestRecallChangedOrWithdrawnReference：working/compared/completed的API/SSR/迟到响应在改版或下架时Prompt和目标都不显示；本人答案仍可读；保留草稿并明确重新开始。
- [ ] 实现明确保存草稿、提交查看参考、人工自评三步；只在成功返回后改变阶段，不自动保存到私人笔记或申请分享。

Run: node tools/verify/run.mjs --cwd frontend -- npm exec -- vitest run src/features/study-review/session.test.tsx src/features/study-review/recall-editor.test.tsx src/features/study-review/comparison.test.tsx src/features/study/note-editor.test.tsx

Expected: 先作答后参考的所有数据边界及原笔记回归GREEN。提交：feat: 实现知识点默写与人工比对。

## Task 6：真实浏览器、容量、恢复与新独立CI

**Files:** 新e2etest/study_review_{fixture,control}.go及测试、新三组Playwright及helpers；改harness.go、auth_fixture.go；新增明确容量测试文件；改两workflow、go-batches及精确预算/语言/兼容政策和测试。

**Interfaces:** 场景study-review由原真实注册/登录/角色流程构造已发布知识与已学记录；控制器复用现有scene预算、隔离随机库及真实客户端。原复位单Exec中五新表排在原列表之前，原语句顺序和RESTRICT保持。

- [ ] 写TestStudyReviewSceneResetAndAtomicity：新表真实FK不会阻塞后续原场景；失败全部回滚；#58真实存储往返仍为1，原六账号/密码参数不改。
- [ ] 三独立文件两视口：flow验证待安排→阅读→自评→队列与历史；recall验证陈述/证明题干、写/存/比对/冻结、公式/双语/离开；security验证所有权/跨标签/原入口交叉/未确认写入/改版/下架/空答限制。
- [ ] 新容量：单用户10000学习与安排项目、多主题关系及不超过50项分页；raw524288、wire589824参考与最大合法64000字节答案/最大元数据的真实Go→Next组合；另测HTML转义密集但合法的16000码点答案，65536仅保守预算/拒绝向量；转义密集参考超wire上限在发起前422且无周期/会话写入。按schema/read/write三条5分钟命令分组，填充与测量分开，不伪造HTTP结果。若达不到8秒动作，补测真实热点并修设计。
- [ ] 真实14快照恢复后Up15：原所有业务表指纹与序列保持、五新表为空、两旧摘要相同；再实际恢复含read/recall/graded/abandoned/receipt的15全库，逐表/序列核对。随机隔离容器/卷仅清理本次UUID。
- [ ] 追加专属study_review_verify Go与frontend作业，原步骤/尺度/预算保持；Go新方法/测试登记新分组；全部新文件和目标精确登记，既有快照与旧阶段不改；新预算/篡改拒绝测试先RED。

Run: 每个Playwright文件单独用run.mjs（清除旧MATH_MASTER_E2E_FRONTEND，使用当前生产前端）；Go用CGO_ENABLED=0及5分钟；node --test tools/verify/*.test.mjs。

Expected: 新真实回归、容量、恢复及所有原工具PASS；不以本机性能冒充CI或线上结果。提交：test: 验证复习默写及原契约完整回归。

## Task 7：运维兼容、独立审查与MR交付

**Files:** 改ops/deploy.py及对应现有测试；新增docs/operations/study-review.md、阶段验收/代码审查报告；精确更新已审查变更登记。

**Interfaces:** 新runtime_study_review_health、verify_study_review_next、verify_study_review_public使用Task3双bool健康；topicSchemaCompatibility保留旧规则并增加15及studyReview能力，16拒绝。原readyz解析不改。

- [ ] 写Ops迁移/能力负例：14规则全保持，15缺Go或Next或公网健康均拒绝，16拒绝，旧14二进制不得回退15；restore/down/清理边界保持。
- [ ] 跑所有原Go分批及新增组、frontend类型/单元/生产构建/依赖审计、全部原浏览器与新增文件、Ops完整测试、兼容及篡改工具。每条命令不超过540秒。
- [ ] 组织一次Native新鲜全分支代码与兼容审查，重点覆盖五条Review Focus、原接口及真实回归证据；阻塞项一次TDD修复、受影响整套GREEN；裁定与延期项如实记录。
- [ ] Git正常推送并创建master MR，附到当前会话；准确最新HEAD所有CI通过再交付。文档写明自评不等于自动数学判定、恢复边界和现场验收步骤。
- [ ] 提取全部裁定/延期项，清理本计划专属scratch和合成测试资源，保留工作树。

Run: node tools/verify/run.mjs -- env TOPIC_COMPATIBILITY_STAGE=managed node --test tools/verify/*.test.mjs；Ops unittest按仓库入口完整运行。

Expected: 无代码/兼容阻塞，全部准确HEAD CI通过，MR可审查。此计划交付不自动合并或部署第二阶段，部署须取得对应授权后使用准确最终merge commit。

## 计划审查与交付

任务覆盖对应Spec第5—9节的每条状态、参数及故障语义；五条Review Focus均有明确拥有任务与测试。方案确认前不以未来测试名或新schema摘要尚未实测冒充已验证实现；实施时按精确签名与实际DDL固定摘要。

详细静态可行性、兼容性证据、独立审查发现与修订记录见[第二阶段方案审查](../reports/2026-10-14-review-recall-plan-review.md)。
