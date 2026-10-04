# 首批内容只读验收

P6a 提供原创草稿、来源定位及技术验收。`draft_ready` 只说明当前草稿满足技术数量、结构、来源与选题口径；P6b 还要落实真实独立数学审查、正常工作流批准及正式发布。隔离数据库的模拟审查不代表真实人员复核。

## 运行前准备

使用固定原字节快照、与之对应的来源报告和对象映射。原始来源与带编辑路径的映射属于私有运维输入，不上传页面或报告附件。不要在命令行写数据库口令；将 PostgreSQL URI 配置到私有环境变量。操作者身份只需要数据库 SELECT 权限。

```sh
CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go -C backend build -o /private/tmp/content-audit ./cmd/content-audit
/private/tmp/content-audit --mode=draft --route=elementary-foundations --version=1 \
  --root=/absolute/repository --snapshot=/absolute/private/snapshot \
  --source-report=/absolute/private/source-report.json \
  --source-map=/absolute/repository/content/source-maps/elementary-foundations.v1.json \
  --code-sha=FULL_40_CHARACTER_GIT_SHA --out=/absolute/new/report-directory
```

上述构建命令从仓库根使用 backend 模块。输出父目录必须存在并使用规范绝对路径；不接受含符号链接的输出父路径，macOS `/var` 等别名应换成解析后的 `/private/var`。输出目录必须全新，目录 0700、两个文件 0600；旧目录内文件不会被覆盖或删除。

发布态命令将 `--mode` 改为 `published`，移除 `--root`，加 `--database-env=MATH_MASTER_AUDIT_DATABASE`。没有 DATABASE_URL 默认值。读取同一个 RepeatableRead/ReadOnly 事务内的两组 head、固定路线版本、冻结正文与作者集合、当前批准和静态选题资格；总截止 8 秒，锁等待 1 秒，最多两个连接。审计不读取学习者答案，不写缓存、事件或后台任务。

## 报告与退出码

| 退出码 | 含义 |
| --- | --- |
| 0 | 草稿 `draft_ready`，或满足真实证据的发布态 `accepted` |
| 3 | `awaiting_review`：真实独立审查或学习验收未齐 |
| 2 | `not_ready` 或非法参数、对象/证据身份不匹配 |
| 1 | 配置、IO、数据库或预算失败；未生成成功报告 |

JSON 与中文 Markdown 使用相同结论和数量。报告只含安全数学身份/摘要、数量、覆盖见证和原因码，不含题面、答案、人员标识、私有编辑路径或连接字符串。达到最小 30 知识、20 模板、300 有效实例仍需逐节点主归属、前置闭包、至少两角度、例子/反例、安全图片、五题覆盖和一次曝光后可覆盖。

`math_master_test_*` 数据库自动 `fixtureOnly=true`，即使省略标记也不能解除；`--fixture-only` 只能加强标记。夹具正式计数恒零且不能 `accepted`。学习者自己的曝光/最近提交并非静态审计计数；浏览器和真实测评联调单独验证这些策略。

## P6b 真实证据

可选 `--evidence=/absolute/private/acceptance-evidence.json` 必须严格匹配 schemaVersion=1、当前 codeSHA、routeSHA、knowledgeHead/questionHead 的 ID 与 SHA。旧版本或旧 head 的 passed 不能复用。没有证据时保留具体待复核原因。

由实际负责人核验每一批准 decision 的真实人员独立性后填写 `reviewAttestations`，每项包含 `decisionId` 与 `independenceVerified`。工具仍会核对真实工作流冻结作者集合与 reviewer 不同、摘要和审查勾选；它无法证明自然人的身份独立性，也不会自动生成这些承诺。

`learningChecks` 记录八类 reading、pass、fail、prerequisites、review、practice_exposure、retake、feedback_correction，各项填写 `name`、`result`（passed/failed/not_run）及实际证据文件 `evidenceSHA256`。保存脱敏原始结果并复算摘要；不得将技术夹具改写成真实教学验证。任何 failed 都使报告 `not_ready`，缺项保持待复核。真实发布和正式 accepted 属于 P6b，生产部署另按 P7 审批。

## 输入与当前可用数量的补充约束

验收输入必须是普通文件；拒绝 FIFO、设备、目录及输入文件的符号链接。读取使用声明长度、前后文件身份/大小/修改时间检查和可取消读取；来源正文的 SHA 与条目解析使用同一份捕获字节，不在摘要核验后重新打开正文。原始 corpus 每文件独立上限64MiB，不能套用4MiB元数据上限；来源映射256KiB、元数据4MiB、安全SVG1MiB、报告8MiB及整条CLI8秒预算不变。未来超过64MiB的来源需要另设计流式导入，当前入口会返回预算失败。

外部验收证据中的 decisionId 和八种 learningChecks 名称必须分别唯一；相同值或冲突值重复都会拒绝，未知检查名称也会拒绝。not_run 不能被后面的 passed 覆盖。发布态的模板必须存在至少一个准确模板身份下当前可用且计入有效池的实例；专属单元/素材退出当前head导致全部实例失效时，该模板不会撑高20模板门槛。来源、审核、正文与两head的原核验仍先于计数。
