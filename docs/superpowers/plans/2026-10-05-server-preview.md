# P7a：服务器预览版实施计划

> **执行要求：**沿用用户已选择的 Native，由当前执行者使用 `superpowers:executing-plans` 逐任务实施；步骤用复选框跟踪。实现结束后使用 `superpowers:requesting-code-review` 做一次独立整分支审查。

**目标：**尽快在 `https://43.135.142.53` 提供可正常登录、阅读现有私有草稿且具备备份恢复能力的服务器预览版。

**架构：**Caddy 接收公网 HTTPS，所有网站及业务 API 请求经 Next.js 现有代理进入 Go，PostgreSQL 仅在内部网络可访问。本机库通过一致性快照复制到新的服务器预览库，先隔离恢复验证，再启动应用和公网入口；按提交号保存镜像、配置与回退记录。

**技术栈：**Go 1.27.1、Node.js 24.17.0、既有 Next.js 16.3.7、PostgreSQL 17.11、Caddy 2.11.7、Docker Compose v2、Python 3 标准库与 shell 入口脚本。

**依据：**[已确认方案](../specs/2026-10-05-server-preview-design.md)。日期：2026-10-05。状态：方案及其兼容性范围已获用户确认；本计划待审阅，尚未实施。此前只读预检基线为 master `9f71cf4f2b29c5101cc87a4511c0b32d786ddeb0`。

## 全局约束

- 开发前通过 SSH 获取最新 master，并从该提交新建 `codex/server-preview` 分支；使用独立工作区，保留已确认的方案与计划。本机 R1 正在使用的两个工作区、数据库及端口不用于测试或重建。
- 保持 Native，不重新选择执行方式；每任务完成相关检查后提交，开发完成后审查、回归并通过 SSH 提交 PR。新 PR 合并按届时明确授权执行。
- 仅改变生产运行、HTTPS origin 与 standalone 构建。既有业务接口、迁移、角色、数学审核和发布规则保持现有行为；原 CI 工作流与兼容性基线不改写。
- Go 构建与测试均设置 `CGO_ENABLED=0`。所有验证通过 `node tools/verify/run.mjs -- …` 限制到 540000ms；Go 测试另设 `-timeout 5m`，浏览器单批沿用 480000ms，不允许单次测试超过十分钟。
- 目标平台固定 `linux/amd64`。构建基础镜像及程序包摘要见下表；禁止浮动 `latest`、不存在的 Caddy 标签或未验证的替代下载。
- Go/Next.js 均使用 `APP_ENV=production`、`AUTH_PUBLIC_ORIGIN=https://43.135.142.53`；Go `HTTP_ADDR=0.0.0.0:8080`，Next.js `HOSTNAME=0.0.0.0`、`PORT=3000`、`GO_API_INTERNAL_URL=http://api:8080`。保留 `__Host-` Cookie 和原开发模式限制。
- 常驻服务名为 `db/api/web/gateway`，项目名 `math-master-preview`；仅 gateway 映射 TCP 80/443，Caddy 管理接口和其余服务不映射宿主机。内存初值依次为 512/512/768/128MiB，构建串行并限制前端 Node 堆。
- 服务器使用 `/opt/math_master/releases/<40位提交号>/`、`shared/`、`backups/`；环境文件和备份分别为 0600、目录 0700。Git、构建上下文和公开证据均不包含凭据、备份、会话或私有数学资料。
- 使用新的服务器数据库凭据；应用账号密码、learner/editor/admin 角色和草稿作者关系由备份保留。禁止重新初始化管理员、重置密码、增加 reviewer、送审、发布或自动重导入不断更新的 Knowledge_JSON。
- 每日备份保留 7 日，每周副本保留 4 周，发版/迁移前另留快照；首次上线及每次发版复制一份到本机。每日自动站外备份留待 P7b，不宣称已经实现。
- 真库回退保留数据卷，不执行迁移 down、`compose down -v` 或自动覆盖恢复。数据库不兼容或恢复不一致时停止写入并保留证据。
- 文档使用中文，页面继续以英文为主；不引入第三方图片素材。本次只验收预览部署，正式数学验收状态继续为未完成。

