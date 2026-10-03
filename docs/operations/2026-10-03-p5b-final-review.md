# P5b 一次整分支独立审查记录

2026-10-04，按已确认 Native 计划完成首轮完整矩阵后，只安排一位新上下文 `gpt-6-astra` reviewer `/root/p5b_final_branch_review`，审查 `dad438d13b37d1053e3bf42e7c32e3829e4518a6..daaafe791ef49295e6b7fcc6beaf908945588409` 的实际整分支 diff、方案、计划、14项裁定与证据。未派实现代理，reviewer未改工作树、index或HEAD。修复由原实现者进行一次，不派第二 reviewer。

## 独立判断与证据

reviewer多遍检查五项 Review Focus：同步资格失效与批准恢复、累计来源与事实守卫、晚提交/租约/补扫、双侧曝光及账户隔离、旧契约/原事实/CI。确认共享锁、登记fence、提交前身份与租约、同事务结果/依赖/通知/断点，原封存答案、五题与4/5门槛保留；旧七迁移和全契约哈希继续保护。31项契约/CI保护和14个顶层Go焦点测试通过（19.59秒），包括两种撤回提交顺序、其他案件限制、旧二进制补扫、模板曝光、滑动窗口和跨账户游标。

| Finding | 原分级／按实际影响复核 | 复现与本轮处理 |
| --- | --- | --- |
| 已批准A→B→C，但worker尚未生成中间结果，数据库只接受直接映射或父结果 | Important／Important | 真实独立批准链复现 `replacement authorized=false`，处理503。正式 `TestCorrectionUnprocessedApprovedChain` RED；有界准确批准链守卫后GREEN，并拒绝缺边、未批准、101引用、冲突和环；原结果/作答不变。 |
| 有效纠错配图仍调用撤回原attempt资源接口，404 | Important／Important | 真实详情effective且assets=1，旧读取404。正式Store、HTTP、UI、proxy RED；新增本人准确有效结果绑定只读SVG，旧attempt保护不变。跨账户、任意SHA、替代再次撤回、答案重叠/30分钟曝光、SVG安全/1MiB/10秒全部回归。 |
| editor误见仅admin登记/重试入口，403使账户边界卸载、草稿丢失 | Minor／Important | 按实际编辑工作中断升级；两项正式页面RED后，登记和重试按admin显示、编辑者保留创建/编辑方案，GREEN。 |

当前修复定向Go GREEN2为9.17秒，UI/proxy13项GREEN为0.83秒，旧契约与CI31项GREEN为0.18秒，类型检查GREEN为2.43秒。一个新Go测试的初次GREEN尝试失败于错误的审计数量预期：两个不同批准source分别保留一份结果审计，重复drain不增；已按既定事实语义校正。首次新增浏览器场景实际PASS，不作为RED证据；最终矩阵必须重新构建再验收。

[一次修复方案及兼容性可行性审查](2026-10-03-p5b-review-fixes.md)列出新增/修改文件和架构。新增一个私有只读配图路径/操作后，纠错与通知共17路径/20操作，旧88路径/220schema/32响应/3安全模式保持保护。第八迁移未曾发布，因此修复其定义不涉及已部署数据库变更。

## 裁定和未判断范围

reviewer逐项认可原14项裁定（内部决策Action、DownTo6、登记fence、旧账户停用模型、草稿参与者独立审核、自动案件夹具、terminal批准依据、版本created_at分页、428/403对齐、从未启用库、schema健康、固定kind与批量依赖、真实事实容量、旧容量UpTo7）。批准链缺口必须另修，不能以裁定掩盖。

reviewer Declined to judge四项均逐条裁定：正式数学资料仍待内容门槛；生产迁移/部署/维护隔离未连接现场；容量结果不外推任意正文、峰值RSS或生产SLO；产品draft PR和最新完整head四项workflow属于本轮审查后的交付义务。判断、理由和错判代价均保存至逐任务账本，并将在交付说明完整列出。

Deferred minors：无。唯一原Minor已升级为Important并处理，没有将其作为润色另开修复轮。

## 最终门槛

独立原始结论为 `Ready to merge? With fixes`。三项修复已有真实失败与定向GREEN，但修复后完整本机矩阵及SSH draft PR最新完整head四项workflow/all jobs仍待完成，不能把首轮旧SHA的成功替代最终验证，也不宣称部署或自行合并。

同一次链修复补充 `TestCorrectionReplacementChainIgnoresUnrelatedFork`：真实其他实例的批准分支冲突造成3.19秒RED，链守卫仅沿当前原实例的确定性边走到终点后，和当前链冲突/环及无中间结果回归一起8.17秒GREEN。其他题目无关分支不阻塞本证据；当前链分叉、环、缺边、超100引用/边仍拒绝。未完成的旧SHA矩阵已由其自身包装器安全中止，不用于最终验收；全部组将在新提交重跑。
