# SDD ledger — plan: docs/superpowers/plans/2026-10-03-correction-impact.md

用户确认：2026-10-03，Native 全部 16 项执行，结束后一次独立整分支审查，SSH draft 产品 PR；产品合并/部署不在本次授权。
Baseline: dad438d13b37d1053e3bf42e7c32e3829e4518a6 (PR #22 merged); branch codex/p5b-correction-workflow.

## 跨任务预检

| producer → consumer | 固定契约 | 预检结论 |
| --- | --- | --- |
| correction → notification | EvidenceRef 单向别名；其他分页独立 | 无循环依赖 |
| correction model → store/httpapi/TS | 明确 nullable、UUID/安全整数、lowerCamel | 按计划类型表逐项同步 |
| equivalence/evaluate → process/projection | 五份原答案、注册算法1、完整累计依赖/案件 | 不重解释原答案 |
| migration → store | 十表、marker、冻结/归属/独立审查守卫 | 不覆盖旧七迁移/旧 qualification trigger |
| cases → learning/practice/assessment | 原证据即时 guard、DB cutoff、新尝试 guard | worker 禁用仍限制 |
| plan proof → result | 真实 publication/实例/参数/作者与独立批准 | 无映射不得授予 |
| jobs/backfill → worker | DB lease/fence、≤50、单并发/共享池 | 根完成不替代逐证据补扫 |
| projection → qualification/views | CorrectionID 可空；新事件与原 unlock 来源 | failed 原事实保持 |
| process → notifications | 结果/依赖/资格/通知/断点一个事务 | 失败全回滚 |
| read/exposure → HTTP/Next/UI | owner WHERE 先游标，受保护 detail 原与替代模板 | 元数据不含文字/答案 |
| API → TS/UI | 19 operations、严格边界/10秒身份总截止 | 同键恢复/内存草稿 |
| runner → server/CLI | 原十连接池、最多2；显式有限 CLI | shutdown 等待任务退出 |
| compatibility → CI | 原 schema 精确逆白名单，旧迁移/用途保持 | 不重写旧基线 |
| review → final regression/PR | 一次独立全分支审查、必要修复再完整验证 | 四个最新 head workflow 全 jobs |

未发现需变更已批准方案的接口冲突。

## 任务状态

Task 1—16: pending; 基线验证完成前不实现产品代码。

Baseline Go/Node/frontend pre: PASS; 24 Node, 283 frontend, old three capacity PASS; browser 17 batches pending. Logs verification.jsonl; no product changes.

Baseline complete: dad438d; 283 frontend / 130 browser (17 batches, 2 viewports, no retries) / 24 Node; all Go and old 3 capacity PASS. Counts [16, 12, 4, 4, 4, 6, 8, 4, 8, 8, 4, 8, 8, 6, 14, 8, 8].
Ruling: NextPlanState 使用内部 Action("approve"/"reject") 区分批准与驳回；Authorize 仅接受计划表中的对外 Action，两个内部动作不授权外部调用 — 原固定签名只给 Action，无法从 decidePlan 判断具体决定 — 判断错误的代价是送审状态被错误批准；负测覆盖 draft 不可批准和所有非 draft 不可修改。
Task 1: complete (commits dad438d..2635ba3, tests: python3 /private/tmp/math-master-p5b-tests.py go task-1-done test ./internal/correction ./internal/notification ./internal/assessment ./internal/learning ./internal/feedback -timeout 5m -count=1 → PASS task-1-done (0.98s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-1-done.log)
Task 2: complete (commits 2635ba3..8ed2514, tests: python3 /private/tmp/math-master-p5b-tests.py go task-2-done test ./internal/correction ./internal/assessment ./internal/question -timeout 5m -count=1 → PASS task-2-done (2.24s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-2-done.log)

Ruling: Task 3 同时调整旧反馈迁移测试 helper 为 DownTo(6)，明确测试 00007 的回退（先安全 Down 空 00008），不修改旧七迁移与断言 — 新迁移改变了单次 Down 的目标 — 判断错误的代价是旧反馈回退保护被漏测；Task3 加跑 TestFeedbackMigration。
Task 3: complete (commits 8ed2514..22b51a6, tests: python3 /private/tmp/math-master-p5b-tests.py go task-3-done test ./internal/store -run '^TestCorrection(Schema|Migration)|^TestFeedbackMigration|^TestLearningSchemaMigrationRoundTrip$|^TestMigrationRoundTrip$' -timeout 5m -count=1 -v → PASS task-3-done (12.96s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-3-done.log)

Ruling: 在既有共享内容锁后、账户锁前增加纠错登记 advisory fence（1296127047）：案件登记独占、个人学习事务共享 — 共享内容锁本身不能排除登记规则案件与资格提交的幻读竞态；该顺序不升级旧锁，读者之间仍并发 — cost if wrong: 锁顺序错误可能导致超时或并发授予失效资格；Task4/7/8并发测试及旧容量锁验证覆盖。
Ruling: 账户停用验证采用既有会话撤销、credential_version失效和must_change_password模型，不新增账户disabled字段 — 当前认证契约没有该字段，新增会改变获批范围外的旧账户协议 — cost if wrong: 运维若仅修改不存在的停用字段不会生效；文档明确既有撤销方式，后台核验现有账户/改密状态。
Task 4: complete (commits 22b51a6..65f8612, tests: python3 /private/tmp/math-master-p5b-tests.py go task-4-done test ./internal/store -run '^TestCorrection(Cases|Cutoff|WithdrawalFence|CommitIdentity|PartialConfig)' -timeout 5m -count=1 → PASS task-4-done (17.41s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-4-done.log)

Ruling: 管理员可按固定 PUT 契约编辑他人草稿；实际草稿编辑者加入冻结作者集合，00008 同时以真实 plan_updated 事件拒绝其自审 — 独立审批必须覆盖实际参与起草的人，同时保持 creator_user_id 不变 — cost if wrong: 管理员编辑后自审会绕过独立验证；专门 RED→GREEN 测试覆盖，旧七迁移不修改。
Task 5: complete (commits 65f8612..4aaee3e, tests: python3 /private/tmp/math-master-p5b-tests.py go task-5-done test ./internal/store -run '^TestCorrection(Plan|Independent|SourceProof|Replay|Sliding|Schema|Migration)' -timeout 5m -count=1 → PASS task-5-done (21.11s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-5-done.log)

Ruling: Task6自动登记撤回案件后，Task4真实来源测试改用真实SHA的旧写入夹具，Task5独立作者测试读取自动案件 — 避免重复创建同一案件而保持原负测；cost if wrong: 自动outbox与旧数据补扫可能未覆盖，Task6原子及legacy测试同时验证。
Task 6: complete (commits 4aaee3e..79ce243, tests: python3 /private/tmp/math-master-p5b-tests.py go task-6-done test ./internal/store -run '^TestCorrection(Enqueue|Lease|Retry|LateTerminal|LegacyTerminal|Rotating)' -timeout 5m -count=1 → PASS task-6-done (12.76s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-6-done.log)
Task 7: complete (commits 79ce243..1ae0994, tests: python3 /private/tmp/math-master-p5b-tests.py go task-7-done test ./internal/store -run '^TestCorrection(Projection|OtherCase|Completion|OriginalBytes)|^Test(Learning|Assessment)' -skip ^TestLearningCapacity -timeout 5m -count=1 → PASS task-7-done (88.68s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-7-done.log)

Ruling: Task8的终结任务plan=NULL可引用同案件实际独立批准的方案，00008同时核验任务owner/原evidence；显式plan任务仍要求相同版本 — Task6在终结时可能尚无批准，RED实证原严格相等守卫误拒合法晚提交 — cost if wrong: 可能绑定其他人的方案或证据，schema及worker归属负测约束；旧七迁移不变。
Task 8: complete (commits 1ae0994..e355c10, tests: python3 /private/tmp/math-master-p5b-tests.py go task-8-done test ./internal/correction ./internal/store -run '^TestCorrection(Process|WithdrawalCommit|CumulativeResults|StaleLease|CrashResume|NotificationDedup|Projection|OtherCase|Completion|OriginalBytes)' -timeout 5m -count=1 → PASS task-8-done (37.53s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-8-done.log)

Ruling: 同一纠错方案 ID 的新版本 created_at 在 00008 插入守卫中至少递增 1 微秒，并增加 UNIQUE(id,created_at) — 固定 {createdAt,id} 游标需能区分同 ID 的所有版本，同时 API 持有同案件锁串行分配；不变更批准的游标格式或旧表 — cost if wrong: 相同创建时间可能漏掉方案版本；精确同钟版本分页 RED→GREEN 测试覆盖。
Task 9: complete (commits e355c10..74818c6, tests: python3 /private/tmp/math-master-p5b-tests.py go task-9-done test ./internal/store -run '^Test(Correction|Notification)' -skip ^TestCorrectionCapacity -timeout 5m -count=1 → PASS task-9-done (101.45s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-9-done.log)
Task 10: complete (commits 74818c6..e5a0a5f, tests: python3 /private/tmp/math-master-p5b-tests.py go task-10-done test ./internal/correction ./internal/notification ./internal/httpapi -run '^Test(Correction|Notification)' -timeout 5m -count=1 → PASS task-10-done (22.8s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-10-done.log)

Ruling: Task11新增错误契约沿用真实认证层的 REAUTHENTICATION_REQUIRED/428 与 PASSWORD_CHANGE_REQUIRED/403，修正 Task10 新 schema 中误写的 REAUTH_REQUIRED，不修改既有认证接口 — 严格客户端否则把可恢复错误降为503，阻断计划要求的重新认证 — cost if wrong: 用户无法识别恢复步骤；Go实际响应/OpenAPI与TS回归RED→GREEN覆盖。
Task 11: complete (commits e5a0a5f..109a0de, tests: node tools/verify/run.mjs --cwd frontend -- npm test -- src/lib/correction src/lib/notification src/lib/api/correction-proxy.test.ts src/lib/api/notification-proxy.test.ts src/lib/learning/schemas.test.ts →    Duration  561ms (environment 59%, setup 15%, transform 14%, import 8%, tests 4%))
Task 12: complete (commits 109a0de..7c23a12, tests: node tools/verify/run.mjs --cwd frontend -- npm test →              learn more: https://vitest.dev/guide/improving-performance#test-environments)
Task 13: complete (commits 7c23a12..9f7e9f3, tests: python3 /private/tmp/math-master-p5b-tests.py front task-13-done test → PASS task-13-done (3.7s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-13-done.log)

Ruling: Task14用内部 ErrNeverEnabled 包装既有 ErrNotConfigured，五方法 WorkerRepository 签名不变 — 必须区分从未启用的安静等待与已启用损坏的显式退出，HTTP仍按同一未配置契约响应 — cost if wrong: 损坏状态可能被静默忽略；真实随机库从旧版本、启用到缺表的回归覆盖。
Task 14: complete (commits 9f7e9f3..15da0e4, tests: python3 /private/tmp/math-master-p5b-tests.py go task-14-done test ./cmd/server ./internal/correction ./internal/config ./internal/cli ./internal/store -run '^Test(Correction|Draining)' -skip ^TestCorrectionCapacity -timeout 5m -count=1 → PASS task-14-done (104.78s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-14-done.log)

Ruling: Task15补齐每事务 schema 健康检查：125约束的原定义/有效状态、17触发器的准确表/函数/延迟绑定、19函数签名、资格唯一索引和必需列；使用PG17.11 catalog减少重复查询但不缓存结果 — 缺约束或错绑触发器必须fail closed，实际兼容负测曾RED — cost if wrong: 损坏schema可能继续授予资格；十种破坏模式覆盖纠错、学习、通知入口。
Ruling: 根据实际1000案件/10000证据容量的8秒事务失败，对不可变case kind选用精确SQL predicate，并将同一结果的dependency逐行INSERT合成一条保留全部行守卫的INSERT SELECT — 不改变50批次/截止/排序/原事实，只减少不相关查询计划和往返 — cost if wrong: 漏扫或原子性损失；完整expected sets、dependency-last故障回滚及终结owner fence RED→GREEN覆盖。自审发现OR需整体括号，已恢复组合谓词边界。
Ruling: 最大历史容量复用真实独立作者/复核/发布事实，再以约束仍开启的SQL构造历史体量，不用1000次绕过配额的API请求 — 保持1000案件/1000批准方案/10000证据和10000通知同源重跑，逐页集合完全相等 — cost if wrong: 测试仅测空壳数据；真实冻结摘要、独立事件、FK/触发器和生产处理方法参与验证。
Ruling: 旧反馈容量的missing migration场景通过仅支持UpTo(7)的测试夹具创建从未启用纠错的数据库，再回退第七迁移；默认其他测试继续完整Up — 先Up8再Down会永久保留纠错启用标记，正确地禁止旧学习回退，不能清标记掩盖此保护 — cost if wrong: 从未启用的旧版本兼容行为被错测；增加无标记/无十表前置断言，保留旧容量数量、锁和原断言。
Task 15: complete (commits 15da0e4..489e150, tests: python3 /private/tmp/math-master-p5b-tests.py go task-15-done test ./internal/store -run '^TestCorrectionCompatibility|^TestCorrectionEnqueueWithdrawalTerminalOwnerFence$' -timeout 5m -count=1 → PASS task-15-done (10.75s) log=/Users/wiw/.codex/worktrees/p5-feedback-design/math_master/.superpowers/sdd/2026-10-03-correction-impact/task-15-done.log)

Task16 implementation/matrix: complete at d6f8c743a9385aeacf97644b2f5f6d6f1b729f60; 44 independently bounded commands, 345 frontend / 144 browser (20 batches) / 44 Node, all Go and five capacities PASS. Final review and SSH draft / latest full-head four-run delivery gates pending.