| 构建输入 | 固定摘要（linux/amd64；基础镜像使用平台摘要） |
| --- | --- |
| `golang:1.27.1-alpine` | `sha256:cd9a32216aee5667f957a62d13a10032a63fd58e14b3f3d9cc8c2122f501e95e` |
| `node:24.17.0-alpine` | `sha256:9e04e3f9c9164cb2c913593ba0733fbd0caf4e9bea474d743ca8a32aa36d98c9` |
| `postgres:17.11`（常规版） | `sha256:e31e3d5327d1806f6177827c9710643e4f35f7ab3f14d26d05332753d3e95ee0` |
| `alpine:3.24` | `sha256:d56c381f961d307a21b3ca004cf1e3910f106644aefb1f43e654c8a56c4fd395` |
| Caddy 官方 `caddy_2.11.7_linux_amd64.tar.gz` | `sha256:727b91701a392de6ebc5027509f548bf39979e5216340d0faed8fa5e69c84f8b` |

Caddy 下载地址为 `https://github.com/caddyserver/caddy/releases/download/v2.11.7/caddy_2.11.7_linux_amd64.tar.gz`；摘要先验证再解包。预检中的 index 摘要及版本核验依据见 [preflight.json](../../operations/evidence/server-preview/preflight.json)。

## 审查重点

1. 环境文件含命令文本、生产 origin 错误或继承本机开发变量：在任何 Docker 变更前拒绝，不执行文件内容、不打印凭据；任务 2 的配置测试覆盖。
2. 备份期间源库仍有写入，或导出中途失败/磁盘写满：清单与 dump 必须来自同一快照，失败不留下可用备份、不覆盖旧目录；任务 3 的快照测试覆盖。
3. 恢复目标误选本机/已有库，或演练清理误删真实卷：拒绝恢复，清理只触及本次创建且标签一致的容器和卷；任务 3 的目标保护测试覆盖。
4. standalone 缺少 CSS/公式字体、运行时 origin 失效，或旧测试证书被当成可信证书：真实镜像和普通 TLS 客户端必须验证这些条件；任务 1、2、5、6 覆盖。
5. 新版本健康检查失败或回退失败：保留上一版本记录、数据库和证据，不继续公开激活；任务 4 的部署失败测试覆盖。

## 文件与依赖

| 文件 | 责任与所属任务 |
| --- | --- |
| `.dockerignore`、`backend/Dockerfile`、`frontend/Dockerfile`、`ops/caddy.Dockerfile`；修改 `frontend/next.config.ts` | 任务 1：可追溯的生产镜像、静态资源和程序白名单 |
| `compose.yaml`、`ops/Caddyfile`、`ops/.env.example`、`ops/common.py` | 任务 2：编排、证书配置、配置校验和统一 Docker 调用 |
| `ops/database-snapshot.py`、`ops/backup.sh`、`ops/restore-drill.sh` | 任务 3：一致性快照、受保护恢复、隔离演练和备份保留 |
| `ops/deploy.py`、`ops/deploy.sh`、`ops/math-master-backup.service`、`ops/math-master-backup.timer`、`docs/operations/server-preview.md` | 任务 4：发版状态、回退、每日备份与中文运行手册 |
| `ops/verify-deployment.py`、`ops/tests/image_smoke.py`、`ops/tests/test_config.py`、`ops/tests/test_snapshot.py`、`ops/tests/test_deploy.py`、`ops/tests/test_verify.py`、`.github/workflows/deployment.yml` | 分别随所属任务实现；任务 5 汇总部署检查及现有回归 |
| `docs/operations/evidence/server-preview/`、开发路线图 | 任务 5、6：无秘密审查/上线结果及实际状态 |

```mermaid
flowchart LR
    T1[1 生产镜像] --> T2[2 内部编排与 HTTPS 配置]
    T2 --> T3[3 一致备份与隔离恢复]
    T3 --> T4[4 部署与回退]
    T4 --> T5[5 回归、审查和 PR]
    T5 --> T6[6 服务器上线与实际验收]
    T6 --> Iteration[后续题库阅读、内容复核与优化]
```

## 任务 1：可运行的生产镜像

