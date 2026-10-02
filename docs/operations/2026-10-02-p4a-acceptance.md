# P4a 可信题库技术验收记录

日期：2026-10-02。基线：master `687f87a0ab7e07fb73c593055b79d5ee3eb416bc`（PR #16），实现分支：`codex/p4a-question-bank`。用户已确认[设计](../superpowers/specs/2026-10-02-question-bank-design.md)和[13项执行计划](../superpowers/plans/2026-10-02-question-bank.md)，使用 Native 逐项实施。功能与本机回归完成，独立审查和最新提交 CI 结果在本记录末尾补充；此状态不能作为部署批准。

## 交付范围与文件

新建独立 question-bank schema、Go `internal/question`、store题库事务、三项CLI、新迁移00005、私有Go端口、OpenAPI具名DTO及Next代理/数据层、八个英文后台页面和真实联调。原ContentPackage、schemaVersion=1、内容摘要/迁移与公开API保留。文件责任与架构见[批准设计](../superpowers/specs/2026-10-02-question-bank-design.md)和[操作说明](question-bank.md)。题库编写、完整有限生成、独立核验、冻结送审、作者隔离、六项复核、双head发布、当前覆盖和永久撤回已实现。P4b练习、五题检测、资格、个人解锁/进度尚未实现。

## 设计第15节的12组验收

| 组 | 验收内容 | 证据与结果 |
| --- | --- | --- |
| 1 | 精确数值/格式/位数/非法选项 | TestNumericModesAndBoundary、TestNumericAnswerRepresentable、TestExactGrade：128/129 Unicode字符、256/257位、零分母、百分号、精确有理数与非法格式全部通过 |
| 2 | 三家族完整组合与独立错误注入 | TestGeneratorFiniteSpace、TestMissingOperandUniqueSolution、TestTemplateChoiceEquivalence、TestVerifierRejectsMutatedGeneratedAnswer；实际31×32=992合法/32×32=1024非法，通用1000/1001边界通过 |
| 3 | 固定身份与可复验归档 | TestQuestionCanonicalIdentity、TestQuestionArchiveTrustBoundaries及CLI往返：同模板参数身份相同，固定SHA/引擎/来源/作者不被外部UUID取代，CLI只导入草稿 |
| 4 | 固定目标与五题覆盖 | TestQuestionFixedReferences、TestFiveDistinctQuestionsCoverAllCore、TestQuestionFrozenSupplementaryObjectives：目标越界、旧版本、误关联和并集足够但五题不足均拒绝 |
| 5 | 全冻结与独立复核事务 | TestQuestionFrozenSubmission、TestQuestionCopiedAuthorCannotApprove、TestQuestionFrozenResponsibilityChange、真实review场景：全实例/目标/来源固定，作者不可自审、竞态单终态、旧冻结字节不变 |
| 6 | 双head及提交时有效身份 | TestQuestionDualHeadActivation、TestQuestionEvidenceEligibility、TestQuestionSessionExpiresWhileWaiting、TestQuestionIdempotencyAtomicity：等待后重新核验会话/资格，失配整体回滚 |
| 7 | 正常替换与永久事实 | TestQuestionReplacementPreservesHistoricalFacts、TestQuestionPermanentWithdrawalFacts、TestQuestionWithdrawalReplayRace和release场景：旧实例退场不伪造撤回，黑名单不经旧批准/重放复活 |
| 8 | 当前可用与历史分离/缺迁移 | TestQuestionHistoricalFactSeparation、TestQuestionCoverageAfterSingleInstanceWithdrawal、TestQuestionKnowledgeWithdrawalStopsOffer、TestQuestionReadinessDoesNotMigrate：动态重算，缺配置固定503、旧公开知识正常 |
| 9 | Go/Node严格边界与隐私 | question_http_boundary.json同源用例、QuestionHTTPContract、QuestionPrivateScopeAndErrors、代理单测/security E2E：原字节、整数/UTF8/重复/未知字段、4MiB/8KiB、Cookie/CSRF/取消/超时、角色撤回通过；学习者无答案读取端口 |
| 10 | 大快照/分页/共享资源 | TestQuestionMaximumLegalWorkflow、TestQuestionPublicationPagination、TestQuestionWrappedPageBudget、TestSharedQuestionValidationSlots、TestSharedContentQuestionRateWindow：101历史、实际返回limit、当前head独立定位，实际同两个槽和既有限流窗口 |
| 11 | 真实全流程浏览器 | authoring 8、review 4、release 8、security 8共28条；1280×900/390×844、键盘Tab/Shift+Tab/Escape及焦点、冲突输入保留、实际提交后超时同键手动重试；真实Next→Go→随机PG，无成功API伪造 |
| 12 | 原功能与兼容契约 | 原公开/阅读16、账户12、内容四批18共46浏览器用例；全部Go与前端原测试保留；原API深相等、原schema/导出摘要/旧迁移未改，两次api:generate零差异 |

