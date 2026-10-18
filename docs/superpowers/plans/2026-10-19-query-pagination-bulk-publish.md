# 查询分页与知识批量发布实施总计划

> 执行者必须使用 `superpowers:executing-plans`（Native）或用户选择后的 `superpowers:subagent-driven-development`，按任务复选框实施。用户已选择 Native，按此计划完成实现并进入最终审查。

**Goal:** 全站查询显示页数、支持图标翻页与跳转，统一行内文字操作，并提供可靠的知识勾选发布和全量发布。

**Architecture:** 公共分页组件适配原偏移量查询和显式页码查询，保留原游标协议。知识发布使用数据库固定清单及持久化回执，由页面串行驱动短事务，支持暂停和恢复。分页与发布分为两个可独立验收的子计划，最后统一审查及创建 MR。

**Tech Stack:** 仓库现有 Go、Next.js / TypeScript、PostgreSQL、Zod、Vitest、Playwright；不升级依赖、不新增图标库或后台任务服务。

**Spec:** [已确认书面设计](../specs/2026-10-19-query-pagination-bulk-publish-design.md)。用户于 2026-10-19 确认书面设计；功能方向及全量范围已在此前确认。

## Global Constraints

- 文档与审查结论用中文；图标为项目自制 SVG。
- 全量发布覆盖整个知识库中当次操作开始时未发布且未删除的知识点，不受当前筛选限制，后续新增不加入。
- 选中发布每次最多 1000 条，请求体限制为 256 KiB；每批最多 20 条，可缩小至 1 条。
- 新增迁移 `00016_knowledge_publish_batches.sql`；不修改 00001—00015 迁移字节，不修改旧表或原 schema 指纹。
- HTTP/事务截止 8 秒、锁等待 1 秒，普通 Next/浏览器请求截止 10 秒；确认回滚后批大小依次为 20、10、5、2、1；临时错误最多连续重试 3 次。
- Go 构建、调试和测试均使用 `CGO_ENABLED=0`；命令由 `tools/verify/run.mjs` 限制最多 540 秒，Go 单组 `-timeout 5m`，浏览器总时限保持 480 秒。
- `pagination=page` 才启用新协议；原游标模式、权限、排序、严格 schema、匿名最新 10 条、原功能跳转和 410 保持。
- 新分页响应为真实 `items/total/limit/offset`，保留所属接口外层 actor/envelope/asOf；计数和取页在同一读取事务内。
- 数据库与浏览器测试使用随机隔离合成库；没有配置 `TEST_DATABASE_URL` 而跳过的仓储测试不得报告通过。
- 已确认开发基线 `02e6a0b12992b754a15f429bcdb185310eb8a463`，当前分支 `codex/query-pagination-bulk-publish`；实施前再次 fetch，若 master 前进，先合入并重新核对迁移编号与契约。

## Review Focus

1. 旧游标锚点正好处于同一时间戳、边界删除或筛选变化时，仍必须校验范围，返回规范页，不泄露其他用户数据（P2/P3）。
2. 浏览器返回旧页面、RSC 服务端传参、两个独立分页区块和迟到响应叠加时，页码、筛选和实际数据必须一致（P1/P4）。
3. 全量清单创建或 COMMIT 响应丢失，不能把未知结果误当回滚，也不能吸收后来新增知识（B2/B3/B5）。
4. 一条大正文拖慢整个批次、管理员身份在运行中变化、两个标签页推进同一操作时，必须准确暂停或恢复（B3/B5）。
5. 迁移 16 安装后，原版本 15 的 Down/恢复测试仍验证原版本；兼容文件白名单、API 摘要及 CI 库存不能漏测或放宽（B1/P6/B6）。

## 拆分、接口与文件责任

