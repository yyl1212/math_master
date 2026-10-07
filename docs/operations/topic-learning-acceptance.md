# 主题学习重构验收记录

本次按已确认规格执行A8项、B8项、C7项，共23项。基线为master bca91cc98d75af53963789f28cdda5ca96871e17；C分支codex/topic-learning-cutover从最新master建立，带入已审查A/B依赖。最终C审查范围从91219bc开始，包含B随后性能与CI拆分依赖。代码交付不代表真实资料审核、合并、部署或生产切换。

分类合同为63个一级、534个二级、4969个具体主题，503辅助和534其他节点单列，共6603；目录数量不作为已发布知识数量。阅读、四状态、私人笔记、主题进度、历史和检索复习构成学习闭环，普通登录进入我的学习。完整操作见[使用手册](website-guide.md)。

## 当前核验

- A/B独立审查和修复记录分别见[分类审查](topic-taxonomy-review.md)、[个人学习审查](topic-study.md)。MR38、MR39为草稿；B2664673的六项远端CI均通过。
- C1—C6的真实隔离数据库、双视口浏览器、125项工具门禁、28项运维及实际镜像、备份恢复均通过。C6镜像源码为e8442c584cc019d520610377d7b87ae4882b6275。
- C7前端462项、类型与生产构建通过；完整业务档案场景证明真实满分诊断没有新完成事件。最终全矩阵、容量及独立终审结果追加于本文。
- 每条验证入口最长540秒；每次Go测试5分钟、数据库事务8秒、锁等待1秒；浏览器单worker、无重试、总480秒，桌面1280×900及手机390×844。
- 所有资料均为原创技术夹具；测试数据库随机math_master_test_*，仅使用专用PostgreSQL17.11容器。真实知识库和账户未进行迁入或切换。

## 保护边界

00010—00012只增加分类侧车、个人学习和明确维护审计；原9份迁移、数学正文、准确来源、封存对象、答案与成绩字节保持。原2184文件逐SHA校验；C只批准26个旧写路径增加410、readyz追加5布尔和3错误合同新增MODULE_RETIRED，其余原API逐条逆向相等。

私人笔记按本人校验，包括管理员访问他人笔记404；账户变化、会话撤销和强制改密清除原账户输入。新进度仅来自明确学习事实，旧检测通过、诊断、曝光或解锁不造完成。旧历史仍按原权限读取，旧写在topics返回410。

## 容量证据口径

公开夹具通过正常内容/主题审核及配对发布生成1000知识点最大内容目录。500技术用户×200准确知识引用生成100000原封存started事实，全部旧SQL约束和触发器启用；不伪造completed。每个源引用的历史技术manifest只保留实际批准的准确成员、submission、decision、frozenDigest，当前公开配对和数学对象不改。该批量体量夹具不冒充100000次人类浏览器行为；真实开始/完成、满分诊断和active档案由业务harness实际命令生成。

迁入每批最多50条，在模式配置锁、原管理员/内容锁、维护串行锁、按顺序账户锁和知识记录锁下进行；源时间/版本/摘要原样入历史。原生学习、笔记、幂等事实优先，不被旧投影覆盖。断点后未映射事实的索引前瞻避免每批扫描全前缀，尾批仍全局重核，包括早于游标的遗漏。

初轮真实容量因逐事实往返与源准备累计触发5分钟超时；批量化保持原时限、计数、守卫和done语义。夹具曾错误把seal编码为字符串及manifest字节遗漏purpose，均被原数据库check拒绝，修正为规范封存/manifest字节，不计产品行为红灯。

## 实施裁决与终审

C阶段逐项裁决、错误成本、一次独立终审及暂缓小项在最终验证后完整追加。A的两项地图小项仍暂缓：搜索卡完整祖先路径、详情子主题/内容分开筛选。B的历史事件中文标签及知识页历史入口已在C5补齐。

真实维护需独立授权和先验备份：按[切换手册](topic-learning-cutover.md)inspect、有限migrate、verify、备份恢复、显式activate；启动、健康检查及部署均不自动迁入或切换。topics库拒绝旧二进制恢复private流量，失败保持maintenance隔离，不Down或删除卷。

## C阶段实施裁决（含错误成本）

