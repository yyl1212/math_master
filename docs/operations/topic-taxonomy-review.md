# 主题分类阶段审查与开发裁决

基线 master：bca91cc98d75af53963789f28cdda5ca96871e17。独立审查范围：bca91cc…3ed6afc，gpt-6-astra/high；审查者未改源码。本记录只涉及阶段 A，个人学习及最终切换继续按 B/C 执行。

## 审查结论与修复

发现 0 项 Critical、3 项 Important、2 项 Minor。三个 Important 均接受并在一次 TDD 修复中处理：

- 正文版本升级后的主题归属以当前已保存知识身份提交；旧主题和来源作为待核对候选，保留编辑输入。版本升级及刷新后旧侧车用例覆盖。B 阶段学习能力先安装但体验仍 legacy 的兼容边界另经 RED→GREEN 验证。
- 已安装分类结构缺损、分类 head 表或单例行缺失时，旧激活拒绝独立切换；已发布历史是持久证据。表缺失和行缺失均已先观察失败。
- 保护触发器禁用时，保存、送审、审核拒绝且原事务整体回滚；只有没有任何分类结构的旧环境才能跳过分类挂点。三种写入故障均已先观察失败。

最终验证：119 项 Node、429 项前端、typecheck/build/diff/gofmt/vet 和完整编译通过；375 个 store 清单测试及其余 Go 包全部有通过记录。363 个原常规用例分五组（91.409/123.405/117.506/73.983/4.826 秒），11 个容量用例独立运行，新增 legacy 能力兼容用例及最后受影响回归亦通过。最大容量批次153.826秒，单测试时限五分钟。82 项真实桌面/手机浏览器4.2分钟通过，单worker、零重试、480秒预算。未部署、合并或批量批准真实课程。

## 全部开发裁决

- Ruling: 计划中文任务标题保留A编号并增加Task N工具标记 — 技能task-start只识别英文数字标题，正文与115个步骤不变 — 若错误只影响提取定位，已按A1—A8逐项映射。

- Ruling: 从最新master新建分支后仅cherry-pick已确认文档提交，不合并依赖MR #37 — 保持开发基线要求并使计划可读；阶段MR明确依赖 #37 — 若该依赖后续变更需重新核对文档差异。

- Ruling: 新增json.mjs共享严格JSON词法解析 — capture及后续adapters需要拒绝重复键、NUL和错误Unicode，旧工具不变 — 若错误可能拒绝合法来源，采用实际元数据烟测及边界回归。

- Ruling: 原来源包字段作为数据归档，未知字段拒绝作用于捕获元数据根合同，不能用网站schema否定异构正文 — 正式发布仍经旧严格DraftInput及审核 — 若错误只影响未接入草稿，不授予公开批准。

- Ruling: 来源登记中previous_catalog_snapshot_before_source8_reconciliation支持其实际{sha256,book_id_count}历史摘要形状，其他before_batchN仍仅接受SHA字符串 — 已读取实际131批字段并做RED→GREEN，未知键仍拒绝 — 若形状以后变化会明确拒绝新批次，不伪造接受。

- Ruling: buildDraftInputs返回{packages,issues}报告，而非裸数组 — 计划同时要求输出缺失映射问题却未定义错误出口；保持每个TopicDraftEnvelope格式，A7消费者取packages — 若错误仅影响新的工具消费者，类型/合同测试会阻止不一致，不影响旧导入。

- Ruling: standalone schema使用RE2可编译的not.pattern约束，不使用JS lookahead或反斜杠u文本 — 当前Go JSONSchema库实际编译拒绝这些语法，已以同批边界测试验证 — 若错误可能错误放行路径，Go/TS额外同样验证路径并拒绝。

- Ruling: taxonomy外键publication快照列使用text，匹配历史SQL主键而非按UUID形状推测 — 未修改原迁移，实际PG迁移测试确认 — 若错误会迁移失败，现已通过。

- Ruling: 新增build-catalogue.mjs将受保护快照转为归一档案并保留署名许可，新testutil/taxonomy.go复用完整原创分类fixture — 填补实际CLI输入转换和跨包成功路径验证，不增加发布权限 — 若错误可能输出被严格Schema拒绝的草稿，不触及真实head。

- Ruling: workflow_tx.go保留新的taxonomy固定错误，并在A5提前增加两份主题归属词典、作者/审核组件测试 — 新工作流错误原来被旧authError吞为不可用，实际RED已复现；纯组件需要可翻译提示 — 若错误影响新API错误映射，A7合同测试覆盖，旧错误分支不变。