| 子计划 | 产物 | 执行顺序 |
| --- | --- | --- |
| [A：查询分页与文字操作](2026-10-19-query-pagination.md) | P1 公共组件；P2/P3 页码接口；P4/P5 页面覆盖；P6 兼容与浏览器验收 | P1 → P2 → P3 → P4 → P5 → P6 |
| [B：知识批量发布](2026-10-19-knowledge-bulk-publish.md) | B1 模型/迁移；B2 固定清单；B3 发布批次；B4 契约/客户端；B5 页面；B6 容量及浏览器验收 | A 完成后 B1 → B2 → B3 → B4 → B5 → B6 |

P1 定义 `pagination.Request/Page/ActorPage`、前端 `PageMeta` 和分页组件的公开接口，两个子计划都消费它们。B 不修改 A 的查询协议。不同包的原 Repository 保持，通过新增窄接口扩展页码或发布能力，避免为旧 mock 强行增加无关方法。

```mermaid
flowchart LR
    P1[公共分页模型和组件] --> P2[当前页码接口]
    P2 --> P3[反馈通知及归档页码]
    P3 --> P4[当前查询页面]
    P4 --> P5[可访问旧列表]
    P5 --> P6[分页兼容与验收]
    P6 --> B1[发布模型和迁移]
    B1 --> B2[固定发布清单]
    B2 --> B3[原子批次]
    B3 --> B4[发布接口与客户端]
    B4 --> B5[选择进度和恢复]
    B5 --> B6[容量和发布回归]
    B6 --> Review[整分支审查 回归 MR]
```

## 实施前与最终交付

- [x] 同步 master、创建分支，提交书面设计 `daac202`，用户确认该设计。
- [x] 用户审阅本总计划及两个子计划，并选择 Native 或逐任务子代理执行。
- [x] 复用当前已隔离工作树，记录最新 master、迁移编号、原 schema 摘要和原协议样例；不重新创建无关工作树。
- [x] 依次执行 P1—P6、B1—B6，每个任务先运行对应失败用例，再最小实现并验证；按任务提交，记录实际测试证据。
- [x] 执行整分支代码审查，逐项处理发现，针对修复重跑相关回归；新增正确性或兼容性问题未解决前不得创建完成态 MR。
- [x] 前端类型检查、构建与单元测试；Go vet/build 及按领域分批测试；真实 PostgreSQL 和桌面/手机浏览器回归，均遵守单次时限。
- [x] 更新中文验收记录 `docs/superpowers/reports/2026-10-19-query-pagination-bulk-publish-acceptance.md`，逐项记录 Spec 第 12 节的 12 条验收结论、测试命令、数量和实际限制。
- [x] Git 提交、push 开发分支，通过 `gh pr create --base master --head codex/query-pagination-bulk-publish --body-file <中文说明文件>` 创建 MR，再将实际 MR 链接附加到本聊天；不自动合并。已创建并附加 [MR #62](https://github.com/yyl1212/math_master/pull/62)。

最终共用验证命令（各条独立执行；数据库测试按子计划明确的组执行）：

```sh
node tools/verify/run.mjs --cwd frontend -- npm run typecheck
node tools/verify/run.mjs --cwd frontend -- npm run build
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go vet ./...
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build ./cmd/...
git diff --check
```

## 计划自审记录

Spec 第 1—7 节由 P1—P5 覆盖；第 8—9 节由 B1—B5 覆盖；第 10 节文件范围在各任务展开；第 11—12 节由 P6/B6 及整分支交付覆盖。五类 Review Focus 均在所属任务有具名测试。

RSC 的链接模式使用可序列化路径和参数，函数只存在于客户端回调模式；旧测试固定到其原迁移目标；正文不进入发布回执；确认回滚与提交未知有不同错误码。深页超过旧 offset 上限时，新增显式页码能力覆盖该接口，旧模式原上限保持。这些是已确认兼容方案的实现约束，不放宽旧合同。

本阶段只提交设计与计划文档，未修改产品代码或执行真实发布。计划自审完成后等待用户确认和执行方式选择，再进入实施。
