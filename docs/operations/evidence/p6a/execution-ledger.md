# SDD ledger — plan: docs/superpowers/plans/2026-10-04-first-content-preparation.md

执行方式：Native；用户已确认设计、兼容边界与实施计划。产品分支 codex/p6a-content-preparation，SSH 最新 master 基线 8bf3c95b9833941182f4c8caa2da752e6d7838ce；已批准文档来自 c65992f/a4b0104。

待办：Task 1—10 已完成，十任务/五十步骤；完整源代码 head 四workflow/六job通过。证据归档提交的准确最终head还需以PR当前检查验收。

Pre-flight 1→2：IndexAudit 三索引联合/来源报告消费同字段，null 声明摘要不改变旧快照致命错误。
Pre-flight 2→3：SourceReport 实际 bytes SHA 与 SourceMap.policyVersion/snapshotId匹配；全部问题计数与选中阻塞分别处理。
Pre-flight 3→4：DraftInput content.Package 单包三十节点前置闭包；素材 ObjectIdentity.Version=null。
Pre-flight 3/4→5→6：Questions 是五个 QuestionPackage，不是 Archive；references 来自本地正文，head=null；blueprint SHA 来自 ValidationReport.Coverage.Blueprint。
Pre-flight 3→7：PublishedFacts 包含一致 head/SHA/审批/撤回/准确主归属；纯层不访问 DB；最大合法体积和元数据预算区分。
Pre-flight 3/7→8：RunContentAudit 显式模式/环境变量名；SELECT只读；fixtureOnly不能取消。
Pre-flight 4/5/6/8→9：新 scene 使用原工作流技术夹具，正式计数始终零，原 runtime DTO不改。
Pre-flight 1—9→10：追加测试/CI，保留旧44Node、350前端、146浏览器/20批/五容量及原预算。
Ruling: 已确认计划尚在未合并的文档 PR #24 — 从最新 master 新建产品分支后带入两个已批准文档提交，不擅自合并 #24 — 若后续单独合并文档 PR，需要核对相同文档变更。
Ruling: Actions启动限制在用户重查时已经解除 — 更新计划的事实描述，最终实现仍核验完整新SHA的CI结果 — 若账号再次受限，MR准确记录阻塞，不放宽测试。

Task 1 RED: 6新增行为失败，旧及安全行为通过；完整日志 task-1-red.log。GREEN: 14/14。新原字节批次 p6a-20261004T045040522486Z，历史 P1 manifest SHA未变。
Task 1: complete (commits a4b0104..c7b41ce, tests: node tools/verify/run.mjs -- node --test tools/content-ingest/source-index.test.mjs tools/content-ingest/snapshot.test.mjs → ℹ duration_ms 434.82575)

基线环境修正：已有容器默认库名 math_master 被测试隔离护栏正确拒绝；仅将私有测试辅助脚本的 URI 路径改为 math_master_test_p6a，连接器仍只创建随机隔离数据库，不放宽产品护栏。新基线结果见 baseline-go-fixed.log。
Task 2 RED: 8/8 来源入口缺失失败；GREEN: 8/8，私有真实报告复算 SHA，选中 205 条、8 项非选中差异、ready=true/publicationApproved=false。
Task 2: complete (commits c7b41ce..7400900, tests: node tools/verify/run.mjs -- node --test tools/content-ingest/source-report.test.mjs → ℹ duration_ms 388.782833)

Task 3: Ruling: 计划文字 ValidateWorkflowDraft 名称在既有代码中不存在 — 使用 content.ValidateWorkflow 这一实际草稿结构/完整性校验并验证 seal，原校验器不改 — 若误判其职责，最终原工作流回归会暴露遗漏。
基线 Go 修正后：15 个指定包全部 PASS，HTTP 62.7s、e2etest 97.6s，均未超预算。

Task 3 RED: 新类型缺失编译失败；补充冻结样例 RED 为文件不存在。GREEN: 路线 29/30、主归属、曝光反例、1000/1001、元数据上界、来源身份、结论/隐私、原工作流离线 head=null 均通过。冻结报告已核对 30/24/450/384/834 草稿与正式零数，不含题面或个人信息。
Task 3: complete (commits 7400900..c399ce9, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentaudit	0.444s)

Task 4 RED: 包和素材缺失；GREEN: 30节点/30单元/70编辑对象/9SVG及旧包摘要通过。原 content-check errors=[]；本地 Chromium 九图目视完成，面积图单位整体改为正方形、十单位图面积一致。作者检查明确不等于实际独立数学批准。
Task 4: complete (commits c399ce9..ba81995, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -run ^TestContentAuditEditorial -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentaudit	0.237s)

Task 5 RED: 两题包不存在；GREEN: 106/295=401 不同草稿实例，11模板/225固定/176参数实例。176实例全 Verify、每模板16、15节点曝光见证、四选项/正确键、数值golden、除零/篡改/伪witness全部验证。
Task 5: complete (commits ba81995..3e5abfb, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -run ^TestContentAuditQuestionsNumbersOperations$ -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentaudit	0.293s)

