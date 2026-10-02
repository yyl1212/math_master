# P5a 版本化反馈与处理验收记录

2026-10-03，用户确认十二项 Native 执行计划后，从最新 master `20a68fa6f087e5e78561bf07c0229e072642e257` 建立隔离分支 `codex/p5a-feedback-workflow`。旧契约基线单独取批准前 master `8400ee56d59fd13ecf23f83d89d7685027c93c2d`。本机完整受验产品/CI提交：`10241b14a4c42cd58b49355e919844e898881672`。实现 MR 的最终完整 head 与四项 GitHub CI 在交付时另行核对，以 MR 当前 head 为准。

## 交付行为

用户可从知识及准确讲解/素材、路线、本人单题练习、检测位置1—5、网站区域提交反馈。报告固定原版本、来源及标题/位置；通过只追加事件追踪五状态、独立处理依据、补充和终态异议重开。列表及全部写回执没有自由文本，讨论只有主动读取后交付；准确原实例/模板与当前未到期检测重合时拒绝，允许交付前在同一事务记账曝光。

处理者只能核验现有独立审核、发布与永久撤回事实，不能处理自己的报告。新生成实例修订使用不同ID，duplicate编号对owner隐藏。原答案、分数、资格、历史解锁不因反馈状态或链接变化。请求失败和原回执重放不扣成功额度，旧序号重放成功回执也不倒退当前状态。跨标签页换账户清除旧草稿，十秒包含新身份等待，只有用户主动以原键重试冻结输入。

架构、五状态依据、额度、曝光、迁移恢复和二进制回退见[操作说明](feedback-workflow.md)。新增四张反馈表在00007，旧00001—00006完整字节及旧API对象逐值保持。

## 完整回归

| 检查 | 实际结果 |
| --- | --- |
| 锁定安装、API生成无漂移、类型、前端生产构建 | 全部通过；没有新依赖 |
| 前端单元 | 53文件273项通过，原135项保留 |
| Node/兼容/资料快照/CI保护 | 24项通过；含原13项、旧75路径/192schema/32响应/3安全方案逐值及改字段/迁移负测 |
| Go纯模块、HTTP、联调harness及testutil | 11包通过，HTTP41.958s、联调101.159s |
| 原store/CLI | 141.793s整批通过 |
| 原学习与检测非容量store | 78.547s整批通过 |
| 反馈非容量store | 27.863s整批通过 |
| 反馈容量 | 10.798s整批通过，含迁移缺失、恢复和非空学习事实不变 |
| 原最大题源、原最大路线 | 90.387s、118.905s通过，原全部计数、共享双槽及排他锁保持 |
| 浏览器 | 124项，17批，全部通过；原100项及新增24项，桌面1280×900/手机390×844，无跳过或重试 |
| gofmt、vet、Go构建、harness、diff | 全部通过 |
| 生产依赖审计 | 0漏洞；首次TLS连接建立前中断，保留代理并显式传给npm后通过 |

每条命令、数量、实际耗时及日志SHA见[结构化验证记录](evidence/p5a/verification-macos.json)。Go各批原总截止五分钟；初始化最多四分钟，后续用原Go总截止的剩余时间；产品动作仍八秒，锁等待一秒。Playwright各批480秒、一个worker、零retry；验证包装器540秒。两个CI job仍30分钟，原14批浏览器和原两项容量没有删减或合并。

## 容量、原事实和失败证据

历史查询夹具为1000工单/10000事件，使用真实批准来源和数据库微秒，全部SQL约束启用；不能声称进行了1000次突破额度的HTTP创建。owner各500工单、review1000工单完整分页，每页50条；100条同微秒讨论连续无重无漏，最大页607731bytes，低于2097152bytes。初始化1.601208333s，最慢操作164.670166ms，两个并发处理恰一成功；线程超过50条仍能补充、解决、重新开启并在sequence105关闭。

工单到sequence104后原sequence101命令重放返回原成功回执，当前状态、事件数和成功消耗不变。私有讨论实际记录读者曝光；元数据没有答案哨兵。真实非空的学习事件、原作答、原结果、资格事件和解锁逐字节保持。缺00007时旧学习动作真实成功，反馈明确503；重新完整迁移后旧学习事实不变。空Down及各类合法非空Down守卫通过。

容量首跑已满足要求，如实记首跑PASS，未制造假RED；实际分页SQL与原参数的EXPLAIN/ANALYZE记录见[容量证据](evidence/p5a/capacity-macos.json)。实现过程的确切失败及修复见[RED→GREEN摘要](evidence/p5a/red-green-summary.md)和[实施账本](evidence/p5a/implementation-ledger.md)。队列的新SSR结果、筛选变化及浏览器返回另有两项确切组件RED→GREEN及两视口真实回归。

## 独立审查与范围

全部功能和本机矩阵完成后使用一次 fresh-context 整分支独立审查，重点覆盖旧回执、原模板重合、自由文本和duplicate隐私、账户切换/身份等待、同微秒游标及滑动窗口。本次审查与最终MR交付门槛正在执行；结论、严重程度判断、必要修复和暂缓小项在[整分支审查记录](2026-10-03-p5a-final-review.md)保存。任何重要修复均由原实现者先做失败回归、再修复和复验，不额外派第二轮审查。

本次只使用本机loopback随机测试数据库；没有修改持续更新的Knowledge_JSON，没有访问生产服务器、执行生产迁移或部署，也没有把技术夹具计入正式数学课程批准数量。P5b独立重算/影响任务/站内通知、P6正式内容质量、P7上线和灾备仍待单独方案与验收。

## 脱敏视觉证据

截图来自自有页面和原创技术夹具，input/textarea的粉红区域为Playwright脱敏遮罩。已目视核对长中文/英文、准确版本/长生成ID、处理依据、序号冲突、owner和handler检测重合阻断、分页以及换账户后的清除；横向宽度由两个视口真实断言验证。

- [手机长文本](evidence/p5a/mobile-feedback-long-text.png)、[桌面长文本](evidence/p5a/desktop-feedback-long-text.png)
- [手机真实修订](evidence/p5a/mobile-feedback-published-revision.png)、[桌面真实修订](evidence/p5a/desktop-feedback-published-revision.png)
- [手机序号冲突](evidence/p5a/mobile-feedback-sequence-conflict.png)、[桌面筛选队列](evidence/p5a/desktop-feedback-filtered-queue.png)
- [手机owner检测阻断](evidence/p5a/mobile-feedback-owner-overlap.png)、[桌面handler检测阻断](evidence/p5a/desktop-feedback-handler-overlap.png)
