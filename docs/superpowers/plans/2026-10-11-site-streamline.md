# 网站功能精简实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans inline. 每任务按失败测试→实现→通过验证记录，不重复已完成任务。

**Goal:** 按用户确认的37项审核收敛知识学习和管理员管理，移除生产旧业务，保留数据。

**Architecture:** 生产当前页面与固定旧入口兼容测试构建分离。新管理复用managed接口；私有历史只增加名称投影，更新提醒统一确认基准。权威managed门禁关闭旧纠错访问与worker写入。

**Tech Stack:** Go1.27.1、CGO_ENABLED=0、Next16.3.8、React19.3、Node24.17、Vitest5、Playwright1.63、PostgreSQL17。

**Spec:** docs/superpowers/specs/2026-10-11-site-streamline-design.md

## Global Constraints

- 保留所有知识、归属、账户、学习、笔记、反馈与旧档案；不重导或清库，不改原迁移。
- 63/534/4969分类、8类型、难度1至5、同原始ID同主题唯一；单文件64MiB且至多100条。
- 多语言名称来源于当前规范目录；数学正文不机器翻译；只给本人私有数据。
- 没有审核或业务版本切换；保留CSRF、幂等、并发编辑保护及历史身份。
- 旧入口兼容fixture固定于base06d04a2eb20cf2a56cfec57283224583659c2ffd；兼容构建不得进入部署包。
- 原39浏览器批次、原Go容量批次与所有预算保持；单次验证不超过10分钟，不skip/重试/only。
- 从最新master已创建codex/site-streamline，原数据需部署前后只读核验。

## Review Focus

- 已下架/删除但本人有历史的知识：保留名称与事件，不泄漏正文、不提供失效阅读链接。
- 多文件部分成功再失败：成功不重导，失败结果未明先查询回执。
- 旧角色账户：正常登录和读本人数据，新管理员权限不能自动获得。
- 内容变更后保存笔记或早期复习ref：提醒不错误消除或永久保留。
- 兼容构建或archive查询试图启用旧生产业务：必须失败，worker不得改旧任务。

---

### Task 1: 公共主题选择与结构化字段
**Files:** 新增frontend/src/components/topic-picker.tsx及测试；新增frontend/src/features/knowledge-admin/structured-fields.tsx及测试。
**Interfaces:** TopicPicker({value:string[],onChange:(string[])=>void,multiple?:boolean,leafOnly?:boolean,disabled?:boolean})；SourceFields/ExplanationFields以当前规范数组受控输入输出，不写网络。
- [ ] 写主题规范名/双语值保持/叶分类限制/已选值加载，以及来源和解释增删保持字段的失败测试。
- [ ] 执行对应Vitest观察RED。
- [ ] 用公开/api/v3/topics和现有LocalizedName实现选择；结构化字段保留已有属性，未知类型不改写。
- [ ] 对应测试通过，类型检查通过，记录验证并提交。
Expected: 当前分类键不变，无raw code主标签，不输出非法/空主题。

### Task 2: 首页 导航与阅读简化
**Files:** frontend/src/app/page.tsx，components/site-header.tsx/auth-status.tsx，features/reading/current-knowledge-view.tsx；新增当前首页组件与CSS、测试。
**Interfaces:** 消费TopicPicker公共名称和公开TopicPage；Homepage仅接收公开概览/登录信号，不接收他人学习数据。
- [ ] 写首页独立、匿名/学习者/管理员菜单、关联名称与不可公开目标无链接、语言切换测试并看RED。
- [ ] 首页站点介绍+主题概览+登录/学习入口；当前主导航按角色只显示现行功能；关联链接复用已加载标题，来源详细引用折叠。
- [ ] 保留语言草稿与数学正文；相关目标无来源时显示本地化不可用而非原始ID。
- [ ] 运行组件测试与构建，记录通过后提交。
Expected: 不存在算法推荐、独立通知/纠错/网站反馈菜单或旧角色管理入口。

### Task 3: 知识上传 列表 编辑
**Files:** features/knowledge-admin/upload.tsx/list.tsx/edit.tsx、测试、样式。
**Interfaces:** 消费Task1选择器和结构化字段；现有previewSource/applyImport/readImport与editToken调用不变。
- [ ] 写多文件一次应用、两种发布选择、部分失败与恢复、重复保护和输入语言保持测试并看RED。
- [ ] 汇总文件预检，保留有效项选择高级详情；一次确认逐文件提交，显示每文件结果与有界恢复。
- [ ] 主题名称筛选；编辑按基本/数学/分类难度/来源解释分组，规范数据保持，删除/发布/恢复接口不移除。
- [ ] 管理组件测试、类型、当前管理浏览器回归通过后提交。
Expected: 无逐文件重复确认，无增大上传限制，无JSON必填编辑界面。

