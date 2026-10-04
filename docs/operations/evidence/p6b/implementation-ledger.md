# SDD ledger — plan: docs/superpowers/plans/2026-10-04-first-content-review.md

执行方式：Native。用户已确认方案、兼容边界和本实施计划。仅准备段 Task1—10 授权开发；正式 R1—R4 依赖实际人员/环境/操作范围，未执行。
分支：codex/p6b-content-review-preparation；SSH最新master：b78108d3dd363eae237dae59f76a7be774b9de85；携带已确认两文档提交后初始HEAD：e4c2b9deb07f6ea285807dd72ab14f7a77c61ddd。
当前文档PR26完整c0f4f6e6c7fb06e68491b27fe3dabf6922023f77四workflow/六job已实际success；历史容量超时未称作已修复。
基线：CGO_ENABLED=0/GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -timeout 5m -count=1 → PASS 1.413s；原日志 /private/tmp/p6b-baseline-contentaudit.log。

## Pre-flight 共享接口

| Producer→Consumer | 产物/约束 | 核对 |
| --- | --- | --- |
| 1→2/3/8 | SelectedDraft/LoadSourcesFromBytes；同字节校验/导出/来源 | 私有校验主体提取，原公开行为保留；根路径仅新入口要求规范 |
| 2→3/4/6 | Scope：958对象、574映射/384派生、205来源、必需检查 | 派生为索引子集，不重复计数；纯层核对原字节与typed值 |
| 3→4/9 | 六现有DraftInput、SourceLink合并、SVG捕获 | 去重键不含LegacyID；API和server作者规则不变 |
| 4→5/6/7 | 私有Bundle/manifest文件摘要、原始未检查登记 | manifest不自摘要；最终登记另存，按完整身份/检查匹配 |
| 5→8 | 原子输出/预算/取消 | 不覆盖，失败只清本次新目录，manifest最后写 |
| 6→7 | frozen/decision/author/reviewer及真实独立性承诺 | 题库archive按frozen identity顺序复算；工具只验证一致性 |
| 7→8/R4 | 原schema1 AcceptanceEvidence/八检查 | 缺真实复核/未执行不输出证据；fixture标记贯穿；当前DB最终核验 |
| 8/9→10 | TestContentReview/纯层/新Node保护 | 原宽入口不跳过新组，原72/350/22批162/六容量保留 |

## TODO

- Task 1: complete
- Task 2: complete
- Task 3: complete
- Task 4: complete
- Task 5: complete
- Task 6: complete
- Task 7: complete
- Task 8: complete
- Task 9: complete
- Task 10: pending

R1—R4: pending external preconditions; formalCounts=0.

Task 1: Ruling: 共享严格解码器只接受其既有封包上限内的limit，不能传原目录解码器10MiB上限 — 新入口目录元数据按计划4MiB读取/扫描，原LoadDraft默认10MiB行为不变 — 错误成本：未来超过4MiB的目录输入需另行审查预算。
Task 1: complete (commits e4c2b9d..cd00cd0, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentaudit	1.617s)

