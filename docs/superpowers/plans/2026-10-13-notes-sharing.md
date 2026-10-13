# 笔记优化与审批分享实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: 使用superpowers:executing-plans按任务在当前会话执行（建议Native），每任务按失败测试→实现→通过验证→提交推进；开始产品开发前，用户须审阅本方案和计划。方案与计划已由用户确认，随后明确授权Native开发与MR交付。

**Goal:** 优化知识笔记编辑与集中检索，交付管理员审批后分享、访客最新10条、登录浏览全部及撤回修改重审闭环。

**Architecture:** 保留原私人笔记表及API，新增独立分享head、不可变提交、事件和回执；所有原笔记新写在同一事务锁下检查分享状态。公开十条和登录全部采用不同精确接口与代理，前端复用安全Markdown/KaTeX，不新增生产依赖。

**Tech Stack:** Go1.27.1、CGO_ENABLED=0、Next16.3.8、React19.3、Node24.17、Vitest5、Playwright1.63、PostgreSQL17；以上为当前项目锁定配置，执行时以最新master锁文件为准，不因本功能升级依赖。

**Spec:** [详细方案](../specs/2026-10-13-notes-sharing-design.md)。基线master dec8b9e86499e7ba5dfe49961dfda87ce10a7921；开发前重新fetch master、复核增量并另建codex/notes-sharing分支，不从过期设计分支直接实施。

## Global Constraints

- 普通笔记默认私有。分享须管理员批准。
- 未登录访客可以查看最新10条已分享笔记；登录用户可以浏览全部已分享笔记。
- 作者更新分享中的笔记前，先撤回分享；修改后重新申请分享并接受管理员审批。
- 首版继续每个用户、每个当前知识点一份私人笔记；pending/shared期间原PUT及DELETE拒绝新写。
- 保留16000个Unicode字符、65536字节及禁止HTML/图片/危险宏规则；不自动保存、分享或审批。
- 不改旧13份迁移、旧笔记列/约束/触发器、原ManagedNote响应、四状态及学习完成事实；noteSharingSchemaIntegrity独立校验。
- 本阶段不包含自由笔记本、附件、协作、评论、点赞、关注、AI、复习排期或默写，不恢复旧知识审核、题库或测评。
- 所有页面与提示中文/英文均支持；数学正文不机器翻译；不制作未经授权的素材。
- 新普通请求正文上限131072字节；详情响应上限1MiB（1048576字节）、分页上限4MiB；HTTP/事务8秒、锁等待1秒、代理/客户端10秒。
- 列表默认20、最大50；q最多128个Unicode字符且512字节；游标最多512字节；分享标题1—120字符，退回/下架理由10—1000字符。
- 每次命令由tools/verify/run.mjs限制最多540秒，Go另设-timeout 5m、CGO_ENABLED=0；现有Playwright globalTimeout=480000、retries=0保持，不skip/only/取消旧检查。
- 先审查方案和兼容例外，再执行；开发后代码审查与回归，通过Git创建MR；合并和生产部署各自通过门禁。

## Review Focus

- 第11条及刚被最新10条挤出的分享：访客直接详情API也拒绝；列表与详情使用相同资格集合（任务3、7）。
- 保存请求超时后笔记已提交：原键确认原操作已执行、返回当前本人笔记且不新增事件；前端阻止新动作，新分享的旧回执不能重启已撤回分享（任务2、5）。
- 撤回已提交而管理员仍开着旧审批页：批准返回冲突，不公开旧正文；两管理员只有一个决定生效（任务2、6）。
- 关联知识不可用：私人读取保留、撤回可执行、公开读取拒绝；分享schema不完整：按方案§6所有分享接口及原笔记新写fail closed，私人读取保留（任务1—3）。
- 第11条详情在切换账户或会话失效后：清空登录专有内容，不退回匿名并保留旧正文；迟到的私有响应丢弃（任务4、6）。

## 文件与接口组织