- Ruling: C计划中文任务标题补标准Task N标记，规格和任务内容保持 — Native task-start必须可靠提取编号，避免任务brief为空 — 若标记错误无法登记正确范围，逐任务读brief核对。

- Ruling: C1增加feedback_write.go的来源化撤回投影，以及feedback/new/page.tsx、topic-location helper/test、topic-view主题反馈入口 — 仅改feedbackConfigured仍会在知识元数据查询无条件解析question_withdrawals，计划漏列此依赖；主题展示定位需页面传递并存现有location，不更改旧API/目标DB形状 — 若分支错误旧实例撤回会漏判，原曝光/撤回及新无旧表具名测试覆盖。

- Ruling: C2增加study_mode事务内共用读取、server/harness接线、各旧routes总截止、旧题库直达详情页和历史服务端client/archive/history-list组件 — 计划只列首页会留下直接写UI，模式额外查询不能重置8秒截止，历史必须有本人保护入口；同一确认退出设计的小范围补全 — 若范围过宽C6逐文件及原API语义审查拒绝，旧阅读/曝光回归继续。

- Ruling: C2 retired响应沿用原private, no-store头且带原固定requestId合同 — 计划“no-store”不表示删除private，新测试最初错误断言单值已核对原合同后修正 — 若头/ID不一致Next严格代理拒绝，原头回归与真实410测试核验。

- Ruling: C2测试harness分类-only配对发布后仅模拟topics，C7再使用真实维护切换 — 本任务在00012及CLI之前须验证旧active界面，不能把直接模拟当正式切换证据 — 若漏C7不能交付正式迁入/切换。

- Ruling: C带入B最新性能修复ea79c94（本分支cherry-pick），最终C审查范围从原B91219bc起包含此修复 — B远端满容量准备真实失败，三容量/普通/Node全部通过后推送新提交，C不使用旧性能实现 — 若依赖差异有问题最终审查和完整矩阵须拒绝；不合并B MR39，最新远端CI继续核验。

- Ruling: C3 MigrationReport增加batchId，先提供仅迁入来源的InspectStudyMigration/MigrationInspection，C4复用后扩展完整切换检查 — C4 expectedMigrationBatchId必须有可信产出，C3已要求inspect/verify而完整切换报告在C4，不能以未实现错误当交付 — 若报告scope不清会误当可切换，C4必须另核pair/兼容代码/历史保护。

- Ruling: 迁入建立sequence=0、LastReadAt=null的旧事实投影；只更新此前由迁入创建且无任何原生事件/回执/笔记/最近阅读的记录，已有新记录完全优先 — 普通native begin可不增序号/事件但会记录阅读与回执，不能只按state或sequence判断归属；来源第一次/最近明确完成照实保留 — 若归属错误会覆盖用户新意图，已有笔记及两批间原生begin具名回归核验。

- Ruling: C3提前补harness三表明确重置与模式读取永久切换cap检查 — 新FK会阻止原账户TRUNCATE，缺已启用C结构不能回退旧写；只在OpenVerified随机库重置 — 若范围错会影响真实数据，harness仍token/loopback且生产server无控制依赖。

- Ruling: 迁入先锁配置、原全局共享锁与独立维护锁，再按UUID排序锁相关账户，沿用记录锁；每批只处理50条，10批上限 — 多个维护进程不能因不同游标交叉锁账户死锁，且与C4同一锁顺序 — 若维护锁冲突1秒明示失败可重跑，不扩大连接池或长期循环。

- Ruling: 00012迁入批次增加数据库递增sequence用于最新批次判定 — completedAt可相同，UUID并非时间顺序，C4必须准确选择最后已提交批次；用户历史仍按原时间+事件ID — 若排序错误会误拒绝或引用旧审计，固定时钟真实旧命令及完整迁入inspect回归覆盖。

- Ruling: C4报告增加activated标志，统计唯一实际切换；重复并发请求可共同确认原审计 — 计划“唯一winner”与“已topics重执行返回原记录”不能按nil错误数量判断，实际mode转换与审计必须只有一条 — 若标志错会误报重复切换，双请求/原ID和原日期重放测试覆盖。

- Ruling: C4增加buildmeta、Store可信构造与Docker VCS_REF编译注入、新模型cutover.go，backupRecord记录需传入审核事实 — 当前容器没有.git，单靠请求codeSHA不可信；沿用原release SHA构建参数，不新增环境口令参数或HTTP切换路由 — 若构建metadata缺失activate明确拒绝，C6临时镜像及C7CLI验收必须核验，真实切换另需授权。

