# 管理员当前知识管理与地图上线验收

日期：2026-10-10（Asia/Shanghai）。全部已授权模块及追加地图修复已完成并上线。网站：https://43.135.142.53；最终功能与工具发布提交：`6ae8ed780bc5e52720233e8221d47d7dfd14228d`。

## 最终行为

- 管理员直接上传、保存、发布、编辑、下架与删除知识；没有知识送审或业务版本选择。原数据ID为唯一身份，同ID同主题不重复，跨主题共享实体，重复已接收输入不撤销人工修正。
- 知识按63个一级主题、534个二级主题、4969个具体主题组织。地图移除无说明的“00-XX · 0”，改为明确“已发布N个知识点”；一级详情明确显示二级主题，二级详情显示具体主题；目录与知识列表分开，无知识时也显示子目录，链接保持稳定。
- 私人学习4状态、笔记与真实时间线，主题进度自动汇总；当前正文修正/下架保留私人数据，过期引用需刷新当前正文后再操作。登录进入我的学习，简单复习检索替代复杂练习入口。
- 编辑SSR字段在客户端接管前禁用，接管后启用，避免初始化覆盖/追加输入。语言切换保留输入。
- 健康与永久模式失败关闭，原迁移1–12和历史数学/审核事实保持；启用后拒绝旧写入、旧迁入与不兼容回退。

## 实际数据与恢复点

- 本次真实导入0、跳过0、冲突0，当前公开知识0。用户说明正式源为云端Knowledge_Standardized/versions/0013共1209条，work/0014未正式；本机无入口，按“下载不到的话暂时先不用真实数据测试”暂缓正式迁入，不发布示例内容，不声称核实了云端数学内容。
- 原30个知识点已移除公开读取，授权学习数据已准确清理：study记录2/事件2/幂等1/legacy链接1/笔记0；legacy学习记录1/事件1/幂等1/依赖2/解锁1。旧不可变知识/审核事实作为历史保留，managed公开源不读它们。
- 账户1、管理员角色、6603分类节点保留（63一级、534二级、4969具体、534目录其他、503辅助）；接口64个根组包含独立“项目其他”。最终数量与清理后相符。
- 清理前后保护指纹相同：`1a73ec011bf815284de8037d66f98e6e70e5fe725203fa37f1c2d17a818d5e80`。准确清理计划：`7dbcc88e1c709b4056a7742616bf946f0e578cdc8e146e5568b7762b835662eb`。
- 生产一致恢复点 `/opt/math_master/backups/precleanup-admin/20261010-2499515/`，113表、6工作区、9素材实际隔离恢复两次，临时容器/卷清除；本机独立主机副本3文件摘要一致。dump SHA：`f6777a7f6915c4e4ba2e5d71e4c6f6ba04f7db51b4b0a1ccac13dd6ed34b9e4e`。
- 原快照manifest `9a3c90343c208175302d6c65ee80b358ea0110868a0353e63c27f61d3c68d4df`；现场保留原快照，兼容副本仅补实测数据库名，dump及113表清单不改，兼容manifest `8d74fb446ea12db2fb67f02bafbccae86d9b5325a25f7d75b1eebc5ed76afd40`。后续代码已直接衔接无名UTF8快照，旧12备份命名规则不放松。
- 独立准备76.00秒，启用0.57秒，准确清理1.20秒，均小于5分钟。后续发布只自动备份/启动/激活，没有再次清理或启用业务。

## 代码、审查与CI