详细文件清单见方案§8。领域对象统一放backend/internal/notesharing；SQL按schema/read/write/tx分文件；前端协议在lib/note-sharing，页面组件在features/notes。现有StudyRepository和ManagedNote不扩字段；新接口通过NoteSharingRepository单独注册。新迁移暂定00014_note_sharing.sql，不改旧迁移；若master占用编号，开工时同步调整两份文档和测试。

共享领域类型（必须由任务1定义，后续任务不得另起同义类型）：

- Query{Q,TopicKey,Status,Cursor string; Limit int}。
- SubmitInput{ExpectedRevision,ExpectedSequence int64; Title string}。
- WithdrawInput{SubmissionID string; ExpectedSequence int64}。
- DecisionInput{ExpectedSequence int64; Reason string}。
- Head{ActorID,KnowledgeID,Status,LastDecisionReason string; Sequence int64; SubmissionID *string; ApprovedAt *time.Time}。
- Receipt{SubmissionID,KnowledgeID,Action,Status string; Sequence int64}。
- OwnNotePage、SharedNotePage、LatestSharedNotePage、AdminNoteSharePage及详情：严格DTO；字段和空值规则完整采用方案§7的DTO表写入OpenAPI；私人页有ActorID；公开页无owner ID、原笔记修订令牌、未分享内容或审批理由。登录详情用AuthenticatedSharedNoteDetail封装actorId和note。
- Access复用knowledgeadmin.Access，不重新解析或创建用户身份；context和幂等键通过原事务方法传入。NoteSharingHealth{Capability,SchemaReady bool}用于独立结构探测，不改变readyz契约。

## Task 1: 领域契约与独立迁移

**Files:** 新增backend/internal/notesharing/model.go、validate.go、repository.go及_test.go；db/migrations/00014_note_sharing.sql；store/note_sharing_schema.go及_test.go。修改api/openapi.yaml、frontend/src/lib/api/generated.d.ts，只新增具名路径与schema。

**Interfaces:** ctx为context.Context、a为knowledgeadmin.Access、kid/submissionID/action为string。NoteSharingRepository的方法及返回值为：ListOwnNotes(ctx,a,Query)→(OwnNotePage,error)，ReadOwnHead(ctx,a,kid)→(Head,error)，Submit(ctx,a,kid,SubmitInput)/Withdraw(ctx,a,kid,WithdrawInput)→(Receipt,error)，Latest(ctx)→(LatestSharedNotePage,error)，ReadPublic(ctx,submissionID)→(SharedNoteDetail,error)，ListShared(ctx,a,Query)→(SharedNotePage,error)，ReadShared(ctx,a,submissionID)→(AuthenticatedSharedNoteDetail,error)，ListAdmin(ctx,a,Query)→(AdminNoteSharePage,error)，ReadAdmin(ctx,a,submissionID,eventsCursor string)→(AdminNoteShareDetail,error)，Decide(ctx,a,submissionID,action,DecisionInput)→(Receipt,error)，ReadSchemaHealth(ctx)→(NoteSharingHealth,error)。SQL检查noteSharingSchemaIntegrity(ctx,tx)→error、noteSharingInstalled(ctx,tx)→(bool,error)由后续store使用。

- [x] 写TestSharingValidateBoundaries：标题120合法/121非法；理由10、1000合法/9、1001非法；正文16000字符及65536字节边界复用原校验；非法UUID、sequence越界、未知JSON字段拒绝。
- [x] 写TestSharingMigrationPreservesManagedDigest：Up13和Up14的原managedSchemaDigest完全相同；空表Down允许，有任一事件/提交/head/回执时Down拒绝；新快照、事件和回执UPDATE/DELETE拒绝。
- [x] 写TestSharingIncompleteSchemaFailClosed：没有14且无新表标记为未安装；有14但缺表/触发器/约束拒绝；任意部分安装拒绝。
- [x] 运行新领域测试与store指定新测试，确认失败由尚未实现行为导致，不把工具/数据库故障当RED。
- [x] 实现模型、四表约束和新表固定摘要；原表及managedSchemaDigest不变。创建新的不可变表触发器，不修改原trigger函数定义。
- [x] 生成API类型，运行相关测试和类型检查；记录准确验证后提交“feat: 定义笔记分享契约与独立数据结构”。