- Ruling: C4扩展C3的CLI inspect/verify为完整切换报告，verify必须同时满足结构、pair、迁入、历史保护及可信版本；内部InspectStudyMigration仍仅检查迁入 — 中间C3空批次不等于可切换，不能保留迁入done就verify成功的旧阶段预期 — 若条件遗漏会误导运维，空配对/无可信代码CLI失败和Store完整检查回归覆盖。

- Ruling: C4固定19张新增表全部约束摘要，并核现有触发器、source链接与永久标记 — 仅表计数不能证明00010-12完整，丢主键也必须拒绝切换；固定schema来自按审核迁移建立的空随机库，不从运行库学习白名单 — 若定义漂移会明确拒绝，缺约束测试及schema catalog等价回归证明；不改原迁移字节。

- Ruling: activate备份参数指向既有700目录/600属主文件，核PGDMP、manifest数据库/版本/源码身份与dump/SHA256SUMS，只在审计留manifest摘要 — 不接受仅输入任意“已备份”文本，跟原ops备份格式一致，文件摘要流式且受总截止 — 若备份过期/不匹配会明确拒绝；恢复有效性仍由原restore-drill及C6验收，不把技术夹具当真实备份。

- Ruling: C带入B CI总预算拆分2664673（本分支cherry-pick），仍只移动新增五步 — B分类容量已远端通过但30分钟job取消最后study容量，124工具门禁RED→GREEN后推送 — 若CI库存错最终C全矩阵拒绝，不增预算、不删原测试，最新B CI持续核验。

- Ruling: C5补topic-area门户、旧纠错详情/plan-editor只读prop、内容TopicDraftEditor/server页面传入模式，以及知识页历史直达和历史筛选事件双语组件 — 只改列表会留下旧直达编辑，原PackageFields必须由真实SSR模式驱动；完整知识阅读/时间线设计在整合时补齐B暂缓两项，不另开B修复遍次 — 若页面未传模式旧写仍410但误显示操作，真实直达/后台/语言回归核验。

- Ruling: C6增加study.SchemaHealth、Store只读health与原handler/health可选能力读者，readyz仅5布尔 — 原readyz只ping不能说明新结构和模式兼容；原fake Pinger/健康合同仍保持，完整固定constraint与永久能力检查生效 — 若健康误报会恢复错误流量，结构损坏目标/安全字段/临时镜像核验覆盖，源码/笔记/用户数据不进health。

- Ruling: C6兼容精确声明73旧接线路径及39新路径，后续仅三原API测试和镜像白名单具名补全，原API语义只27路径/3错误合同改变 — C2明确批准410，readyz仅追加布尔；原files/9迁移/数学/来源/答案/成绩保持，其他API全部逐条逆向校验 — 若范围过宽全树PIN/旧故障注入/独立终审拒绝，没有自动按可见文件放行。

- Ruling: C6原API对象门禁只按固定目标语义SHA逐条逆向，加三具名旧测试入口及已批准路径的原始Git快照 — 未匹配条目不逆向，故障注入应报真实被改字段而不是先遇新增合法410/枚举；实际全文件字节门禁仍独立拒绝任何未知变化 — 若逆向过宽会隐藏篡改，新增不批准错误码/身份字段测试及全部原API/迁移/生成/成绩故障注入通过。

- Ruling: C6提前添加ui-language-cutover真实双语场景并精确登记 — 语言库存只接受具名ui-language文件，不能把普通topic spec当语言完成；新退出/档案/提醒/知识纠错必须真实覆盖 — 若缺覆盖门禁明确失败，原一worker/480秒及双视口不改。

- Ruling: C6镜像只额外打包分类安装和有限主题维护两命令，保留harness排除及原非root/tmpfs/TLS/资产检查 — 正式维护不能依赖未打包的A安装命令，新CCLI也需实际二进制 — 若白名单错误临时镜像及最终CI拒绝，真实镜像不推送部署。

- Ruling: C6功能与全部工具门禁通过后先提交以固定真实源码SHA，再构建临时镜像/隔离恢复，task-done只在该实测通过后登记 — 可信VCS_REF必须对应提交，不能给未提交树冒充发布SHA；这是本任务提交与镜像步骤的最小顺序调整 — 若镜像失败继续同任务修复，不提前声称C6完成，不真实部署。