Task 6 RED: 后三题包缺失；GREEN: 五包24/450/384/834，重复0、30蓝图/30主节点曝光覆盖通过；原参数逐个Verify，完整纯层测试PASS。来源映射逐对象单行排版，实际约204KiB以内，符合256KiB；原数学摘要不变。
Task 6: complete (commits 3e5abfb..17ce995, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentaudit	0.832s)

Task 7 RED: 入口缺失；GREEN: 真实双head屏障/SELECT角色/6种撤回/4种证据损坏/纠错限制/1s锁等待通过。最大200/10000/1000题库和1000知识/4000单元/200路线/1000资产同时最大正文通过；32/8MiB+1拒绝。容量总66.54s，最大内容读取1.698s，命令仍≤8s；查询计划及原始体积见 task-7-capacity.log。
Task 7: complete (commits 17ce995..02be3b9, tests: python3 /private/tmp/math-master-p6a-test.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run ^TestContentAudit -skip ^TestContentAuditCapacity$ -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/store	13.831s)

Task 8 RED: 新命令未定义；GREEN: 参数/无默认DSN/旧目录保护/输出权限/秘密脱敏/SELECT角色/旧代码证据拒绝通过。真实原字节来源 CLI draft_ready/0（30/834，正式零）；隔离真实五包批准发布 CLI awaiting_review/3 自动fixtureOnly。构建及中文运维说明完成。
Task 8: complete (commits 02be3b9..015c699, tests: python3 /private/tmp/math-master-p6a-test.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/cli -run ^TestContentAudit -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/cli	4.703s)

Task 9: Ruling: 新真实双目标文本令原学习面板 select 的固有宽度在390px视口溢出（RED截图999px页面宽） — 新增一个局部CSS宽度约束，修复真实数据布局；组件接口、API、学习策略均不变 — 长选项显示受原生select限制，完整350前端/旧146浏览器及最终独立审查复核兼容性。

Task 9 RED: Go新场景缺失，两浏览器组各8失败；GREEN: 30节点首次/实际单题曝光/五题真实提交、原draft/question/learning/feedback/correction重置通过。两视口reading8/8（40.6s）、learning8/8（1.1m）；九图真实加载与脱敏截图、0未确认自动重试。长公式沿用原long场景，图文与固定v2完整。布局RED→局部CSS→生产构建/移动GREEN。
Task 9: complete (commits 015c699..9d1cd05, tests: python3 /private/tmp/math-master-p6a-test.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/e2etest -run ^TestContentAcceptance -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/e2etest	18.935s)

Task 10: Ruling: 旧CI保护将浏览器总数锁死20且广泛store的skip字符串固定，直接追加会让保护本身失败并重复新容量 — 仅允许两组具名新增浏览器、原skip追加ContentAudit并独立运行新常规/容量，保留44旧测试及其全部负测；同步两旧CI测试的精确预期 — 不允许删除旧批、跳过原测试或增大预算，独立审查核验此分类。

Task 10: Ruling: Playwright把旧 reading.spec.ts 参数当子串正则，实际会额外选中新 content-acceptance-reading.spec.ts — config将裸文件参数同时约束为准确basename glob，旧20批命令不变且仅执行原146，新两批执行16；原workers/retries/480000不变 — 非裸文件正则参数仍使用原默认全集，实际 --list 和全22批复核范围。
Task 10 回归发现百分数测试助手将分数直接附%而原输入仅接受十进制百分数；修正技术助手为精确有限十进制，并对834实例全部GradeAnswer校验，产品判分/数学正文不改。

Task 10: Ruling: 批准计划的Step4要求整分支审查先于Step5提交/MR，Native通用流程则把审查置于task-done之后 — 以批准计划交付门为准，审查包包含base..HEAD及Task10全部未提交补丁/新增文件，完成一次审查与验证后再提交/MR/task-done — 若补充包遗漏则存在未审查改动风险，记录其文件清单与摘要并由执行者核对，不增设复审席位。

