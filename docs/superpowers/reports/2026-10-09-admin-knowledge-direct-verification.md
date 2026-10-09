# 管理员当前知识管理验收记录

日期：2026-10-10（Asia/Shanghai）。主体功能已上线，生产已启用 managed；线上最终验收发现的旧草稿代理410映射修复正在 MR44 CI，完整 HTTPS 验收待补丁上线后完成。

## 实际上线与数据

- 网站：https://43.135.142.53；主体 MR43：https://github.com/yyl1212/math_master/pull/43。
- MR43 精确 HEAD `97d9c8ba886f1341c45f7b5fca7d17da80977e23` 的14项CI全部成功后合入，部署固定提交 `2499515704ba8f8da980423dfcb150611aae5d36`。生产归档2698条目逐文件内容/路径/权限摘要与本机Git归档一致。
- prepare/start/activate均成功；数据库13，新模块健康字段 capability/schemaReady/managedMode均true，原topic五字段均true。
- 当前公开知识0条；原30已移除公开读取，旧学习记录及学习时间线已按授权准确清理。原不可变知识/审核事实作为历史保留，新公开源不读取它们。
- 清理study_records2、study_events2、study_idempotency1、study_legacy_event_links1、study_notes0；旧learning_records1、learning_events1、learning_idempotency1、learning_evidence_dependencies2、learning_unlocks1。
- 账户1、管理员角色及6603分类节点保留。分类为63一级、534二级、4969具体主题，另有534目录其他节点及503辅助节点。主题汇总接口64项包含独立“项目其他”。
- 保护指纹前后一致：`1a73ec011bf815284de8037d66f98e6e70e5fe725203fa37f1c2d17a818d5e80`。准确清理plan SHA：`7dbcc88e1c709b4056a7742616bf946f0e578cdc8e146e5568b7762b835662eb`。
- dot用户说明正式数据在云端 Knowledge_Standardized/versions/0013，共1209条；work/0014未正式。当前本机没有文件入口，按用户“下载不到的话暂时先不用真实数据测试”执行：真实导入0、跳过0、冲突0，未发布演示知识，不宣称核实了云端数学内容。

## 恢复点与独立步骤

- 最新一致快照：生产 `/opt/math_master/backups/precleanup-admin/20261010-2499515/`；本机异地副本在受保护私有目录保存。
- 两次实际隔离恢复，113表、6工作区、9素材核对；演练临时容器和卷已清除。dump SHA256：`f6777a7f6915c4e4ba2e5d71e4c6f6ba04f7db51b4b0a1ccac13dd6ed34b9e4e`。
- 原快照manifest SHA：`9a3c90343c208175302d6c65ee80b358ea0110868a0353e63c27f61d3c68d4df`。原工具为异名恢复省略database.name，维护CLI要求它；保留原恢复点，生成兼容副本，补充SELECT current_database()实测的math_master_preview，dump与全部其他清单字段不改。兼容manifest SHA：`8d74fb446ea12db2fb67f02bafbccae86d9b5325a25f7d75b1eebc5ed76afd40`；实际三文件双主机摘要完全一致。
- 首次准备未产生成功回执（清单缺库名）。重新独立准备耗时76.00秒，恢复/异地凭证通过后启用0.57秒、准确清理1.20秒；各步在5分钟以内。没有扩大时间预算或清理范围。

## 开发与审查证据

