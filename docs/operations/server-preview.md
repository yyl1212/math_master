# 服务器预览版运行手册

日期：2026-10-05。目标入口：`https://43.135.142.53`。本手册对应 P7a；网站已于北京时间 2026-10-05 18:36 通过外部可信 HTTPS 检查开放，运行提交 `610aefff8da3358194a6ec8693f168c0f0cb5a39`。实测结果见 [rollout.json](evidence/server-preview/rollout.json)。正常登录和工作区恢复已由本人确认；知识点逐条切换、搜索及全部 SVG 视觉验收仍待补充，因此完整预览验收保持 pending。

本阶段复制现有账号与六个私有草稿，保留密码、权限、作者关系和未审阅状态。公开课程仍按独立数学复核与既有发布流程推进；不能把网站可访问计为内容审核完成。

## 1. 目录与请求路径

```mermaid
flowchart LR
    User[浏览器] -->|HTTPS 80/443| Gateway[Caddy]
    Gateway --> Web[Next.js 页面及同源代理]
    Web --> API[Go 权限与业务 API]
    API --> DB[(PostgreSQL)]
    DB --> Backup[私有备份与隔离恢复]
```

- `/opt/math_master/releases/<40位提交号>/`：Git 归档、生产镜像构建输入和运维工具。
- `/opt/math_master/shared/.env`：下一次 prepare 的私有配置输入，0600。
- `/opt/math_master/shared/configs/<提交号>.env`：该版本的冻结配置，0600，不覆盖不同内容。
- `/opt/math_master/shared/deployment.json`：当前/上一提交、实际运行提交、恢复基线及阶段，0600。
- `/opt/math_master/current`：成功 production 检查后的相对符号链接；失败不会前移。
- `/opt/math_master/backups/{daily,weekly,predeploy}/`：0700，文件 0600，目录发布禁止覆盖。

仅网关映射公网 TCP 80/443；Go、Next.js、数据库和 Caddy 管理接口不映射宿主机。测试控制程序不在生产镜像中。原有 SSH 防护和其他服务不在本次修改范围。

## 2. 准备依赖与代码归档

先完成代码审查、原有回归、新部署 CI 和具体 PR 合并授权。部署使用最终合并提交，不使用未提交的本机验证镜像。

Ubuntu 安装步骤：仅安装 Docker/Compose 所需依赖，不执行整机升级。先检查实际软件版本、已有服务和 80/443 占用情况，再由 ubuntu 执行对应 sudo 操作。目录统一由 ubuntu 持有，shared/backups 为 0700；不添加 docker 组权限。

```bash
sudo apt-get update
sudo apt-get install --no-install-recommends docker.io docker-compose-v2 docker-buildx
sudo systemctl enable --now docker
sudo install -d -m 0700 -o ubuntu -g ubuntu /opt/math_master
sudo install -d -m 0700 -o ubuntu -g ubuntu /opt/math_master/releases /opt/math_master/shared /opt/math_master/backups
```

本机用 `git archive <准确提交>` 生成归档，经已核验主机指纹的 SSH/SCP 加密传输；SHA-256 在两端一致后解包到新的 release。服务器不保存 GitHub token、本机服务器密码、R1 临时管理员密码或浏览器会话。已存在的 release 不盲目覆盖。

## 3. 私有配置

从 `ops/.env.example` 创建 `shared/.env` 并设 0600。数据库/用户名为 `math_master_preview`，新数据库密码由 `python3` 的 `secrets.token_hex(32)` 生成并直接写入私有文件；不要输出到终端、命令参数或 Git。

`DB_LC_COLLATE` 和 `DB_LC_CTYPE` 使用源快照实际值。源库须为 PostgreSQL 17.11、UTF8、libc locale provider；编码/locale/版本不一致时停止恢复。生产 Go/Next 的 APP_ENV、内部监听和内部 API 地址由 Compose 固定；origin 为 `https://43.135.142.53`。

文件只接受六个键、重复/未知键拒绝，禁止引号、命令替换等可执行语法。配置不是 shell 脚本，不使用 source/eval。每个版本有自己的配置副本，回退不会错误地套用下一版本配置。

## 4. 创建新备份及真实恢复

本机 R1 数据库保持运行；捕获时只读。只复制数据库，不重导入仍在更新的原始 Knowledge_JSON，也不运行 admin-init。

```bash
node tools/verify/run.mjs -- python3 ops/database-snapshot.py capture \
  --container math-master-r1-review-db \
  --database math_master_review_r1 --user math_master_review \
  --out "$MATH_LOCAL_PRIVATE_BACKUP" --revision "$MATH_SOURCE_API_REVISION"
```

