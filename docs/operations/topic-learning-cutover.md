# 主题学习切换与兼容恢复

本手册交付维护流程，开发验收只操作随机隔离库。正式迁入、数学批准、切换和部署须由操作者按对应授权执行。服务启动不会自动迁移或切换。

## 顺序与前提

1. 记录当前Git提交、镜像revision、数据库迁移版本、公开知识及分类配对指针。
2. 使用原`ops/database-snapshot.py capture`生成一致性备份，并按原恢复演练验证可恢复。保留700备份目录、600属主文件`database.dump`、`manifest.json`、`SHA256SUMS`。
3. 在维护隔离下显式应用00010—00012；原00001—00009、原冻结数学、来源、答案和成绩不修改。安装新结构不会改变体验模式。
4. 用可信发布镜像安装分类批次并通过正常角色完成准确主题归属、独立审核和配对发布。空知识发布允许保持`knowledgeHead=null`，分类发布必须有效。
5. 有限执行迁入，保存每次JSON报告及最后`batchId`。每批1—50事件、每命令1—10批，重跑来源链接去重，`done=false`表示仍有工作，不得当作全库完成。
6. 执行`inspect`与`verify`。结构、pair、迁入、历史保护及可信编译版本都应满足，未映射旧事件必须为0。
7. 显式`activate`，核对实际模式、唯一审计ID和原日期；再检查健康及全部保留功能，最后开放流量。

## 有限命令

`DATABASE_URL`仅使用既有受保护环境/Compose配置，不作为命令行参数，也不输出至日志。生产镜像通过原`VCS_REF`把Git SHA写入编译元数据；请求中的`code-sha`不能替代可信版本。未提供或不匹配时拒绝切换。

```sh
/app/bin/topic-learning-maintenance inspect
/app/bin/topic-learning-maintenance migrate --batches=5 --limit=50
/app/bin/topic-learning-maintenance verify
```

迁入只取旧`started/completed`，保留原来源ID、知识版本和数据库时间。检测通过、诊断和解锁不能生成新完成。已有新状态、最近阅读、回执或笔记优先，迁入只补历史。旧事件当时没有分类则分类指针保持null。

切换必须提供核对过的完整参数：

```sh
/app/bin/topic-learning-maintenance activate \
  --expected-pair='{"knowledgeHead":null,"taxonomyHead":"<准确UUID>","taxonomyVersionId":"<准确64位SHA>"}' \
  --code-sha='<当前发布的40位GitSHA>' \
  --migration-batch='<inspect核对的最后UUID>' \
  --reason='本次切换的明确维护理由' \
  --backup-record='/已验证私有备份目录'
```

备份目录应以只读挂载供维护进程访问，进程UID须与备份属主一致；使用Compose原私有配置时按实际属主选择维护命令的`--user`。校验包括PGDMP、数据库名称、迁移/源码身份、创建时间、dump摘要和SHA256SUMS，只把manifest摘要写入切换审计，不复制备份内容。

旧交互写先持配置行共享锁，切换先持独占锁再按原全局顺序。切换在锁内再次检查未迁入事件；早先`done=true`之后新增旧事件仍会拒绝切换，继续迁入后重新inspect。并发调用只有一次实际转换，后续返回原审计，不更新日期。

## 健康和恢复

`/healthz`仍表示进程存活，`/readyz`核对数据库及新结构；新增`topic`仅含taxonomy、study、retirement、schemaReady、topicsMode五个安全布尔，不含用户内容、笔记或来源路径。

恢复应用时先隔离gateway/web，仅启动目标api检查实际编译能力。数据库要求新结构时旧程序不能恢复私有流量，topics模式必须有retirement能力；缺失/损坏结构保持维护隔离。永久能力标记也纳入最低兼容版本，不能利用空Down留下的版本号恢复旧写。

仅恢复兼容二进制，保留数据库及证书卷、原snapshot/migration核验。不要执行真实Down、清学习数据、删除永久标记或把旧active改成用户放弃。发生不兼容或未知版本时记录manual-recovery-required，先修复结构或选择兼容发布，再重新核验。