**文件：**新增 `.dockerignore`、三个 Dockerfile、`ops/tests/image_smoke.py`；修改 `frontend/next.config.ts`。

**接口：**从仓库根目录构建；输出 `math-master-api:<revision>`、`math-master-web:<revision>`、`math-master-gateway:<revision>`，三者含 OCI revision 标签。API 镜像工作目录 `/app`，仅提供 `/app/bin/{server,migrate,admin-init,correction-maintenance}` 与 `/app/db/migrations`；web 入口为 standalone 的 `node server.js`；gateway 提供 `/usr/bin/caddy`。镜像烟测 CLI：`python3 ops/tests/image_smoke.py --revision <40位提交号>`，成功退出 0，报告版本和通过项，不输出环境值。

- [ ] **1. 写镜像烟测。**验证四个 Go 程序可找到、不存在 `e2e-harness`，web 在容器内部 `/login` 返回 200 且其引用的 CSS/公式字体能读取，gateway 报告 `v2.11.7`；临时容器使用随机名称并只清理自己创建的对象。
- [ ] **2. 运行失败检查。**用现有验证包装器运行上述烟测，预期因待构建镜像缺失而失败，记录具体缺失项。
- [ ] **3. 实现生产构建。**Go builder 同时复制 `backend/` 与被 go.mod 引用的 `schemas/`，使用显式程序白名单和 `CGO_ENABLED=0`。Next.js 配置增加 `output: "standalone"`，保留 `poweredByHeader: false`；复制 standalone 与 `.next/static`，当前没有 `frontend/public/`，按可选目录处理。Caddy 按固定摘要校验官方静态程序包并安装 CA 根证书；全部 runtime 以非 root 用户运行并具备所需目录权限。
- [ ] **4. 构建并验证。**按 API、web、gateway 顺序，每次单独通过包装器执行 `docker build --platform linux/amd64`，标签使用实际提交号；web 构建设 `NODE_OPTIONS=--max-old-space-size=1536`。烟测退出 0；构建上下文确认排除 `.git`、`.superpowers`、`.env*`（示例除外）、`config/*.local.json`、输入资料、dump、缓存及测试结果。
- [ ] **5. 提交任务。**只提交本任务文件，提交信息 `build: 增加服务器预览生产镜像`。

## 任务 2：内部编排、配置校验和 HTTPS 网关

**文件：**新增 `compose.yaml`、`ops/Caddyfile`、`ops/.env.example`、`ops/common.py`、`ops/tests/test_config.py`。

**接口：**`common.load_env(path: Path) -> dict[str, str]` 严格解析非可执行配置；`common.docker(args: list[str], sudo: bool = False) -> subprocess.CompletedProcess[bytes]`；`common.compose(root: Path, revision: str, args: list[str], sudo: bool = False, acme: str = "production") -> subprocess.CompletedProcess[bytes]` 选择 `releases/<revision>/compose.yaml` 和 `shared/configs/<revision>.env`。失败输出仅含阶段/退出码；原始输出不进入公开日志。服务器 `--sudo` 使用 `sudo -n docker`，不授予 docker 组权限。

配置允许键仅为 `AUTH_PUBLIC_ORIGIN`、`POSTGRES_DB`、`POSTGRES_USER`、`POSTGRES_PASSWORD`、`DB_LC_COLLATE`、`DB_LC_CTYPE`。数据库/用户名固定 `math_master_preview`，密码为新生成的 64 位小写十六进制；locale 来自源快照，接受单个合法 locale 标识。DATABASE_URL 由 Compose 这些值生成，不在命令参数中传递；生产监听和 APP_ENV 固定在编排文件。动态 release/ACME 参数由已校验的 CLI 生成，不继承调用方的开发配置。