验证命令：

    node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/notesharing -timeout 5m -count=1
    node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run 'TestSharing(Migration|IncompleteSchema)' -timeout 5m -count=1
    node tools/verify/run.mjs --cwd frontend -- npm run api:generate

## Task 2: 本人笔记列表、申请、撤回与审批事务

**Files:** 新增store/note_sharing_read.go、write.go、tx.go及对应_test.go；修改knowledge_admin_notes.go及_test.go、knowledge_admin_tx.go；httpapi/knowledge_admin_routes.go新增ErrLocked/ErrNotReady错误映射及定向测试。

**Interfaces:** 实现任务1的ListOwnNotes、ReadOwnHead、Submit、Withdraw、ListAdmin、ReadAdmin、Decide。新增noteShareWriteGuard(ctx,tx,owner,kid)；调用前原study record及note已锁定。未安装分享功能时返回nil，部分安装或损坏返回notesharing.ErrNotReady；pending/shared返回notesharing.ErrLocked。

- [x] 写TestSharingPrivateOwnerIsolation、TestSharingSnapshotExactSavedRevision、TestSharingNotesLiteralSearchAndPagination，检查他人/管理员不能读未提交正文、原笔记ref原样捕获、父主题与跨主题去重、正文不是SQL通配。
- [x] 写TestSharingSaveSubmitRace、TestSharingWithdrawApproveRace、TestSharingTwoAdminDecisions：用真实事务同步点控制竞争，断言只能冻结一次、撤回后不能批准、相同head.sequence只有一个决定。
- [x] 写TestSharingSaveReceiptBeforeLock、TestSharingReceiptCannotRepublish、TestSharingWrongKeyInput、TestSharingLockedDelete：原成功保存重试不误报锁定，新键禁止编辑；撤回后旧批准回执不复活；原键不同payload冲突；分享中DELETE不能绕过。另写TestSharingLockedOriginalNoteHTTP精确断言409及NOTE_SHARING_LOCKED，保留旧错误code断言。
- [x] 写TestSharingOldSaveReplayKeepsCurrentNote：第一次保存A，后续合法保存B，再重试A原键；返回当前B且不新增事件、不覆盖B。保留原保存幂等语义，新分享回执的冻结结果不套用到原笔记接口。
- [x] 写TestSharingDomainErrorsSurviveKnowledgeTx：ErrLocked/ErrStale/ErrIdempotency/ErrNotPublic/ErrNotFound/ErrInvalid/ErrNotReady经真实事务返回后errors.Is保持；修改knowledgeError完整透传7类新领域错误，不让authError折为503。原笔记HTTP另新增ErrNotReady映射；旧knowledge/auth错误不改名。
- [x] 写TestSharingWithdrawUnavailableKnowledge：源下架仍能撤回；私人读取保留；原下架知识保存/删除语义不变。
- [x] 观察上述真实数据库测试RED，再按方案锁序和状态机实现。原保存只在新写处分支增加guard；原DTO、事件kind、修订递增及幂等逻辑不重写。
- [x] 对快照、head更新、事件与回执采用一个事务；每次重提交新UUID，管理员允许明确审批本人申请但不能编辑正文。
- [x] 测试GREEN，核对数据库时间、所有权与失败回滚；提交“feat: 实现笔记分享审批与撤回事务”。

验证将下面三个组各自单独运行，均加-timeout 5m、-count=1并由540秒工具包装：SharingPrivate/SharingSnapshot/SharingNotes；SharingSave/SharingWithdraw/SharingTwo；SharingReceipt/SharingWrong/SharingLocked/SharingOldSave/SharingDomainErrors及原TestManagedPrivateNotesDuringUnpublish。各组不合并为超过10分钟的单次命令。

## Task 3: 最新10条及登录全部的读取权限

