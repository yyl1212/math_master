# 共享夹具初始化修复实施计划

> 使用superpowers:executing-plans按用户授权的Native方式执行；先完成方案/兼容审查，再TDD开发。最终修复MR与生产部署继续同一任务。

**Goal:** 将原共享fixture的存储清理合并为单次Exec及单一提交，保持完整流程和预算，并使新MR门禁通过后继续生产部署。

**Spec:** [方案](../specs/2026-10-13-fixture-initialization-design.md)。

**Architecture:** 原控制器/守卫→单一存储复位事务→原基础账号及四账户真实流程→原场景。生产程序与业务模块不改。

## Global Constraints

- 原表列表、RESTRICT及原目录CASCADE范围不变；不关闭fsync/结构约束、不缓存密码/schema/会话、不复制批准结果。
- 账户、角色、认证参数、全原用例和原预算保持；Go使用CGO_ENABLED=0。
- 原1—14迁移、API、旧快照及历史基线不改；仅精确追加本补充目标。
- 生产保持旧版本，完整门禁通过前不执行start/activate。只用合成随机库验证实现。

## Review Focus

- 目录清理前移是否删除新认证记录或改变账户角色/会话语义（Task1/2）。
- 原RESTRICT是否被意外扩大为CASCADE；外键错误时永久标记与旧数据是否完整回滚（Task1）。
- 一个Exec中的语句执行顺序、真实pgx上下文/参数及取消回滚是否保持（Task1）。
- 是否把本机未复现的CI长尾当作确定原因，或以放宽预算/少测代替远端门禁（Task2）。
- 修改是否仅落在测试包且不能进入生产依赖或恢复旧生产接口（Task2）。

## Task 1: 单事务存储复位与真实回归

**Files:** auth_fixture.go、workflow_fixture.go、新fixture_storage_reset_test.go、精确兼容目标及本方案审查。

**Interfaces:** resetAccounts/resetWorkflow/workflowAccounts保持原签名；私有resetFixtureStorage(context.Context,*sql.DB,bool) error；私有initializeFixtureAccounts(context.Context,*auth.Service,*auth.AdminService) error。

- [ ] 先实现真实pgx驱动计数包装，完整转发上下文、参数和事务，不伪造数据库返回。
- [ ] TestFixtureStorageResetTransport通过原resetWorkflow观察存储Exec=4而目标1，真实RED。
- [ ] TestFixtureStorageResetRollback通过真实外键阻塞账户清理，比较永久标记、singleton、原表指纹及序列；原实现模式已变化应RED。
- [ ] TestFixtureStorageResetContracts验证全部六个账户、原角色、原真实密码/独立盐、旧会话失效；standalone保留目录，workflow清空目录后正常真实流程。
- [ ] 最小抽取存储清理与基础账户初始化；一个Tx中按原顺序Exec全部原存储语句，可选目录语句最后执行；保留原RESTRICT/CASCADE及守卫。原账户逻辑原样调用。
- [ ] 重跑新用例、原场景复位/managed/分享及全部e2etest/testutil；记录真实结果和相同环境计时，提交。

Run: `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/e2etest ./internal/testutil -timeout 5m -count=1`

Expected: 原实现真实4≠1及回滚失败RED；最小修改后新用例及全部原隔离流程PASS，无预算调整。

## Task 2: 兼容、独立审查、MR及恢复部署流程

**Files:** 中文审查/验收记录、精确兼容政策；新MR说明。原生产与浏览器/CI代码不改。

**Interfaces:** 消费Task1全部原函数及测试；产出同HEAD代码/兼容审查、完整CI及最终merge commit。

- [ ] 原e2e独立命令执行note-editor-experience、note-sharing、note-sharing-security、ui-language-notes及受影响原内容/题库/managed场景；固定旧版目标保持。
- [ ] Go vet、原API/全部14迁移字节、生产依赖退场、87项工具通过；只登记受审的精确目标。
- [ ] 一次Native新鲜独立全补充分支审查；按实际影响重分级，仅一轮TDD修复并完成受影响整套；不重复审查已验证未改的笔记实现。
- [ ] 正常Git推送、创建并附加新MR；最新HEAD六工作流/全部检查通过后合并master。
- [ ] 以最终merge commit新建release并按已确认生产运维流程重新验证备份/恢复、prepare、start、数据保留和activate；公网40次只读及分享权限/健康通过。旧119f0b5候选不激活。
- [ ] 最终证据落到新MR及私有生产验收记录；保留工作树，清理本任务专属测试容器及本计划scratch。

Run: `node tools/verify/run.mjs -- env TOPIC_COMPATIBILITY_STAGE=managed node --test tools/verify/*.test.mjs`

Expected: 全部PASS，独立审查无阻塞，最新修复CI全部通过，生产最终准确merge commit与迁移14正常；不提前声明任何门禁通过。
