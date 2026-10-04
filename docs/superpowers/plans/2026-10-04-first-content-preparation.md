# P6a：首批初等数学草稿与验收工具实施计划

> **供实施者使用：** 必须使用 superpowers:executing-plans，在当前会话按任务及复选步骤执行；沿用用户已选择的 Native 方式。实现结束进行一次独立整分支审查及回归验证。

日期：2026-10-04。状态：P6 书面设计与兼容性边界已获用户确认；本实施计划待审阅，尚未开始产品实现。设计分支 codex/p6-content-design，基线 master 为 8bf3c95b9833941182f4c8caa2da752e6d7838ce。计划批准后先通过 SSH 获取最新 master，在干净隔离工作树新建 codex/p6a-content-preparation；记录实际基线，不能把本文件旧 SHA 当作最新代码。

**Goal（目标）：** 完成可追溯的 30 节点原创草稿、题库、原始资料批次检查、准确的只读验收工具和隔离联调。P6a 的交付结论为草稿技术就绪并等待真实独立数学复核；P6b 才完成正式内容审核、发布和 30/20/300 验收。

**Architecture（架构）：** Node 固定原字节并生成来源报告；Go 复用现有内容、题库精确校验和 FiveQuestionCover，分别检查草稿与一致事务读出的发布事实；运维 CLI 生成 JSON 与中文 Markdown。网站消费既有 API，所有模拟发布仅在随机隔离库中执行。

**Tech Stack（技术栈）：** Go 1.27.1（CGO_ENABLED=0）、Node 24.17.0、PostgreSQL 17.11、仓库已锁定的 Next.js/TypeScript、Node test、Go test、Playwright；不引入新依赖。

**Spec（已批准方案）：** [P6 首批内容与质量验收设计](../specs/2026-10-04-first-content-acceptance-design.md)，尤其第五、七、八、十一节。

## Global Constraints（全局约束）

- 正式门槛保持至少 30 个不同知识 ID、20 个不同且当前有效的批准模板、300 个不同有效实例；每节点至少 10 个有效实例、5 个检测候选，并有恰好五题覆盖全部核心目标的见证。正式数量与草稿数量分别报告，固定题与生成实例分别报告。
- 本次编写目标锁定为 30 知识、30 学习单元（各至少两种角度，共至少 60 角度）、1 条固定路线、9 个原创 SVG、24 个有不同教学目的的模板、450 道原创固定单选题。每模板 4×4=16 个有效组合，目标 384 生成实例、834 不同实例；实际以独立校验和去重结果为准，不能以算式代替实测。
- 每节点固定题 15 道：5 道只检测目标 0，5 道只检测目标 1，5 道同时检测两个目标。后一组的题意必须真的同时要求两项能力；不合适时调整题目内容，不能只补 coverage 字段。所有题的主 knowledge 为本节点准确版本，单元为本节点准确版本，禁止靠其他节点 secondary coverage 计入检测数。
- 每个 blueprint 的 coreObjectiveIndices=[0,1]、ruleVersion=1、questionCount=5、passCount=4；来源列出本节点全部固定题准确 ID/版本和模板准确 ID/版本。参数引擎仅 rational_arithmetic、rational_comparison、missing_operand，generatorVersion/verifierVersion 均为 1。不增加求解器或改动既有判分、输入语法、曝光冷却、最近一次提交排除、资格或角色规则。
- 固定题无数值见证时只做结构与答案键完整性检查，数学正确性仍待逐题人工复核；不得给概念题附上无关数值见证。生成器与独立校验器继续分别执行；源码内 review 标签、AI 检查和测试账号不能证明实际人员独立性。
- 知识包一次装入完整前置闭包，避免现有 content 校验器不允许跨包关系引用的问题。题库拆五包：numbers、operations、fractions、decimals、ratios。逐包不超过 50 模板、200 固定题、1000 生成实例、100 blueprint；总库不超过现有 200/10000/1000 和正文 32 MiB、manifest 8 MiB 限制，不放宽容量。
- 原 source、P1 快照、旧十节点草稿、旧技术题库、00001—00008 迁移、公开 API/OpenAPI/生成类型和数学摘要用途保留。只读工具不导入、迁移、审核、发布、修改学习记录或触发纠错任务；输出不带用户答案、凭据、会话、内部 Library URL 或私有工单正文。
- 源目录是只读操作参数，首次执行重新固定完整原字节；快照写到全新的 content/snapshots.local/p6a-<UTC时间>。时间名称实际生成且不得复用。预检的 2214/2213 和八项差异仅为历史时点，不能成为恒定断言。
- Native 按任务顺序执行。每任务五步；Go 单次 5m、浏览器 8m、包装命令 9m、CI job 30m，所有单次测试小于 10 分钟。新测追加到既有矩阵，保留旧 44 项 Node、350 前端单测、146 浏览器用例/20 批及五项真实数据库容量验证的行为覆盖。
- 当前 Actions 因账号付款或支出限额未启动，属于交付门槛。可以完成本地实现与 MR；准确新 head 的远端 CI 未真正成功前不能声称通过或合并，不能用修改工作流、购买服务或重复重跑处理账号限制。

## Review Focus（审查重点）

| 易错输入/失败类别 | 必须覆盖的具名测试及归属 |
| --- | --- |
| 源库采集中变动、旧一/两索引输入、第三索引同路径冲突 | Task 1：snapshot legacy digest / third-index conflicts / source-changed cleanup |
| legacy/章节重复、条件不同、选定条目缺失，AI 标签被当批准 | Task 2：source report preserves provenance / selected-only blocking / AI is not approval |
| 全站数量或 secondary coverage 假达标，曝光后核心目标失去覆盖 | Task 3：TestContentAuditRouteCounts / TestContentAuditPrimaryBinding / TestContentAuditExposureWitness |
| 读到混合 head、正文 SHA 不符、撤回或审批证据缺失 | Task 7：TestContentAuditConsistentHeads / TestContentAuditWithdrawal / TestContentAuditEvidenceMismatch |
| 随机测试库伪装正式报告、CLI 写入/泄密、复用旧联调夹具使旧场景失效 | Task 8/9/10：TestContentAuditFixtureCannotAccept / TestContentAuditReadOnly / 新浏览器组 / CI 保护 |

## 文件职责与架构图

