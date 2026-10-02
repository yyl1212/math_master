# P5a 一次整分支独立审查与修复记录

审查范围：`20a68fa6f087e5e78561bf07c0229e072642e257..4a5ced873a5b76601bac373057b51a7ce80cef0c`。一位 fresh-context reviewer 只读审查，不修改工作树、不追加代理；独立兼容/CI保护11项通过，以实际React组件和替代网络/身份返回复现五项 Important。抽查已保存完整矩阵及原创脱敏截图。审查时结论为 **With fixes，当前HEAD不可合并**，没有Critical。

| 分级 | 已复现问题 | 原实现者处理 |
| --- | --- | --- |
| Important/P1 | 无auth-change通知的SSR A→B复用Provider及表单，A私有草稿以actor B发送 | 实际组件RED→GREEN；共同Provider按actor原子重建，核验前旧草稿消失，新账户只能发送新输入 |
| Important/P2 | POST成功后最新元数据GET失败/截止，pending提前丢失，新POST使用新key重复创建 | GET错误/十秒截止两项钩子RED→GREEN；保留原key及已确认状态，另有晚到导航实际RED→GREEN |
| Important/P2 | 一次身份服务错误永久保留error，同actor SSR refresh不重验，Reload无效 | 显式Reload、focus及同actor SSR三项RED→GREEN；同账户SSR保留草稿 |
| Important/P2 | FEEDBACK_TARGET_STALE后不能刷新同来源并保留文本，两种动作均继续旧来源 | 4000字草稿/新版本/原冻结输入及key回归RED→GREEN；真实发布v2两视口联调通过 |
| Important/P2 | Reload report status无条件清owner输入；sequence key重建处理面板清回复和依据 | owner刷新及handler冲突/依据两项RED→GREEN；保留原键重试，明确新请求后使用新序号 |
| Minor/P3，暂缓 | 检测重合提示只有Reload，缺少返回检测专用入口 | 原导航仍可到达，不进入本次重要问题修复 |
| Minor/P3，实施者视觉核对补充，暂缓 | 临时身份故障恢复后，共用顶部账户栏可能暂显Accounts unavailable | 原AuthStatus只在挂载/身份通知重读；反馈表单已用新鲜身份恢复，统一顶部状态同步另行处理 |

精确位置：`frontend/src/features/feedback/feedback-account.tsx:30`、`pending-command.ts:36`、`new-form.tsx:13—14`、`status.tsx:11—14`、`discussion-panel.tsx:22—27`；以上行号对应审查HEAD。五个SSR页共同使用账户Provider。

五项重点中，后端旧序号原回执、准确原模板检测重合、自由文本/duplicate隐私、长讨论及同微秒分页/权限WHERE/滑动窗口均通过。账户切换及前端恢复存在上述重要问题，因此修复前不通过账户重点。数据库提交前身份复核及原十秒预算有效，后端来源、事务、不可变历史、曝光和配额无需改变。

审查接受既有三项实施判断：按合法FK状态验证四表非空Down、version=1游标只描述边界且权限WHERE收紧、隔离harness账户重置同时清四反馈表。

## 审查保留判断与实施者裁定

1. P5b重算、影响任务和通知保留后续阶段，本轮确认不改变原资格。若裁定错误，需要另外实现，不能宣称P5a已提供。
2. revision_published仅连接现有独立发布事实，不承诺完整供题或资格恢复。若裁定错误，用户恢复资格预期需要另审契约。
3. 进入表单前的历史公开快照按批准ContextQuery解析当前ID，历史定位协议未批准；加载后的过期来源纳入本轮修复。若裁定错误，历史页面反馈会指向当前版本，需要新协议及兼容审查。
4. 正式数学质量、生产部署和灾备保留P6/P7验收。若裁定错误，技术夹具可能被误当作可上线证明。
5. 未创建MR及其精确HEAD四项CI列最终交付门槛，不判为产品缺实现。若裁定错误，未被远端验证版本可能被交付，因此交付前必须逐项核实。

## 修复和最终验证

原实现者完成一次五项重要问题RED→GREEN修复，不安排第二轮审查；十个新增回归分别复现并覆盖问题，42项反馈UI/钩子及全量283项前端通过。具体测试在`review-regressions.test.tsx`及`pending-command.test.ts`。完整Go/容量与17批130项浏览器复验全部通过，零skip/retry；24项Node/兼容及全部构建检查通过。受验重要修复提交`175c61f41df6c1c1f974c0ce3e9a023c54361951`，逐项命令及日志摘要见验收证据。后端及浏览器各用独立随机测试库，未修改来源资料或生产数据。MR、合并和生产部署各有独立门槛，本次授权仅实现、验证、SSH推送和创建draft MR。

## MR交付

[draft MR #21](https://github.com/yyl1212/math_master/pull/21)已SSH推送并附到当前任务。首轮完整交付head `6c550d9aa68bc4a5bc2a15cd35b8168cc5ab2d61` 的Go/前端push/PR四项CI全部SUCCESS，精确SHA、事件和run链接见[首轮CI证据](evidence/p5a/ci-initial-delivery.json)。本次后续文档提交只保存验收与完成记录；最终以MR最新完整head四项CI另行核对，在MR正文记录最终SHA及run链接。未以旧提交通过代替最新提交检查。