- Ruling: C7抽取已有publishTopicCatalogueFixture共用正常审核段，真实旧资料包仅新增准确主题侧车并保留全部旧事实；harness使用固定可信技术runtimeSHA构造，不从control请求接收codeSHA — reset函数会删除旧历史不能用于真实迁入测试，复用正常业务段避免复制不同校验 — 若旧资料字节变化harness/原数据指纹及后端最终回归拒绝，实际CLI/真实编译版本已C4/C6验证。

- Ruling: C7容量100000源事件均为明确started，500用户×200准确历史知识引用，当前公开1000知识仍走原完整审核；所有SQL约束/触发器启用 — 批量体量夹具不是人类100000次浏览器动作，不能写completed冒充或从成绩推完成；实际业务harness separately真实完成/满分诊断 — 若夹具源不合法原守卫拒绝，已实际拒绝错误seal，迁入须全部保留来源/时间并不制造completed。

- Ruling: C7迁入改为同一事务批量账户/记录锁、来源链接读取及记录/事件/链接写入，模式行先锁并完整验证A/B/C，不逐批读取公开数学scope；游标后有未映射事实则done=false，仅尾批全局重核 — 100000真实源容量触发5分钟超时，原逐事实往返与重复全前缀扫描不能满足计划；每批仍50、8秒，原生记录优先和不可变来源审计不变 — 若批处理覆盖原生状态或漏迟到早时间事件，会误迁，原并发/断点/时间/回滚回归及新增容量验证拒绝。

- Ruling: C7容量源准备使用每知识一个技术历史manifest，成员准确复制实际批准submission/decision/digest，保持原全部SQL守卫、公开1000知识最大manifest和配对不变 — 原源准备独占181秒，不能用放宽5分钟预算解决；该隔离体量夹具不作为发布或人类完成证据 — 若准确审核来源失真，原learning_content_approved/封存触发器拒绝；真正用户动作另由完整业务harness核验。

- Ruling: C7部署workflow沿用C6已核验原镜像/隔离恢复入口，不作无效果修改；新增容量独立step和真实切换/档案/双语浏览器step — 计划列出的deployment/study_control无需新增逻辑，现有harness统一scene足以接线 — 若CI漏场景，库存门禁和完整最终矩阵拒绝。

- Ruling: C7维护CLI新增仅migrate接受的512字节明确--cursor JSON，并更新有限命令手册 — 原CLI每次命令从nil开始，超过500事件会反复停在已映射前缀；Store已有准确时间+UUID断点但CLI缺传递口；新增真实CLI审计游标测试实际RED退出2 — 若手册/参数错误会阻塞大型维护，严格字段/时间/UUID/操作范围校验及原Store跨批回归核验，不自动无限循环。

- Ruling: C7按Native先本机完整矩阵/任务提交与task-done，再一次整分支审查、必要一次修复、草稿MR触发远端最新SHA CI；交付仍等待同SHA所有CI成功 — 原“CI与终审后创建MR”存在触发/顺序循环，A/B同样采用此完成门禁，不省略审查或CI — 若后续未通过不得报告23项完成或合并部署。

## C7本机完整矩阵

最终22批后端全部退出0；244项浏览器、46份文件、36分组、两个视口全部通过，最大批耗时122.139秒；前端462/462、工具125/125、运维28/28、OpenAPI生成无差异、类型、生产构建、Go vet、所有命令构建、diff检查通过。真实切换知识修订提示也通过15.7秒双视口。临时公开故障提示回归真实RED后恢复原ContentState/重试且保持fail closed，原分组16/16 GREEN。

休眠日志曾墙钟跳跃触发浏览器分组12总超时；系统pmset确认深度休眠/开盖唤醒，原完整日志归档。原预算及断言重跑后8/8、57.6秒通过，其余已通过分组保留，未调整系统电源设置；环境中断不计通过证据。

100000迁入容量最终初次GREEN全程244.325秒，源准备117.379秒，迁入40.487秒，最大50事件批150.728毫秒；原当前1000知识/6603目录、500用户、0虚构completed、0用户发布提醒fanout、重跑去重通过。任务结束按同一容量命令再执行验证，完整结果由本阶段ledger归档记录。