`MATH_SOURCE_API_REVISION` 为实际正在运行的本机 Go 版本，前端实际版本另记在私有交接记录。备份包含全部公开 schema 表及历史；dump 与表清单使用同一 exported snapshot。清单记录数据和迁移版本、工作区 revision/status、素材摘要、角色、head 数量。备份目录、清单和 dump 均为私有资料。

传输 `database.dump`、`manifest.json`、`SHA256SUMS`，在服务器检查权限和摘要。先做隔离演练，随后才恢复到空预览库：

```bash
/opt/math_master/releases/"$MATH_PREVIEW_REVISION"/ops/restore-drill.sh \
  --backup "$MATH_SERVER_PRIVATE_BACKUP" --sudo
```

演练创建随机 PostgreSQL 容器/卷、无宿主端口、无外部网络。实际恢复采用 `--no-owner --no-acl --single-transaction --exit-on-error`，逐表摘要和序列检查通过才算成功；清理限定本次 UUID 及标签。损坏备份、其他项目容器和非空目标均拒绝。

## 5. 部署四个阶段

```bash
MATH_PREVIEW_ROOT=/opt/math_master
# MATH_PREVIEW_REVISION 设为最终合并提交的完整 SHA。
"$MATH_PREVIEW_ROOT/releases/$MATH_PREVIEW_REVISION/ops/deploy.sh" prepare \
  --root "$MATH_PREVIEW_ROOT" --revision "$MATH_PREVIEW_REVISION" --sudo
```

prepare 冻结配置、串行构建 api/web/gateway，并仅启动 db。PostgreSQL 就绪检查使用 TCP，等待初始化临时服务器退出及最终服务启动。

从实际 Compose 获取准确 db 容器 ID，再调用 `database-snapshot.py restore --container <id> --database math_master_preview --user math_master_preview --backup <目录> --sudo`。不得替换成 R1 容器或对已有数据使用 pg_restore --clean。目标恢复和 inspect 成功后运行：

```bash
"$MATH_PREVIEW_ROOT/releases/$MATH_PREVIEW_REVISION/ops/deploy.sh" start \
  --root "$MATH_PREVIEW_ROOT" --revision "$MATH_PREVIEW_REVISION" \
  --baseline "$MATH_SERVER_PRIVATE_BACKUP" --sudo
"$MATH_PREVIEW_ROOT/releases/$MATH_PREVIEW_REVISION/ops/deploy.sh" activate \
  --root "$MATH_PREVIEW_ROOT" --revision "$MATH_PREVIEW_REVISION" --acme staging --sudo
```

start 先核对恢复基线，关闭 gateway/web/api 的公网入口与写入，停写后保留发版前快照，再显式向上迁移和启动 api/web。更新存在短暂停机窗口，候选版本通过健康与可信 HTTPS 验证后恢复入口。首次快照的来源提交记录为原本机运行版本，Compose 使用候选版本配置；两个版本概念分开。此时 current 尚未指向新版本；首次失败后保留已核验的基线来源，修复故障后可重新 start。

staging 的证书为测试证书。查看该次网关日志中的证书获取成功结果，确认实际 80/443 入站挑战可达后再切换：

```bash
"$MATH_PREVIEW_ROOT/releases/$MATH_PREVIEW_REVISION/ops/deploy.sh" activate \
  --root "$MATH_PREVIEW_ROOT" --revision "$MATH_PREVIEW_REVISION" --acme production --sudo
```