**Files:** store/note_sharing_read.go及_test.go；新增httpapi/note_sharing_routes.go、note_sharing_json.go、note_sharing_test.go；修改application.go、backend/cmd/server/main.go。

**Interfaces:** 实现任务1的Latest、ReadPublic、ListShared、ReadShared、ReadSchemaHealth；注册Notes NoteSharingRepository。新范围先于现有study分派，其他v3路径仍走原路由。分享错误使用原HTTP错误形状并新增具名错误码。独立GET /api/v3/note-sharing/health仅返回两个bool，结构未就绪503，旧readyz响应形状不变。

- [x] 写TestSharingGuestLatestTenAndDirectDetail：真实批准12条，latest恰为后10条；第11条直接public详情404，登录详情200；撤回或知识下架后补入下一条，原共有ID不重复。
- [x] 写TestSharingPublicReadWindowBeforeID：负例证明不能按请求ID先过滤再LIMIT10；批准时间相同按UUID排序稳定。
- [x] 写TestSharingReadAfterWithdrawal、TestSharingKnowledgeRestoreVisibility：撤回提交后新事务无正文；源知识恢复仅恢复仍shared的快照，withdrawn/hidden不恢复。
- [x] 写TestSharingRoutePermissions、TestSharingRequestBounds：无会话401、旧editor/reviewer管理403、CSRF/Origin拒绝、GET正文/转义/重复参数/未知字段拒绝、长度边界、跨筛选游标400。
- [x] 写TestSharingStoreHTTPExactErrors：真实store完成withdraw后旧approve返回409 NOTE_SHARE_STALE；同键不同payload返回409 IDEMPOTENCY_CONFLICT；非法title返回422 INVALID_FIELD；缺分享结构返回503 NOTE_SHARING_NOT_READY；原note PUT锁定返回409 NOTE_SHARING_LOCKED。必须穿过knowledgeTx的错误转换，不能只mock Repository的返回值。
- [x] 写TestSharingHealthRealSchema：真实Up14后GET health返回200且严格两bool为true；缺表/约束返回503且schemaReady=false；未知查询参数和GET正文拒绝，响应不含笔记字段。实现Latest/ReadPublic/ReadSchemaHealth的公开只读事务，不误调用要求身份的knowledgeTx。
- [x] 实现统一SQL资格谓词与同一REPEATABLE READ事务；公共接口无全库total及游标。登录只给当前有效分享，管理员专门路径读历史。
- [x] HTTP原普通时限保持；动作请求不接受正文快照；私有与公开响应都no-store，严格限制响应大小且不返回截断JSON。
- [x] 对应Go测试GREEN，go vet相关包与原managed API回归通过；提交“feat: 加入笔记分享接口与访客十条访问限制”。

测试命令分别运行store SharingGuest/SharingPublic/SharingRead/SharingKnowledgeRestore和httpapi SharingRoute/SharingRequest/SharingResponse/SharingStoreHTTP/SharingHealth；每条使用tools/verify/run.mjs、CGO_ENABLED=0、GOTOOLCHAIN=go1.27.1、-timeout 5m。

## Task 4: 前端严格协议、代理和身份边界

**Files:** lib/note-sharing/{types,schemas,client,server-client}.ts及测试；lib/api/note-sharing-proxy.ts及测试；app/api/v3/[[...segments]]/route.ts；必要的i18n错误码声明。

**Interfaces:** noteShareRequest<T>(path,method='GET',input?,key?):Promise<T>；readNoteShare<T>(path,cookie=''):Promise<Result<T>>；createNoteSharingProxy(raw,options,fetcher):请求处理器；isNoteSharingPath(path):boolean。旧managedRequest及schemaForPath规则保持。新模块明确校验ActorID，不将公开列表当作登录数据。