- [ ] **1. 写配置失败测试。**`test_rejects_executable_or_insecure_env` 验证 `$()`、反引号、重复/未知键、HTTP origin、非 0600 文件、符号链接均在 Docker 调用前被拒绝且没有副作用；`test_ignores_inherited_development_settings` 验证调用者的 APP_ENV/DATABASE_URL 不覆盖部署配置；`test_only_gateway_publishes_ports` 从实际 Compose 解析结果核对端口和持久卷。
- [ ] **2. 运行失败测试。**`node tools/verify/run.mjs -- python3 -m unittest discover -s ops/tests -p 'test_config.py' -v`，预期尚未实现 common/Compose 而失败。
- [ ] **3. 实现 common 与编排。**固定项目名，PostgreSQL 卷名 `math-master-preview-pgdata`；Go `/readyz`、web 容器内 `/login`、db `pg_isready` 为健康检查。gateway 独享外部网络，api/web/db 使用内部网络；健康依赖不隐式执行迁移。Compose 读取指定 env 文件，清除相关继承变量；sudo 仅显式转交已经校验的非秘密 release/ACME 变量。
- [ ] **4. 实现网关配置。**Caddy 全部请求反代 web，不另建直达 Go 的公网路由；显式 `issuer acme`、`profile shortlived`，staging 使用 `https://acme-staging-v02.api.letsencrypt.org/directory`，production 使用 `https://acme-v02.api.letsencrypt.org/directory`。两者证书卷分别为 `math-master-preview-caddy-staging`、`math-master-preview-caddy-production`；新卷目录授予 runtime UID 写权限，容器内设置 `net.ipv4.ip_unprivileged_port_start=0` 以允许非 root 网关监听 80/443，不改变宿主机参数。
- [ ] **5. 验证配置。**配置测试及容器中的 `caddy validate` 退出 0；实际 Compose 解析只在内存核对，不打印密码，staging/production 卷名不同。后续真实验证不得复用测试证书。
- [ ] **6. 提交任务。**提交信息 `feat: 配置服务器内部编排与公网 HTTPS`。

## 任务 3：一致性备份与受保护恢复

**文件：**新增 `ops/database-snapshot.py`、`ops/backup.sh`、`ops/restore-drill.sh`、`ops/tests/test_snapshot.py`。

**接口：**Python CLI 提供 `capture --container <id/name> --database <name> --user <name> --out <新目录> --revision <sha> [--sudo]`、`inspect --container … --database … --user … --backup <目录> [--sudo]`、`restore --container … --database … --user … --backup <目录> [--sudo]`、`drill --backup <目录> [--sudo]`。均退出 0 表示成功；capture 生成受限的 `database.dump`、`manifest.json`、`SHA256SUMS`。`backup.sh --root <目录> [--sudo] --kind daily|weekly|predeploy` 调用 capture 并保留相应副本；`restore-drill.sh --backup <目录> [--sudo]` 调用 drill。数据库访问经容器内 Unix socket，不读取应用登录密码。

manifest schemaVersion=1：数据库编码/locale/PostgreSQL 版本、实际源运行提交、迁移版本、创建时间、dump 摘要；每个 public 表的行数和规范化行摘要；工作区 id/revision/status/包版本、素材字节摘要、账号角色和公开 head 数量。完整清单为私有文件，只向 Git 输出数量和一致性布尔结果。目录已存在时拒绝覆盖。

- [ ] **1. 写备份与恢复保护测试。**`test_snapshot_stays_consistent_during_write` 在随机隔离库并发插入后证明 dump 与清单来自同一快照；`test_failed_capture_is_not_published` 模拟导出失败/写入失败，旧备份不变、无成功目录；`test_rejects_corrupt_or_nonempty_target` 验证损坏摘要、locale 不同、错误容器和已有 public 对象均不执行恢复；`test_drill_cleanup_keeps_foreign_volumes` 验证只清理匹配本次标签的对象。
- [ ] **2. 运行失败测试。**通过包装器运行 `python3 -m unittest discover -s ops/tests -p 'test_snapshot.py' -v`，预期 helper 未实现而失败。数据库集成测试使用新建随机测试容器，禁止连接本机 R1。
- [ ] **3. 实现快照与保留。**持有一个只读 repeatable-read 事务并执行 `pg_export_snapshot()`；pg_dump `--format=custom --snapshot` 与清单查询导入同一 snapshot，全部结束后才释放事务。固定 UTC/ISO/UTF-8，对规范化行排序后流式计算摘要，不打印原始行；同文件系统临时目录成功后原子提交。保留 7 个日副本和 4 个周副本，predeploy 不参与自动轮换。
- [ ] **4. 实现受保护恢复。**先验 dump 摘要和 PG/locale；restore 仅接受 `math-master-preview` 的 db 容器/固定库名，或本次随机演练标签及随机库名，且目标必须为空。使用 `pg_restore --no-owner --no-acl --exit-on-error`，随后 inspect 逐表、逐工作区和素材核对；序列验证不落后于相应主键，不对非 MVCC 序列宣称逐字节快照一致。drill 创建无宿主端口、`--network none` 的新 PG 容器与新卷，实际完成恢复；数据库密码经受限临时 env 文件传递。
- [ ] **5. 验证一致性及目标保护。**全部快照单元测试退出 0；随机 PG 容器的实际并发写入/导出/恢复测试退出 0，清单与恢复库摘要一致。所有临时对象清理后本机 R1 和服务器预览卷均未被删除。
- [ ] **6. 提交任务。**提交信息 `feat: 增加一致性备份与隔离恢复演练`。