- 全前端500/500（含生产发现的新增2个代理案例）、tools142/142、Python34/34；typecheck/build/vet通过。Go全部核心16包、普通数据库135、原学习/审核/反馈/分类/迁入回归通过；具体长测试以各批5分钟运行。
- 十万事实容量准备155.383秒、迁入28.406秒、清理0.387秒，各独立5分钟预算；原合法最大学习容量87.342秒通过。
- 管理端真实隔离浏览器8例、学习6例、managed双语2例、原工作区12例通过。手机管理表格内部滚动和知识地图无页面横向溢出。
- 原整分支一次fresh独立审查2P1+5P2全部定为Important并修复，均RED→GREEN；套件发现的marker DELETE及目录表达式解析差异也已复现并修复，原迁移1–12字节和时间线断言保持。
- 生产补充MR44：https://github.com/yyl1212/math_master/pull/44。旧草稿后端410被代理原schema转503，且旧停用分支缺请求ID（无旧数据泄漏），只增加限定字段/文案/请求ID/4096字节的410分支，异常同时abort+cancel。针对性兼容审查发现流取消遗漏，已0≠1 RED→GREEN；旧schema和正常响应不改。后端停用分支补服务器请求ID、header/body一致且克隆请求；空ID/反射来访ID RED→GREEN，独立兼容审查通过。
- 准确154文件路径登记与294文件源摘要，保留旧快照及全部12原迁移；不放行目录、未知文件、未知API字段或任意摘要。

新增验证：请求ID补充完整HTTP回归137.961秒通过；新旧CLI1.615秒通过，直接篡改proof数据库的拒绝断言通过；准备工具接受原无名快照，旧12流程仍严格命名，独立兼容审查无阻塞。

用户追加地图问题：原接口已有二级主题，但显示无层级标题及“编码·0”。新增展示修复保留API/目录/链接，明确一级、二级、具体主题标题、发布知识点单位和独立空状态；3项unit RED→GREEN，桌面/手机三级浏览及原学习流程8/8、24.8秒通过。该修复随MR44更新，尚未部署。

## 线上验证与当前限制

- 实际匿名浏览器核对地图63/534/4969与0发布，桌面/手机可访问；管理页显示登录提示，旧editor/publications/review跳账户；私人管理API401、无泄漏。
- 内置CUA工具超时，使用已安装Playwright的独立匿名上下文验收，未改动生产知识或读取用户会话。没有重置密码、伪造会话或宣称完成实际用户登录。
- 完整ops/verify-deployment.py在旧草稿代理状态码发现上述503，暂未通过；补丁上线后须重新核对系统CA、IP SAN、HTTP跳HTTPS、cookie属性、公式字体、40请求及旧草稿410。

## 实施取舍（按ledger原顺序）