- [x] 写proxy.public GET不转发Cookie/private和admin必须转发Cookie的测试；unknown路径/方法拒绝；非上传请求不得使用64MiB和60秒预算。
- [x] 写proxy.healthPublicExactSchema：health在新代理中仅public GET，不转发Cookie；恰为两个bool，未知字段/非bool拒绝；缺分享schema的503原样可核验；原知识代理不新增任意公开前缀。
- [x] 写schemas拒绝未知键、超长正文、ActorID不一致、公开DTO含私人字段的测试；原ManagedNote schema照旧接受原响应。
- [x] 写失效会话、第11条详情和迟到响应测试：401清空内容，503显示不可用，不匿名降级；中断前保留原幂等键，不自动重发审批。
- [x] 观察Vitest RED后实现新代理分派与协议。公共latest/public详情精准列为public，shared全列表和详情精准列为private。
- [x] 定向Vitest、类型检查通过；提交“feat: 加入笔记分享前端协议与精确代理”。

    node tools/verify/run.mjs --cwd frontend -- npm test -- src/lib/note-sharing src/lib/api/note-sharing-proxy.test.ts
    node tools/verify/run.mjs --cwd frontend -- npm run typecheck

## Task 5: 笔记编辑体验、我的笔记与阅读布局

**Files:** features/study/managed-note-editor.tsx、managed-controls.tsx、managed-learning-view.tsx、managed-pages.tsx、app/learn/page.tsx及测试；features/reading/current-knowledge-view.tsx；新增features/notes/note-toolbar.tsx、note-preview.tsx、note-leave-guard.tsx、own-note-list.tsx、share-controls.tsx及测试；styles/notes.module.css；双语文案与覆盖清单。

**Interfaces:** NoteToolbar({value,onChange,selection,disabled})只处理文本；NotePreview({body})复用SafeMarkdown；NoteLeaveGuard({dirty,actorId,onDiscard})仅在当前脏笔记页注册离开保护，取消保留输入/URL、确认离开调用onDiscard并注销；OwnNoteList({page,filters})消费OwnNotePage；ShareControls({head,noteRevision,hasUnsaved,onWithdraw,onSubmitted})消费新分享API。保留ManagedNoteEditor原调用能力，通过新增受控共享状态读取完成界面锁定，不在原笔记DTO扩字段。

- [x] 写note.editor.previewAndFormula、selectionInsertion、limitBoundaries、languageKeepsDraft、pendingReceiptBlocksSubmit、withdrawBeforeEdit、spaNavigationUnsaved、accountChangeDropsLateResponse。
- [x] 写ownNoteList.ownerSearchPagination、unavailableNoteReadOnly、shareStatusDisplay；原学习与历史栏目链接及查询仍保持。
- [x] 写currentReading.bodyBeforeEmptyNote、mobileReadingDefault、safePreviewRejectsMarkup；数学正文数据不改写，阅读提示移除JSON/每行格式说明。
- [x] 观察RED后实现编辑/预览、格式按钮、字符与状态提示、我的笔记列表、分享预览与撤回操作；保留原保存按钮及准确冲突草稿。
- [x] 未保存时提交分享先提示保存，不自动丢弃输入；分享待审/已分享时后端锁定与前端一致。页面局部守卫覆盖所有离开入口，包括顶部logo/知识地图、账户/管理菜单及浏览器后退/前进；不要仅保护阅读/笔记内部链接或仅用beforeunload。保持Next历史状态，正常返回不残留重复导航；语言、预览、同页锚点、新标签打开不误拦截，不引入全站导航框架或正文持久存储。
- [x] 定向Vitest、原managed笔记与语言测试、类型检查GREEN；提交“feat: 优化私人笔记编辑与集中检索”。

    node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/notes src/features/study/managed-pages.test.tsx src/features/study/managed-learn-page.test.tsx

## Task 6: 分享阅读页、审批页与必要导航

**Files:** 新增app/notes/page.tsx、notes/[id]/page.tsx、admin/note-shares/page.tsx、admin/note-shares/[id]/page.tsx；features/notes/shared-note-list.tsx、shared-note-view.tsx、admin-share-queue.tsx、admin-share-view.tsx及测试；components/site-header.tsx/auth-status.tsx及测试；page-titles、messages、文案覆盖清单。