- Ruling: TopicFields明确接收单个知识AssignmentInput及保存回调，DraftEditor同时接收可选主题视图/刷新回调 — 原计划仅列DraftTopicView无法编辑未归类知识或传回当前修订，依照单知识8KiB设计补齐 — 若错误仅影响新接线，A7真实代理和页面测试验证。

- Ruling: 提前补充e2etest/auth_fixture.go的显式taxonomy重置表清单，复用A8 taxonomy_fixture.go — 全量测试实际13项reset失败，SQL诊断0A000明确新增外键拒绝旧TRUNCATE；仍只作用于OpenVerified随机隔离库 — 若遗漏表则场景重置明确失败，不使用宽泛CASCADE。

- Ruling: 全套Go分为常规回归和独立容量组运行，完整容量回归纳入A8阶段验收 — go test ./... -timeout 5m实际在累计306秒时终止于CorrectionCapacityNotifications；已完成包除修复前e2etest reset外无行为失败，不能将整体超时称为通过 — 若容量组漏跑阶段不验收，每组仍严格小于10分钟。

- Ruling: 发布正文新增私有ReleaseDocument/ReleasedAssignment，以准确数学SHA、冻结归属摘要和同一决定绑定；对外仍仅ReleaseView；新增ReadTopicRelease供A7消费 — 既有计划只有准备/激活接口却列了读取路由，私有审核依据不能塞入公开视图 — 若合同有误A7严格响应与匿名边界会拒绝，原正文字节不改。

- Ruling: command-controls.tsx仅导出既有PasswordDialog供成对发布复用，并补充发布词典和UI行为测试 — 密码核验需沿用原可访问对话框且绝不能自动激活，RED→GREEN已验证 — 若错误影响旧对话框，414项前端回归已覆盖。

- Ruling: 在A7补齐受保护GET /api/v2/admin/publications分页历史、TopicReleasePage合同及Store查询 — 计划保留发布管理却只列单条读取/准备接口，分类仅调整不会进入旧知识快照历史，需要可查的成对发布历史 — 若新增合同不一致将由生成类型、严格响应及真实HTTP回归拒绝；旧v1路径/schema保持原样。

- Ruling: 新taxonomy管理预检与旧publication服务共享验证槽和限流，补齐原固定内容错误到TaxonomyResult，保持原错误状态/消息 — 新模块不能绕过既有2并发预算或吞掉审核/版本错误 — 若调用者误配置共享槽会阶段容量门禁失败，服务器和harness使用同一服务实例。

- Ruling: 新增lib/taxonomy/protocol.ts统一路由、查询字节、2MiB响应、原固定错误和8KiB Go转义预算，及对应3个测试文件 — 计划列出多个客户端/代理但未列其共同词法入口，集中后防止不同入口放宽边界 — 若错误会同时拒绝新入口，当前8项和Go合同检查覆盖，旧实现不修改。

- Ruling: 旧KnowledgeMap/DomainView保留纯组件兼容，新SSR在分类尚未发布时使用旧目录；最终topics模式下C阶段须关闭该fallback — A阶段分类导入与配对初发之前不能破坏原可读流程，已确认渐进切换 — 若C漏掉模式约束可能显示旧导航，C验收必须覆盖缺失分类时拒绝fallback。

- Ruling: 完整原创测试fixture使用实际63一级代码但原创名称，并加入13C60深链接；提前补A8 fixture数据形状 — 计划A7真实浏览器要求13C60/中文搜索，旧连续00—62占位fixture没有该深链接 — 若代码漏掉实际主题则严格6603父子/计数测试会拒绝；未复制原资料正文。

- Ruling: 辅助/other未指定level时默认2/3，主类默认1；界面自动层级匹配类别，知识标题命中也定位其审核主题及祖先 — 旧默认1会把辅助503/other534误显示为0，真实RED已复现 — 若语义偏离，显式level仍保持精确过滤，主类计数不变。

- Ruling: 新增TopicDraftEditor/TopicPublicationWorkspace客户端适配器、受保护SSR三页面及topic-transfer导入器 — Next服务器不能把命令回调序列化给组件，A2的topic-draft侧车必须有实际消费者；导入只设置未保存候选、不自动提交，网络重试复用输入/键 — 若错误会卡在归属就绪/严格来源校验，编辑/导入RED→GREEN已覆盖。

- Ruling: 提前实现A8真实topic-catalogue场景、harness taxonomy服务接线和场景测试 — A7要求真实Go/PostgreSQL浏览器，不能等A8才有数据；场景只在loopback/token/OpenVerified随机库中用原创数学及真实审核/配对发布 — 若隔离缺失会有测试副作用，原harness隔离保护和新匿名管理401断言保留。

