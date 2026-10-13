# 旧纠错事务入口性能优化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: 使用superpowers:executing-plans按任务Native执行；先观察真实失败，再最小修改并验证。用户已授权追加性能优化并在方案/兼容审查通过后开发，不重复执行方式确认。

**Goal:** 减少原纠错完整锁入口的三次多余数据库往返，保留全部事务语义并使原容量与MR最新HEAD门禁通过。

**Architecture:** 使用一条具有MATERIALIZED生产者依赖的SQL按原顺序取得完整锁。模式读取、schema核验、每证据处理、独立提交/回滚仍沿原路径。

**Tech Stack:** 当前锁定Go1.27.1、CGO_ENABLED=0、pgx/v5、PostgreSQL17；不升级或新增依赖。

**Spec:** [补充方案](../specs/2026-10-13-legacy-correction-fence-design.md)。实现起点为笔记分支9db0520，目标仍为MR #57/master。

## Global Constraints

- 原1—13迁移、14迁移、schema守卫、API/DTO、评分、资格、审批和幂等规则保持。
- 完整锁顺序设置1秒超时→admin→content→knowledge；模式读取在后续单独语句完成。
- contentLocks=false只取knowledge锁，不新增admin/content锁或超时设置。
- 不缓存数据库模式/schema结论；不合并不同证据事务；worker最大50、原事务8秒和批次30秒保持。
- 原1000案件/10000证据/200批/全量分页、Go5分钟、命令540秒、浏览器480秒/单worker/零重试及全部旧检查保持。
- 先审查兼容例外再实现，精确登记目标字节，不整体重置基线。只更新现有MR，不合并或生产部署。

## Review Focus

- owner/角色及永久managed激活在锁等待期间变化，后续读取不能沿用旧快照（Task1）。
- knowledge锁超时或取消后，已取得的admin/content锁必须随回滚释放（Task1）。
- 锁账本入口contentLocks=false不能新增content锁或等待admin锁（Task1）。
- 数据库部分安装/永久启用标记不一致，模式读取仍拒绝而不能走缓存或回退（Task2原回归）。
- 性能改善不能来自删除数据、减少200批/通知页、改变历史数学字节或修改超时（Task2）。

## Task 1: 单次往返的有序锁入口

**Files:** 修改backend/internal/store/managed_business_mode.go；新增managed_correction_fence_internal_test.go、managed_correction_fence_test.go；追加精确兼容目标。

**Interfaces:** 保留managedCorrectionGuard(context.Context,*sql.Tx,bool) error、managedBusinessMode及所有原调用者。测试入口ManagedCorrectionGuardForTest(context.Context,*Store,bool) error、ManagedCorrectionLockIDsForTest() []int64仅编入测试二进制。

- [ ] 写真实驱动Exec计数用例TestManagedCorrectionFenceTransportBound：完整入口只有1次Exec往返；原四次路径应失败。包装必须转发原pgx上下文、参数类型和事务方法，不能伪造模式返回。
- [ ] 写TestManagedCorrectionFenceOrderedLocks：另一连接占有content排他锁时，等待者已取admin而未取knowledge；占knowledge时已取admin/content。先释放阻塞锁，再断言成功。
- [ ] 写TestManagedCorrectionFenceFreshModeAfterWait：占有knowledge排他锁，等待者阻塞后通过同一控制事务完成永久managed激活，提交后等待者返回correction.ErrRetired。
- [ ] 写TestManagedCorrectionFenceBookkeepingContract：占admin/content排他锁，不取content入口仍成功；与原设置一致。
- [ ] 写TestManagedCorrectionFenceCancellationRollback：取消或1秒锁超时，原错误保留；回滚后控制连接能立即取得此前的admin/content排他锁。
- [ ] 运行定向用例，观察原路径四次Exec的RED；其余原语义用例作为兼容基准。
- [ ] 仅替换完整锁入口为明确依赖的单条SQL；模式读取和false分支不改。重跑上述真实用例及原managed激活/纠错锁/旧租约竞争回归。
- [ ] 登记仅此文件准确兼容例外及测试白名单，运行精确兼容工具；提交。

Run: `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestManagedCorrectionFence|^TestManagedActivation|^TestCorrectionStaleLeaseAtFinalCommitFence$' -timeout 5m -count=1`

Expected: 新往返断言在原实现为4≠1而失败；修改后全部PASS，实际锁等待/激活/回滚断言不删。

## Task 2: 完整语义、容量、兼容及MR交付

**Files:** 精确兼容政策、中文补充审查/验收文档；现有MR说明。产品功能及原CI/容量测试不改。

**Interfaces:** 消费Task1原函数及全部原验证入口；产出同环境原完整容量结果、最终代码/兼容审查和MR最新HEAD CI。

- [ ] 按原5分钟命令分别运行纠错/通知非容量、Impact容量、Notifications容量；所有原断言、规模及原始封印比对保留。记录整用例/worker/metadata/通知阶段实际耗时。
- [ ] 若整用例改善不足约10%，先写实际热点和补充设计审查；禁止用扩大预算、删数据、关闭约束、跨事务缓存或空重跑替代优化。
- [ ] 运行原managed/分享相关真实SQL与HTTP边界、Go vet及87项精确兼容/原CI工具。验证原13+14迁移字节和原API语义不变。
- [ ] 对本补充9db0520..HEAD范围进行一次Native独立代码/兼容审查；复核阻塞项，一轮TDD修复并通过受影响完整套件。原笔记整分支独立审查证据保留，不重复审查其已验证未改代码。
- [ ] 更新现有MR #57的最终范围与实际验证，正常Git推送；核对每项远端检查对应最新HEAD，失败先定位，不盲目重跑。
- [ ] 最终性能和CI门禁通过后关闭本补充任务，再完成笔记计划Task7；保留工作树，清理本任务专属测试容器及仅对应的临时目录。

Run（各独立原预算）:

    node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -run '^Test(Correction|Notification)' -skip '^TestCorrectionCapacity' -timeout 5m -count=1
    node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestCorrectionCapacityImpact$' -timeout 5m -count=1 -v
    node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestCorrectionCapacityNotifications$' -timeout 5m -count=1 -v
    node tools/verify/run.mjs -- env TOPIC_COMPATIBILITY_STAGE=managed node --test tools/verify/*.test.mjs

Expected: 全部PASS；容量完整集合、200批及封印不变，实际收益有记录，MR最新HEAD CI全部通过；未合并或部署。