Task 1 steps: 1.1 tests / 1.2 API and unsafe-input RED / 1.3 implementation / 1.4 selected + whole suite GREEN / 1.5 commit done. Original defaults unchanged; 16 captured files, selected v2+v1, safe file/asset/source boundaries verified.
Task 2: Ruling: 原字节空白变化不代表输入与解码值矛盾；负例改为真实字段变化 — 映射修订按实际原字节 SHA 记录，允许合法空白 — 错误成本：后续消费者必须校验原字节 SHA，不能仅比解码对象。
Task 2 steps 2.1–2.4: 全量、修订模板 SHA、末尾参数、205 来源、五证明及非法身份测试完成；RED API 与行为日志保留；GREEN 全包 1.968s。
Task 2: complete (commits cd00cd0..59a95e5, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentreview	1.648s)
Task 3: Ruling: 路线汇总出处不能扩散到每个知识点；导入用对象本身的准确知识关联，要求每条路线出处已有该关联 — 全部路线出处仍在 sources.json，避免无关引用及现有 256KiB 来源数组超限 — 错误成本：仅有路线引用但无知识关联的新来源需先明确映射，否则拒绝导入。
Task 3 steps 3.1–3.4: 已完成来源合并、现有严格导入解码、未知/缺资产/漏用途/超限/错误范围测试；RED 与 GREEN/实测导入字节日志保留；全包 3.045s。
Task 3: complete (commits 59a95e5..f1bcd76, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentreview	2.710s)
Task 4: Ruling: manifest.files 不收录 review-register.json/register 分片，登记根及分片绑定最终 manifest 原字节 SHA，并由根保存分片 SHA — 避免 manifest 与登记相互摘要形成不可求解循环；全部导入/材料/图/来源仍受 manifest 摘要约束 — 错误成本：验证器必须单独验证登记根和分片及完整行集。
Task 4: Ruling: 私有 ReviewManifest 增加 requiredChecks[] 准确集合 — 无证明对象身份无法从 SHA 推断 proof 要求；必须把必需检查集合纳入 manifest 绑定，防止最终登记删项 — 错误成本：私有 schema=1 的读写两端和文档须同步；现有运行时契约不变。
Task 4 steps 4.1–4.4: 1163 行初始未检查、958 材料身份、完整 834 实例/五证明/九图/205 来源、HTML/Markdown 惰性文本及分页/确定性摘要通过；全 contentreview 5.122s。
Task 4: complete (commits f1bcd76..eedbb9a, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentreview	4.774s)
Task 5 steps 5.1–5.4: 已有目录/文件/symlink/FIFO 不覆盖；写/关闭/取消失败只清理自身目录；文件数256/文件8MiB/总256MiB临界及+1、摘要/安全路径/父symlink/0700与0600测试通过。8MiB实际写入耗时见 task-5-output.log；全包 5.172s。
Task 5: complete (commits eedbb9a..617b958, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentreview	4.766s)
Task 6: Ruling: 增加仅离线调用的 question.DecodeOperationalJSON（8MiB）共用原扫描器/严格形状，新增 int32 范围检查；原 DecodeStrictJSON 仍限定4MiB — 私有8MiB文件预算不能用原4MiB入口实现，拒绝扩大HTTP上限或复制扫描算法 — 错误成本：需在整分支兼容审查验证该入口无运行时调用、原解码行为和所有旧测试保持。
Task 6: Ruling: evidence-root 包含原复核包的不可变 files，验证时复算这些 SHA，冻结 SourceMap 与其中六导入逐字字段对照；Verification 增加内部 reviewComplete 诊断 — manifest 中只有对象身份无法独立重建每个映射别名的准确 Note；用已绑定导入消除猜测，诊断用于测试和交接 — 错误成本：操作手册必须要求保存/带入完整私有包，缺文件会拒绝校验；外部既有 Evidence 不变。
Task 6 steps 6.1–6.4: 原顺序 CanonicalFrozen、严格 Archive/作者/来源/批准/全量行集、未签/未检查/returned/空依据全测通过；原始导入先 ValidateAndSeal 后比较规范化包 SHA。行为全测21.510s；旧question2.213s、contentaudit2.818s；8MiB离线临界及原4MiB保护测试通过。
Task 6: complete (commits 617b958..9183630, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentreview	26.273s)
Task 7: Ruling: 未落实实际双 head 时，ReleaseContext 两字段可同时留空并保持 awaiting_review；半填或非法身份仍拒绝 — 缺环境不能靠编造 head 填模板，且不能产生 AcceptanceEvidence — 错误成本：操作方须在正式执行前补齐真实双 head，并重做绑定全部八文件。
Task 7 steps 7.1–7.4: 全八记录、12类最终上下文差异、原字节/附件SHA/路径/重复/状态/观察/签署、64附件及+1、实际256MiB读取及+1通过；failed不改passed；缺项不写证据，完整字节由原DecodeEvidence解码。全纯层67.381s，正式计数仍0。
Task 7: complete (commits 9183630..b6527fa, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/contentreview	67.612s)
Task 8 steps 8.1–8.4: 两子命令真实文件、完整fixture/v2明确选择、退出0/3/2、未知/重复/DB/发布参数、规范根/FIFO/预算/取消及脱敏日志通过；cli新组8.625s，CGO_ENABLED=0命令构建完成。中文全量清单/手册含架构及正式前置；构建输出使用既有忽略bin目录。
Task 8: complete (commits b6527fa..a6df3d7, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/cli -run ^TestContentReview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/cli	8.397s)