## 回归与环境保护

- Go核心8包116顶层测试通过，store/cli103顶层测试通过，共219；store/cli最长整批129.565秒。`go vet ./...`、`go build ./cmd/...`通过，均CGO_ENABLED=0、GOTOOLCHAIN=go1.27.1。
- 前端33文件92测试、typecheck、生产build通过；`npm audit --omit=dev`为0漏洞。工具/资料快照9测试通过。浏览器74真实用例通过，一worker、retries=0，单批最长49.7秒。所有命令经540秒wrapper，Go5分钟内部deadline；没有延长预算或缩小最大容量以假通过。
- 基线原35路径、83schemas、11responses、3securitySchemes及全局security逐项深相等；00001—00004、原content/catalogue schema及content/digest无差异，原导出往返与摘要测试通过。生成两次API文件无Git差异。
- 只读确认真实开发库：`math_master|2|t`，即迁移版本2且question_packages不存在。本次不自动升级、初始化账户或发布真实数学内容。随机16位后缀测试库残留0，runtime.local.json不存在；原Knowledge_JSON未写入，资料快照保持独立。

## 最大合法容量及SQL计划

技术夹具包含10个真实已发布目标节点；每批20模板/1000实例/100蓝图，合并200模板/10000实例/1000蓝图，单节点池1000。模板及实例规范正文33,063,620 bytes（约31.53 MiB，≤32 MiB），manifest5,179,825 bytes（≤8 MiB），单包/冻结/题目其他限额同时满足。全部数据由真实编辑→生成→独立批准→prepare→activate创建，不用SQL伪造批准。

| 操作 | 最终本机耗时 | 累计分配bytes | 操作后堆bytes | 进程峰值驻留bytes |
| --- | --- | --- | --- | --- |
| 单模板最大可组成合法992实例 | 51.074 ms | 112,808,640 | 10,951,904 | 103,743,488 |
| 单批1000实例创建+验证+双次核验送审 | 1.8076—1.963727 s | 609,622,688—664,620,872 | 约38—43 MB | ≤105,857,024 |
| Prepare 200/10000/1000 | 1.560131 s | 1,329,877,240 | 103,244,456 | 271,089,664 |
| Activate完整候选 | 0.915208 s | 671,266,984 | 153,151,712 | 272,105,472 |
| 动态coverage完整题库 | 0.678419 s | 750,968,832 | 129,206,136 | 272,285,696 |

实际每个动作断言<8秒。累计分配代表GC周转量，不能当作驻留；Darwin getrusage.maxrss单位bytes，Linux测试记录KiB换算后的bytes。这里是本机单进程样本，P7仍需实际服务器并发/容量/恢复验收。

真实EXPLAIN ANALYZE BUFFERS：成员分页使用question_publication_members_pkey Bitmap Index/Heap Scan后top-N heapsort，读取11,200成员，返回100，执行2.444 ms、937共享命中/0磁盘读取/0临时写；按ID读取sealed当前发布使用question_publications_pkey Index Scan，执行0.005 ms、2共享命中。新历史/changes/正文筛选索引另由实际store行为回归覆盖，未修改旧表索引。

## 实施裁定（全量、原顺序）

以下完整保存执行账本所有Ruling及判断错误的成本，便于兼容性审核：