| 新增/修改文件 | 单一职责 |
| --- | --- |
| tools/content-ingest/source-index.mjs、source-index.test.mjs（新增）；snapshot.mjs、snapshot.test.mjs（修改） | 三索引声明联合与旧快照兼容，采集安全 |
| tools/content-ingest/source-report.mjs、source-report.test.mjs（新增） | 固定批次的来源问题、选择和处置；不转写数学正文 |
| content/source-maps/elementary-foundations.v1.json（新增） | 可复核来源到准确原创对象的映射，生成实例继承准确模板来源 |
| content/packages/elementary-foundations.v1.json；content/assets/elementary-foundations/*.svg（新增） | 30 节点、30 单元、固定路线和 9 原创图 |
| content/questions/elementary-foundations-{numbers,operations,fractions,decimals,ratios}.v1.json（新增） | 五个 QuestionPackage 草稿，不伪造 Archive、发布 head 或作者账户 |
| backend/internal/contentaudit/{model,decode,draft,coverage,report}.go 及对应 *_test.go（新增） | 有界输入、准确口径、草稿构建、覆盖和安全报告 |
| backend/internal/contentaudit/testdata/（新增） | 小型脱敏来源/覆盖夹具，不能存书籍原文或账户秘密 |
| backend/internal/store/content_audit.go、content_audit_test.go、content_audit_capacity_test.go（新增） | 一致、有界、只读数据库事实，真实并发与容量验证 |
| backend/internal/cli/content_audit.go、content_audit_test.go；backend/cmd/content-audit/main.go（新增） | 明确操作参数、私有环境凭据、报告与退出码 |
| backend/internal/e2etest/content_acceptance_fixture.go、content_acceptance_fixture_test.go（新增）；harness.go（修改） | 仅增加独立的 content-acceptance 技术场景 |
| tests/e2e/content-acceptance-{reading,learning}.spec.ts（新增） | 新数据结构的双视口真实网站联调 |
| tools/verify/content-acceptance.test.mjs、content-acceptance-ci.test.mjs（新增）；.github/workflows/{backend,frontend}.yml（修改） | 旧契约保护及新验证批次；保留旧工作流步骤 |
| docs/content/p6a-source-decisions.md、p6a-editorial-checklist.md；docs/operations/content-acceptance.md、evidence/p6a/（新增）；路线图（修改） | 中文来源处置、技术证据和 P6b 人工交接 |

~~~mermaid
flowchart TD
    Source[持续更新资料，只读] --> Snapshot[全新原字节快照]
    Snapshot --> Report[三索引来源报告 v1]
    Report --> Map[原始定位及原创对象映射]
    Map --> Draft[30 节点与五题库包草稿]
    Draft --> Offline[Go 离线校验与覆盖]
    DB[(明确选择的数据库)] --> Read[单个一致只读事务]
    Read --> Audit[准确路线验收]
    Offline --> Audit
    Audit --> Output[JSON 与中文 Markdown]
    Draft --> Isolated[随机隔离库及双视口联调]
    Isolated --> Technical[fixtureOnly 技术证据]
    Draft -. P6b 真实独立复核 .-> Publish[既有双 head 发布]
    Publish -. 后续正式验收 .-> DB
~~~

## 固定知识版本、目标与前置图

所有节点 domainIds=[elementary-mathematics]、topicIds=[elementary-mathematics-arithmetic]。数字约定 N₀={0,1,2,…}；整除与因数使用正整数除数；分数分母非零，首批计算以非负有理数为主；有余数除法 a≥0、d>0、0≤r<d。关系表中的编号指对应行准确 ID/版本，仅添加 prerequisite 边，路线 nodes 按 1—30 排列。

知识字段 objectives 使用下表两句英文，索引 0/1 固定；中文名称进入 titleZh。陈述、证明、例子可以经数学审查修订，若改变目标、版本或前置图需先更新计划和审核依据。

| # | ID / 版本 | 中文主题 | objective 0；objective 1 | 前置编号 | 来源核对记录（Manes） |
| --- | --- | --- | --- | --- | --- |
| 1 | natural-numbers / 1 | 自然数 | Count and represent nonnegative whole numbers.; Distinguish a number from its numeral and state the zero convention. | 无 | p3-k001、p2-k008 |
| 2 | zero / 1 | 零 | Interpret zero as an empty count.; Explain zero as a placeholder without confusing it with an absent digit. | 1 | p3-k001、p2-k010 |
| 3 | place-value / 1 | 位值 | Evaluate a base-ten numeral by place value.; Explain exchanging ten units for one higher unit. | 1、2 | p2-k001、p2-k002 |
| 4 | number-line / 1 | 数轴 | Locate nonnegative whole numbers on a uniformly spaced line.; Interpret distance from zero and equal unit intervals. | 1、2 | p3-k012、p3-k013 |
| 5 | comparing-numbers / 1 | 比较与排序 | Compare two nonnegative whole numbers.; Explain an ordering using place value or position. | 3、4 | p3-k012、p2-k005 |
| 6 | rounding-estimation / 1 | 舍入与估算 | Round a nonnegative value to a stated place using half-up rounding.; Use an estimate to judge whether a result is reasonable. | 3、5 | p6-k028（近似背景，舍入约定原创明确） |
| 7 | addition / 1 | 加法 | Compute the total of combined nonnegative quantities.; Model addition as directed movement on a number line. | 3、4 | p3-k003、p3-k013 |
| 8 | addition-properties / 1 | 加法性质 | Apply commutativity and associativity of addition.; Distinguish these laws from invalid subtraction rules. | 7 | p3-k021、p3-k022、p3-k023 |
| 9 | subtraction / 1 | 减法 | Compute the amount remaining after removal.; Interpret subtraction as the difference between two quantities. | 5、7 | p3-k005、p3-k019 |
| 10 | addition-subtraction-inverse / 1 | 加减互逆 | Find a missing addend.; Recover an unknown starting quantity from a difference. | 7、9 | p3-k019、p3-k028 |
| 11 | multiplication / 1 | 乘法 | Compute a total from equal groups.; Interpret multiplication as a rectangular array or area. | 7、8 | p3-k007、p3-k015 |
| 12 | multiplication-properties / 1 | 乘法性质与分配 | Apply commutativity, associativity and identity of multiplication.; Use distributivity and recognize invalid distributive claims. | 11 | p3-k024、p3-k025 |
| 13 | division / 1 | 除法 | Find a share when a total is divided into equal groups.; Find the number of groups of a given size with a nonzero divisor. | 9、11 | p3-k008、p3-k029 |
| 14 | division-remainder / 1 | 有余数除法 | Determine quotient and remainder satisfying the division identity.; Interpret a remainder in context and check its bounds. | 5、13 | p3-k011、p3-k030 |
| 15 | order-of-operations / 1 | 运算顺序 | Evaluate a stated expression using parentheses and operation precedence.; Identify and explain an incorrect order of evaluation. | 7、9、11、13 | p3 运算记录仅作背景；约定与例题原创推导 |
| 16 | factors / 2 | 因数与倍数 | Identify positive factors and nonnegative multiples.; Explain the factor-multiple relationship using exact division. | 11、14 | p4-k008、p3-k011（定义与性质原创展开） |
| 17 | divisibility / 1 | 整除 | Apply divisibility tests for 2, 5 and 10.; Justify these tests from base-ten place value. | 3、16 | p2-k013、p2-k015、p2-k002；5/10 的证明原创 |
| 18 | fractions / 2 | 分数与整体 | Interpret numerator and nonzero denominator relative to a unit whole.; Distinguish equal parts from unequal parts and changes of the whole. | 1、13 | p4-k001、p4-k003、p4-k011 |
| 19 | fraction-number-line / 1 | 分数数轴 | Locate a nonnegative fraction using equal unit subdivisions.; Interpret improper fractions on a consistent unit scale. | 4、18 | p4-k012、p4-k005 |
| 20 | equivalent-fractions / 2 | 等值分数 | Produce an equivalent fraction by a common nonzero factor.; Explain why additive changes to both entries need not preserve value. | 11、18 | p4-k007、p4-k031 |
| 21 | simplifying-fractions / 2 | 约分 | Reduce a fraction by common positive factors.; Justify value preservation and recognize invalid cancellation. | 16、20 | p4-k008、p4-k031 |
| 22 | comparing-fractions / 2 | 分数比较 | Compare fractions by an exact common representation.; Explain a valid benchmark or common-unit comparison. | 19、20 | p4-k013、p4-k014、p4-k016 |
| 23 | fractions-same-denominator / 1 | 同分母加减 | Add fractions expressed in the same unit.; Subtract such fractions and explain why the denominator is retained. | 7、9、18 | p4-k009 |
| 24 | fractions-unlike-denominator / 1 | 异分母加减 | Add fractions after constructing a common denominator.; Subtract fractions using equivalent units and explain the conversion. | 16、20、23 | p4-k010 |
| 25 | fraction-multiplication / 1 | 分数乘法 | Interpret a product as scaling a nonnegative quantity.; Interpret a product as the area of a rectangle with fractional side lengths. | 12、18 | p4-k019、p4-k020 |
| 26 | fraction-division / 1 | 分数除法 | Interpret division as measuring fractional groups.; Solve a missing fractional factor and state the nonzero condition. | 13、25 | p4-k024、p4-k026、p4-k028 |
| 27 | decimal-place-value / 2 | 小数位值 | Evaluate a finite decimal using powers of ten.; Explain trailing zeros and distinguish place value from digit value. | 3、18 | p6-k002、p6-k004 |
| 28 | decimal-fraction-conversion / 2 | 分数小数转换 | Convert a finite decimal to an exact fraction.; Determine whether a reduced fraction terminates and convert terminating cases. | 20、21、27 | p6-k003、p6-k010、p6-k013 |
| 29 | ratios-proportions / 1 | 比与比例 | Compute and interpret a unit rate.; Find a proportional scale factor while keeping units consistent. | 11、26 | p8-k002、p8-k004（数学模型背景） |
| 30 | percentages / 2 | 百分数 | Express a part-to-whole ratio as a percentage.; Compute a percentage of a stated whole and identify that whole. | 28、29 | p6-k003、p8-k002 仅作背景；百分数定义与推导原创 |

所有记录 ID 的完整前缀均为 manes-，如 p3-k001 实际为 manes-p3-k001。这张表只规定核对线索，不能宣称背景记录完整证明了新增内容；缺失记录或成立条件不匹配进入来源报告。旧 numbers/arithmetic 的概括性概念不改造成细分概念；其 v1 与另八个复用 ID 的 v1 均保留。

单元固定 ID ef-<knowledgeId>-unit / version 1；角度类型至少 intuitive 与 formal，配一个含步骤的例子、一项误区或反例、一项问题分解/合理性核查。路线为 elementary-foundations / version 1，英文标题 Elementary foundations，中文标题 初等数学基础。

9 个 SVG 的文件名与资产 ID 对应：place-value-exchange、whole-number-line、equal-parts、fraction-number-line、equivalent-fraction-strips、fraction-product-area、decimal-place-table、ratio-double-line、percent-grid；统一资产前缀 ef-，versionless 资产以实际 SHA 固定。前两图分别绑定节点 3/4，其余绑定 18/19/20/25/27/29/30；借用其他节点图时也必须保留准确资产及对应单元引用，不伪造归属。

## 锁定模板与有限参数域

模板 ID=ef-<下表键>、version=1。参数为表中集合的笛卡尔积，均 16 个组合；引擎先归一化，题面不能假设保留原分数字符串或小数格式。模板只声明本表所列实际目标，固定题补足其余目标。

缩写 A=rational_arithmetic，C=rational_comparison，M=missing_operand；除了 C 为 single_choice/answerFormat=null，均 numeric/answerFormat=rational；percent-rate 为 numeric/percentage。A/C 参数名 left/right，M 为 known/result；M 的 unknownSide 按表固定。除法 A 使用 nonzero_divisor，所有首批非负计算使用 nonnegative_result，C 无该约束；M multiply 的 known 集合不含零。解释写模型、单位与步骤，并使用 {{answer}} 展示准确结果；不在题面放答案占位符。

| 模板键 | 主节点；目标 | 引擎/操作/unknownSide | left 或 known | right 或 result | 教学目的 |
| --- | --- | --- | --- | --- | --- |
| compare-whole | 5；0 | C/compare/null | 2,7,12,20 | 2,8,10,21 | 准确比较，包括相等 |
| add-combine | 7；0 | A/add/null | 2,4,6,8 | 1,3,5,7 | 合并两组数量 |
| add-movement | 7；1 | A/add/null | 2,4,6,8 | 1,3,5,7 | 起点与正向移动 |
| subtract-removal | 9；0 | A/subtract/null | 10,12,14,16 | 1,2,3,4 | 从整体移除 |
| subtract-difference | 9；1 | A/subtract/null | 10,12,14,16 | 1,2,3,4 | 两量差/距离 |
| inverse-addend | 10；0 | M/add/right | 1,2,3,4 | 10,12,14,16 | 已知总数与一部分 |
| inverse-start | 10；1 | M/subtract/left | 1,2,3,4 | 2,4,6,8 | 从差及移除量恢复起点 |
| multiply-groups | 11；0 | A/multiply/null | 2,3,4,5 | 2,4,6,8 | 组数与每组量 |
| multiply-area | 11；1 | A/multiply/null | 2,3,4,5 | 2,4,6,8 | 长方形两边与平方单位 |
| divide-sharing | 13；0 | A/divide/null | 12,24,36,48 | 2,3,4,6 | 已知份数求每份量 |
| divide-measuring | 13；1 | A/divide/null | 12,24,36,48 | 2,3,4,6 | 已知每组量求组数 |
| compare-fractions | 22；0 | C/compare/null | 1/6,1/3,2/3,5/6 | 1/4,1/2,3/4,1 | 准确分数比较 |
| same-unit-add | 23；0 | A/add/null | 1/8,3/8,5/8,7/8 | 1/8,3/8,5/8,7/8 | 相同单位相加 |
| same-unit-subtract | 23；1 | A/subtract/null | 9/8,11/8,13/8,15/8 | 1/8,3/8,5/8,7/8 | 相同单位相减，包括假分数 |
| common-unit-add | 24；0 | A/add/null | 1/3,2/3,4/3,5/3 | 1/5,2/5,3/5,4/5 | 三分之一与五分之一统一单位 |
| common-unit-subtract | 24；1 | A/subtract/null | 4/3,5/3,7/3,8/3 | 1/5,2/5,3/5,4/5 | 转换后相减 |
| fraction-scale | 25；0 | A/multiply/null | 1/3,2/3,4/3,5/3 | 1/5,2/5,3/5,4/5 | 分数倍缩放 |
| fraction-area | 25；1 | A/multiply/null | 1/3,2/3,4/3,5/3 | 1/5,2/5,3/5,4/5 | 分数边长的面积 |
| fraction-measuring | 26；0 | A/divide/null | 1/2,3/4,5/4,3/2 | 1/4,1/3,1/2,2/3 | 有几个分数单位 |
| fraction-missing-factor | 26；1 | M/multiply/right | 1/4,1/3,1/2,2/3 | 1/2,3/4,5/4,3/2 | 已知乘积求另一因子 |
| ratio-unit-rate | 29；0 | A/divide/null | 6,12,18,24 | 2,3,4,6 | 每单位量 |
| proportion-scale | 29；1 | M/multiply/right | 2,3,4,6 | 12,24,36,48 | 比例缩放因子 |
| percent-rate | 30；0 | A/divide/null | 1,2,3,4 | 10,20,25,50 | 明确要求百分数语法的占比 |
| percent-part | 30；1 | A/multiply/null | 20,40,60,80 | 1/10,1/5,1/4,1/2 | 已知整体与以分数给出的百分率求部分 |

同引擎、同参数的两模板必须具有不同题意、模型、单位及目标，解析体现区别；作者和独立复核者在清单中逐对说明理由。单纯换故事名称或重复模板换数字不能增加有效题型数。24 是草稿计划，正式至少 20 须由实际复核认可。

固定题 ID=ef-<knowledgeId>-fixed-01 至 -15 / version 1；01—05 覆盖 [0]，06—10 覆盖 [1]，11—15 覆盖 [0,1]。蓝图 ID=ef-<knowledgeId>-assessment / version 1。每题四个非空、文本不同的选项，ID a/b/c/d；正确选项分布不能恒定，正文、答案和解析原创。以下批次数字由模板分配计算，必须在 Task 5/6 验证实际值：

| 包后缀 / 节点 | 模板 | 固定题 | 生成实例 | 总实例 |
| --- | --- | --- | --- | --- |
| numbers / 1—6 | 1 | 90 | 16 | 106 |
| operations / 7—15 | 10 | 135 | 160 | 295 |
| fractions / 16—24 | 5 | 135 | 80 | 215 |
| decimals / 25—28 | 4 | 60 | 64 | 124 |
| ratios / 29—30 | 4 | 30 | 64 | 94 |
| 合计 | 24 | 450 | 384 | 834 |

## 跨任务接口与结论规则

这些是新增内部工具契约，不写入 OpenAPI，不变更旧数学摘要。除说明外，JSON 数组不为 null，缺失与未知字段拒绝；对象/文件 SHA 为 64 小写十六进制，codeSHA 为本仓库完整 40 位 Git SHA；版本正 int32，路径安全相对定位，时间 UTC RFC3339。Go 字段名按 JSON tag 的首字母大写对应（如 draftCounts→DraftCounts），不是另造一套属性名。

| 类型 / 接口 | 精确约定 |
| --- | --- |
| IndexAudit（JS） | {indices:string[],packageCount:number,primaryFiles:string[],declarations:{path:string,claims:{index:string,sha256:string|null}[]}[],issues:{code:string,path:string}[]}；readIndexDeclarations(captured:Map<string,Buffer>):IndexAudit。排序与去重确定，声明冲突保留全部 claims |
| SourceReport（JSON v1） | {schemaVersion:1,policyVersion:1,snapshotId,selectedFiles:[{path,sha256,datasetId,recordIds:string[]}],issues:[{code,path,indexClaims:[{index,sha256:string|null}],actualSha256:string\|null,blocksSelected:boolean}],ready:boolean,publicationApproved:false}；原始文件定位只在编辑记录保留，公共摘要删除定位 |
| SourceMap（JSON v1） | {schemaVersion:1,policyVersion:1,snapshotId,sourceReportSha256,sources:[{id,path,recordId,datasetId,fileSha256,publicSources:content.Source[],use:fact_check/background/original_derivation,legacyIds:string[],reviewStatus:string,conditionsNote:string}],objects:[{kind,id,version:int\|null,sha256,sourceIds:string[],origin:original}]}。素材 version=null；知识/单元/路线 SHA 取 content.Digest，模板取 CanonicalTemplate，固定题用 Seal 得到的 fixed Instance identity SHA（objects.kind=instance），蓝图取 ValidateAndSeal 返回的 ValidationReport.Coverage 中 Blueprint.SHA256，素材取原字节 SHA；禁止统一重新发明数学哈希。生成实例从准确模板及参数追溯，不重复保存 384 份来源 |
| contentaudit.Request | {Mode:Mode,Route:content.VersionRef,FixtureOnly:bool,CodeSHA:string,At:time.Time}；Mode=draft/published，完整 Git SHA 必填 |
| contentaudit.SourceBundle | {SnapshotID,ReportSHA:string,Report:SourceReport,Mapping:SourceMap}；LoadSources(ctx context.Context,snapshotDir,reportFile,mapFile string)(SourceBundle,error)。验证 manifest snapshotId、全部选定文件原字节和条目定位，report SHA、映射政策一致；不读取 live source 代替快照 |
| contentaudit.DraftInput / DraftFacts | Input={Catalogue:catalogue.Catalogue,Content:content.Package,Questions:[]question.QuestionPackage,AssetsRoot:string}；Facts={Content:content.Snapshot,Path:content.Path,References:question.ReferenceSnapshot,Sealed:[]question.SealedPackage,QuestionReports:[]question.ValidationReport}。LoadDraft(ctx,root string)(DraftInput,error) 加载锁定文件；CheckDraft(ctx,Input)(DraftFacts,error) 用本地真实正文摘要构造 references，knowledgeHead=null，不伪造发布 UUID；调用既有 ValidateWorkflowDraft/ValidateAndSeal 与 question.ValidateAndSeal |
| contentaudit.Head / ApprovalFact | Head={ID,SHA256:string}；ObjectIdentity={Kind,ID:string,Version:*int,SHA256:string}，资产 Version=nil，其他为准确版本；ApprovalFact={Space:content/question,Object:ObjectIdentity,Evidence:publication.MemberEvidence,AuthorIDs:[]string,ReviewerID:string,ChecksComplete:bool,FrozenMatches:bool}。资产对应原 SQL member 的 version=1 仅为数据库内部历史约定，不能在工具身份中变成可增长的数学版本。正文/审核私有说明不出报告 |
| contentaudit.PublishedFacts | {CatalogueVersion:int,CatalogueSHA:string,KnowledgeHead,QuestionHead:*Head,Path:content.Path,PathSHA:string,Content:content.Snapshot,Bank:question.Candidate,Approvals:[]ApprovalFact,EligibleInstances:[]question.Identity,Excluded:[]Exclusion,FixtureOnly:bool}。所有事实同一事务；Bank 先验证完整有界 head，再按准确路线过滤；EligibleInstances 使用现有主归属、蓝图、单位/素材、批准、撤回及纠错限制谓词，不读任何学习者答案 |
| contentaudit.AcceptanceEvidence | {schemaVersion:1,codeSHA,routeSHA,knowledgeHead,questionHead,fixtureOnly,reviewAttestations:[{decisionId,independenceVerified:boolean}],learningChecks:[{name,result:passed/failed/not_run,evidenceSha256}]}。head 为 Head\|null，学习检查名固定 reading/pass/fail/prerequisites/review/practice_exposure/retake/feedback_correction；只保存结果及证据摘要。读取实际文件，不能默认生成全部 passed；真实人员独立性由 P6b 人工核实并对准确 decision 留证，不以另一个账号证明 |
| contentaudit.Report | {schemaVersion:1,mode,fixtureOnly,createdAt,codeSHA,context:{catalogueVersion,catalogueSha,route:content.VersionRef,routeSha,knowledgeHead:*Head,questionHead:*Head},source:{snapshotId,policyVersion,reportSha,complete:boolean,unresolvedCount:int},draftCounts:Counts,formalCounts:Counts,nodes:[]NodeReport,quality:{twoAngles:boolean,examples:boolean,assets:boolean,independentReview:boolean,learningComplete:boolean},conclusion:Conclusion,reasons:[]Reason} |
| Counts / NodeReport | Counts={knowledge,templates,fixedInstances,generatedInstances,effectiveInstances,duplicates,excluded:int}；NodeReport={knowledge:question.Identity,blueprint:*question.Identity,effectiveInstances,assessmentInstances:int,core:[]int,fiveWitness:[]string,afterPracticeWitness:boolean,ready:boolean,reasons:[]Reason}；Reason={code,path:string}，Exclusion={object:ObjectIdentity,code:string}。不带题面、选项、答案或个人 ID |
| 纯函数 | EvaluateDraft(ctx context.Context,req Request,facts DraftFacts,sources SourceBundle)(Report,error)；EvaluatePublished(ctx,req Request,facts PublishedFacts,sources SourceBundle,evidence AcceptanceEvidence)(Report,error)；WriteReport(jsonOut,markdownOut io.Writer,report Report) error。实现位于 coverage/report.go，消费者不重算统计 |
| 数据库 / CLI | Store.ReadContentAudit(ctx context.Context,route content.VersionRef)(contentaudit.PublishedFacts,error)；cli.RunContentAudit(ctx context.Context,args []string,stdout,stderr io.Writer) int；新 main 只传递 args/context/writer |

SourceReport/SourceMap/AcceptanceEvidence 在 model.go 中用对应结构体和 JSON tag 定义，decode.go 严格检查重复键、无效 UTF-8/NUL、int32、未知字段及截断输入。每个原索引未提供摘要时声明 claim 为 null，声明值原样保留；实际对象/文件摘要仍严格校验。helper 不新增原 snapshot 的缺摘要致命错误，来源报告对选中缺摘要给具体待核对原因，非选中问题按 blocksSelected=false 隔离。元数据文件最多 4 MiB，SourceMap 256 KiB，报告输出最多 8 MiB。复用现有 JSON 检查器时须确认能够覆盖这些规则，不直接用 permissive json.Unmarshal。原字节资料快照不限于这些元数据大小，不把完整书籍装入工具报告。

结论固定：输入损坏/不可核验返回错误，不生成成功报告；可核验但存在结构、来源、数量、覆盖或学习失败为 not_ready；draft 模式草稿技术合格为 draft_ready，formalCounts 全零；published 模式技术满足但缺少当前批准或真实独立复核/学习证据为 awaiting_review；仅所有正式门槛、当前准确审核及学习证据满足且 fixtureOnly=false 才 accepted。fixtureOnly 的 published 报告正式数量全零，最高 awaiting_review；模拟通过量只列 draftCounts。后续满足业务最低门槛即可，不强制正式达到编写目标 834。

## Task 1：三索引固定快照与兼容保护

**Files：** 新增 source-index.mjs/source-index.test.mjs，修改 snapshot.mjs/snapshot.test.mjs，均在 tools/content-ingest/。
**Interfaces：** Consumes 原 Map<string,Buffer> 与既有 snapshot 命令参数；Produces readIndexDeclarations(captured):IndexAudit，schemaVersion=1 原 manifest 和独立声明审查事实，供 Task 2 使用。

- [ ] **Step 1：编写失败测试。** legacy digest 测试逐字节固定 files 数组及 snapshotId；旧一/两索引输入得到原 primaryFiles/packageCount。第三索引独有主文件进入结果；三份相同声明只计一路径，同路径两个 SHA 保留两条 claim 并列 mismatch。路径逃逸/符号链接失败；测试子进程用同步屏障在首轮采集后改写/删除文件，断言 SOURCE_CHANGED_RETRY、输出目录清理、前批 SHA 不变。

```js
assert.deepEqual(audit.indices, ['Knowledge_JSON_Index.json', 'Incremental_Knowledge_JSON_Index.json', 'MSC2020_Incremental_Knowledge_JSON_Index.json']);
assert.equal(conflict.declarations[0].claims.length, 2);
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs -- node --test tools/content-ingest/source-index.test.mjs tools/content-ingest/snapshot.test.mjs`；新第三索引/声明联合测试失败，原五项 snapshot 测试继续作为基线。
- [ ] **Step 3：实现联合读取。** helper 固定识别三份索引，其中第一份仍必需，第三份可缺；同路径声明不 last-wins。snapshotId 继续为 hash(JSON.stringify(files))，保留旧字段、错误约束、0700/0600、完整原字节及二次字节核验。详细政策不塞入原摘要含义。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2，全部通过；额外比对 P1 manifest 原字节 SHA，历史文件不修改。执行全新来源快照时输入 `/Users/wiw/Documents/math_master/Knowledge_JSON/`，输出新私有目录，保存真实 stdout，不提交原资料。
- [ ] **Step 5：提交。** `git add tools/content-ingest/source-index.mjs tools/content-ingest/source-index.test.mjs tools/content-ingest/snapshot.mjs tools/content-ingest/snapshot.test.mjs`；`git commit -m "feat: 支持三份来源索引并保留快照兼容"`。

## Task 2：来源选择、隔离与追溯报告

**Files：** 新增 tools/content-ingest/source-report.mjs/source-report.test.mjs、docs/content/p6a-source-decisions.md。
**Interfaces：** Consumes Task 1 IndexAudit 和新快照 manifest/files；Produces buildSourceReport(snapshotDir:string,selectedFiles:string[]):SourceReport 及 CLI `--snapshot --selected --out`；--selected 指含所选安全路径的 JSON 数组，--out 为全新文件，0700 父目录/0600 文件。

- [ ] **Step 1：编写失败测试。** 选中 Manes 样式的 corpus，205 只是当前来源观测，不写死未来数量。断言所选文件 SHA 和每个记录 ID 均可定位；相同标题/legacy ID 的不同条件不自动合并，主文件与章节分别保留来源。选中摘要冲突、缺失记录或文件阻止 ready；非选中高级资料差异 blocksSelected=false 不阻止 ready；human_reviewed=false/AI review 标记不改变 publicationApproved=false。索引 instructions 中含命令时不执行，源目录树摘要不变。

```js
assert.equal(report.ready, true); // 仅非选中路径有差异
assert.equal(report.publicationApproved, false);
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs -- node --test tools/content-ingest/source-report.test.mjs`；新报告入口缺失导致明确失败。
- [ ] **Step 3：实现报告。** 本批唯一默认选择需由 --selected 明确给出 Manes corpus 相对路径（方案第三节）；从 snapshot files 读取 schema_version/dataset_id/knowledge_points，不读取 live source。保留 provenance、review_status、legacy 与问题两侧 SHA 的私有出处，公共摘要只列数量/原因。记录本轮实际八项或更新后差异及处置，条件不明标记待核对，不自动改写用户文件。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2；对 Task 1 新批次生成 source-report.json，复算输出 SHA 并确认 selected ready 的原因真实。中文版处置文档记批次、政策、选中条目与背景来源边界，不能写成发布批准。
- [ ] **Step 5：提交。** `git add tools/content-ingest/source-report.mjs tools/content-ingest/source-report.test.mjs docs/content/p6a-source-decisions.md`；`git commit -m "feat: 生成可追溯的来源选择与隔离报告"`。

## Task 3：纯验收口径、来源校验与草稿构建

**Files：** 新增 backend/internal/contentaudit/{model,decode,draft,coverage,report}.go、对应 *_test.go 和 testdata/。原校验器不修改。
**Interfaces：** Consumes Task 2 SourceReport、既有 content/question 类型；Produces本计划跨任务表全部纯 Go 类型和 LoadSources/LoadDraft/CheckDraft/EvaluateDraft/EvaluatePublished/WriteReport，供 Task 4—8 使用。数据库事实通过参数传入，纯层不打开数据库。

- [ ] **Step 1：编写失败测试。** TestContentAuditRouteCounts：其他路线 300 知识/5000 实例不能让目标 29 节点达标，同 ID v1/v2 只计当前版本；精确重复实例只计一次，并另报重复。TestContentAuditPrimaryBinding：主归属 A、secondary coverage B 的实例不计 B 检测。TestContentAuditExposureWitness：排除任一模板的全部实例或任一固定题后，仍必须恰好五题覆盖 [0,1]；仅一模板 15 实例的反例为 false。TestContentAuditSourceIdentity：错 snapshot/report/对象 SHA、record 缺失和元数据大小+1失败。TestContentAuditConclusions：draft_ready 正式数全零；缺批准/真实独立证明 awaiting_review；fixtureOnly 即使模拟全部通过仍不能 accepted。JSON/中文摘要不含答案、个人 ID/私有路径。

```go
if r.Conclusion != DraftReady || r.FormalCounts.EffectiveInstances != 0 { t.Fatal(r) }
if secondaryNode.AssessmentInstances != 0 { t.Fatal(secondaryNode) }
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -timeout 5m -count=1`；新增类型/函数缺失失败。
- [ ] **Step 3：实现纯层。** draft.go 按内容实际摘要构造空 head ReferenceSnapshot 后调用 question.ValidateAndSeal；不使用要求已发布 head 的 BuildCandidate。来源定位读取固定批次并核对所有实际对象来源，不把条件不同记录按标题合并。coverage.go 精确筛选、按准确身份及已归一化 QuestionBody 的相同正文去重后调用 FiveQuestionCover，正文指纹仅用于本工具去重，不变更任何持久数学摘要；静态 afterPracticeWitness 对每种模板/固定题曝光类别逐次检查；report.go 按结论优先级与安全字段输出。来源缺口和数学判断保留具体原因，不伪造审核事件。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2；边界测试实际覆盖现有合法单节点 1000 候选与 1001 拒绝，元数据上界与+1。冻结小型 JSON/Markdown 报告样例，两个格式 context/counts/conclusion 一致。source.unresolvedCount 记录全部问题，source.complete 与 not_ready 仅依据 blocksSelected=true 的未解决问题，不能因其他板块隔离问题阻止首批。
- [ ] **Step 5：提交。** `git add backend/internal/contentaudit`；`git commit -m "feat: 建立首批内容的纯验收口径与草稿校验"`。

## Task 4：完整知识、学习单元与原创 SVG 草稿

**Files：** 新增 content/packages/elementary-foundations.v1.json、content/assets/elementary-foundations/ 下锁定九图、content/source-maps/elementary-foundations.v1.json、docs/content/p6a-editorial-checklist.md；新增 backend/internal/contentaudit/editorial_test.go。
**Interfaces：** Consumes Task 2 固定来源与 Task 3 校验器、锁定知识表；Produces 一个完整前置闭包 content.Package、SourceMap 及真实 assets bytes，供 Task 5/6/9 消费。

- [ ] **Step 1：编写失败测试。** TestContentAuditEditorialRoute 断言 30 不同 ID、8 个 v2/22 个 v1、30 单元每个两种不同角度及例子、1 路线准确顺序、9 个安全 SVG；前置边严格等于知识表且无环。本任务测试直接读取单一内容包并调用原 content 校验器，尚不要求五个题库文件存在。固定数学反例核对 0、位值 305、同整体等值 1/2=2/4、错误 1/2+1/3≠2/5、非零除数、2.50=2.5、百分率整体变化；由具名 golden 案例与实际正文/图片标签对照，不声称程序证明全部 prose。检查源标签 AI 不成为实际数学批准，旧包 SHA 不变。

```go
if len(p.Knowledge) != 30 || len(p.Units) != 30 || len(p.Assets) != 9 { t.Fatal(p.ID) }
if p.Paths[0].ID != "elementary-foundations" || p.Paths[0].Version != 1 { t.Fatal(p.Paths) }
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -run '^TestContentAuditEditorial' -timeout 5m -count=1`；草稿与原创素材缺失失败。
- [ ] **Step 3：原创编写。** 严格按 30 行目标/版本/前置图编写英文正文与中文术语；theorem 有证明、条件明确；九图纯 SVG，viewBox 与刻度/等分/面积/百分格符合数学量，标注文字及原创归属。每个知识、单元、路线、素材写 SourceMap 当前正确摘要与来源用途；公开 source 引用原公开出处，编辑路径不注入页面。草稿 version 在送审前可编辑，批准后修改必须新版本。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2；逐图渲染目视核对（使用本地浏览器），对单包执行 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go run ./cmd/content-check --catalogue ../content/catalogue/domains.json --package ../content/packages/elementary-foundations.v1.json --assets ../content/assets`，不导入数据库。清单明确当前是作者校验，独立复核未完成。
- [ ] **Step 5：提交。** `git add content/packages/elementary-foundations.v1.json content/assets/elementary-foundations content/source-maps/elementary-foundations.v1.json backend/internal/contentaudit/editorial_test.go docs/content/p6a-editorial-checklist.md`；`git commit -m "feat: 编写三十节点初等数学原创草稿"`。

## Task 5：数与四则运算题库

**Files：** 新增 content/questions/elementary-foundations-numbers.v1.json、elementary-foundations-operations.v1.json、backend/internal/contentaudit/questions_test.go；更新 SourceMap 和编辑清单。
**Interfaces：** Consumes Task 4 准确知识/单元/资产及 Task 3 CheckDraft；Produces 两个 QuestionPackage：11 模板、225 固定题、15 blueprint、176 生成实例，401 不同实例。没有作者账户或假 Archive 的 sourceResponsibility。

- [ ] **Step 1：编写失败测试。** TestContentAuditQuestionsNumbersOperations 断言两个包分别 1/90/16/106 与 10/135/160/295，每模板 16 个生成实例都通过 VerifyInstance；每节点目标 [0,1] 的固定题组准确。人工独立核算 golden：12÷3=4，未知 x−3=4 得 x=7，零数量/除零/舍入中点为固定题正确选项与条件。篡改一个生成正确答案必须校验失败，零除数和概念题假 witness 被拒绝；模板目的不能仅换数字。

```go
if len(sealed[0].Instances) != 106 || len(sealed[1].Instances) != 295 { t.Fatal("wrong draft quantities") }
if totalTemplates != 11 || totalFixed != 225 { t.Fatal("wrong authoring batch") }
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -run '^TestContentAuditQuestionsNumbersOperations$' -timeout 5m -count=1`；新题包缺失失败。
- [ ] **Step 3：编写两包。** 按锁定参数表、固定题 ID 分组及蓝图规则原创题面/解析；使用实际 Knowledge Ref、Unit Ref 和公开 original source。固定题的正确答案由作者逐题核算并留清单，后续待真实复核；生成实例由现有 Seal 产生，不手造 instance ID。更新每个 template/固定 instance/blueprint 的准确 SourceMap 对象 SHA。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2；完整遍历 176 参数实例、四选项唯一、正确键分布、十五节点的首次与任一一次曝光五题见证均通过。将包/封存/参数实例实际大小写清单，仍满足原容量，不只查数组长度。
- [ ] **Step 5：提交。** `git add content/questions/elementary-foundations-numbers.v1.json content/questions/elementary-foundations-operations.v1.json backend/internal/contentaudit/questions_test.go content/source-maps/elementary-foundations.v1.json docs/content/p6a-editorial-checklist.md`；`git commit -m "feat: 编写数与四则运算原创题库草稿"`。

## Task 6：分数、小数、比例及百分数题库

**Files：** 新增 content/questions/elementary-foundations-{fractions,decimals,ratios}.v1.json；修改 questions_test.go、SourceMap、编辑清单。
**Interfaces：** Consumes Task 4—5；Produces 其余 13 模板、225 固定题、15 blueprint、208 生成实例，完整五包 24/450/384/834；全路线 EvaluateDraft 可交付 draft_ready。

- [ ] **Step 1：编写失败测试。** TestContentAuditQuestionsFractionsRatios 断言三包实际 215/124/94 实例及全部 30 blueprint；同分母经归一化后仍显示八分之一单位，异分母确为 3/5 两域。golden 独立核算 1/3+1/5=8/15、3/4÷1/2=3/2、1/4=0.25、3/10=30%、60 的 25%=15；非有限小数不能声称有限，零分母/不一致整体/非法约分的固定单选有明确唯一答案。TestContentAuditCompleteDraft 断言 actual draftCounts=30/24/450/384/834、正式数全零，逐节点 effective≥15、五题和最坏一次模板/固定题曝光仍覆盖核心。

```go
if r.DraftCounts.Templates != 24 || r.DraftCounts.FixedInstances != 450 || r.DraftCounts.GeneratedInstances != 384 || r.DraftCounts.EffectiveInstances != 834 { t.Fatal(r) }
if r.FormalCounts.Knowledge != 0 || r.Conclusion != DraftReady { t.Fatal(r) }
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -run '^TestContentAudit(QuestionsFractionsRatios|CompleteDraft)$' -timeout 5m -count=1`；未完成题包失败。
- [ ] **Step 3：编写三包。** 完成 225 独立固定题及 13 模板。不能让 numeric 判分强制它本来不检查的最简分数/特定表示形式；表示与约分要求放在原创固定单选。percent-rate 题面明确百分数输入，percent-part 参数归一化呈现分数比例并解释百分率；原精确语法不改。补全对象来源与双目标题实际理由。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2，再运行完整 contentaudit 测试。重新实测五包总数、全参数独立验证、每节点主归属、单次曝光组合及素材/来源完整性；记录草稿数量及待人工复核，重复项不以换 ID 填数。
- [ ] **Step 5：提交。** `git add content/questions/elementary-foundations-fractions.v1.json content/questions/elementary-foundations-decimals.v1.json content/questions/elementary-foundations-ratios.v1.json backend/internal/contentaudit/questions_test.go content/source-maps/elementary-foundations.v1.json docs/content/p6a-editorial-checklist.md`；`git commit -m "feat: 补齐分数小数比例与百分数题库草稿"`。

## Task 7：一致、有界、只读发布事实

**Files：** 新增 backend/internal/store/content_audit.go、content_audit_test.go、content_audit_capacity_test.go。
**Interfaces：** Consumes Task 3 PublishedFacts/ApprovalFact 和原数据库表/读取核验函数；Produces Store.ReadContentAudit(ctx,route content.VersionRef)(contentaudit.PublishedFacts,error)，不增加账户参数、HTTP endpoint 或迁移。数据库操作者访问是明确的运维权限，不变成匿名应用读取能力。

- [ ] **Step 1：编写失败测试。** 在 testutil 随机 PostgreSQL 库中 TestContentAuditConsistentHeads 用事务同步屏障并发切换两个 head，结果必须完整来自同一快照或失败，不能拼接。TestContentAuditWithdrawal 核验准确路线、知识前置、单元/资产、模板/实例/蓝图撤回及纠错规则限制的排除；不同路线有效数据不能撑满目标数量。TestContentAuditEvidenceMismatch 篡改 body SHA/审查 frozenDigest/作者集合/当前批准，报告失败或对应不就绪。TestContentAuditReadOnly 以仅 SELECT 数据库身份运行，对比读取前后所有业务表行数/事件/任务均不变。最大合法 bank 200/10000/1000 及 1000 节点引用、32/8 MiB 元数据上界守卫，不静默截断并算绿。

```go
if facts.KnowledgeHead.ID != oldKnowledgeHead && facts.KnowledgeHead.ID != newKnowledgeHead { t.Fatal(facts) }
if businessRowsAfter != businessRowsBefore || writesObserved != 0 { t.Fatal("audit mutated state") }
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestContentAudit' -skip '^TestContentAuditCapacity$' -timeout 5m -count=1`；入口缺失失败；测试使用 TEST_DATABASE_URL 已配置的隔离工具，不连接生产。
- [ ] **Step 3：实现读取。** BeginTx RepeatableRead/ReadOnly，一次 ctx≤8s，statement_timeout≤8s/lock_timeout=1s；固定目录、准确路线 SHA、两个 head、冻结作者与审批事实，复用原 canonical/证据检查。不能调用会另开事务的公开读取方法、写入式 learning 缓存或后台补扫。先验证 bounded head 完整性，再按 route 精确过滤；资格谓词对齐 assessment_sources 及 correction 的元数据限制，实例正文只用于精确核验，用户答案完全不读。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2；另跑 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestContentAuditCapacity$' -timeout 5m -count=1 -v`。记录真实正文大小、时长、查询和预算。锁/读取超限必须失败关闭；不能增大截止或删除最大合法测试。
- [ ] **Step 5：提交。** `git add backend/internal/store/content_audit.go backend/internal/store/content_audit_test.go backend/internal/store/content_audit_capacity_test.go`；`git commit -m "feat: 增加一致有界的只读内容验收事实"`。

## Task 8：只读 CLI、双格式报告及运维说明

**Files：** 新增 backend/internal/cli/content_audit.go/content_audit_test.go、backend/cmd/content-audit/main.go、docs/operations/content-acceptance.md。
**Interfaces：** Consumes Task 3 Evaluate/WriteReport/LoadSources 与 Task 7 Store.ReadContentAudit；Produces cli.RunContentAudit(ctx,args,stdout,stderr) int。

- [ ] **Step 1：编写失败测试。** TestContentAuditCLI 用新随机库检查零参数/重复参数/未知参数/越界版本/未配置显式 DSN、已有输出目录拒绝且不删除用户文件；draft 模式不读数据库。TestContentAuditFixtureCannotAccept 强制识别 math_master_test_* 数据库，即使操作者省略 --fixture-only 仍 fixtureOnly=true、正式数零、不能 accepted。未提供真实 AcceptanceEvidence 时 published 最高 awaiting_review；证据 head/代码 SHA 不一致不能套用旧 passed。只读 SELECT 账号正常报告，写入尝试为零；错误日志不泄露 DSN/私有记录。

```go
if exit != 3 || !r.FixtureOnly || r.Conclusion == Accepted || r.FormalCounts.EffectiveInstances != 0 { t.Fatal(r, exit) }
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/cli -run '^TestContentAudit' -timeout 5m -count=1`；新命令缺失失败。
- [ ] **Step 3：实现接口。** 必选 --mode=draft/published、--route=elementary-foundations、--version=1、--snapshot、--source-report、--source-map、--code-sha、--out；draft 另必选 --root，published 另必选 --database-env=变量名，只从该私有环境读 postgres URI，没有 DATABASE_URL 默认。--evidence 可选：缺省有具体待复核/学习未验证原因；--fixture-only 只可加强技术标记，不能取消自动识别。context 总预算沿用 8s；连接池最多 2，库读取只读。--out 原子创建全新 0700 目录，report.json/report.md 0600，stdout 只给安全 conclusion/数量摘要。退出 0=draft_ready/accepted，3=awaiting_review，2=not_ready/非法输入，1=配置/IO/预算失败；失败不留下成功报告。
- [ ] **Step 4：验证 GREEN。** 重跑 Step 2；实际 draft CLI 给 draft_ready/0，正式数零，双格式一致；隔离 published CLI 给 awaiting_review/3，fixtureOnly=true。构建新 binary，中文运维文档给无秘密命令和 P6b 真正人员、审批 decision、准确 head、学习检查证据的填写流程，明确技术工具不证明人的独立性。
- [ ] **Step 5：提交。** `git add backend/internal/cli/content_audit.go backend/internal/cli/content_audit_test.go backend/cmd/content-audit/main.go docs/operations/content-acceptance.md`；`git commit -m "feat: 提供安全只读的首批内容验收命令"`。

## Task 9：新数据结构的隔离学习与阅读联调

**Files：** 新增 backend/internal/e2etest/content_acceptance_fixture.go/content_acceptance_fixture_test.go、tests/e2e/content-acceptance-reading.spec.ts/content-acceptance-learning.spec.ts；仅修改 harness.go 的场景分发。
**Interfaces：** Consumes Task 4—8 全部草稿和既有 fixtureAccounts/工作流/服务；Produces `questionControls.contentAcceptanceScene(ctx context.Context) error`，scene 名 content-acceptance，仅受原 loopback/token/random DB 防护控制。原 runtime JSON/公开 API 和其他场景不变。

- [ ] **Step 1：编写失败测试。** TestContentAcceptanceFixture 断言随机库三十节点/五题库结构实际进入现有送审/技术批准/发布流程，head 与版本准确，运行记录 fixtureOnly=true、非正式人工批准；同进程切换原 draft/question/learning/feedback/correction 场景仍可用。浏览器分组：reading 的双视口英文/中文术语、九图、长公式、30 路线及 fixed-version 回顾；learning 的首次五题、4/5通过与3/5失败、前置不足、一次练习后检测、手动重测/题不足显示、反馈及撤回纠错。草稿中的答案不经公开元数据泄露。

```ts
await expect(page.getByRole('heading', { name: 'Elementary foundations' })).toBeVisible();
await expect(page.getByText('Five eligible questions are not available.', { exact: false })).toBeVisible(); // 耗尽场景，沿用现有文案
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/e2etest -run '^TestContentAcceptance' -timeout 5m -count=1`；场景缺失失败。完成新浏览器断言后分别用既有 npm e2e 入口运行两文件，预期新场景未建立而失败，不能写 skip。
- [ ] **Step 3：实现技术场景。** 在全新随机库内用既有测试账号和准确 expectedHead/双 head 发布草稿；仅在测试进程使用模拟工作流，不能创建真实人员承诺。完整 DAG 以准确前置授予或既有检测测试帮助函数安排学习者，真实选题/提交仍调用现有服务。为两新 spec 各设置四个业务场景，每个 desktop/mobile 均执行，新增共16用例；reading 包含两组视觉/固定版场景与两组来源/公开边界，learning 包含通过失败前置、曝光补测、版本回顾、反馈纠错场景。不改渲染器或学习策略来迎合夹具。
- [ ] **Step 4：验证 GREEN。** 重跑 Go 测试并按两批分别跑 `node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-acceptance-reading.spec.ts`、同入口 learning.spec.ts。真实 store 补核三十节点首次/一次曝光后准确主归属的五题见证；两视口16项全通过，旧场景测试仍通过。对九图实际显示目视复核并归档脱敏截图，报告始终 fixtureOnly。
- [ ] **Step 5：提交。** `git add backend/internal/e2etest/content_acceptance_fixture.go backend/internal/e2etest/content_acceptance_fixture_test.go backend/internal/e2etest/harness.go tests/e2e/content-acceptance-reading.spec.ts tests/e2e/content-acceptance-learning.spec.ts`；`git commit -m "test: 验证首批草稿的隔离阅读与学习闭环"`。

## Task 10：CI 追加、兼容保护与实施交付

**Files：** 新增 tools/verify/content-acceptance.test.mjs/content-acceptance-ci.test.mjs，修改 .github/workflows/backend.yml/frontend.yml、路线图；新增 docs/operations/evidence/p6a/ 下脱敏证据、manifest.json 和自审记录。
**Interfaces：** Consumes Task 1—9 及原 CI 步骤/时间预算；Produces 新保护测试、新独立验证批次和有完整 SHA/命令/结果的交付证据。不改变旧 api/generated、迁移、引擎或前端组件接口。

- [ ] **Step 1：编写失败测试。** content-acceptance-ci.test.mjs 必须捕获新 contentaudit/CLI/store audit容量/两浏览器批次缺失，同时保护全部旧步骤、工具链、CGO_ENABLED=0、5m/8m/9m/30m预算。content-acceptance.test.mjs 核对旧快照 schema/原ID/旧包与迁移字节、没有公开新 API、测试入口不进入 server 生产 import。先定义稳定的基线文件清单及 SHA，避免把整个未来代码树固定为不能修改；负测删除新批次或更改旧保护文件确实失败。

```js
assert.equal(oldBrowserBatches, 20);
assert.deepEqual(newBrowserFiles, ['content-acceptance-reading.spec.ts', 'content-acceptance-learning.spec.ts']);
assert.equal(changedMigrationFiles.length, 0);
```

- [ ] **Step 2：验证 RED。** `node tools/verify/run.mjs -- node --test tools/verify/content-acceptance.test.mjs tools/verify/content-acceptance-ci.test.mjs`；新增 CI 批次尚未连接而失败。
- [ ] **Step 3：追加工作流与记录。** backend 加 contentaudit/CLI/store audit/容量独立命令和来源报告测试，避免同一 audit 容量在广泛集成批重复；为新矩阵保留既有 job 30m，不用超时掩盖性能。frontend 加两新 browser 批次，20旧批原样保留。若总 job 实测不能在原预算完成，以新增独立 job 分组，并在兼容保护中锁定旧行为；不延长原 job。归档实际草稿数量、来源处置、CLI结论及待复核事实，路线图 P6a/P6b状态准确。
- [ ] **Step 4：验证 GREEN 与交付审查。** 重跑 Step 2，再执行下方完整矩阵。一次独立整分支审查覆盖五项 Review Focus、计划/方案符合性、SQL只读一致性与数学材料抽查；修复实际问题后只重复受影响验证及必要完整回归。全部本地结果、红绿证据、代码 SHA、原始安全 stdout 和实际待复核清单归档，不生成假的正式验收。最终完整 head 的远端 CI 实际启动/结果另核验；账号限制未解除时 MR 明确阻塞，不能声称正式 accepted。
- [ ] **Step 5：提交与实现 MR。** 只添加此任务列出的实际文件，`git commit -m "test: 接入首批内容验收与兼容回归"`；通过 SSH 推送 codex/p6a-content-preparation，用 gh --body-file 创建实现 MR 并附加到本会话。MR正文围绕来源、原创草稿、只读工具、隔离结果和实际局限；未经另一次合并授权不合并。P6b 真实数学审核与 P7 部署分阶段交接。

## 完整验证矩阵与证据要求

每命令单独运行，使用 tools/verify/run.mjs；不能将所有测试包塞进一次超过十分钟的测试。TEST_DATABASE_URL 只用于 testutil 生成的随机隔离库；缺本地环境先修复开发测试环境，不退化为 mock 或跳过数据库测试。

| 验证项 | 命令/固定要求 |
| --- | --- |
| 旧+新增 Node | 原七文件 Node命令（44 基线项）保留，加 source-index、source-report、content-acceptance、content-acceptance-ci 四个新 test 文件 |
| Go 格式/静态 | 既有 gofmt 检查；`node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go vet ./...`；`git diff --check` |
| 纯验收与来源 | `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentaudit -timeout 5m -count=1` |
| 旧 Go / SQL / CLI | .github/workflows/backend.yml 的所有旧分组、五项容量原样保留；新 audit 的常规与容量分别按 Task 7/8命令运行，防止宽泛 store 分组重复容量 |
| 前端 | 既有 api:generate +生成文件无diff、typecheck、npm test（350基线）、build、npm audit --omit=dev；全部经既有包装入口 |
| 浏览器 | 保留 frontend workflow原20批/146基线；追加两批/16新用例，workers=1、retries=0、desktop1280×900/mobile390×844、8m全局截止；构建后使用真实 harness |
| 构建/CLI | 既有 `go build ./cmd/...`；对实际固定批次运行 draft CLI，对隔离库运行 published CLI，分别核对 0/3退出、正式数零、原始JSON与中文摘要一致 |
| 新数据库容量 | Task 7最大合法bank、读取预算、错SHA/混合head/撤回/审批负测；实测30/24/450/384/834草稿数量，不把旧容量fixture数量当正式量 |
| 交付审查 | 整分支独立审查、必要修复、所有证据完整性；准确最终SHA远端各workflow/job的真实结论 |

计划实施记录保存执行基线、任务/50步骤完成状态、测试时长、审查问题与解决证据；源码/JSON/诊断脚本执行 git diff --check，原始日志不能为消除空白而改写。新正式报告不存在时明确记录“未生成正式 accepted；真实人员复核待落实”。

## 可行性自审与交接

1. 单一知识包30节点/30单元/9素材低于现有100/200/16，所有前置引用在同包；五题库包最大135固定题、160生成实例，低于200/1000，24模板/834实例/30蓝图低于整库上限。包字节与实际运行时间仍由 Task 4—7实测，不以估计替代。
2. 每节点15固定题保留两个目标的多样题源，参数模板额外增强计算练习；一次曝光任一模板不消耗固定题来源。一固定题或最近五题排除后覆盖仍需测试真实谓词；不承诺无限立即重复检测。
3. Manes目前205记录与版本/条件已读取，正文 review.human_reviewed=false；数学事实核对、背景线索和原创推导分别记来源用途。不存在因缺少自动证明引擎而阻止P6a的必要改造；审核资源仍为P6b门槛。
4. 对照方案十三节完成覆盖：意图/方式由Global Constraints；来源由Task1/2/3；内容由Task4；题库由Task5/6；只读报告由Task3/7/8；独立复核/发布为Task8说明及P6b交接；文件/兼容由职责表与Task10；验证由Task7/9/10。五项 Review Focus 均有具名负测，没有未处理的阻塞性设计缺口。
5. 自审由本会话完成，核对每任务五个复选步骤、跨任务类型/命名一致、准确主归属、模板级冷却、原始资料不变、技术报告不能accepted。此自审不替代未来产品整分支审查或真实数学批准。

审阅本计划后，按已保留的 Native 方式从最新 master 开始 P6a 实现。P6a交付时仍需真实独立复核者逐项审核条件/证明、两角度、固定题答案、24模板题意与全参数、九图及来源；审批、发布和正式达标报告属于P6b。当前文档 MR #24 不意味着已执行实现、内容发布、MR合并、账号付费或生产部署。

本次计划验证记录：十任务/五十复选步骤、三十不同节点、八个 v2/二十二新 ID、全部前置边拓扑顺序、二十四模板的 4×4 参数域/非负与非零约束、384/834 编写目标算术及三十六个本地文档链接检查通过；仅验证计划决策与文档，未生成上述产品数据或执行产品回归。SSH 复核最新 master 仍为 8bf3c95b9833941182f4c8caa2da752e6d7838ce。
