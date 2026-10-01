# P3b 内容工作流技术验收

日期：2026-10-01。范围：工作区、固定送审、独立复核、manifest 准备、激活、撤回、英文后台及严格 Go/Next.js 边界。设计和实施计划均已确认，基于 master 2d5ca6f 在隔离分支 codex/p3b-content-workflow 实施。

## 实现与保护

十项任务覆盖 publication 契约、迁移 00004、同事务授权/幂等、SVG 与固定版本校验、19 个私有 HTTP 操作、严格客户端和七类动态后台页面。schemaVersion=1、P1 包摘要、公开 DTO 和原账户边界保持；管理员不自动拥有编辑/审核角色。

使用可修改工作区、不可修改送审和审核、不可修改 manifest 分离责任。复制继承作者，作者不能批准；当前资格、会话、DB 时间在锁等待之后及提交前重验；审计、幂等结果和 head 在同一事务完成。永久撤回与反向前置闭包防止旧候选重新发布。

真实开发库、源 Knowledge_JSON 和既有离线快照均未用于测试发布。所有迁移、账户、批准和公开测试 head 仅在严格命名并校验连接的随机测试库；正常退出由既有 helper 清理。本次未部署或完成真实数学内容审核。

## 容量与性能实测

本机 macOS、Apple silicon、Go 1.27.1 CGO_ENABLED=0、Docker PostgreSQL 17.11；数字仅描述本机技术夹具，不代表生产吞吐或可用内容。

容量组合同时达到 1000 知识、4000 单元、200 路线、16000 关系、1000 原创 SVG、规范 JSON 33,554,432 字节及唯一 SVG 10,485,760 字节。63 个独立合法批次经真实送审/复核，按每次最多 20 个准备和激活；最终 6200 固定成员。超限测试单独保持其余约束合法。

首次失败定位到继承证据的 LATERAL 查询：3971 个旧成员被展开 3971 次，移除 15,764,870 次不匹配，单查询 4936.55 ms。改为单次 MATERIALIZED 展开后完整流程通过，四组 prepare 约 1.09/2.21/3.29/3.47 秒，最长操作 3.47 秒，原 8 秒截止不变。Go 累计分配约 1.06/2.00/3.02/3.25 GB，是各操作累计分配而非驻留内存；整测试进程最大 RSS 621,428,736 字节（约 593 MiB）。

最大 PublicationView 为 3,454,247 字节，低于 4 MiB。公开 knowledge/path 约 183/182 ms，分别满足既有 3/5 秒预算；6200 成员查询使用 publication_member_lookup 索引，执行 0.727 ms。新公开边界测试证明原校验遗漏 data 包装和换行，修复后完整 HTTP JSON 精确 10 MiB 可读，+1 拒绝；公开格式未变。

基准各跑 -benchtime=3x：WorkflowValidation（一个合法最大工作区批次）约 10.26 ms/op、22,403,154 B/op、108,725 allocs/op；WorkflowSnapshot（完整最大组合）约 791.74 ms/op、2,017,397,045 B/op、6,304,404 allocs/op。基准进程最大 RSS 508,018,688 字节（约 484 MiB）。双槽同时执行实际最大校验于 0.88 秒完成，第三槽立即拒绝；安全唯一 SVG 10 MiB + 1 被拒绝，JSON/count 超限、锁等待取消均通过。包含并发容量场景的完整 store/cli 回归最大 RSS 791,085,056 字节（约 754 MiB）。生产资源与真实人员独立数学验收仍在 P6/P7，不能以技术批准替代。

## 最终验证状态

技术验收通过：工具 9/9；Go vet、auth/content/publication/config/httpapi/e2etest/testutil 全通过；store/cli 全部 70 个顶层测试通过（store 77.728 秒、cli 0.951 秒）；Go 构建成功。前端类型检查、23 文件 69 项单元测试、生产构建、依赖审计通过，审计为 0 漏洞。生成类型无差异，Go 全目录格式及 git diff --check 通过。

真实浏览器两种视口：content-authoring 4/4、content-review 4/4、content-release 2/2、content-security 6/6；旧 catalogue/reading 16/16、auth/auth-security 12/12，共 44/44。所有批次 workers=1、retries=0，分别在 480 秒内完成。成功路径没有 route.fulfill。

验收后只读检查：随机测试库剩余 0；真实开发库仍有 10 个知识点、公开 head 0、迁移版本 2，没有账户表或内容工作流表；原始冻结资料目录和 manifest 保留。CI 已加入 publication 包和四个独立内容浏览器步骤，保留旧回归。

独立整分支审查和 PR 精确 head CI 尚待完成；交付状态随后补记。