1. Ruling: 校验摘要使用独立用途 question-validation-v1 — expectedDigest须绑定输入、固定引用和全部实例，与题包/送审不同，计划未单列该用途 — 若错误会导致客户端摘要契约需修订。
2. Ruling: 当前发布/列表的描述兼容性仅在新question契约内处理 — 已批准设计要求旧ContentPackage及旧端口字节保持 — 若错误会影响旧导入和API消费者。
3. Task 3: Ruling: 用真实三家族31×32=992/32×32=1024测试生成器，再对通用预算函数测试1000/1001和4/5参数 — 首版每家族恰有两个命名参数，≤32值时无法组成1000/1001，不能虚构第四参数家族 — 若错误会遗漏通用组合预算边界。
4. Task 3: Ruling: 固定模板摘要采用独立用途question-template-v1 — qi身份必须绑定模板规范字节SHA并排除题包包装ID，计划未列模板用途标识 — 若错误会改变后续固定模板与实例身份。
5. Task 3: Ruling: 单模板规范实例字节累计超过4MiB立即拒绝 — 任一模板已超整包冻结上限时不可能合法，先阻止无效物化膨胀；跨模板合计和完整冻结仍由任务4判断 — 若错误可能提前拒绝本应合法的极大单模板。
6. Task 3: Ruling: 原创技术夹具引用fractions v2而不复用v1空目标 — 现有v1固定知识的Objectives为空且不能覆盖蓝图，新固定版本需另行审核；本夹具没有批准或公开 — 若错误会导致示例在真实发布前需要调整引用。
7. Task 4: Ruling: 每个固定蓝图保留一行带准确蓝图身份的覆盖依据，无蓝图知识另列未就绪行 — 设计不限定同知识只能有一个蓝图，不能遗漏其余待复核配置；全局题量仍按身份去重 — 若错误，P4b需进一步规定节点默认蓝图选择。
8. Task 4: Ruling: KnownObject中素材的version固定为0，仅知识/单元使用正数学版本 — 既有素材身份为ID＋SHA且没有数学版本，新题库不伪造素材版本或改变旧规则 — 若错误，新DTO素材判别分支需修订。
9. Task 4: Ruling: 固定所有被引用知识的核心与补充目标文本，并计入原4MiB冻结预算 — 补充目标在审核页也必须对应固定版本，不能改取当前版本文本 — 若错误，会扩大某些合法题包的冻结字节并要求分解过大的目标集合。
10. Task 4: Ruling: 蓝图固定正文采用独立用途question-blueprint-v1 — 规范蓝图SHA须与模板/实例/题包区分，计划的用途清单遗漏该独立成员 — 若错误，蓝图固定摘要及后续manifest需修订。
11. Task 5: Ruling: 提前更新任务13的随机E2E账户重置清单，显式纳入18张题库表 — 新外键使原重置在全套回归中被PostgreSQL拒绝；修复仅作用于已验证随机测试库且不用宽泛CASCADE — 若错误，会改变后续测试场景保留的题库夹具数据。
12. Task 6: Ruling: 补充question/archive.go的严格DTO解码、归档核验和实际引擎版本助手，以及store/question_references.go的批量公开引用解析 — Task6既有DTO已完整定义；让工具和后续HTTP共用严格边界，避免在CLI复制契约与信任逻辑 — 若错误，会扩大内部文件责任并影响后续调用接口。
13. Task 6: Ruling: 离线check使用--archive及--references两个文件，import只接受archive并重新查数据库当前公开引用，export新目录输出archive.json — 计划未定义CLI参数，Archive不含公开资格证明；离线快照不能授权发布 — 若错误，导出者需另外保留离线参考快照，工具使用说明需调整。
14. Task 6: Ruling: 作者预读后按既定顺序锁定账户，并在内容锁内重新计算作者；新增作者未预锁时拒绝本次修改 — 保持UUID排序用户锁在操作者会话之前，同时防止等待期间继承资料变化 — 若错误，罕见并发编写需要一次用户主动重试。
15. Task 6: Ruling: 在新迁移增加题包正文JSONB索引，作者继承查询先按完整待复用成员预筛选，再核对准确正文 — 避免草稿普通读写展开全部历史题包；仅新题库索引和查询调整 — 若错误，增加题包入库索引成本或影响作者查询性能。
16. Task 7: Ruling: HTTP校验摘要增加question-validation-proof-v1包装，绑定workspace/revision及可信作者和legacy标记 — 纯数学摘要不涵盖平台来源责任，不能让复制作者变化继续使用旧证明 — 若错误，客户端校验摘要及离线工具摘要的区分需调整。
17. Task 7: Ruling: 送审预读完整规范内容以发现应锁作者，取得内容锁后再次完整生成核验 — 原始草稿参数和规范参数字节不同，单纯预读原始正文会遗漏复制作者；锁内重算保证当前引用和数据一致 — 若错误，双次生成增加送审耗时，仍受同一8秒总期限限制。
18. Task 7: Ruling: editing工作区相同revision仅允许服务端来源不明标记单向增加及gate刷新 — 新继承的可信记录可能带legacy标记，来源正文、作者和修订输入不由客户端更改 — 若错误，数据库工作区不可变边界需收紧。
19. Task 7: Ruling: 实例缩页预算用实际序列化助手直接测试4MiB边界，真实冻结送审另测完整分页及权限绑定 — 合法送审的全部实例已位于4MiB冻结载荷内，无法构造合法单送审溢出页夹具；保持实际缩页保护 — 若错误，测试对未来页面扩展的覆盖不足。
20. Task 7: Ruling: 同一数学题包的新修订可更正sourceMap，旧送审来源逐字节保留 — 题包正文身份不含sourceMap，现有任务5的来源包含约束误把初次导入映射当成永远不可纠正；作者及legacy继承仍不可清除 — 若错误，来源更正可能需要改为新题包版本。
21. Task 8: Ruling: 内部BaseManifest/Candidate增加Templates与Blueprints正文数组 — 原接口仅给manifest身份和实例，无法检查旧蓝图的固定题源或替换模板闭包；批量载入当前版本正文，不修改任何旧公开DTO — 若错误，增加题库内部候选内存与后续接口维护成本。
22. Task 8: Ruling: 显式重新选取相同数学身份的成员采用本次批准依据，未选中的成员才继承当前head的历史依据 — 保证激活仍校验新选审核者资格，数学身份未变不增加差异计数 — 若错误，重新选择会改变当前审核依据，但历史manifest始终保留原证据。
23. Task 8: Ruling: 差异计数只比较成员kind/ID/version/SHA，题包归属及批准依据单独固定在manifest — 同一数学版本更换来源责任不等于数学替换，避免误导后续历史作答资格判断 — 若错误，管理页差异计数不展示仅审核依据变化，需通过成员依据详情查看。
24. Task 9: Ruling: 为纯内部当前可用/历史事实函数增加仅_test.go编译的授权测试包装，不增加P4a用户端口或Repository方法 — 计划要求分离这两种消费语义，但既定外部测试包无法直接调用私有助手 — 若错误，会增加测试接口维护成本。
25. Task 9: Ruling: 历史事实助手允许指定当时的题库publication，否则寻找该固定版本最早的已发布批准依据 — 现有HistoricalFacts单一Approval字段未带尝试上下文；P4b记录尝试后须传其实际publication以绑定当时证明 — 若错误，无上下文查询的批准依据可能不同于用户当时看到的发布。
26. Task 9: Ruling: 来源不可用的实例从实时题池排除，模板题源仍存在且其余实例可组成五题时蓝图可继续就绪 — 覆盖与可出题按有效实例重新计算，不能只沿用冻结ready，也不能因单个不可用题无条件停掉正确余题 — 若错误，需要进一步明确P4b对部分题源不可用时的配置策略。
27. Task 9: Ruling: 未配置蓝图的已发布节点只载入身份/前置关系并给出未就绪行，题库绑定知识仍读取完整固定目标 — 避免每次覆盖报告展开全部知识正文；知识回顾仍通过原固定知识入口 — 若错误，无题库配置节点的补充目标索引不会在覆盖报告列全。
28. Task 9: Ruling: 为固定历史版本和差异查询增加新题库成员历史索引及changes JSONB索引 — 历史查询不能逐份展开所有manifest/送审，只筛出与准确kind/ID/version/SHA相关的不可变记录 — 若错误，增加发布写入索引成本。
29. Task 9: Ruling: 完整响应包装参与分页字节预算，复用可选包装回调于发布/成员/差异/撤回/覆盖 — 4MiB是完整JSON限额，单独只计Page会漏掉head与统计元数据 — 若错误，会降低部分页面有效limit，客户端须按返回limit继续。
30. Task 12: Ruling: 文件表列出的八个实际页面全部实现，另增question-state.tsx及command-controls.tsx复用状态与手动重试对话框 — 任务标题写七页但具体路径为八个，按明确路径执行；共享助手避免权限错误状态及重试逻辑复制 — 若错误，会增加两个内部组件的维护范围。
31. Task 12: Ruling: 提前实现任务13的question_fixture.go及harness题库共享服务装配，先观察场景400失败再修复 — 任务12真实尺寸和键盘检查需可用的真实Next→Go→随机PG题库流程，不能伪造业务成功 — 若错误，会让任务13部分装配验证前移，后续仍必须运行全部四类浏览器回归。
32. Task 13: Ruling: 在任务13 Step5完成实现后先执行一次独立整分支审查，最终完成行延后到SSH PR及最新SHA CI真正完成 — 任务13本身包含审查/交付，而执行技能流程把审查放在全部任务完成行之后；优先满足实质审查与完成契约，不提前标记交付完成 — 若错误，机械记账顺序偏离技能流程，但不减少审查或验证。

## 独立审查、CI与保留事项

待整分支独立审查后填入结论、修复和全部暂缓小项。最新技术提交四项CI将通过PR/check链接交付，不以设计或旧提交CI替代。正式知识/题库数量仍为0；夹具不计入P6的20批准模板/300生产实例。不部署服务器，P4b须另行设计安全题面DTO、实际publication尝试、检测证据与历史解锁。
