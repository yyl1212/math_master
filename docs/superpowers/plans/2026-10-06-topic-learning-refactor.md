# 主题学习重构总执行计划

> **For agentic workers（致执行代理）：** REQUIRED SUB-SKILL：按用户选择使用 superpowers:executing-plans（Native，推荐）或 superpowers:subagent-driven-development。按任务逐项执行，以 - [ ] 跟踪步骤。计划审核及执行方式未确认前，不写产品代码。

**Goal（目标）：** 按三个可独立验证的阶段交付已确认设计，并在所有依赖就绪后完成体验切换。

**Architecture（架构）：** 阶段A建立分类和知识归属，阶段B提供独立个人学习，阶段C迁入真实历史并退出测评写入口。三个计划共享明确接口与资源边界，先后实施，不用阶段成果代替整体验收。

**Tech Stack（技术栈）：** Go 1.27.1、PostgreSQL 17.11、Next.js 16.3.7、React 19.3.0、TypeScript 5.9.3、Node.js 24.17.0；沿用现有 Vitest、Playwright、Zod、goose、pgx，不新增运行依赖。

**Spec（设计）：** [用户已确认的主题学习重构设计](../specs/2026-10-06-topic-learning-refactor-design.md)，2026-10-06 获书面确认。本计划待审。

## 全局约束

- 分类固定为 63 个一级主题、534 个二级主题、4,969 个具体主题；503 个辅助分类和534 个其他兜底节点单列，完整节点6,603 个。
- 个人状态仅 unlearned、learning、completed、reviewing；实际阅读开始，主动完成，已完成知识主动复习后回到已完成。
- 主题按当前公开可用知识去重；复习不降低完成率，零分母显示暂无内容；分类移动、版本变化不清除笔记或完成事实。
- 旧正文、摘要、来源、批准、原答案和原成绩保持；00001—00009 历史迁移保持原字节。缺失能力拒绝相关操作，不删除永久标记或历史来恢复。
- 控制请求8 KiB；笔记16,000 Unicode 标量且64 KiB、请求128 KiB；私有完整响应2 MiB；普通列表默认20最多100，历史默认20最多50。
- Go 请求8 秒、锁等待1 秒、Next 与客户端10 秒总截止；原账户时限保持。普通读写保留个人120/30、全局600/120 每分钟预算，后台重操作共用原工作槽。
- 每工作区100 知识、当前快照1,000 知识等原内容容量保持；6,603 分类节点不占知识容量。不增加运行依赖、连接池或常驻导入服务。
- Go 验证使用 CGO_ENABLED=0、GOTOOLCHAIN=go1.27.1、-timeout 5m；命令由 tools/verify/run.mjs 限540 秒；浏览器单 worker、零重试、批次480 秒，桌面1280×900、手机390×844。
- 数据库测试只用 TEST_DATABASE_URL 和 testutil.Database 创建的随机 math_master_test_* 库；缺配置导致 skipped 时不报告验证完成，不使用真实开发库或服务器。
- 页面沿用 en、zh-CN 功能词条和受限 Markdown/KaTeX/原创 SVG。GET、SSR、预加载和草稿预览不写学习记录；私人笔记不进入日志或公开材料。
- 未确认命令仅主动同键重试；不得自动重发 POST。每个业务任务先写行为失败测试，确认失败，再最小实现、目标回归和提交。
- 开发前更新 master 并新建 codex/ 分支；三阶段先后交付，每阶段审查与回归后创建 MR，不擅自合并依赖或部署。

## Review Focus 审查重点

1. 分类目录完整但知识未发布：目录可浏览，学习分母只取当前有效公开知识（A4、B4）。
2. 分类/内容两head变化：准备候选失效，任何读取不得返回混合版本（A6）。
3. 用户在另一个标签页写笔记或换账号：保留当前输入并显示冲突，私人内容不跨身份（B5、B6）。
4. 旧通过事实没有阅读完成：迁移保留原成绩但不生成新完成记录（C3）。
5. 真实环境schema/代码不匹配：拒绝相关写入，保留数据库与维护隔离（C4、C6）。

## 设计批准与执行选择