Task 9: Ruling: 旧示例已占 elementary-foundations/v1 的不同路线内容；正例采用同一旧目录/知识且 Paths=[] 的首次路线隔离基线，反例保留旧路线验证必须拒绝 — 不覆盖已发布不可变路线，也不改变生产规则 — 错误成本：实际环境有旧同版本路线时，R2 必须追加路线版本及准确清单/映射并重新复核。
Task 9 steps 9.1–9.3: 正常 auth/session/CSRF/idempotency、知识与五题包导入/固定送审/独立审核/双 head 发布/正常导出及完整原始字节身份对照通过；旧同版本路线和错误知识 head 拒绝，非法 SVG 草稿可编辑但不能送审，技术身份与整体 awaiting_review 保持。真实流程测试 RED→GREEN 6.093s。
Task 9 smoke: 固定快照首批原字节 prepare=0，958对象/384派生/205来源/64映射/5证明/1163登记；未落实真实审批与 heads 的 verify=3，未生成验收证据，正式计数0。公开 preparation.json 的 codeSHA 为实际运行 a6df3d7424766ea29a758fa3c126d3507d8b1cbc，不重标后续文档提交。
Task 9 step 9.4: 原宽集成入口（skip 原样）通过：store 147.438s、cli 10.596s；未漏 TestContentReview，正式数量仍0。
Task 9: complete (commits a6df3d7..5e72307, tests: python3 /private/tmp/math-master-p6a-test.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run ^TestContentReview -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/store	6.509s)

Task 10 steps 10.1–10.2: 284项既有运行时/契约/数学字节基线；新 CI/兼容9测从缺新增纯层批次的真实 RED 转 GREEN。所有旧六容量/22浏览器及改名/删组/skip/CGO/预算负例先从通过副本逐项破坏再拒绝。
Task 10 matrix correction: 首次 go vet 检出新 CLI 测试三处跨包未键控结构字面量；只改测试为显式字段，原始 vet RED 日志单独保留，未改运行时契约。
Task 10 matrix progress: 原 Node72+新9=81/81，前端350/350、生成/typecheck/build/audit无差异全部通过；原Go内容/HTTP638、宽集成190、学习测评142、反馈48全部零跳过；容量题源本次89.819s和最大路线124.169s通过，未宣称旧慢查询根因已修复。
Task 10 matrix evidence: Go 使用 -json 仅改变输出表示，保存实际 command 与原 workflowCommand，统计 Test pass/fail/skip（含具名子测试）。每批预算和所有筛选原样。公开原日志以 gzip 保存并复算摘要；绝对工作树前缀脱敏时分别保留原 SHA/存储 SHA，完整原始字节另保存在受保护私有目录。
Task 10 matrix progress: 新纯层99/99、原内容验收纯层59/59、原只读Store/CLI19/19、纠错通知164/164全通过且零跳过；现已完成14/16批Go矩阵，六容量剩末两项。
Task 10 evidence correction: 临时矩阵记录器新增 harness 模式时误改了计数分支缩进，首次仅 Python 语法错误、未启动测试；修正临时脚本后，当前分支隔离 harness 构建通过0.429s。测试/产品未受修改。
Task 10: Ruling: Task10 文件清单未重列 Task8 的 CLI 测试，go vet 实际发现三处跨包未键控字面量 — 将这三处测试改为显式字段名纳入本任务窄范围修复，不改输入/输出/运行时；vet RED→GREEN，全 CLI 已由宽集成入口回归 — 错误成本：字段名若写错会构建失败或测试行为变动，当前完整回归已覆盖。
Task 10 Go矩阵: 16/16批全部 exit0，六项容量全部实际通过；1376个Test pass结果事件（含子测试）、fail0/skip0。当前分支正常隔离 harness 已重建，开始原22浏览器批次。
Task 10 browser progress: 8/22批、58项 passed，failed0/skipped0；逐批正常启动独立harness。
Task 10 browser progress: 14/22批、100项 passed，failed0/skipped0。
Task 10 review focus supplement: 请独立核对 verifier 边界的 manifest 原字节绑定，包括外部JSON重排字段/空白后与重新编码摘要的关系；此为待审查行为，不预判严重级别。
Task 10 browser progress: 原前20批/146项全部 passed，failed0/skipped0；最后首批完整路线两批进行中。
Task 10 steps 10.1–10.3: 9新保护RED→GREEN；原72+新9=81 Node、350前端、22批162浏览器、16批Go1376结果及全部六容量全部通过，零跳过。现代码与完整本地矩阵完成，10.4唯一fresh整分支审查及10.5 SSH draft MR/准确最终head CI交付门槛待执行，不宣称整个Task10交付完成。
