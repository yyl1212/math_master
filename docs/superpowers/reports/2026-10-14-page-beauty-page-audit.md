# 第三阶段逐页视觉分析

基线：master `bf8e02c2193ffca1c6534b1991399cacce62ce12`。分析范围为49个页面入口、2个学习工作区变体和全局状态；不把旧业务源文件当作当前生产页面。

本轮23个当前入口全部做源码检查。另使用本机独立合成数据，采集25个代表页面/状态×中英文×1280/390/320，共150张基线画面；每张记录文档宽度、页头高度、标题数和文字尺寸。反馈详情及管理员反馈详情本轮仅作源码审查，实施时补真实讨论状态图。缺内容、权限、冲突等其他状态的完整矩阵也是实施验收工作，不能由这些基线图推断已经通过。

私有本机证据：`/private/tmp/math-master-page-audit-20261014-sl0qvkjk/metrics.json` 及同目录截图。截图只含仓库原创技术夹具及虚构账户，不含生产资料。临时证据可能清理；本文保留可重现条件及主要数值，实施时T1会生成受版本管理的合成基线摘要及选择性截图。

## 1. 共性发现

| 编号 | 已见证据 | 设计决定 | 实施任务 |
| --- | --- | --- | --- |
| G01 | 中文标准夹具：1280页头104px，390/320为223.296875px | 语言/账户同一utility区域，减少手机分行；保留DOM阅读顺序 | T2 |
| G02 | 当前Markdown14px，手机13px；阅读、笔记、比对共享 | 当前study排版17/16px；旧默认排版和安全渲染保持 | T3 |
| G03 | 复习沿用全局大h1，学习页另有38px覆盖；h2也不一致 | 首页/普通页/正文三级标题尺度，纯展示PageHeading | T2—T6 |
| G04 | muted暖底4.497:1；装饰边界白底1.345:1 | 次要文字加深；控件边界与卡片装饰分开 | T2 |
| G05 | 原工具栏按钮、元数据字号和提示样式分散 | 功能文字≥13px，按钮目标44px、表单48px，状态文字保留 | T2—T6 |
| G06 | 150个样本无页面级横溢；管理员数据表已有局部横滚 | 保留局部滚动能力，扩大文字后重新验证，不能删列或裁公式 | T3、T6、T7 |

## 2. 当前公开页面

“文件”均相对于 `frontend/src/`。P0优先服务阅读，P1整理主要发现入口，P2补齐账户和管理。

