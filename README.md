# 数学学习成长网站

主题学习重构采用63个一级主题、534个二级主题、4969个具体主题，503个辅助分类及534个其他兜底节点单列，完整目录6603个节点。网站围绕知识阅读、四状态、私人笔记、自动主题进度、时间线和检索复习组织，普通登录进入“我的学习”。[完整功能与使用方法](docs/operations/website-guide.md)、[切换维护手册](docs/operations/topic-learning-cutover.md)和[已确认总计划](docs/superpowers/plans/2026-10-06-topic-learning-refactor.md)说明边界与流程。

分类阶段通过草稿MR #38、个人学习通过草稿MR #39交付，退出/迁入/兼容代码已实现并在隔离环境验证，本机完整回归已通过（22批后端、244项双视口浏览器、462项前端、125项工具、28项运维），一次独立终审的两项问题及CI库存遗漏已修复，修复后463项前端、126项工具及60项相关浏览器通过；最新提交远端CI为最终交付门禁。正式资料仍须正常数学审核，代码交付不代表已经合并、部署或执行真实切换。下文P1—P7保留为既有交付历史，旧测评新写仅在legacy模式回归，topics模式明确返回410。

本项目旨在搭建一个全面的数学学习成长网站，以内容正确性、主题覆盖与知识点学习为核心，逐步覆盖零基础学习到学术研究与知识分享。