用户在2026-10-06 正式设计及草稿MR #37交付后回复“确认”，授权编写实施计划。设计正文标记已确认；本计划及三份分计划尚需审查。推荐 Native：任务共享配对发布、状态与迁移接口，由当前会话顺序实施，最后一次独立整分支代码审查。未获得执行方式选择前不启动实现，也不自动派发子代理。

| 阶段 | 计划 | 任务 | 可审查成果 |
| --- | --- | --- | --- |
| A | [主题分类与知识归属](2026-10-06-topic-taxonomy.md) | A1—A8，共8项 | 三级目录、来源适配、冻结归属、配对发布和主题页面 |
| B | [个人主题学习](2026-10-06-topic-study.md) | B1—B8，共8项 | 学习状态、主题进度、笔记、时间线、复习和登录入口 |
| C | [旧模块退出与学习迁移](2026-10-06-topic-learning-cutover.md) | C1—C7，共7项 | 反馈解耦、旧写退出、真实事实迁入、显式切换和恢复兼容 |

总计23 个任务、每任务5 个可检查步骤，共115 个步骤。A提供 taxonomy.KnowledgeRef/PairRef/公开查询及配对事务，B消费这些接口并提供study命令与记录，C消费A/B合同完成能力切换；不并行修改共享接口。实现每阶段先核对依赖已审查，起点master没有依赖时创建明确依赖的草稿MR，不能以用户“确认设计”推定合并授权。

```mermaid
flowchart LR
    A1[捕获与适配 A1 A2] --> A2[分类合同与查询 A3 A4]
    A2 --> A3[归属与配对发布 A5 A6]
    A3 --> A4[页面与门禁 A7 A8]
    A4 --> B1[独立学习与事件 B1 B3]
    B1 --> B2[进度 笔记 时间线 B4 B5]
    B2 --> B3[页面 登录 回归 B6 B8]
    B3 --> C1[反馈与退役 C1 C2]
    C1 --> C2[迁入与切换 C3 C4]
    C2 --> C3[整合 兼容 完整验收 C5 C7]
```

## 类型来源与阶段依赖

A3定义分类基本类型、SourceRecordRef={sourceId,workFamilyId,recordId,path,sha256}及CapturedBatch/Version，A4定义TopicSummary/TopicDetail/KnowledgeSummary，A5定义DraftTopicInput/View，A6定义PrepareInput/ActivateInput/ReleaseView。A7新增管理代理与真实网络客户端，A5/A6组件先以具名回调验证。A4的受控CLI只安装未发布分类，A6管理员会话才授予公开head。

B1定义StudyRecord/Detail/CommandInput/ReviewInput/NoteInput/NoteView/NoteReceipt、ListQuery和HistoryQuery；Overview、TopicProgress、HistoryPage在B4确定字段，并回填model.go与OpenAPI生成类型。HistoryPage={actorId,items:StudyEvent[],nextCursor:*string}，StudyEvent={id,kind,knowledge:KnowledgeRef,taxonomyVersionId,occurredAt,originEventId?,noteRevision?}；Overview={actorId,topics:[]TopicProgress,recent:[]StudyEvent,reminders:[]StudyContentReminder,unclassified:[]StudyDetail}。B5消费B1的笔记合同；C3定义MigrationReport/LegacyCursor，C4定义CutoverInput/Report。不允许新建同名但字段不同的平行DTO。

## 模式与发布规则

topic_learning_state 在00010创建，experience_mode 初始legacy。新taxonomy API在已发布分类时可用；首次配对发布建立taxonomy_heads，此后知识激活必须配对。00011只启用新schema并在goose永久记录曾启用，不改体验模式；00012只增加迁入和切换结构，不自动退出旧接口。topics模式只能由完成所有检查的维护命令显式启用。

前两个阶段的浏览器可在随机库以内部harness控制模拟topics界面；生产server绝不能导入控制模块，真实库在三个阶段就绪前保持原值。阶段C的切换检查必须重新核验完整schema、匹配head、迁入完成和兼容代码，不能因为测试控制能设置模式就认为正式可切换。

分类和来源覆盖记录完整，不代表网站正文都已整理或发布。基线文件绑定dot已接受的一次读取，A1每次接入重新捕获已接受批次，不能要求正在更新的源文件永远等于设计时摘要，也不能混合新覆盖矩阵和旧主文件。

## 共同验证入口