### 后端实际命令 / 退出码 / 秒

```text
all-nonstore	0	125.884	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test $(cd backend && go list ./... | rg -v '/internal/store$') -timeout 5m -count=1
go-1	0	54.651	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 node ../tools/verify/go-batches.mjs --batch 0 --size 70 --slots 4 --skip '^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit|Taxonomy|Topic|Study)|^Test(WorkflowCapacityEnvelope|QuestionMaximumLegalWorkflow|QuestionCandidateCapacity)$'
go-2	0	37.265	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 node ../tools/verify/go-batches.mjs --batch 1 --size 70 --slots 4 --skip '^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit|Taxonomy|Topic|Study)|^Test(WorkflowCapacityEnvelope|QuestionMaximumLegalWorkflow|QuestionCandidateCapacity)$'
go-3	0	0.748	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 node ../tools/verify/go-batches.mjs --batch 2 --size 70 --slots 4 --skip '^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit|Taxonomy|Topic|Study)|^Test(WorkflowCapacityEnvelope|QuestionMaximumLegalWorkflow|QuestionCandidateCapacity)$'
go-4	0	0.649	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 node ../tools/verify/go-batches.mjs --batch 3 --size 70 --slots 4 --skip '^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit|Taxonomy|Topic|Study)|^Test(WorkflowCapacityEnvelope|QuestionMaximumLegalWorkflow|QuestionCandidateCapacity)$'
go-5	0	36.967	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestWorkflowCapacityEnvelope$' -timeout 5m -count=1 -v
go-6	0	25.113	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestQuestionMaximumLegalWorkflow$' -timeout 5m -count=1 -v
go-7	0	1.442	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestQuestionCandidateCapacity$' -timeout 5m -count=1 -v
go-8	0	87.544	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^Test(Learning|Assessment)' -skip '^TestLearningCapacity' -timeout 5m -count=1
go-9	0	33.214	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedback' -skip '^TestFeedbackCapacity' -timeout 5m -count=1
go-10	0	10.963	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedbackCapacity$' -timeout 5m -count=1 -v
go-11	0	87.95	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearningCapacitySourceVolume$' -timeout 5m -count=1 -v
go-12	0	118.788	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearningCapacityMaxPool$' -timeout 5m -count=1 -v
go-13	0	15.064	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -run '^TestContentAudit' -skip '^TestContentAuditCapacity$' -timeout 5m -count=1
go-14	0	61.288	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestContentAuditCapacity$' -timeout 5m -count=1 -v
go-15	0	132.97	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -run '^Test(Correction|Notification)' -skip '^TestCorrectionCapacity' -timeout 5m -count=1
go-16	0	138.118	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestCorrectionCapacityImpact$' -timeout 5m -count=1 -v
go-17	0	42.317	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestCorrectionCapacityNotifications$' -timeout 5m -count=1 -v
go-18	0	37.748	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/taxonomy ./internal/store ./internal/cli -run '^Test(Taxonomy|Topic)' -skip '^Test(TaxonomyCapacity|TopicCutoverCapacity)' -timeout 5m -count=1
go-19	0	91.389	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestTaxonomyCapacity6603With1000Knowledge$' -timeout 5m -count=1 -v
go-20	0	38.938	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/study ./internal/store -run '^TestStudy' -skip '^TestStudyCapacity' -timeout 5m -count=1
go-21	0	93.658	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestStudyCapacityReadPages$' -timeout 5m -count=1 -v
```

### 浏览器实际命令 / 退出码 / 秒