| ID/优先级 | 页面与现状 | 本阶段改进 | 文件/任务 | 关键保留与验收 |
| --- | --- | --- | --- | --- |
| A01/P1 | `/`：平静清楚，但大标题/上下留白较重，少量主题时卡片区显空 | 保持独立学习首页；控制hero高度，已发布数量降为元信息，主题卡片与三步说明统一节奏 | `features/catalogue/current-home.tsx`、`styles/current-home.module.css`；T5 | 登录/匿名CTA去向、最多6个概览主题及真实数量不改；0/1/6主题不伪造填充 |
| A02/P1 | `/knowledge`：搜索、层级标题、卡片及分页近距离排列 | 搜索区域、结果区、署名区形成明确间隔；卡片主标题/已发布数量分层，搜索空态有一致留白 | `features/catalogue/{current-catalogue,catalogue-search}.tsx`；T5 | topics/knowledge两种模式、scope、q、offset与清空行为保持；MSC署名和许可链接完整 |
| A03/P1 | `/topics/[id]`：子主题与知识点都用相同大卡片，层级辨识弱 | 主题名、层级/数量、子主题网格、知识列表分组；分页紧邻各自内容 | `features/catalogue/current-catalogue.tsx`；T5 | 两个分页offset/knowledgeOffset独立，project-other映射与无结果语义保持 |
| A04/P0 | `/knowledge/[id]`：桌面阅读/笔记双栏；手机页头占高，正文小，部分空分节仍显示标题 | study正文、固定分节节奏与标题比例，学习操作压成清晰工具区；空的可选分节不显示标题，原非空字段保留 | `features/reading/current-knowledge-view.tsx`、`features/study/managed-controls.tsx`、`styles/notes.module.css`；T3 | 1100断点、mode=notes、原学习动作及更新标记；匿名内容和私人内容分界不改，公式容器局部滚动 |
| A05/P1 | `/notes`：卡片标题/摘要/作者/时间与“个人笔记”标记视觉权重接近 | 标题与摘要为主，作者/知识来源/时间为次；访客登录提示、会员筛选及空态统一 | `features/notes/shared-note-list.tsx`、`styles/notes.module.css`；T5 | 匿名最新10条、会员游标/筛选、批准后可见和撤回失效保持；不新增社交元素 |
| A06/P0 | `/notes/[id]`：阅读卡片已有最大宽度，正文仍用小字号 | 采用study正文；标题、作者/批准时间、内容、关联知识、返回链接形成层级 | `features/notes/{shared-note-view,note-preview}.tsx`；T3/T5 | 可见性复核、auth-change清理、内容更新提示及不可用状态保持 |
| A07/P2 | `/login`：双栏介绍+表单与全局页头重复留白 | 收紧介绍宽度与表单节奏，增强字段/错误/主按钮层级；手机一列 | `features/auth/credentials-form.tsx`、`styles/auth.module.css`；T5 | 自动填充、原错误文本和焦点、登录/强制改密跳转保持；不新增登录方式 |
| A08/P2 | `/register`：复用凭据表单，提示与协议/错误区需一致 | 与登录同一规格，保留字段顺序与所有说明 | 同A07；T5 | 原密码/Unicode/重复账户校验，不新增注册字段或默认勾选 |

## 3. 个人学习与复习

| ID/优先级 | 页面与现状 | 本阶段改进 | 文件/任务 | 关键保留与验收 |
| --- | --- | --- | --- | --- |
| A09/P0 | `/learn`：四状态概览、主题侧栏、筛选和列表都套卡片；手机顶部占高 | 小而清楚的页标题、本地栏目和状态概览；侧栏与列表层级分明，状态文字/数值对齐，列表动作易找 | `features/study/managed-learning-view.tsx`、`styles/managed-learning.module.css`；T4 | 原completed+reviewing进度口径、mode=review旧入口、选择/分页、更新标记保持 |
| A10/P0 | `/learn?view=notes`：本人笔记列表与分享/编辑状态同屏 | 笔记标题/知识名为主，保存时间/分享状态为次，明确编辑入口和空态 | `features/notes/own-note-list.tsx`；`ManagedNotesPage`身份宿主保持；T3/T4 | 私人隔离、原游标、保存/分享/撤回与修改规则保持，列表不变公开笔记 |
| A11/P1 | `/learn?view=history`：长筛选区后是普通编号列表、ID收在details | 时间轴用细线和文字层级，日期/知识名/动作清晰；筛选和两来源历史分区 | `features/study/managed-history.tsx`；T4 | 原六事件、原时刻、details中的ID、cursor/legacyCursor独立保持；不合并事实或更换排序 |
| A12/P0 | `/learn/review`：标题大，五入口、筛选与asOf分散；空队列时垂直留白较多 | 五入口统一数值/选中样式，筛选及asOf组织紧凑，卡片主次动作与更新提示分明 | `features/study-review/queue.tsx`、`styles/study-review.module.css`；T4 | due含changed；五入口/20与50分页/主题代码字段不改；未完成优先语义保持 |
| A13/P0 | `/learn/review/[id]`：读、作答、比对、自评共享页；比对h2大、正文小，自评可能较靠下 | 标题收敛；题干、作答工具、保存/比对、冻结提示、自评有稳定视觉顺序；双栏参考栏与作答栏等比例、正常文档滚动 | `features/study-review/{session,recall-editor,comparison,self-rating}.tsx`；T3/T4 | 揭示前无参考、冻结不改答、空答仅忘记、800断点及键盘可达；不自动跳步/打分/保存 |
| A14/P1 | `/learn/review/history`：安排、评分和当时/当前时间信息较多 | 历史条目先显示知识/模式/自评，再分别显示当时安排和当前安排，不靠颜色暗示答案正确 | `features/study-review/history.tsx`；T4 | 原历史与当前安排不混同；本人作答按原权限读取，筛选/游标保持 |
| A15/P2 | `/account`：窄卡片适合阅读，但身份、改密与会话动作层级可加强 | 身份概要、改密表单、会话管理分组；注销/撤销危险动作与普通保存区分 | `features/auth/account-panel.tsx`、`styles/auth.module.css`；T5 | 强制改密、撤销身份、对话框焦点与退出后私人清理保持；不新增设置 |

