# 旧纠错事务入口性能优化补充方案

## 目标与授权

笔记阶段 MR #57 的最终提交 `9db0520256043118258784894ee6e9fae997d250` 有15项CI通过，剩余原纠错容量用例两次在通知分页阶段达到5分钟预算。用户已授权追加旧纠错性能优化：先补方案与兼容审查，通过后开发，沿用Native。原数据规模、断言、超时和笔记阶段行为继续保留。

本补充只优化既有事务入口的数据库往返，不恢复旧业务页面，不修改评分、资格、审批、幂等、schema守卫或原SQL函数。原1—13迁移与未发布14保持原字节，不新增迁移或产品依赖。

## 已观察证据

- 原完整1000案件、10000证据、200实际处理批次与完整通知分页：当前本地153.32秒、准确master基线148.63秒；同HEAD push CI曾298.07秒通过，PR两次在300秒终止。
- 测试准备的流水线传输和批量准备临时试验分别156.52/153.37秒，无明显收益，均未进入产品分支。
- 当前真实用例的Go CPU剖析：总墙钟153.36秒、CPU样本21.33秒，系统调用和调度等待为主，JSON编码未表现为主要热点。
- `managedCorrectionGuard(contentLocks=true)` 每次执行 `SET LOCAL lock_timeout`、admin共享锁、content共享锁、knowledge共享锁四个独立往返。容量worker保留每证据独立事务，入口约调用一万次，因此可减少约三万次往返。
- SQL统计仅在专属合成测试容器启用，不作为产品配置或依赖。该模块配置参考[PostgreSQL17文档](https://www.postgresql.org/docs/17/pgstatstatements.html)。

## 选择与架构

推荐只合并上述锁入口。它保留全部原操作和检查，范围最小。进一步批量化账户、案件或结果读取需要更复杂的快照、错误与并发审查，首轮不采用；跨事务缓存状态或改旧SQL函数会扩大兼容边界，排除。

```mermaid
flowchart LR
  A[原每证据独立事务] --> B[一条SQL内设置1秒锁超时]
  B --> C[admin共享锁]
  C --> D[content共享锁]
  D --> E[knowledge共享锁]
  E --> F[下一条语句读取当前业务模式]
  F --> G[原schema核验与证据处理]
  G --> H[原独立提交或回滚]
```

SQL使用明确的MATERIALIZED生产者依赖，每层从前一层读取，最终结果消费knowledge层，避免依靠CTE书写次序。`set_config('lock_timeout','1s',true)`等价于原事务内的SET LOCAL；原项目`correctionSystemEnter`已经使用同样的设置及锁依赖模式。

模式读取必须保持在锁语句完成后的下一条SQL，以保留READ COMMITTED等待后重新取快照的行为。不能把模式读取加入同一CTE或缓存其结果。`contentLocks=false`仍仅取knowledge锁，不新增admin/content锁，也不提前设置锁超时。

## 文件范围

| 文件 | 职责 |
| --- | --- |
| `backend/internal/store/managed_business_mode.go` | 只修改managedCorrectionGuard完整锁入口；原managedBusinessMode及不取content锁分支保留 |
| `backend/internal/store/managed_correction_fence_internal_test.go` | 测试专用入口、真实驱动的Exec计数包装及锁ID；不编入产品二进制 |
| `backend/internal/store/managed_correction_fence_test.go` | 一次往返、锁顺序、等待后模式读取、取消/超时回滚、不取content锁分支的真实数据库验证 |
| `api/topic-learning-compatibility-baseline.json`与对应政策固定摘要 | 只追加受审文件的准确目标及新增文件；保留原基线、快照、规模和原CI命令 |
| 补充方案、计划、审查及验收记录 | 记录剖析、兼容审查、实际收益、完整门禁与MR最新HEAD |

不改原容量测试及历史造数函数，不改前端或原笔记/分享实现，不调整服务超时、worker批次或内存边界。

## 兼容与验收

1. 完整锁顺序仍为设置超时→admin→content→knowledge；共享锁类型、ID和事务生命周期不变。
2. knowledge锁等待期间发生managed激活，解锁后的下一条模式读取仍拒绝旧纠错操作；部分schema继续拒绝，未启用legacy继续按原行为判断。
3. 不取content锁的入口不等待admin/content排他锁。每次事务独立检查模式和完整schema，不缓存数据库结论。
4. 取消、锁超时及查询错误原样传播，整个事务回滚并释放此前取得的锁。
5. 原完整容量用例、所有纠错/通知语义、原managed与分享边界、精确兼容门禁及最新HEAD的CI必须通过；规模1000/10000、200批、每批50、Go5分钟和命令540秒均不变。
6. 同环境记录整用例与worker阶段收益，初轮期望至少约10%的整用例改善。若收益不足，不放宽门禁、不取消旧检查；先更新实际热点及补充设计，再扩大改动。

## 开发前审查

执行者审查确认：已有数据库和驱动支持该SQL，项目已有同模式生产者依赖；未新增接口、schema或依赖；模式读取的语句边界、原错误优先级和不取content锁分支均明确。剩余的实际执行顺序、阻塞后快照和回滚风险由列明的真实数据库用例验证，无设计阻塞。用户已明确授权审查通过后继续开发，不重复询问执行方式。