```text
browser-0	0	46.942	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- catalogue.spec.ts reading.spec.ts
browser-1	0	30.405	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- auth.spec.ts auth-security.spec.ts
browser-2	0	42.579	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-authoring.spec.ts
browser-3	0	32.467	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-review.spec.ts
browser-4	0	32.43	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-release.spec.ts
browser-5	0	39.934	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-security.spec.ts
browser-6	0	36.596	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-authoring.spec.ts
browser-7	0	38.445	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-review.spec.ts
browser-8	0	50.406	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-release.spec.ts
browser-9	0	49.765	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-security.spec.ts
browser-10	0	24.171	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-progress.spec.ts
browser-11	0	34.528	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-practice.spec.ts learning-assessment.spec.ts
browser-12	0	57.963	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-diagnostic.spec.ts learning-security.spec.ts
browser-13	0	26.866	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-review-regressions.spec.ts
browser-14	0	74.765	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-user.spec.ts
browser-15	0	47.312	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-review.spec.ts
browser-16	0	63.641	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-security.spec.ts
browser-17	0	27.822	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- correction-user.spec.ts
browser-18	0	19.202	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- correction-review.spec.ts
browser-19	0	46.755	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- notification-security.spec.ts
browser-20	0	42.728	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-acceptance-reading.spec.ts
browser-21	0	62.825	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-acceptance-learning.spec.ts
browser-22	0	8.616	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-public.spec.ts
browser-23	0	8.197	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-auth.spec.ts
browser-24	0	21.549	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-content.spec.ts
browser-25	0	21.312	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-question.spec.ts
browser-26	0	18.617	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-learning.spec.ts
browser-27	0	8.947	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-feedback.spec.ts
browser-28	0	15.146	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-correction.spec.ts
browser-29	0	13.161	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-navigation.spec.ts
browser-30	0	18.397	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-taxonomy.spec.ts
browser-31	0	122.139	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-study-navigation.spec.ts topic-study-progress.spec.ts topic-study-notes.spec.ts topic-study-security.spec.ts ui-language-study.spec.ts
browser-32	0	37.26	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-retirement.spec.ts topic-feedback.spec.ts topic-content-corrections.spec.ts
browser-33	0	16.006	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-learning-cutover.spec.ts
browser-34	0	7.991	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-learning-archive.spec.ts
browser-35	0	14.248	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-cutover.spec.ts
```

### 后续裁决

- Ruling: C7三公开首页/地图/旧板块模式未知时使用原ContentState unavailable及原重试按钮，继续不查询legacy fallback — 完整旧catalogue双视口真实RED，StudyUnavailable属私人学习且无重试，原公开可恢复故障界面必须保留；精确更新既有三具名页面SHA — 若错误会让topics故障回退旧目录，模式null直接return ContentState，不读旧API；全浏览器、兼容门禁核验。

## 唯一独立终审与一次修复

审查者为fresh gpt-6-astra/high，只读审查91219bc..3991b5c，未派生代理。Critical0、Important2、新增Minor0；核心来源迁入、配置屏障、旧保护与隔离恢复可靠。审查发现混合反馈列表被受限旧工单阻断，以及reviewed=true仍显示待回顾；CI构建版本/health测试遗漏按持续保障影响升为Important，同一遍次修复。

- 混合反馈：真实双场景RED（question_heads/learning_records）→GREEN；在分页前只过滤旧保护不可用的practice/assessment，不改targetValidity/API，不删数据；本人/审核游标、另一账户隔离、旧讨论空输出保护均验证。原反馈全套39.257秒、独立容量11.495秒通过。
- 回顾：组件真实RED→GREEN；正式切换后v2真实发布、开始并完成复习，overview reviewed=true，待回顾root链接消失双视口17.6秒通过，历史事件仍保留；GET/语言无写。
- CI：具名库存测试真实RED→GREEN；原5分钟Go批次新增buildmeta/health两包，原包和预算不改。

修复后全部nonStore/CLI/HTTP/health/buildmeta125.186秒、前端463/463、工具126/126、相关浏览器60/60、类型/生产构建/Go vet/全部命令构建通过。此前未改模块完整22后端/244浏览器证据保留，最终同SHA远端全矩阵另执行。没有二次审查或第二修复遍次。A两既有minor仍保留，B两历史界面小项已C5解决。

### 修复后实际命令 / 退出码 / 秒

```text
feedback	0	39.257	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedback' -skip '^TestFeedbackCapacity' -timeout 5m -count=1
feedback-capacity	0	11.495	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedbackCapacity$' -timeout 5m -count=1 -v
all-nonstore	0	125.186	node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test $(cd backend && go list ./... | rg -v '/internal/store$') -timeout 5m -count=1
browser-0	0	76.077	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-user.spec.ts
browser-1	0	46.841	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-review.spec.ts
browser-2	0	62.662	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-security.spec.ts
browser-3	0	8.671	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-feedback.spec.ts
browser-4	0	11.873	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-feedback.spec.ts
browser-5	0	118.208	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-study-navigation.spec.ts topic-study-progress.spec.ts topic-study-notes.spec.ts topic-study-security.spec.ts ui-language-study.spec.ts
browser-6	0	13.539	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-content-corrections.spec.ts
browser-7	0	17.882	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-learning-cutover.spec.ts
browser-8	0	7.771	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-learning-archive.spec.ts
browser-9	0	13.392	node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-cutover.spec.ts
```