Final review: fresh gpt-6-astra / p6a_final_whole_branch_review，只读base..9d1cd05+Task10 pending，无子审查；Critical0，Important3，Minor2。数学独立枚举384参数组合、24模板与每节点固定01/06/11/15抽查，技术审查不是全量人员数学批准。原始完整Go16命令与浏览器20/146+2/16均PASS。
Final re-grade: Important1重复证据、Important2来源IO预算/同字节、Important3无eligible实例模板计数均保留Important，进入同一修复批；Minor1发布manifest回退与Minor2中文摘要信息保持Minor，真实CLI未发现绕过且JSON保留详情，后续再优化。
Final: minor (deferred): published纯层缺manifest时从SourceMap回退blueprint SHA；当前Store已提供核验manifest，未来调用方需要更明确契约。
Final: minor (deferred): 中文Markdown报告尚未显示双head和逐节点具体原因；完整JSON已保留，摘要可在后续补充。
Final: Ruling: 审查者不能判定真实人员数学批准与自然人独立性 — P6a只完成技术机制，交P6b真实人员核验；当前正式数零 — 若人员门被误当技术门，可能造成未经独立复核的错误发布。
Final: Ruling: 审查者未逐条判定205来源出处/许可/条件匹配 — 保留不可变SHA与映射线索，P6b逐条复核来源，不声明已许可或正式批准 — 若线索错误，需退回对应内容/题目而不能正式上线。
Final: Ruling: 审查者只抽查固定题、正文与部分图，未全面认证450题/全部证明/九图 — 参数全量验证与抽查支持技术交付，全量人员数学复核留P6b — 若未查出的数学错误存在，P6b必须修订后重新验证。
Final: Ruling: 同节点多个已发布blueprint的选择未判定 — 本批每节点唯一，工具对多蓝图失败关闭；后续支持多蓝图需另设计选择规则 — 若未来合法多蓝图进入本路线，工具会拒绝报告，需明确适配。
Final: Ruling: 审查时最终Go矩阵/证据/远端CI尚未全部可判定 — 执行者补齐原始矩阵与manifest，并在push后核验准确完整head所有job — 若远端未通过，应保留MR阻塞，不能用本地或旧PR替代。
Final: Ruling: 生产部署/迁移/真实发布/MR合并未判定 — 本次授权只创建实现MR，不执行这些后续动作 — 若过早执行会产生授权外副作用，分阶段交接。
Final: Ruling: 来源正文不能套用4MiB元数据预算，但按声明大小全量捕获仍需内存上界 — 原始corpus独立64MiB上界、声明大小+实际文件一致性、8秒取消/普通文件、同份捕获字节摘要与解析；SVG原1MiB/其他原预算不变 — 若未来单来源超过64MiB会失败关闭，需另设计流式大文件入口，本批实际来源在界内。
Final: fixed Important1重复验收证据 — TestContentAuditEvidenceConflicts完整正向与两顺序/同值/冲突值/未知名称 RED→GREEN；纯层、Store、CLI全部常规验收组PASS。
Final: fixed Important2来源IO预算与同字节 — TestContentAuditSourceReadBudget/SourceSpecialFiles/SourceDeclaredBytes/SourceCorpusCapacity/DraftSpecialAsset/DraftCancellation RED→GREEN；普通文件、单份正文、独立64MiB、取消错误传播；全验收常规组与最大容量PASS。
Final: fixed Important3失效模板门槛 — TestContentAuditTemplateEligibility/TemplateDependencyEligibility RED→GREEN；模板专属unit v1→v2使生成实例不可用，15固定题仍保留；24→19模板报告not_ready，原先accepted；全部验收组PASS。
Final fix commit: 4d27dc8f1564bd0ed5b5fa0d4ac1a5a8c504c8ab；3/3 Important已修复；Critical0；2 Minor保留；单次修复批，无复审。Node68/68、原350前端与162浏览器、原Go完整矩阵、修复后全验收常规与容量66.68s、vet/build通过，后续准确远端head完整CI另核验。
Task 10 本地交付提交: 2c587c6df0b592b8cd74e954be2347c753f99abe；CI新增分组/兼容保护/完整本地证据入Git，最终7项新保护PASS。待SSH推送、实现MR与准确完整head远端CI；当前不记complete。
Task 10: SSH推送完成，实现PR #25 https://github.com/yyl1212/math_master/pull/25 已创建并附加。完整2c587c6df0b592b8cd74e954be2347c753f99abe 的push/PR四workflow、六job实际startedAt已核验；仍in_progress，不预宣称success。

Task 10 远端 RED: 完整2c587 head的push frontend在新阅读组第6项准备场景失败（5.2s），其余7项通过；PR frontend及两backend均通过，仅5/6 job success，不记complete。失败附件页面未导航，无页面断言错误；原传输助手把所有异常隐藏为相同文本，无法从该历史日志区分底层异常，截止原因由5秒边界与慢响应复现支持。
Task 10: Ruling: 新content-acceptance真实工作流准备30知识点/五包834题，继承旧5秒场景传输预算导致CI在准备阶段边界失败 — 将助手抽至tests/e2e/scene-control.ts，仅该具名新场景上限10秒，其他场景仍5秒；单次用例30秒、8m/9m/30m总预算和零重试不变。慢HTTP5.5秒真实响应、旧场景预算、失败单次调用与脱敏均RED→GREEN验证 — 若未来准备超过10秒仍失败，需改善夹具性能；历史异常无法恢复其具体原因，因此保留原始RED与这一推断的证据边界。
Task 10 准备修复 GREEN: Node72、前端350/typecheck、真实原公共阅读16+新阅读8+新学习8全部通过；三批36.477/40.858/66.044秒。原first-head 5/6失败保留，待新完整head六job。
Task 10: complete (commits 9d1cd05..fca745e, tests: node tools/verify/run.mjs -- node --test tools/verify/content-acceptance.test.mjs tools/verify/content-acceptance-ci.test.mjs → ℹ duration_ms 5584.788166)

P6a 收尾归档：全部15条Ruling与两项minor按原序保存；私有来源输入已留在被忽略固定快照的p6a-inputs.local.json，原快照/files和用户资料不变。只清理本计划scratch，保留工作树、其他计划和快照；最终归档head的六CI结果在PR #25与会话交付记录中核验，避免报告自引用Git SHA。