**Interfaces:** SharedNoteList({page,loggedIn})、SharedNoteView({detail})；AdminShareQueue({page,filters})、AdminShareView({detail})；页面SSR先核验会话并选择精准公共/登录路径，私有/管理数据复用账户边界。

- [x] 写guestTen/loginAll、oldLinkUnavailable、sharedNoteSafeFormula、adminExactSnapshotAndSequence、returnedReasonVisibleToOwner、adminCannotEditBody、ownSubmissionNeedsExplicitApproval。
- [x] 写withdrawWhileAdminPageOpen、sessionRevokedRemovesEleventhNote、accountSwitchDropsPrivateApproval、publicNoCacheHeaders；旧审批页收到409后刷新状态，保留输入理由，不重发批准。
- [x] 实现动态分享页与审批页：最新列表不静态生成，不混用公开/登录返回；批准、退回、下架明确点击，有结果待确认时仅用同请求确认。
- [x] 主导航增加“分享笔记”；管理员现有菜单增加“笔记审批”；个人笔记仍在我的学习内，手机菜单不新增多行常驻管理入口。
- [x] 原语言切换、草稿保持、角色导航与新组件测试GREEN；生产构建通过；提交“feat: 交付分享笔记阅读与管理员审批页面”。

    node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/notes src/components/auth-status.test.tsx
    node tools/verify/run.mjs --cwd frontend -- npm run build

## Task 7: 兼容审查、真实回归与MR交付

**Files:** 新增backend/internal/e2etest/note_sharing_fixture.go；注册harness.go；tests/e2e/note-sharing.spec.ts、note-sharing-security.spec.ts、note-editor-experience.spec.ts；修改ops/deploy.py及ops/tests/test_deploy.py；恢复兼容修订补充ops/database-snapshot.py及ops/tests/test_snapshot.py；精确兼容基线与工具摘要；.github/workflows/frontend.yml、backend.yml新增有界批次；docs/operations/note-sharing.md、website-guide.md、README.md及验收报告。

**Interfaces:** 新scene命名note-sharing，完全复用原隔离数据库、真实Go/Next和账户fixture；生成不少于12条已批准分享，使用原创数学技术夹具，不导入或批准生产笔记。不得通过新增测试入口回启生产旧业务。

- [ ] 写真实桌面/手机流程：本人编辑和公式预览→保存→申请→admin批准→访客10/登录全部→作者撤回→修改→重新审批；检查原学习进度与事件只按真实学习动作变化。
- [ ] note-editor-experience.spec.ts补真实SPA离开测试：填未保存正文，依次点击logo/知识地图/账户，取消后原URL和输入保持；浏览器后退取消后仍保留，确认后离开且再次返回不出现旧草稿或多余历史项；语言/预览/同页锚点不弹提示。每个用例使用隔离上下文，不用mock替代真实Next导航。
- [ ] 写真实安全流程：第11条直接API/URL、匿名分页搜索绕过、跨用户原笔记、会话撤销、旧审批批准、知识下架、同请求重试；前端禁止不能代替HTTP断言。
- [ ] 保留原managed-knowledge-study、ui-language-managed、site-streamline、生产退场及固定legacy完整回归；仅给修改的具体文件新增审查后的目标SHA，新迁移纳入清单，原13份迁移摘要逐一对照；不得整体重置基线。
- [ ] 运维先写test_fourteen_requires_note_sharing_health、test_old_managed_binary_cannot_serve_fourteen、test_start_and_activate_probe_sharing、test_future_migration_rejected：14缺noteSharing/探测404/结构false必须拒绝，14完整能力通过，15拒绝，1—13原判定不变。观察RED后修改topicSchemaCompatibility上限和显式noteSharing能力；新增runtime_note_sharing_health(root,revision,sudo)→dict并由verify_topic_runtime在requiredMigration>=14时调用。test_start_and_activate_probe_sharing还断言start先只探Go，web启动就绪后才探Next，activate后才探公网HTTPS；不在web启动前要求代理可达。不扩大原readyz JSON、不删除迁移一致回退检查。
- [ ] 新schema缺失/损坏、冻结快照不可变、Down拒绝有数据、不同代码/迁移版本回退拒绝和恢复表摘要/序列均有隔离证据。原readyz响应形状不扩字段，新增note-sharing API的schema检查作为发布前探测。
- [ ] 运行前端全测试、类型、API生成一致性、生产/固定兼容构建；Go按原批次预算执行，新增批次独立；每个浏览器文件独立命令，双视口原globalTimeout保持480秒。
- [ ] 使用superpowers:requesting-code-review进行一次独立整分支代码与兼容审查；修复阻塞项，仅重跑受影响范围，再记录最终commit的验证。
- [ ] 通过Git提交、推送codex/notes-sharing并创建指向master的MR，附到当前任务；必需CI必须对应最新HEAD。MR说明写最终行为、兼容变更、测试和生产迁移边界。
- [ ] 用户审阅并完成合并门禁后，正式发布另按既有运维流程准备备份、隔离恢复、整组代码与14迁移，核对所有原笔记指纹和原学习记录；代码MR通过不代表已部署。

    node tools/verify/run.mjs --cwd frontend -- npm run e2e -- note-sharing.spec.ts
    node tools/verify/run.mjs --cwd frontend -- npm run e2e -- note-sharing-security.spec.ts
    node tools/verify/run.mjs --cwd frontend -- npm run e2e -- note-editor-experience.spec.ts
    node tools/verify/run.mjs -- env TOPIC_COMPATIBILITY_STAGE=managed node tools/verify/topic-learning-compatibility.mjs managed
    node tools/verify/run.mjs -- python3 -m unittest discover -s ops/tests -p test_deploy.py