### 审查暂未裁决的行为与最终裁决

审查者将A两既有界面小项、生产实际迁入/备份恢复/上线、备份最大年龄/与新源码SHA等值要求，以及DDL管理员主动替换函数的完整防篡改列为Declined to judge。下面完整列出执行者裁决及错误成本，不将这些行为静默丢弃。

- Ruling: task-done后先push已授权codex开发分支触发最新提交CI，与唯一独立终审并行；草稿MR仍在审查/必要一次修复后创建，最终门禁要求最新相同SHA全部绿 — push本身即触发原codex/** workflow，可满足计划CI先启动而不等待MR；不合并、不部署，也不让审查者改HEAD — 若出现修复新提交，则必须重新等待新SHA CI，不能用旧绿覆盖。

- Final: Ruling: 混合列表在分页SQL前只过滤保护能力不可用的practice/assessment来源，不新增targetValidity枚举/改旧API；直接元数据和讨论仍拒绝 — 修复C1混合页阻断且避免虚构withdrawn/replaced；本人/审核同一稳定时间+UUID游标 — 若过滤错误会漏显示或乱分页，真实双角色分页/另一账户0项和旧讨论空输出测试核验，数据不删，能力恢复后重新可见。

- Final: Ruling: CI建议提升为Important，并同遍次加入buildmeta与health两包原有限Go测试 — 可信版本与实际readyz恢复门禁需要后续持续验证，本机证据不能替代CI — 若库存错未来回归可能漏检，新增具名CI库存行为RED→GREEN，不增预算、不删原包。

- Final: Ruling: A两项既有界面minor维持，不重复修复未变A实现 — 主题详情已有路径，原筛选可完成学习，符合已审查阶段范围 — 若用户需搜索卡完整祖先或子主题/内容独立筛选，需要后续明确小项，不影响本次四状态与事实准确性。

- Final: Ruling: 生产迁入/真实备份可恢复性/实际上线不由此审查批准，维持隔离代码交付 — 无生产操作授权，真实验证需要维护窗口及对应备份 — 若误把隔离证据当生产成功将影响真实业务，文档/最终报告明确未合并部署切换。

- Final: Ruling: 不自定备份最大年龄或要求备份源码SHA等于新程序SHA — 合理升级前备份可来自旧代码；保留权限/格式/DB名/版本范围/摘要及真实隔离恢复，正式恢复仍由操作者选择并验证备份适用性 — 若选用不适用旧备份可能丢后续业务数据，真实操作必须按手册核对维护事实，校验不冒充任意备份都可恢复。

- Final: Ruling: 不扩展到抵抗DDL管理员主动替换同名函数的完整防篡改 — 用户/API无DDL入口，设计以受信数据库维护者为边界；保留已声明约束/触发器/永久标记检查 — 若DB管理员主动篡改超出当前模型，需要另外安全设计，不能声称抵抗已掌握DB管理权限的攻击。

- Final: Ruling: 修复后重跑受影响完整套件（反馈Store含独立容量、全部nonStore/CLI/HTTP/health/buildmeta、全部前端与工具、相关反馈/主题/双语浏览器），未改模块沿用已完整通过22后端/244浏览器证据；最新SHA全量CI另完整执行 — 不重复无变化大容量，保留Native一次修复绿色套件及最终全量门禁 — 若影响范围漏评，远端完整旧/新矩阵必须拒绝，不能拿旧SHA绿色替代。

最终容量在已提交3991b5c再次按Native task-done执行，254.242秒通过。23实施任务本机全部完成，最终交付继续等待相同最新SHA全量CI。首次push被自动审批以目的地信任证据不足拒绝；核验当前账号与自有public仓库ADMIN/PUSH、同仓库已附MR37/38/39及无私有runtime/凭据模式命中后，重审允许同一git push动作。未使用旁路。正式库、数学批准、合并与部署未执行。