## 4. 反馈与管理员

| ID/优先级 | 页面与现状 | 本阶段改进 | 文件/任务 | 关键保留与验收 |
| --- | --- | --- | --- | --- |
| A16/P2 | `/feedback`：普通列表+元数据；空态较平 | 标题/来源/状态清晰分层，更新时间不抢正文，保留新建入口 | `features/feedback/ticket-list.tsx`；T5 | 本人范围、原分页和标签保持；日期本阶段只调整展示样式，不扩大为时间语义改造 |
| A17/P2 | `/feedback/new`：长表单及来源信息集中在一张卡 | 来源说明与填写区分开，计数/限制为辅助层，错误和未确认提示靠近操作 | `features/feedback/new-form.tsx`；T5 | managed知识和site来源、不可变上下文、原最大长度及未知结果确认保持 |
| A18/P2 | `/feedback/[id]`：本轮源码审阅，讨论需要显式加载；记录信息密集 | 状态概要、讨论时间线、补充信息三段，保留加载/下一页和原详细记录 | `features/feedback/discussion-panel.tsx`；T5 | 不自动加载原私人讨论或改事件顺序；实施补有讨论/未确认/权限失效画面 |
| A19/P2 | `/review/feedback`：管理员队列筛选与列表沿用内容工作台样式 | 与本人反馈同一标题/状态规格，筛选有独立区域和选中反馈 | `features/feedback/ticket-list.tsx`；T6 | 当前管理员权限与处理筛选保持，旧reviewer角色不升级 |
| A20/P2 | `/review/feedback/[id]`：本轮源码审阅，处理原因与讨论区需要区分 | 将处理表单、结果依据、原事件时间线分组，危险/不可逆动作保留原确认 | `features/feedback/{discussion-panel,review-panel}.tsx`；T6 | 不改变状态流转、处理依据、本人/管理员边界或命令回执；实施补全真实状态图 |
| A21/P2 | `/admin/knowledge`：手机先显示较高上传卡，再筛选和横滚表格 | 上传、筛选、结果三组节奏统一；文件状态与预检结果清楚；保留单份表格DOM并给局部滚动区可读名称/焦点 | `features/knowledge-admin/{list,upload}.tsx`；T6 | 不删列、不用隐藏列冒充适配；多文件逐项预检/确认、未知结果、人工修正保护保持 |
| A22/P2 | `/admin/knowledge/[id]`：长字段组有legend，视觉仍接近连续输入流 | 基本信息、分类难度、正文解释、来源等现有组加强标题/间距；普通保存与删除分区 | `features/knowledge-admin/edit.tsx`；T6 | 各组始终挂载、保留顺序和输入；editToken、保存、删除确认与手工修正语义保持 |
| A23/P2 | `/admin/note-shares`：状态筛选、待审批卡片按钮与全文标题接近 | 状态/提交时间为辅助层，标题清楚，审核操作易辨认 | `features/notes/admin-share-queue.tsx`；T6 | pending/shared/returned/withdrawn/hidden原筛选/游标与管理员权限保持 |
| A24/P2 | `/admin/note-shares/[id]`：笔记正文、审批意见、动作和历史同页 | 正文按study阅读，审批区和历史区分组；退回/隐藏原因及未确认结果明显 | `features/notes/admin-share-view.tsx`；T3/T6 | 只处理冻结分享快照，不能变成私人笔记任意读取；sequence、理由规则及重试边界保持 |
| A25/P2 | `/admin/users`：列表+详情双栏和原生对话框 | 用户身份、角色、改密/撤销动作层级统一；手机单列、焦点和详情不丢 | `features/auth/admin-users.tsx`；T6 | 原角色解析/展示、管理员权限、原确认和焦点恢复保持；不提供新授权能力 |