### 恢复兼容修订

真实117表备份恢复发现，原check函数通过未限定schema的调用依赖其他public函数，而pg_restore将会话search_path置空，导致COPY失败。方案保持原13份迁移、函数定义和结构摘要不变；任务7补充恢复工具修复：pg_restore生成0600私有临时SQL，仅替换经首部校验的会话路径为pg_catalog,public，再由psql的ON_ERROR_STOP及single-transaction整体执行。原备份文件和正文不改写，非法首部拒绝执行，提取与执行共享530秒预算。

可行性审查依据为真实隔离失败及PostgreSQL17的SQL脚本输出和psql事务模式；兼容验收补充嵌套check、正文内相同语句保留、中途错误整体回滚、117表摘要/序列及两个结构摘要。恢复时需要额外私有临时磁盘空间，磁盘不足必须在执行前失败或整体回滚。新14的固定摘要为2179de9c9ef5b78ec31debb6d5f4fb6c，正文check采用平铺AND，避免BETWEEN与外层AND重建时的目录表达式变化；仍只接受单一固定结构摘要。此修订纳入最终独立代码与兼容审查。

## 方案到任务的覆盖映射与停止条件

| 方案章节 | 实现任务 | 必须形成的证据 |
| --- | --- | --- |
| §1—3范围与架构 | 1、4、7 | 无新依赖、原API/退场边界和文件清单审查 |
| §4笔记及页面 | 4—6 | 组件验证、双语、双视口与键盘/320px检查 |
| §5状态机 | 2、6、7 | 真实事务竞争、撤回重审、旧链接不可复活 |
| §6数据与事务 | 1、2、7 | 原schema摘要不变、新schema守卫、幂等和恢复 |
| §7权限及缓存 | 3、4、6、7 | 最新10条详情资格、会话失效、代理和no-store |
| §8文件范围、§9验收 | 全部，终审在7 | 精确兼容例外、最终HEAD回归、独立审查与MR |

开始产品开发前，详细方案和计划必须经用户审阅；本轮可行性和兼容性审查不能替代开发后的测试。执行中若发现接口、锁序、数据保留或访问范围需要改变，先修订方案并审查该兼容变更，不能临场扩大范围。

Native执行建议沿用当前会话逐任务实施，最后一次独立整分支审查。此处为建议执行方式，用户确认后采用；不预先创建执行子任务或启动产品实现。