## 任务 4：部署、失败处理、回退和日常备份

**文件：**新增 `ops/deploy.py`、`ops/deploy.sh`、备份 service/timer、`docs/operations/server-preview.md`、`ops/tests/test_deploy.py`。

**接口：**`deploy.sh prepare|start|activate|rollback --root <目录> --revision <40位提交号> [--sudo]`；activate 另需 `--acme staging|production`，首次 start 另需 `--baseline <已恢复快照目录>`。Python 主函数 `main(argv: list[str]) -> int` 复用任务 2 的 common 和任务 3 的 CLI；各阶段加同一部署锁，不通过 shell eval/source 读取配置。prepare 将已校验的 `shared/.env` 保存为 0600 的 `shared/configs/<revision>.env`，同提交的不同配置拒绝覆盖；之后只读取版本副本。`shared/deployment.json` 记录尝试、已验证基线、当前/上一 release、对应配置摘要和非秘密结果；`current` 仅在 production 健康核验后指向成功 release。

- [ ] **1. 写部署失败测试。**`test_invalid_release_does_not_mutate` 验证非法 sha、路径穿越/符号链接、并发部署在 Docker 前被拒绝；`test_failed_health_keeps_current_release` 验证失败不前移 current；`test_rollback_keeps_database_and_cert_volumes` 验证未调用 down/down -v/迁移 down；`test_secrets_are_not_logged` 用包含密码的失败输出验证公开日志只含阶段及退出码。
- [ ] **2. 运行失败测试。**通过包装器运行 `python3 -m unittest discover -s ops/tests -p 'test_deploy.py' -v`，预期 deploy 尚未实现而失败。
- [ ] **3. 实现部署状态机。**prepare 串行构建三镜像并只启动 db；start 首次必须 inspect 与 baseline 一致，随后留 predeploy 备份、显式 `/app/bin/migrate --dir /app/db/migrations up`、启动 api/web 并等待健康；后续 start 必须确认已登记的预览库并先备份。activate 选择固定 CA/卷并启动 gateway，production 的普通 TLS/健康检查成功后才更新 current。失败有上一版本时恢复其应用镜像/配置并重新核验，无上一版本时停止应用写入及公网入口，保留 db；回退失败停止写入并报告具体阶段。
- [ ] **4. 实现备份 timer 与运行手册。**service 使用 ubuntu 和 `--sudo`，调用已确认 current 的 backup；timer 每日 `02:00:00 Asia/Shanghai`，`Persistent=true`，周日生成周副本。手册写明 env 创建、版本归档、源 locale、首次恢复、staging/production 切换、正常登录、备份下载、证书到期/续期检查及失败回退；首次无 previous 时不能承诺自动回退到旧服务。
- [ ] **5. 验证部署与回退。**部署单元测试退出 0；在临时测试 root 通过 Docker 调用模拟器执行准备、启动、健康失败及回退，检查确切命令、数据保留条件、current 和上一 release/config。不让测试修改真实 Compose 项目；真实 Docker/PG 恢复在任务 3、5 验证，实际部署及 systemd 验证在任务 6 完成。
- [ ] **6. 提交任务。**提交信息 `feat: 增加服务器部署回退与每日备份`。