各命令独立执行，不串成长时间单批。所有命令在仓库根运行；frontend命令通过包装器切换cwd。数据库测试先准备专用TEST_DATABASE_URL和现有随机库保护，不能把真实DATABASE_URL用作测试连接。

| 类别 | 命令 |
| --- | --- |
| Node资料与门禁 | node tools/verify/run.mjs -- node --test tools/topic-ingest/*.test.mjs tools/verify/topic-learning-compatibility.test.mjs |
| 分类纯层 | node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/taxonomy -timeout 5m -count=1 |
| 学习纯层 | node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/study -timeout 5m -count=1 |
| 新普通数据库 | node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^Test(Taxonomy|Topic|Study)' -skip 'Capacity' -timeout 5m -count=1 |
| 容量分别单跑 | 各阶段任务明确的TaxonomyCapacity6603With1000Knowledge、StudyCapacityReadPages、TopicCutoverCapacity，不并行挤占预算 |
| 工具和HTTP | node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/cli ./internal/httpapi -timeout 5m -count=1 |
| 生成合同 | node tools/verify/run.mjs --cwd frontend -- npm run api:generate |
| 类型 | node tools/verify/run.mjs --cwd frontend -- npm run typecheck |
| 前端单元 | node tools/verify/run.mjs --cwd frontend -- npm test |
| 前端生产构建 | node tools/verify/run.mjs --cwd frontend -- npm run build |
| 静态检查 | node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go vet ./... |
| 命令构建 | node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/ ./cmd/... |
| 运维 | node tools/verify/run.mjs -- python3 -m unittest discover -s ops/tests -p 'test_*.py' -v |

浏览器先完成Go命令与frontend生产构建，再逐批运行node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- 对应.spec.ts；每批最多480秒，两个视口固定、无重试。资料与真实用户数据不用于fixture。新用例每份文件是独立批次；原测试分组保持既有预算，超时先诊断，不减少测试或放宽时限。

## 原回归与兼容审查

保留旧账户/内容/公开阅读、旧学习/题库/反馈/纠错的legacy模式回归；topics模式增加新体验与明确退休矩阵。旧安全历史、曝光、撤回、角色和不可变字节用例继续运行。旧业务预期只有topics模式下明确退出的操作可变为410，不删除整个测试文件来达到绿色。

既有兼容门禁包含旧文件摘要与CI字节保护。实现任务需要精确兼容映射，明确允许的文件、批准原因、目标摘要和阶段；原baseline保持，新增映射只能描述已批准变化。新增CI分批步骤不替换旧步骤；对要求精确旧字节的验证器追加精确审查后的模式例外，不对任意文件放行。

设计有意新增/v2及旧410，这些变化以已批准设计为依据。若实施发现需要改变分类数量、状态、隐私、历史事实、发布容量或退出规则，先回到设计审查，不把意外变化塞入计划。

## 需求到任务映射

| 设计要求 | 任务 |
| --- | --- |
| 三级分类、辅助类别、稳定代码、跨主题去重 | A1、A3、A4、A7 |
| dot持续更新、异构主文件、来源和隔离 | A1、A2、A8 |
| 冻结归属、审核、配对激活与撤回 | A5、A6 |
| 简单状态、完成事实与版本提醒 | B1—B4 |
| 私人笔记、冲突、删除、时间线 | B4—B7 |
| 主题进度、检索复习、正常登录入口 | B4、B7 |
| 来源相关反馈与内容纠错 | C1、C5 |
| 退出路线测评、新旧历史边界 | C2、C3 |
| 迁移幂等、有限切换、兼容恢复 | C3、C4、C6 |
| 账户权限、语言、素材安全、资源预算 | 各任务全局约束、A8、B8、C7 |
| 独立整分支审查、回归和MR | A8、B8、C7 |

## 计划自查

23 个任务均有文件、消费或产出接口、具名失败测试、运行命令、实现决策、通过检查和提交动作。三份分计划的Review Focus各五项均映射到具名用例。原schema与数据的保留、复杂模块退出、私人权限及发布事务预算分别有对应任务；不存在由“分类已确认”直接运行真实迁入或部署的步骤。

执行方式在用户审查计划时确定；此记录不表示115 个步骤已经实施。代码开发阶段按复核过的计划填写checkbox与实际验证结果。