- Ruling: 中文任务标题改为 Task N: 中文名 — 技能脚本仅识别 Task 标题，不改变任务范围 — 若错误仅影响任务提取。
- Ruling: 增加 backend/internal/knowledgeadmin/testdata/valid-source.json — CI必须携带准确规范金样，不能依赖本机交付目录 — 成本为新增一个精确登记的测试素材。
- Task 1: Ruling: dataset_version 按准确源 schema 使用 int — 金样和 schema 均为正整数，不能用字符串承接 — 若错误所有合法文件无法解码；已通过类型定位修正。
- Task 1: Ruling: SourceCoreSHA 另外排除 original_binding/content_origin/original_type — 这些是来源声明，正文一致而来源变化不能造成 ID 冲突，设计优先 — 若错误跨来源重复会误冲突；TestSourceCoreExcludesBinding 已 RED。
- Task 1: Ruling: 服务端源规范金样不重读 dot 原书本机路径，原字节核验仅记录 false — 设计明确没有附件不得宣称原字节核实 — 若错误会误报来源证明；上传/manifest核验在任务3实现。
- Task 2: Ruling: 当前 DTO 增加明确 topicKeys，新增关联不改写已接收规范正文的分类声明 — 同 ID 可多主题，而 sourcePoint 不能混入 project:other 或互斥分类；原主题证据保留来源表 — 若错误多主题知识会无法再次编辑；TestManagedCrossSourceOtherMembership 已 RED。
- Task 4: Ruling: 管理员列表使用明确 KnowledgeSummary DTO，SQL只取摘要 — 满足列表不取整库正文，并避免以清空正文的无效SourcePoint作为公开契约 — 若错误影响管理列表客户端类型，详情接口保持完整源点。
- Task 5: Ruling: 反馈补充 validation.go、feedback_write.go、feedback_routes.go 和未发布00013的label分支 — 原计划列出的文件未覆盖输入验证、回执标签与有效性计算；不改00001–00012及原分支 — 若错误会阻断managed反馈或产生旧引用空指针；所有新增路径精确登记。
- Task 6: Ruling: 增加 auth-status.tsx、feedback/page-data.ts 与 auth_fixture.go 的精确改动 — 菜单需识别独立 managed 模式，反馈需接入当前引用，隔离 fixture 必须清理新增关联测试数据；生产不清除永久标记 — 若错误仅影响对应入口与测试隔离，任务8精确登记。
- Task 6: Ruling: 上传刷新恢复仅在 sessionStorage 保存操作 ID，不保存文件内容、笔记或来源路径 — 从服务器按当前账号查询准确回执，原始文件与重试键仅驻内存；未提交预览需重新选择原文件 — 若错误不能自动恢复未提交文件，界面明确待继续而不假报成功。
- Task 6: Ruling: 新增 globals.css 的管理表格/编辑字段样式 — 原全局 panel 仅布局而没有表单样式，手机表格需容器内滚动 — 若错误仅影响管理视觉，精确文件登记。
- Task 7: Ruling: 新增 knowledge_admin_health_export_test.go 与迁移13结构整体指纹 — 仅表存在不能保证唯一/私有/永久触发器仍有效；钉住约束、字段、函数与触发器的目录表达式 — 若错误将拒绝运行而不放行不完整库。readyz.content.capability表示当前二进制支持此协议，managedMode表示实际永久激活，原topic五字段保持。
- Task 7: Ruling: 原30清理移除公开入口与准确学习数据，保留 immutable knowledge_versions/publication_members/审核事实 — 设计明确旧审核历史不得改写，managed公开源完全不读取旧内容；无需物理拆除历史FK — 若错误旧审核备份仍存在，但无公开读取入口；收据明确 RemovedPublicKnowledge。
- Task 7: Ruling: MigrateLegacyStudyBatch 在永久managed能力开启后拒绝旧迁入 — 清理后不能通过维护命令复活已清除状态；原legacy/topics行为不改 — 若错误会阻止未迁入旧记录的再次迁入，当前历史只读仍保留。
- Task 7: Ruling: 新增 cli/knowledge.go、CLI测试、backend/Dockerfile命令清单 — 原计划仅入口文件无法复用受保护配置与备份验证；新CLI禁任意SQL/数据库URL参数，仅读取0600输入与凭证，原Topic备份上限12仍保持 — 若错误新维护命令无法随部署镜像使用，所有文件精确登记。
- Task 8: Ruling: 增加明确144路径清单 admin-knowledge-approved-paths.json — 逐项来自任务1–8文件和已记录补充路径，不使用当前工作树自动放行 — 若错误遗漏合法路径会使CI拒绝，额外文件不能通过。
- Task 8: Ruling: 跨100条拆分文件的已核实关系保留声明，不要求目标同文件存在 — 设计要求不存在目标只显示文字、不造占位知识，合法快照可能跨文件；同文件已存在目标仍检查source版本和前置环 — 若错误关系不会自动跳转，内容证据仍由dot负责。来源未知占位字符串补充校验，两个RED已观察。
- Task 8: Ruling: verifier自身policy SHA字面量在源文件摘要中归零 — 原值仍按整个policy字节核验，归零只消除policy→工具→policy摘要循环，工具其余字节精确钉住 — 若错误会拒绝而不放行未登记工具代码。新验收文档/计划仅精准路径登记，持续记录实际部署证据；原已有文档字节规则不放松。
- Task 8: Ruling: 增加严格mode-client钩子、原导航/作者测试的明确legacy模拟、语言覆盖清单及managed双语浏览器场景 — 原测试会把新的模式请求当作账户请求，错误中断旧鉴权去重断言；提取钩子后保留原全部断言，并新增managed隐藏旧菜单断言 — 若错误仅影响测试与菜单，不放宽权限/模式失败关闭。五个精确文件加入149路径清单与方案。
- Task 8: Ruling: 整分支独立审查安排在全部实现与主要验收后、MR合入和部署前 — 任务8本身含部署，不能等部署后才执行技能的最终审查；遵守已确认方案的上线前审查顺序 — 若错误会推迟上线，不在未审代码上进行生产变更。
- Task 8: Ruling: 用户最新“下载不到的话暂时先不用真实数据测试” — 本次改为允许空库启用新模块，正式0013迁入暂缓，不发布演示内容；准确旧30/授权学习清理仍用最新备份+恢复异地凭证 — 若误解会使公开知识暂为空，原内容/审核事实及一致备份保留且可通过兼容当前管理重新导入；不能回退永久旧模式。方案与设计同步更新，并纳入本次唯一fresh审查补充提交。
- Final: Ruling: reviewer未判断实际0013内容 — 用户已暂缓真实数据测试，本次不导入1209，也不宣称格式/数学已核实 — 成本为真实迁入留待获取正式文件后执行。
- Final: Ruling: reviewer未判断生产backup/offsite/activation/cleanup现场 — 在本次部署前由本人用最新实际证据核验，不能以代码/模拟测试代替 — 成本为现场失败时保持库与备份、拒绝扩大清理。
- Final: Ruling: 增加workflow_tx.go与content_error.go准确范围 — HTTP410前置检查不能约束已排队旧请求，事务拿切换锁后必须复验且错误映射410 — 若错误旧请求可能落库；新回归已RED观察，正常legacy流程保留。
- Task 8: Ruling: 原始快照为跨数据库恢复而省略 database.name，维护 CLI 要求该字段；保留原快照，生成受保护的兼容副本并补充 SELECT current_database() 实测名称，dump 和全部113表清单字节/语义不改，双主机校验并再次恢复原 dump — 不修改或绕过应用验证，兼容副本记录原始 manifest 摘要 — 若错名称或副本摘要不符则 CLI 拒绝，原恢复点完整保留。首次准备未产生成功回执，第二次准备单独重新计时最多5分钟。
- Task 8: Ruling: 生产 HTTPS 旧草稿代理返回503而后端410（无内容泄漏）— 增加 content-proxy.ts/test 两个准确路径的严格410专用边界，原schema/正常合同不变；源码字节重新登记，独立 MR 和针对性兼容审查后部署 — 若错误会隐藏停用原因或接受非规范错误，未知字段/文案/ID/headers RED→GREEN覆盖。该问题是生产新发现，另行针对性审查，不复审原7项修复。
- Task 8: Ruling: 实际后端停用响应在serveContent前返回，requestId空或反射来访头 — 在serveManagedLegacyRetirement决定响应后生成服务器ID并克隆请求，header/body一致，未启用legacy仍返回false不动；原已有knowledge_current_routes.go/test准确摘要更新 — 若错误会使严格代理继续503或污染来访请求；TestManagedLegacyRetirementResponseBoundary header空/forged RED→GREEN，进行针对性兼容补充审查。
- Task 8: Ruling: 后续prepare与CLI直接接收真实可移植快照格式 — 仅maxMigration13+math_master_preview+省略name+UTF8接受；受保护proof必须目标匹配，显式wrong/null/empty名称拒绝，旧Topic12严格名称不变；prepare固定同一生产目标且拒绝显式错名 — 若错会误拒绝原快照或减弱源库检查，因此旧协议和证明/摘要断言均保留；正确物理路径下旧代码RED→GREEN、Python实际metadata形状KeyError RED→GREEN，进行独立兼容审查。不修改原snapshot捕获/恢复工具，当前现场兼容恢复点仍保留。
- Task 8: Ruling: 用户指出编码/0及二级层级不可识别，实际接口和浏览器确认子主题已有但无层级标题 — 保留目录/API/稳定链接，仅明确二级/具体主题区块、发布数量单位及知识空状态，新增准确catalogue回归路径并随MR44发布 — 若错误会造成浏览误解，不改分类归属或发布知识；用户已两次“继续”，不重复请求批准。

## 暂缓的小项

- Final: minor (deferred): managed个人页匿名访问目前显示通用不可用提示，页面头部登录入口正常，私人API返回401且没有内容泄漏；登录后的完整流程已由真实隔离浏览器回归，后续可把匿名提示改为明确登录提示。