## 任务 5：部署验收器、相关回归和代码交付

**文件：**新增 `ops/verify-deployment.py`、`ops/tests/test_verify.py`、`.github/workflows/deployment.yml`；更新 `docs/operations/evidence/server-preview/`。

**接口：**验收 CLI `python3 ops/verify-deployment.py --origin https://43.135.142.53 --out <新证据文件> --requests 40 --concurrency 2`，成功退出 0；每请求超时 5 秒，默认信任系统 CA。JSON schemaVersion=1，记录 revision、TLS/IP SAN/到期时间、重定向、公开页/资源、匿名权限/Cookie属性、请求数量/失败数/p50/p95/max；不保存 Cookie、CSRF、响应正文或凭据。`check(origin: str, requests: int, concurrency: int) -> dict` 供单元测试替换 HTTP 传输，负载仅 GET 公开只读路径。

- [ ] **1. 写验收器失败测试。**`test_untrusted_tls_fails` 拒绝未知 CA/错误 IP SAN，不使用跳过校验；`test_anonymous_draft_is_private` 核对 `/api/v1/content/drafts/904d550b-226b-48c0-98e0-610af339826c` 为 401，阅读入口返回现有 `Sign in to view your account` 提示和 `/login` 链接且无草稿正文；`test_missing_static_asset_fails` 拒绝 CSS/KaTeX 字体 404；`test_evidence_omits_session_material` 验证敏感响应内容不写证据。
- [ ] **2. 运行失败测试。**通过包装器运行 `python3 -m unittest discover -s ops/tests -p 'test_verify.py' -v`，预期尚未实现验收器而失败。
- [ ] **3. 实现验收器。**HTTP 必须跳转到固定 HTTPS origin，login/register 返回 200，匿名 auth/context 的生产预认证 Cookie `__Host-mm_preauth` 具备 Secure、HttpOnly、SameSite=Lax、Path=/ 且无 Domain。请求头沿用网站要求；私有验证与只读负载分开计数。首次 TLS 签发失败不能用测试证书替代验收。
- [ ] **4. 验证运维与既有回归。**包装运行全部 `ops/tests/test_*.py`，退出 0。分别执行 `node tools/verify/run.mjs --cwd frontend -- npm test -- --maxWorkers=1`、同 cwd 的 `npm run typecheck`、`npm run build`、`npm run api:generate`（生成文件无差异）。Go 执行 `node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/config ./internal/httpapi ./cmd/server -timeout 5m -count=1`；集成测试的 TEST_DATABASE_URL 仅指向独立的 `math_master_test_*` 测试库并使用既有随机隔离机制。包装运行 `node --test tools/verify/*.test.mjs tools/content-ingest/*.test.mjs` 的现有 13 项文件；按原 workflow 构建 `bin/e2e-harness`，分别包装 frontend 的 `npm run e2e -- auth.spec.ts auth-security.spec.ts` 和 `npm run e2e -- content-authoring.spec.ts`，覆盖 desktop/mobile，前后确认 R1 未被触及。
- [ ] **5. 增加独立部署 CI。**新 workflow 执行 Python 运维单元测试、镜像构建/烟测和随机隔离 PG 恢复检查，工作量拆成每命令不超过 9 分钟的 job；不改原 frontend/backend workflow 或基线。核对 standalone 的运行时认证/私有 SVG，验证服务端文件没有依赖本机绝对路径。
- [ ] **6. 审查整分支。**使用 requesting-code-review 做一次独立整分支审查；修复阻塞/重要问题并重跑受影响检查，保存无秘密结论。
- [ ] **7. 提交并交付 PR。**提交信息 `test: 验证服务器预览部署与既有功能`。SSH 推送分支，创建中文 PR 并 attach_artifact；核对实际 head 的新旧 CI 全部成功及无冲突，取得具体合并授权后 squash 合并并记录确切部署提交。

## 任务 6：真实服务器上线及数据/运维验收

**文件：**更新中文运行手册、开发路线图和无秘密 `docs/operations/evidence/server-preview/rollout.json`；实际配置/数据/日志只存本机受限目录和服务器 shared/backups。