## 5. 14个跳转入口逐项保留

| 原页面 | 当前目标/条件 | 本阶段处理 |
| --- | --- | --- |
| `/domains/[id]` | `/knowledge` | 保留跳转 |
| `/learning-history` | `/learn?view=history`，保留允许的过滤参数 | 保留参数投影和两个历史游标 |
| `/notifications` | `/learn` | 保留跳转，不恢复独立通知中心 |
| `/editor` | managed模式且有效管理员→`/admin/knowledge`，其余→`/account` | 保留模式、身份和角色条件 |
| `/editor/drafts/[id]` | 同上 | 保留角色条件 |
| `/editor/drafts/[id]/preview` | 同上 | 保留角色条件 |
| `/review` | 同上 | 保留角色条件 |
| `/review/[id]` | 同上 | 保留角色条件 |
| `/admin/publications` | 同上 | 保留角色条件 |
| `/admin/publications/[id]` | 同上 | 保留角色条件 |
| `/admin/withdrawals` | 同上 | 保留角色条件 |
| `/review/corrections` | `/review/feedback` | 保留跳转 |
| `/review/corrections/[id]` | `/review/feedback` | 保留跳转，不恢复档案 |
| `/review/corrections/[id]/plans/[planId]/[version]` | `/review/feedback` | 保留跳转，不恢复计划 |

## 6. 12个退场入口逐项保留

下列入口仍返回现有RetiredEntry展示。统一共用状态外观和返回链接即可，不新建旧业务页面、不移除其退出提示：

| 原页面 | 原页面 | 原页面 |
| --- | --- | --- |
| `/paths/[id]` | `/practice/[id]` | `/assessments/[id]` |
| `/assessments/[id]/result` | `/corrections/[id]` | `/editor/questions` |
| `/editor/questions/drafts/[id]` | `/review/questions` | `/review/questions/[id]` |
| `/admin/question-publications` | `/admin/question-publications/[id]` | `/admin/question-withdrawals` |

## 7. 共用状态与验收表

| 状态/边界 | 视觉要求 | 不能丢失的行为 |
| --- | --- | --- |
| loading/error/not-found | 与正文同一内容宽度、标题/说明层级；加载动画支持reduced-motion | 原role、错误边界、不伪造空结果 |
| 匿名/无权/强制改密 | 原因与下一步明确，和真正空列表区分 | 不输出私人正文，不通过隐藏CSS保留私人DOM |
| 保存中/未确认/冲突/失败 | 按钮、文字和提示有稳定位置，禁用仍可辨识 | 原请求确认方式、同幂等键和身份清理保持 |
| 内容改版/下架/笔记冻结 | 重要提醒与正文分层，用户草稿仍能看到 | 官方参考隐藏、私人答案保留及原重新开始流程 |
| 长标题/大量条目/公式 | 标题自然换行，局部公式/表格可滚动；上下文不被省略 | 不截断输入/数学源，不改变分页，不复制交互DOM |
| 中英文/放大/键盘/高对比 | 允许换行、焦点可见、DOM顺序稳定 | 中文/英文切换不丢输入；forced-colors仍使用原生选择器箭头 |

实施逐行记录T1基线、所属任务、改后截图/行为证据及最终审美结论。23个当前入口、2个工作区变体、14个跳转、12个退场必须全部闭环；反馈详情的基线缺口在T1补齐。