公网 IP 客户端可能不发送 SNI；网关在全局配置 `default_sni {$PUBLIC_HOST}`，明确使用已配置的公网 IP 证书。配置依据为 [Caddy 官方 default_sni 文档](https://caddyserver.com/docs/caddyfile/options#default-sni)。实际生产验收继续使用系统 CA 与 IP SAN 校验；镜像测试的一次性证书仅由该测试上下文显式信任。

两个 CA 使用独立证书数据卷，切换时重建 gateway。可信 HTTPS、公开入口、静态资源、匿名权限和 Cookie 检查成功后才设置 current。不要用测试 CA 或忽略证书错误代替 production 验收。

## 6. 验收与正常登录

在普通外部客户端运行 `ops/verify-deployment.py --origin https://43.135.142.53 --out <新私有证据文件> --requests 40 --concurrency 2`，单请求 5 秒。检查 HTTP 跳转、可信 IP SAN/到期日、公开页、CSS/KaTeX 字体及匿名草稿 API 401。阅读页匿名状态沿用登录提示，不要求自动重定向。

用户在真实 HTTPS 登录页用 yyl1212 与当前密码登录。核对 learner/editor/admin、草稿状态与 revision、标题搜索、知识点切换及原创 SVG；操作人员不提取 Cookie 或重置密码。正常登录未确认时，完整预览验收保持 pending。

网站要求云侧入站允许 TCP 80/443（IPv4 来源 `0.0.0.0/0`）。2026-10-05 实际外部探测两端口均可达；主机 UFW 未启用，现有防护链只处理 SSH，因此本次无需新增防火墙规则。保留现有 SSH 防护。

另从外部验证 3000/8080/5432/2019 没有应用服务暴露，记录容器内存、重启次数、实际迁移版本、40 次只读请求的失败数及延迟。数据数量以此次新快照为准，不用旧截图替代。

## 7. 每日备份与站外副本

```bash
sudo install -m 0644 "$MATH_PREVIEW_ROOT/current/ops/math-master-backup.service" /etc/systemd/system/math-master-backup.service
sudo install -m 0644 "$MATH_PREVIEW_ROOT/current/ops/math-master-backup.timer" /etc/systemd/system/math-master-backup.timer
sudo systemd-analyze verify /etc/systemd/system/math-master-backup.service /etc/systemd/system/math-master-backup.timer
sudo systemctl daemon-reload
sudo systemctl enable --now math-master-backup.timer
sudo systemctl start math-master-backup.service
sudo systemctl list-timers math-master-backup.timer
```

每天北京时间 02:00 执行，补执行启用，周日按 Asia/Shanghai 判断周副本。保留最近 7 个有效日日期与 4 个有效周；predeploy 快照不自动轮换。损坏或未知文件不参与删除。

首次服务器备份再次做隔离恢复，之后经 SSH 复制到本机 0700 私有目录，并验证摘要；每次发版都复制站外副本。记录复制时间与实际位置。当前不是每日自动站外备份，独立存储与自动传输留待 P7b。

## 8. 证书续期、重启与回退

Caddy 自动管理显式 Let’s Encrypt `shortlived` 证书；公网 IP 证书寿命约 160 小时。保留 production 证书卷、外部 80/443 入站与 ACME 出站访问。用普通 TLS 检查记录实际 notAfter，检查网关错误和容器重启，重启四服务后再验证证书和数据保留。首次配置验收不等于已等待过完整续期周期。

### 重启次序及内容服务验收

本次直接并行重启四容器后，容器均 healthy，但内容接口返回 `503 CONTENT_NOT_CONFIGURED`。数据库表及数据完整；数据库就绪后重新启动 API，接口恢复 401，本人刷新工作区确认可用。应按以下次序执行，不能仅凭 healthy 判断内容服务已恢复：

1. 停止 gateway/web/api 的入口与写入。
2. 重启 db，并通过 `compose up -d --wait --wait-timeout 120 db` 等待真实数据库健康。
3. 使用该版本冻结配置启动 api/web 并等待健康，再启动 gateway；不要将 db 与 api 同时 restart。
4. 普通外部 TLS 客户端验证可信证书、页面、静态资源、安全 Cookie，匿名草稿 API 必须返回 401；503 表示内容服务未就绪，不表示匿名成功读取。
5. 本人刷新已登录工作区并读取草稿，确认 revision 和未审阅标识。会话、密码和原数据保留。

调用 Compose 时复用 `ops/common.py` 的 `compose(root, revision, args, sudo=True)`，选择 `shared/deployment.json` 中 current 的冻结配置，不 source 环境文件。整机重启后的内容模块自动恢复仍需后续兼容性方案和专门回归；若发生该 503，按上述次序恢复。当前手册不声称容器探针覆盖所有业务功能。

本次证书实际到期时间为北京时间 2026-10-12 08:30:46，Caddy production 证书卷已持久化并配置自动续期；尚未等待过一个完整续期周期。每日备份服务已实际执行成功，首个每日备份的服务器隔离恢复及本机站外副本 SHA 校验通过。

明确回退的上一版本完整 SHA，执行 `deploy.sh rollback --root /opt/math_master --revision <上一版本> --sudo`。先核对真实迁移版本与该 release 的迁移文件版本；相符后只换 api/web/gateway 镜像及对应配置，并验证健康/TLS。数据库卷、证书卷和用户数据保留。

手动与自动回退的健康/TLS失败都停止 gateway/web/api，并保存具体失败阶段。若结构不兼容、恢复不一致或回退仍失败，停止应用写入、保留备份与现场，单独制定从隔离恢复结果切换的方案。禁止真库迁移 down、compose down -v 或自动覆盖恢复。首次无旧服务时，失败结果为停止公网入口及写入，不能承诺退回不存在的版本。
