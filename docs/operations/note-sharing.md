# 笔记优化与审批分享

私人笔记继续按账户与知识点保存。作者可将已保存的内容申请分享，管理员明确批准后进入分享页；未提交的私人正文不向管理员开放。本阶段不包含复习排期、默写、自由笔记本或附件。

## 使用流程

1. 在知识阅读页打开“我的笔记”，使用编辑、格式按钮及公式预览；点击保存后才可申请分享。离开时有未保存内容会提示，取消后保留草稿。
2. 在“我的学习”的“我的笔记”中按知识、主题和分享状态查找本人笔记。关联知识不可用时保留原笔记，禁止新编辑。
3. 点击“申请分享”，填写标题并检查已保存正文。提交后冻结当次内容，状态为待审批；待审批和已分享期间禁止修改或删除原笔记。
4. 管理员进入 `/admin/note-shares`，阅读冻结内容，明确批准，或填写至少 10 个字符的理由退回。已分享内容可以填写理由下架。管理员本人提交的笔记同样需要明确批准。
5. 作者先撤回，再修改、保存并重新申请。每次重审生成新的提交标识，旧分享链接失效。退回或下架后的理由在本人笔记中可见。

`/notes` 对访客只展示最新 10 条当前有效的已批准分享。第 11 条及被挤出窗口的直接链接也不可匿名访问。登录用户可以搜索、分页浏览全部有效分享。知识撤回会隐藏关联分享；知识恢复只恢复仍处于已分享状态的内容。

## 数据与接口

迁移 `00014_note_sharing.sql` 新增 `note_share_submissions`、`note_share_heads`、`note_share_events`、`note_share_receipts`。提交正文、事件和幂等回执不可更新或删除；head 保存当前状态与序列。原私人笔记表、学习状态和已完成事实保持原契约。

公开接口为 `GET /api/v3/shared-notes/latest`、`GET /api/v3/shared-notes/public/{submissionId}`；登录全部列表与详情使用独立路径。所有分享响应禁止缓存，私人响应带 `private, no-store`。新 `GET /api/v3/note-sharing/health` 只返回 `capability` 与 `schemaReady` 两个布尔字段。

请求结果未确认时，界面仅允许以原幂等键确认同一请求。原保存重试返回当前本人笔记；分享旧回执不会复活已撤回内容。过期审批序列返回冲突，管理员需要检查刷新后的状态。

## 发布和回退

代码审查通过后，正式发布仍需执行现有备份与发布流程。发布前保存一致性数据库备份，并在隔离恢复环境比较全部表摘要、工作区与序列；原私人笔记及学习记录也必须核对。不得在已有分享数据时执行迁移 Down。

Go、Next 与公网入口必须成组发布，并明确声明 `noteSharing` 能力及迁移上限 14。部署检查顺序为 Go 启动后探测分享结构，Next 启动后探测代理，切换可信 HTTPS 入口后再次探测；任一检查为 404、结构损坏或能力缺失都拒绝继续。原 `/readyz` 返回形状不变。

恢复工具通过私有临时 SQL 设置会话级 `pg_catalog, public` 搜索路径，以支持原嵌套约束函数；只调整经校验的脚本首部，正文、函数定义和原备份摘要不变。执行保持单事务并遇错停止，提取和执行共享 530 秒预算。需要为两份临时 SQL 预留空间，临时文件随流程清理。脚本输出与事务模式参见 [PostgreSQL17 pg_restore](https://www.postgresql.org/docs/17/app-pgrestore.html) 和 [psql](https://www.postgresql.org/docs/17/app-psql.html)。

迁移 14 的数据库不能回退到仅声明迁移 13 的旧二进制。部署工具仍检查目标发布与数据库迁移一致性；恢复备份与回退代码属于独立运维操作。不得通过关闭结构检查或删除分享表绕过门禁。

## 验证入口

使用专用 `TEST_DATABASE_URL`，每条命令最多 540 秒，Go 设置 `CGO_ENABLED=0` 和 `-timeout 5m`。浏览器测试复用随机数据库和真实 Go/Next；不批准或导入生产用户笔记。

```sh
node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/notesharing ./internal/store ./internal/httpapi -run '^TestSharing' -timeout 5m -count=1
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- note-sharing.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- note-sharing-security.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- note-editor-experience.spec.ts
node tools/verify/run.mjs --cwd frontend -- npm run e2e -- ui-language-notes.spec.ts
node tools/verify/run.mjs -- python3 -m unittest discover -s ops/tests
```