P1 内容基础层已于 2026-10-01 通过 [PR #5](https://github.com/yyl1212/math_master/pull/5) 合并到 master，已完成独立审查及回归；提供 Go 服务、正式目录、严格校验、PostgreSQL 版本存储及草稿导入导出。P2 英文只读前端已实现五页面与安全数学阅读，已完成独立代码审查问题修复及本机回归，CI 结果见 PR；P3a 已实现账户、会话与角色管理，已完成独立审查与本机回归，功能交付见 [PR #10](https://github.com/yyl1212/math_master/pull/10) 及其最新提交 CI；P3b 内容工作流已按十项任务实现，已完成整分支回归、独立审查及三项问题修复，通过 [PR #14](https://github.com/yyl1212/math_master/pull/14) 交付，技术提交的四项 CI 全部通过，详见 [技术验收记录](docs/operations/2026-10-01-p3b-acceptance.md)；学习记录和部署继续按后续阶段开发。技术方案采用 Go 业务后端、Next.js / TypeScript 前端和 PostgreSQL；首版先建立 16 个学习板块的方向地图，做扎实初等数学学习路线。

P3b 已于 2026-10-01 通过 PR #14 合并，master 086ed7a 的后端和前端 CI 均通过。P4 已确认分为“可信题库”和“学习检测”两次交付；[P4a 可信题库设计](docs/superpowers/specs/2026-10-02-question-bank-design.md)已获用户书面确认，通过 [PR #15](https://github.com/yyl1212/math_master/pull/15) 合并。[13 项执行计划](docs/superpowers/plans/2026-10-02-question-bank.md)已确认并通过 PR #16 合并，沿用 Native。P4a 可信题库已实现，一次独立审查的四项重要问题已修复，全部本机回归与容量验证通过，通过[PR #17](https://github.com/yyl1212/math_master/pull/17)交付，技术提交四项CI全部通过，最终文档提交检查见PR最新状态；详见[技术验收记录](docs/operations/2026-10-02-p4a-acceptance.md)。个人练习、检测、解锁与进度继续在 P4b。

## 设计文档

[总体方向与首版设计](docs/superpowers/specs/2026-09-30-math-learning-platform-design.md)包含知识点、解锁与回顾、内容和题库审核、反馈纠错、架构、文件范围与验收标准。

[开发路线图](docs/superpowers/plans/2026-09-30-development-roadmap.md)按可信内容底座、英文页面、账户审核、学习检测、反馈纠错、首批数据验收、部署试运行推进。[P1 执行计划](docs/superpowers/plans/2026-09-30-content-foundation.md)给出首阶段的文件、接口、验证和提交步骤，记录首阶段实现与验收。[P2 英文只读网站执行计划](docs/superpowers/plans/2026-10-01-english-readonly-frontend.md)包含五页面、素材接口、文件安排、架构及六项实施任务，已按用户确认的计划实现六项任务，已完成整分支回归和独立审查问题修复。2026-10-01 较早批次已读取用户提供的本地 `Knowledge_JSON` 目录，包含 31 个资料包、8,722 条数学候选记录及 104 条软件能力记录；[检查报告](docs/content/2026-10-01-knowledge-json-inspection.md)记录文件校验、结构差异与重复内容。[资料来源清单](docs/content/source-inventory.md)保留本地路径、Library 入口与 ZIP 整理流程，后续本次快照已增至 35 个资料包、905 个文件；正式数学内容仍需分批编写、去重并独立复核。

[英文页面设计与预览说明](design/README.md)包含 16 板块数据、页面层级、样例路线及预览启动方式。后续直接在项目中设计与开发，已取消 Figma 同步。

```bash
python3 -m http.server 8897 --bind 127.0.0.1 --directory design
```

打开 <http://127.0.0.1:8897/preview/#map>。只提供 `design` 目录，页面中的讲解和学习记录均为演示数据。


## P1 开发入口

详见 [内容基础层操作说明](docs/operations/content-foundation.md) 和 [OpenAPI](api/openapi.yaml)。目录为 16 板块、56 主题；10 个初等数学知识点均为草稿，其中 9 个仍是正文骨架。零结构错误不代表独立数学审核通过，P1 尚无公开数学内容。

从项目根操作，先按 `.env.example` 创建本机 `.env` 并填写开发数据库凭据：

```sh
set -a
source .env
set +a
docker compose -p math-master-p1 -f compose.dev.yaml up -d --wait db
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/ ./cmd/...
./backend/bin/migrate --dir db/migrations up
./backend/bin/content-check --catalogue content/catalogue/domains.json --package content/packages/elementary-fractions.v1.json --assets content/assets
./backend/bin/content-import --catalogue content/catalogue/domains.json --package content/packages/elementary-fractions.v1.json --assets content/assets
./backend/bin/server
```

浏览 `http://127.0.0.1:8080/api/v1/domains`。草稿知识与路线返回 404，建设状态由有效发布快照派生；数据接口不读取 `design/data`。P2 已接入英文页面，P3b 提供独立审核和发布工作流，实际草稿仍须人员复核。

```sh
node --test tools/verify/run.test.mjs tools/content-ingest/snapshot.test.mjs
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go vet ./...
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/content ./internal/config ./internal/httpapi -timeout 5m -count=1
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -timeout 5m -count=1
```

测试必须设置专用 `TEST_DATABASE_URL`；每次创建并清理自己的随机 `math_master_test_*` 库。每条验证命令最多 540 秒，超时终止宽限 5 秒。

## P2 英文网站入口

详见 [英文只读网站操作说明](docs/operations/english-readonly-frontend.md) 和 [P2 验收记录](docs/operations/2026-10-01-p2-acceptance.md)。生产构建接入 Go API，提供首页、知识地图、板块详情、路线与知识阅读；支持英文/中文目录搜索、真实建设状态、精确版本前置图、受限 Markdown/KaTeX 和有效发布素材。实际公开数学内容仍为 0，测试发布只发生在随机隔离数据库。

```sh
node tools/verify/run.mjs --cwd frontend -- npm ci
node tools/verify/run.mjs --cwd frontend -- npm run build
cd frontend
npm run start -- --hostname 127.0.0.1 --port 3000
```

启动前按操作说明设置仅供服务端使用的 `GO_API_INTERNAL_URL` 并启动 Go 服务。匿名阅读不创建学习记录，个人解锁与进度属于 P4。

## P4a 可信题库入口

详见[题库操作说明](docs/operations/question-bank.md)。英文编辑、独立六项复核、双head发布、覆盖与永久撤回后台接入 Go/PostgreSQL；三项 CLI 只检查与导入导出草稿。新迁移00005须操作者显式启用，本次保留开发库和持续更新的资料。技术夹具不计入正式题量。

## P3b 内容工作流入口

详见 [内容操作说明](docs/operations/content-workflow.md)。英文编辑、复核、发布与撤回后台接入 Go/PostgreSQL；知识、公理、定理、来源及原创 SVG 按固定版本审核。上线前需操作者显式迁移、授予人员角色并完成真实数学复核，本次仅在随机测试库验收。

[知识草稿阅读页说明](docs/operations/knowledge-draft-reading.md)：在工作区列表或编辑器点击 **Read saved draft**，可搜索中英文标题和 ID、逐点阅读已保存版本、查看私有原创配图并提交一般网站反馈。页面展示真实 revision 和草稿状态，沿用现有权限与发布流程。

## P3a 账户入口

详见 [账户与权限操作说明](docs/operations/account-foundation.md) 和 [P3a 验收记录](docs/operations/2026-10-01-p3a-acceptance.md)。英文注册、登录、账户与管理员页面接入真实 Go/PostgreSQL；密码修改、角色变更和人工重置撤销旧会话。账户能力独立于数学内容发布和学习进度。

启用前按操作说明备份、显式迁移并配置相同认证 origin；首次管理员通过隐藏终端输入的 admin-init 创建。没有默认真实管理员或口令。本次隔离验收保留开发库和资料快照。

## 目录结构

```text
math_master/
├── .gitignore                 # 排除私有配置和本机文件
├── README.md                  # 项目说明
├── config/
│   ├── server.example.json    # 可提交的服务器配置模板
│   └── server.local.json      # 本机私有服务器配置，不提交 Git
├── backend/                   # Go 服务、命令与业务校验
├── frontend/                  # Next.js / TypeScript 英文网站与账户页面
├── tests/e2e/                 # 真实 Go / PostgreSQL 浏览器回归
├── schemas/                   # 唯一正式 JSON 契约及嵌入模块
├── content/                   # 正式目录、未审核草稿及原创素材
├── db/migrations/             # 显式数据库迁移
├── api/openapi.yaml           # 公开只读及私有账户契约
├── tools/                     # 资料快照与验证时限
├── compose.dev.yaml           # 本机 PostgreSQL 17.11
├── design/                    # 英文设计预览、16 板块数据与历史设计脚本
└── docs/
    ├── content/               # 资料入口、来源清单与知识点映射
    └── superpowers/
        ├── specs/             # 中文设计文档
        └── plans/             # 分阶段路线图与执行计划
```

## 配置结构

```mermaid
flowchart TD
    Project[数学学习成长网站项目] --> Config[config 配置目录]
    Config --> Example[server.example.json 配置模板]
    Config --> Local[server.local.json 本机私有配置]
    Example --> Git[Git 版本管理]
    Local --> Private[仅本机使用，Git 忽略，权限 600]
```

`config/server.local.json` 已保存本次提供的服务器地址、用户名与密码。SSH 端口暂按默认值 `22` 配置，后续可按实际服务器设置调整。本次未连接服务器或验证 SSH 服务。

| 字段 | 含义 |
| --- | --- |
| `host` | 服务器地址 |
| `port` | SSH 端口 |
| `username` | 登录用户名 |
| `password` | 登录密码，仅存于本机私有配置 |

在其他开发环境中，可从模板创建自己的私有配置：

```bash
install -m 600 config/server.example.json config/server.local.json
```

随后填写实际配置。私有配置包含明文密码，权限 `600` 仅允许文件所有者读写；请勿强制添加到 Git。模板中不保存密码。

## Git 工作流程

远程仓库为 `git@github.com:yyl1212/math_master.git`，本地远程名称为 `origin`。仓库初始分支为 `master`，本次配置工作在 `chore/project-init` 分支进行。首次同步时远程仓库为空，因此先建立 `master` 基线，再推送开发分支并创建面向 `master` 的 PR（即 MR）。

后续开发可按以下步骤更新基线并创建分支：

```bash
git switch master
git pull --ff-only origin master
git switch -c codex/your-feature
```

远程访问优先使用本机已配置的 SSH 密钥，GitHub CLI 用于创建 PR。令牌不写入项目文件或远程地址。

后续开发遵循项目约定：先更新远程 `master`，再新建开发分支；按审查过的方案与计划实施，完成代码审查、回归验证后，通过 Git 托管平台创建 MR。每次测试不超过 10 分钟，项目文档使用中文。
