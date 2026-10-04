# P6b 设计文档 CI 查询超时诊断

日期：2026-10-04。诊断 head 为 bea6f17047ecc968a1801f23eeba48a70051af37，代码基线 master b78108d3dd363eae237dae59f76a7be774b9de85。该分支只改文档，产品、依赖、工作流及容量测试与 master 一致。完整机器记录见[元数据](ci-diagnosis.json)。

CI 当前可以启动。master 的 [Go push](https://github.com/yyl1212/math_master/actions/runs/37190264295)和[前端 push](https://github.com/yyl1212/math_master/actions/runs/37190264336)全部成功。设计 head 的四 workflow/六 job 为五成功、一失败；失败为 [PR Go verify](https://github.com/yyl1212/math_master/actions/runs/37190823622/job/111402432022)，不是账号启动限制。

失败步骤为“最大合法学习题源与证据”，原命令为：

```sh
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestLearningCapacitySourceVolume$' -timeout 5m -count=1 -v
```

[原失败日志](ci-failed.log)显示 09:08:44.453 UTC 开始，09:09:43.567 已创建最大数据，09:12:44.889 在 1000 条旧/新准确审批证明对照处 context deadline exceeded，总测试240.71秒。源码 newLearningCapacityFixture 使用四分钟准备 context，该对照完成后才换到五分钟测试截止并执行八秒运行操作。报错中的 0/false/false 是 QueryRow 扫描遇到超时后的默认值，不能据此断言审批算法对照不一致。

相同 head 的 [Go push](https://github.com/yyl1212/math_master/actions/runs/37190820415)通过，对照查询从日志时间相减约12.74秒，1000/allEqual/allApproved 为真；master 同对照约9.52秒且通过。一次本地完全相同测试通过，91.81秒，同样完成1000条对照；[公开节选](local-capacity-excerpt.log)保留数量/结论，完整原日志在受保护本地保存并登记 SHA。这次技术夹具正式计数为零。

已定位失败到准备阶段对照查询，但具体为何该 runner 查询显著变慢尚未证实，失败查询的 EXPLAIN/资源诊断不足。本地成功只表示未复现，不代表已修复；未修改代码、容量、四分钟准备、五分钟测试、八秒运行验证或工作流，也未请求 Actions 重跑。

实施交付必须核验准确新 head 的四 workflow/六 job。再次出现同阶段失败时，保留当前错误与时间、同数据/上下文复现、获取旧函数与优化函数的计划/统计/资源证据，再审查最小根因修复；保持1000条完整对照及全部原运行检查。不得删除对照、减少容量、增加截止或靠反复重跑获得一次绿色来宣布修复。