- Ruling: 新门禁包含全部master受跟踪文件与精确新增白名单，并给12个既有受保护改动保存master字节供旧门禁逆向比较；扩展admin-review-compatibility.mjs，旧基线JSON不改 — 否则既有API生成类型与冻结事务挂点的合法新增会被旧逐字节检查拒绝；实际旧门禁RED→GREEN及篡改测试通过 — 若逆向路径过宽会掩盖变化，固定目标SHA/策略SHA及全树检查共同约束，整分支审查必须核对。

- Ruling: 提前补齐缺失/禁用不可变触发器能力拒绝，并增加taxonomy_control只读测试统计 — 表存在并不等于保护可用，满容量故障实际暴露；控制只在原随机库token服务注册，生产server无依赖 — 若探针错误会拒绝新功能，已验证正常满容量与故障路径；不允许用旧流程绕过修复。

- Ruling: 新CI按固定目标SHA逆向恢复旧workflow后运行原命令/时限门禁；新增topic单独批次，仅从原宽集成剥离新容量用例并单独执行；旧测试逻辑及旧SHA记录保留 — 旧CI门禁要求逐字节工作流，直接追加会误判；普通包累计超过五分钟已证实需独立容量 — 若逆向过宽会掩盖预算变更，目标SHA/策略PIN及变异测试约束，最终审查须核对。

- Ruling: A8步骤4的独立整分支审查/MR按Native技能的final review/finish门禁执行，在Task8实现与回归提交后运行 — 否则task-done需审查而技能需task-done之后审查形成循环；阶段交付仍必须审查/MR结束 — 若收尾未做不能声称阶段A交付。

- Final: Ruling: 对部分缺表/禁用保护的已安装分类环境拒绝旧事务，只有所有分类表均未安装才兼容跳过 — 数据损坏不能解释为未启用，否则正文和归属证据会分离 — 若错误会拒绝损坏旧环境的写入，需修复迁移/保护后继续，不允许静默降级。

- Final: Ruling: 个人学习、笔记、时间线、登录和退役不作为 A 缺失 — 按已确认 B/C 计划继续开发，不将 A 功能当作整体交付 — 若分期衔接有误会在 B/C 验收阻塞，不能宣称整体完成。

- Final: Ruling: topics 模式 legacy fallback 与最终迁移回退留给 C 验收 — A 保留初始化前可读兼容，C 必须有故障回归关闭 fallback — 若 C 遗漏会显示旧导航，交付门禁必须拒绝。

- Final: Ruling: 真实数学正确性、来源许可充分性和部署不由本次技术审查批准 — 只审受保护数据接入，真实资料仍走内容审核，当前无合并部署授权 — 若错误可能引入未经批准的课程，本阶段不产生真实公开批准。

- Final: Ruling: 只用已发布分类历史或 topics 模式识别缺失配对，study_enabled 单独不代表已启用主题体验 — B2 要先安装能力而 B7/C 保持旧模式，新增实际 RED 显示该标记误阻塞健康的旧激活；故障结构仍由完整能力检查拒绝 — 若错误可能在首次配对前允许旧流程，但当前模式仍 legacy 且已有分类发布必拒绝。

- Final: Ruling: 新增中文 topic-taxonomy-review.md 保存裁决与审查证据并精确加入白名单 — Native 收尾必须移除短期工作区，用户仍需要可审计的全部决定 — 若遗漏会丢审查上下文，提交前按 ledger 自动收集全部 Ruling/Minor。

- Ruling: 将远端常规store清单按70项精确分为四个槽，最大工作流/题库容量单独执行，新分类由已有专门批次覆盖 — 全部原用例保持，超过四槽直接门禁失败，不增加单批五分钟或工作槽 — 若划分错误会漏回归，纯分组覆盖/真实清单与现有workflow逆向保护共同核验。

## 远端 CI 分批修复

首轮远端旧宽集成批次在 300.016 秒累计超时，没有报告行为断言失败。按完整清单改为每批最多70项，超过四槽直接失败；最大内容工作流、最大题库工作流及题库候选容量改用独立的原五分钟命令，分类仍在已有独立批次覆盖。Example与Fuzz种子保留。新分组覆盖/边界经过RED→GREEN，122项完整Node通过；常规110项按70/40分批实际通过，耗时50.932/33.944秒。代码业务未改。远端重跑结果以当前MR最新提交为准。

- Ruling: backend checkout显式fetch-depth0，保留固定基线archive变异断言 — 默认浅克隆无基线对象导致git archive返回128；不删除保护或改为当前HEAD — 若历史拉取失败CI明确拒绝，应用与数据库不改。

## 暂缓的小项

- Final: minor (deferred): 搜索结果补充完整可点击祖先路径；当前进入详情可查看。

- Final: minor (deferred): 主题详情增加有公开内容/全部切换并在分页前筛选；当前可浏览全部含零内容主题。
