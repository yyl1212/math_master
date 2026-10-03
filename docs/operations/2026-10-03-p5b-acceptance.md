# P5b 本机验收记录

2026-10-04，用户已确认16项/80步 Native 执行。产品基线为文档 [PR #22](https://github.com/yyl1212/math_master/pull/22) 合并后的 master `dad438d13b37d1053e3bf42e7c32e3829e4518a6`，隔离分支 `codex/p5b-correction-workflow`。完整本机受验产品/CI提交为 `d6f8c743a9385aeacf97644b2f5f6d6f1b729f60`；后续证据文档提交没有产品行为变化。一次独立整分支审查及SSH draft产品PR的最新四项CI仍为交付门槛，本文不预记审查或远端成功。

## 已实现行为

真实撤回/判分案件即时限制，准确已发布实例方案独立批准，原答案累计重判，六种处置、当前资格统一投影、持久租约/断点/有限补扫和站内通知已落地。英文私有页面提供本人纠错回顾、独立处理和通知，既有结果/知识/历史可进入纠错记录。关闭worker仍限制；永久启用标记和健康核验使损坏数据库拒绝个人学习。Original result、原回执字节和历史解锁保留，不能用新结果覆盖旧分数。

## 完整本机矩阵

| 检查 | 实际结果 |
| --- | --- |
| 44条独立包装器命令 | 全部退出码0；精确SHA、参数、时间、日志SHA见结构化记录 |
| 前端单元 | 70文件345项通过，保留原283项 |
| Node/入口/兼容/资料快照/CI保护 | 44项通过，保留原24项；旧88路径/220schema/32响应/3安全方案保护 |
| 桌面与手机浏览器 | 20批144项通过，保留原17批130项，追加三批14项，无跳过或重试 |
| 核心/HTTP/server退出/harness/testutil | 102.65s通过；原11包、两个新包与cmd/server均运行 |
| 原store与CLI | 141.39s通过，原Question等测试未额外skip |
| 学习/检测非容量 | 82.73s通过 |
| 反馈非容量 | 28.45s通过 |
| 纠错/通知/新CLI非容量 | 112.21s通过 |
| 原反馈容量 | 11.21s通过，原数量/锁/断言保留 |
| 原最大题源、原最大路线 | 92.15s、121.73s通过；原测试文件字节未改 |
| 新1000案件/1000批准方案/10000证据容量 | 254.19s通过；200批次、全部expected sets相等 |
| 新10000通知与10000同源重跑 | 43.04s通过；去重、owner分页、首次已读与额度保持 |
| 锁定安装、API生成、类型、生产构建、harness | 全部通过；generated无漂移，无新增依赖 |
| gofmt、vet、Go构建与diff | 全部通过 |
| 生产依赖审计 | 0漏洞 |

[完整命令和日志摘要](evidence/p5b/verification-macos.json)、[证据摘要清单](evidence/p5b/manifest.json)、[逐任务提交与裁定](evidence/p5b/implementation-ledger.md)均可复核。Go每批5m、浏览器8m/一个worker/零retry、包装器9m、每CI job30m保持不变。CI原verify与三项旧容量保留，新增独立correction_verify job，只运行新非容量/两项容量；前端保留原17组完整顺序再追加3组。

## 容量、旧事实和真实失败

历史容量复用真实独立作者、复核及发布事实，全部FK、不可变/独立审核及延迟守卫开启；并非1000次绕过配额的API创建。10000证据由生产方法连续处理，每批50条且断点单调前进，原seal/答案摘要保持不变；无准确映射的撤回诚实返回retake_required，不凭空授予资格。所有案件、方案、结果和两位owner的通知分页直到全部，expected sets完全相等。

最新影响容量：保留堆增量335768bytes、单批最高分配23577080bytes，均低于该一题技术夹具的32MiB断言；总分配量为4686570664bytes。这不是峰值RSS或所有真实数学正文大小的承诺，单正文和完整响应仍按各自字节上限处理。10000通知生产append后再同源append10000次，最终仍10000条；首次已读时间、同键回执及跨owner拒绝均真实验证。

RED→GREEN包括缺check/FK/unique/function、触发器错表/错函数/禁用、约束同名变弱、FK未验证、资格唯一索引缺失，以及终结outbox的OR优先级扩大范围。真实容量首跑在补扫8秒事务失败，按固定case kind减少无关查询计划、每结果合并依赖INSERT后通过，全部行守卫与末行故障回滚保留。通知容量首跑本来就PASS，虽历史日志名为red，未制造假失败。证据目录保留这些原日志。

旧反馈missing migration场景真实从UpTo(7)建立从未启用库，再Down到6；不清永久标记，原学习成功、反馈未配置、重新Up不改变旧事实。曾启用后空Down仍拒绝个人学习，非空十表Down整体拒绝。真实原failed成绩与提交回执不变，不能冒用旧failed结果插入旧资格事件。

## 脱敏视觉与边界

18张截图来自自有页面和原创技术夹具，其中新增7场景的桌面/手机共14张，原检测/练习各两视口共4张。input/textarea粉红区为Playwright脱敏遮罩，已目视核对手机独立审批、已读通知、重测和旧结果/练习；长版本/摘要及导航未出现横向溢出，实际浏览器也验证宽度。

- [手机独立批准](evidence/p5b/mobile-correction-independent-review.png)、[桌面独立批准](evidence/p5b/desktop-correction-independent-review.png)
- [手机通知已读](evidence/p5b/mobile-correction-notifications.png)、[桌面通知已读](evidence/p5b/desktop-correction-notifications.png)
- [手机诚实重测](evidence/p5b/mobile-correction-retake.png)、[手机owner隔离](evidence/p5b/mobile-notification-owner-isolation.png)
- [手机原3/5失败结果](evidence/p5b/mobile-original-assessment-result.png)、[手机原练习](evidence/p5b/mobile-original-practice-revealed.png)

未改Knowledge_JSON，未批准或发布正式数学内容，未访问生产服务器、运行生产迁移或部署。技术夹具不计P6内容数量。恢复边界见[操作说明](correction-workflow.md)，审查结论将在[一次整分支审查记录](2026-10-03-p5b-final-review.md)保存；产品合并与生产部署需另行授权。