- MR43：https://github.com/yyl1212/math_master/pull/43，HEAD97d9c8b14项CI全成功，主体合入部署2499515。
- MR44：https://github.com/yyl1212/math_master/pull/44，最终HEADfbd4d698d871f60f7f82ddf4ff93094aa1cd8b41全部16CI成功，地图与协议修复合入f9e8eeb。
- MR45：https://github.com/yyl1212/math_master/pull/45，最终HEAD1390414601ddccbf524707564c980d8c7c6ed9f6全部16CI成功，最终合入与发布6ae8ed7。唯一一次当前学习CI初始化超时在操作前，未放宽原限时，onlyjob重跑后全成功。
- 全前端501/501、Python部署35/35、类型/生产构建与兼容门禁通过；原工具142及后续verify87批次均通过。Go核心16包、数据库135、原学习/反馈/审核/分类/迁入等回归通过，完整HTTP137.961秒，新旧维护CLI和proof目标拒绝通过。
- 原容量准备155.383秒、迁入28.406秒、清理0.387秒，分别独立5分钟；学习容量87.342秒。CI资源边界触发的5分钟prepare不计通过，保持预算后取得成功结果。
- 最初整分支fresh独立审查2P1+5P2均按Important修复，RED→GREEN；后续新发现的退休代理流取消、请求ID、快照格式、地图、CI拆分、流式跳转base/template/noscript及编辑水合均针对性审查与回归，无未处理阻塞。
- 前端原整体job两次30分钟取消后拆独立verify/verify_later，前驱49条原命令逐字保留，20/19浏览器批次独立PG/harness，30分钟job、9分钟命令、8分钟浏览器预算保持，诊断独立7天，命令库存门禁和源码摘要拒绝删检查/改预算。
- 准确154文件路径、294源码摘要登记，保留旧快照及12原迁移，不放行未知文件或API字段。

## 真实线上结果

- 最终归档2699条目逐文件路径、权限、内容与Git树一致，树SHA：`a4984f572a13af01c27be879313eaa78e3ce8d7e7f618cbeca4f1e25986bc9fa`；最终prepare/start/activate原工具自动全成功，current指向6ae8ed7。
- 严格系统CA/IP SAN、HTTP到HTTPS、cookie属性、旧草稿410、账户跳转、CSS与公式字体通过；40请求/并发2/失败0。
- 实际匿名Playwright桌面1280与手机390两组三级目录、零发布也显示子主题、稳定链接、统计无编码、无横向溢出全部通过。内置CUA工具超时，使用独立匿名浏览器；没有读取用户cookie、重置密码、伪造会话或宣称完成真实生产用户登录。
- f9切换曾因Next流式200+唯一固定账户meta被旧验收误判而停入口；严格内部核实无私人草稿后恢复，用已提交317d049源码SHA在服务器独立40请求核验后由原publish_current发布。最终6ae内置修复后的严格验证器，activate原样成功；接受合法3xx或唯一Next0/1秒/account标记，拒绝外链/base改写/惰性meta/重复/错类型/私人草稿。
- 桌面/手机延迟全部JS验证SSR编辑可见但禁用→接管启用→完整替换→语言切换不追加：4/4，15.2秒；地图+原学习8/8，24.8秒，管理员8例及原工作区12例通过。

## 实施取舍（ledger原顺序）

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
- Task 8: Ruling: 同HEAD PR前端两次whole30分钟限时，push全部通过 — 拆verify/verify_later两个独立隔离构建/PG任务，所有前驱f11执行命令逐字保留，30分钟job/9分钟batch均不加预算，新精确库存门禁+兼容审查 — 若错误丢检查会被命令库存和源码摘要拒绝，不能以重复取消当成功或无限重跑。
- Task 8: Ruling: 新版激活验收把Next.js合法流式200+唯一__next-page-redirect/1秒/account当异常，自动恢复也因同一误判停入口 — 在已审功能代码上严格实测无私人正文的固定账户跳转后恢复入口；验证器只接受规范3xx或唯一Next标记0/1秒/account，原安全检查保持，新增负例及审查 — 若误放宽会接受任意200或泄露，故固定目标/标记/次数/类型与私人草稿拒绝全部验证，不伪报旧工具已PASS。
- Task 8: Ruling: CI双语手机编辑输入出现新文本+原正文追加，源无ready门禁导致SSR可输入与client水合同步竞态 — 仅编辑fieldset在SSR禁用，React客户端接管后启用并save guard，原正文/语言状态不改；SSR-disabled缺失实际RED+延迟script browser验证 — 若错会阻止编辑或导致输入完整性问题，原字段值保留和全部原语言断言继续执行，不用重跑侥幸掩盖产品缺口。

## 暂缓小项

- Final: minor (deferred): managed个人页匿名访问目前显示通用不可用提示，页面头部登录入口正常，私人API返回401且没有内容泄漏；登录后的完整流程已由真实隔离浏览器回归，后续可把匿名提示改为明确登录提示。

最终HTTPS验证UTC时间：2026-10-09T21:08:19.827333+00:00，p95 647.674ms，证书到期 2026-10-15T08:26:12+00:00。