**接口：**输入任务 5 的确切部署提交及任务 1—5 的已通过结果，使用已知主机指纹的 SSH 加密传输 Git 归档和新备份。最终输出实际 HTTPS 入口和预览验收记录；`previewAccepted=true` 只能在下面所有必要项完成后设置，正式内容验收 `formalContentAccepted=false`。

- [ ] **1. 复核实施现场。**确认最新部署提交、服务器 80/443 及磁盘/内存、本机 R1 的实际运行版本与当前六草稿状态；源为 `math-master-r1-review-db` 的 `math_master_review_r1`，只进行只读导出，不读取过期管理员密码或用户会话。
- [ ] **2. 准备服务器运行环境。**仅安装 Docker/Compose 所需包并启动 Docker，建立独立目录、新的随机数据库密码及受限 env；不升级无关软件、不改 SSH 防护链或已有 Netdata。Git 归档传入准确 release，记录包摘要；prepare 退出 0，source locale 与新 db 一致。
- [ ] **3. 创建和传输新快照。**调用 capture，以同一 snapshot 的实际六工作区 revision/status、yyl1212 角色、素材和全表摘要为准；通过 SSH 传输并校验 SHA256SUMS。预期基线为 30 知识点/30 单元/9 SVG，题库 450 固定题/24 模板/30 蓝图；有新变动时记录新基线，不伪造旧计数。全量 dump 包含学习、审核和反馈历史。
- [ ] **4. 完成恢复演练与真实恢复。**先在服务器调用 restore-drill，实际恢复和 inspect 全通过；随后 restore 到空预览库。启动应用前再次 inspect，角色、工作区、素材及所有表摘要一致，`publication_heads/question_heads` 保持源状态；管理员不重新初始化。失败不进入 start，也不修改本机库。
- [ ] **5. 启动内部应用。**start 使用准确 baseline，显式向上迁移成功，api/web 健康；记录实际镜像和迁移版本、内存及重启次数。
- [ ] **6. 验证 IP 证书与外部入口。**activate staging 完成 ACME 挑战；确认云侧 80/443 入站可达，若被阻断只定位具体规则处理。切换独立 production 证书卷，签发 IP SAN 证书，用普通 TLS 客户端完成自动验收器；不安装全局测试 CA 或忽略证书错误。另从外部确认 3000/8080/5432/Caddy 管理端口没有应用服务暴露。
- [ ] **7. 核对正常登录和私有阅读。**打开真实 HTTPS 登录页，由用户用当前密码正常登录 yyl1212；核对 learner/editor/admin、未勾选 reviewer，30 个知识点切换、标题搜索、revision/草稿标识及 9 SVG 阅读。只记录通过项，不提取 Cookie；如登录待用户完成，入口可交付但完整预览验收仍为 pending。
- [ ] **8. 验证重启、备份与站外副本。**服务器 `systemd-analyze verify` 通过后启用备份 timer、实际运行一次并检查下次执行时间；服务器首个备份再做隔离恢复，复制到本机受限站外目录并核对摘要。重启四服务后验证数据、健康与同一可信证书保留；记录续期配置、160 小时左右证书的实际 notAfter 及人工复查命令，不声称已经等待过一次完整自动续期。
- [ ] **9. 完成有界负载与交付。**40 次只读请求、并发 2，报告实际失败数/延迟、容器峰值内存和数据库连接；失败检查修复后仅重跑受影响项。全部必要条件满足才设置 previewAccepted=true，更新实际入口、运行手册及路线图，通过独立证据 PR 保存无秘密结果；公开数学内容数量与正式验收按真实状态报告。

## 计划自审结论

方案的首次交付、四服务架构、文件/兼容性范围、数据迁移、TLS、备份恢复、回退、测试和运行验收分别落实到任务 1—6；五个重点风险均有所属测试或现场验证。镜像、Compose、快照、部署与验收器接口名称一致；六任务各自有检查结果，没有未定义的执行组件。未发现书面计划阻塞；公网证书、构建资源和正常登录仍需现场验证，预检不替代运行证据。

计划批准后先执行任务 1—5，再进入服务器实际安装、复制与公网激活。此文不构成已经上线或已经完成数学审核的声明。