### Task 4: 私有历史名称与更新提醒
**Files:** backend/internal/knowledgeadmin/study.go，internal/study/history.go或对应DTO，store/knowledge_admin_history.go/study_history.go/knowledge_admin_study.go；frontend/lib schemas/types；managed-history/pages；api/openapi.yaml/generated。
**Interfaces:** ManagedHistoryEntry增加title/titleZh/available显示投影；旧历史增加可选title/titleZh；同一确认ref选择方法用于详情/列表/概览。
- [ ] 写名称与下架历史owner、原身份不变、笔记不清提醒、旧review之后新completion清提醒测试并看RED。
- [ ] 分页后LEFT JOIN名称不返回正文；本人旧历史按原版本身份取名称或归档占位。
- [ ] 按完成/复习时刻选最新确认，无确认用最早事件；三个读取投影一致，不改笔记写入。
- [ ] 历史主题用名称，编号详情折叠；早期记录单独折叠展示、保留两个独立游标，避免重放。
- [ ] 生成契约并同步严格schema；对应Go<=5m、前端与浏览器通过后提交。
Expected: events/notes/state不变，新增字段只显示，旧客户端契约同批发布。

### Task 5: 反馈合并与角色收敛
**Files:** feedback/ticket-list.tsx/review-panel.tsx/page-data.ts，auth/admin-users.tsx，相关backend反馈与角色事务；测试。
**Interfaces:** managed业务权限admin-only；响应仍可读取原四角色，角色写入界面只学习者/admin；反馈当前target仍site或managed-knowledge。
- [ ] 写旧reviewer不能处理当前反馈/授予权限、admin可正常处理、旧角色可读账户测试并看RED。
- [ ] 网站反馈并入本人反馈；队列简洁动作，提供对应知识编辑入口，去掉旧版本/撤回依据输入。
- [ ] 管理员明确授予角色；不自动升级旧editor/reviewer；保留重置密码和会话功能。
- [ ] 权限/反馈/角色回归通过后提交。
Expected: 后端权威授权与前端一致，旧反馈与原身份不清理。

### Task 6: 生产退场与隔离兼容构建
**Files:** 旧editor/review/发布/纠错/通知/路线/练习/测评/题库/domains页面；knowledge_current_routes.go、correction_tx.go/jobs.go、runner；新增固定legacy fixture清单/build脚本，playwright config和生产退场spec。
**Interfaces:** 生产旧页面只最小跳转/退场壳且不import旧业务；MATH_MASTER_E2E_FRONTEND只用于测试选择独立已验证构建，不影响生产配置。
- [ ] 写managed旧纠错GET/POST/retry退场、worker停止不改任务、prod依赖清单无旧组件的失败测试并看RED。
- [ ] 固定base原页面/header及必要旧业务组件的准确快照作为兼容构建输入；独立临时前端构建复用当前共享库，不复制旧后端。
- [ ] 生产旧页面替换为不导入旧组件的壳；通知转学习；旧纠错全部退场；实际历史读仅保留必要study/history/note。
- [ ] 纠错用户/系统事务按权威managed状态及原锁序停止，runner识别退出；原任务不清空、不伪造完成。
- [ ] 原39批次在固定隔离兼容构建中完整运行；新增生产真实导航/退场/权限/历史/批量上传双视口验证。
- [ ] 更新精确冲突测试（知识退场“纠错是unrelated”的旧预期），对应通过后提交。
Expected: 新生产产物无旧业务页面依赖，测试兼容副本不能用于生产启动。

### Task 7: 兼容审查 完整验证与上线
**Files:** tools/verify准确白名单、基线目标与工作流新增检查；语言覆盖；验证报告。
**Interfaces:** 消费Tasks1-6最终源，基线保留原快照、API原语义和全部迁移；仅登记本次已审范围。
- [ ] 检查37项覆盖与共享接口，记录每个兼容例外；不得整体重置或以跳过断言通过。
- [ ] 前端全测试/类型/双构建/浏览器、Go关联范围、验证工具与独立代码审查通过。
- [ ] 创建Git MR并附到当前任务；精确最终HEAD必需CI通过才合并。
- [ ] 固定Git树校验、备份、生产构建与部署；独立HTTPS和真实账户只读验证。
- [ ] 核对知识与归属指纹，学习/笔记/历史未丢失；完成报告并归档本计划记录。
Expected: 每个成功声明有实际证据，部署不包含兼容构建、凭证或真实测试数据副本。
