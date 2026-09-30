# 数学学习成长网站

本项目旨在搭建一个全面的数学学习成长网站，以内容正确性、内容覆盖与数量和丰富的学习路径为核心，逐步覆盖零基础学习到学术研究与知识分享。

当前完成 Git 仓库与服务器配置初始化，已形成首版设计、英文页面预览与分阶段开发计划，生产网站尚未实现。技术方案采用 Go 业务后端、Next.js / TypeScript 前端和 PostgreSQL；首版先建立 16 个学习板块的方向地图，做扎实初等数学学习路线。

## 设计文档

[总体方向与首版设计](docs/superpowers/specs/2026-09-30-math-learning-platform-design.md)包含知识点、解锁与回顾、内容和题库审核、反馈纠错、架构、文件范围与验收标准。

[开发路线图](docs/superpowers/plans/2026-09-30-development-roadmap.md)按可信内容底座、英文页面、账户审核、学习检测、反馈纠错、首批数据验收、部署试运行推进。[P1 执行计划](docs/superpowers/plans/2026-09-30-content-foundation.md)给出首阶段的文件、接口、验证和提交步骤，供实施前审阅。已读取用户提供的本地 `Knowledge_JSON` 目录，包含 31 个资料包、8,722 条数学候选记录及 104 条软件能力记录；[检查报告](docs/content/2026-10-01-knowledge-json-inspection.md)记录文件校验、结构差异与重复内容。[资料来源清单](docs/content/source-inventory.md)保留本地路径、Library 入口与 ZIP 整理流程，正式内容仍需转换、去重并独立复核。

[英文页面设计与预览说明](design/README.md)包含 16 板块数据、页面层级、样例路线及预览启动方式。后续直接在项目中设计与开发，已取消 Figma 同步。

```bash
python3 -m http.server 8897 --bind 127.0.0.1 --directory design
```

打开 <http://127.0.0.1:8897/preview/#map>。只提供 `design` 目录，页面中的讲解和学习记录均为演示数据。

## 目录结构

```text
math_master/
├── .gitignore                 # 排除私有配置和本机文件
├── README.md                  # 项目说明
├── config/
│   ├── server.example.json    # 可提交的服务器配置模板
│   └── server.local.json      # 本机私有服务器配置，不提交 Git
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
