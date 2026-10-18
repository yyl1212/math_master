# 知识批量发布实施计划

> 执行者必须使用 `superpowers:executing-plans` 或用户选择后的 `superpowers:subagent-driven-development`，按任务复选框实施；具体执行方式由总计划审阅确定。

**Goal:** 管理员跨页勾选发布或一键发布整个知识库的未发布知识，准确查看结果，并在暂停、刷新和请求中断后恢复。

**Architecture:** 使用独立三表持久化操作、固定 ID/令牌清单及批次回执，复用原知识锁、内容校验和发布审计。页面串行请求短批次；已确定失败与结果未知分别处理，旧单条发布和数据源身份保持。

**Tech Stack:** 仓库现有 Go、PostgreSQL、Next.js / TypeScript、Zod、Vitest、Playwright，不引入常驻 worker。

**Spec:** [已确认设计](../specs/2026-10-19-query-pagination-bulk-publish-design.md)，消费[总计划](2026-10-19-query-pagination-bulk-publish.md)及[分页子计划](2026-10-19-query-pagination.md)的公共接口。

## Global Constraints

- 总计划约束全部适用；全量为整个库当时未发布且未删除的知识，后续新增不加入；选中每次最多 1000 条，创建输入最多 262144 字节。
- `batchSize` 为 1—20、默认 20；8 秒事务、1 秒锁等待、10 秒客户端截止保持；缩批为 20/10/5/2/1，最多连续 3 次临时重试。
- 00001—00015 原字节、三个既有 schema 摘要、私人学习及笔记事实保持；新增迁移 16 独立检查。
- Go 均 `CGO_ENABLED=0`，单命令最多 540 秒、Go 每组 `-timeout 5m`；10001 条容量和真实浏览器分组运行。

## Review Focus

总计划第 3—5 项分别在 B2/B3/B5、B3/B5、B1/B6 落实；重点核对未知提交、令牌变化、大正文、身份撤销与旧版本恢复。

## B1：发布模型、独立三表与迁移兼容

**Files:** 新增 `backend/internal/knowledgeadmin/publish_batch.go`、`publish_batch_test.go`；新增 `db/migrations/00016_knowledge_publish_batches.sql`；新增 `backend/internal/store/knowledge_publish_schema.go`、`knowledge_publish_schema_test.go`；精确修改 `backend/internal/store/study_review_schema_test.go`、`study_review_restore_test.go` 的版本 15 测试夹具目标，不修改原断言；修改 `backend/internal/e2etest/{auth_fixture.go,fixture_storage_reset_test.go}`。更新 `docs/superpowers/specs/2026-10-19-query-pagination-bulk-publish-design.md` 的实施状态。

**Interfaces:** 新增 `PublishMode=selected|all`、`PublishItemState=pending|published|skipped|failed`；`PublishSelection{ID,EditToken string}`；`PublishCreateInput{Mode PublishMode;Items []PublishSelection}`，all 禁止携带 items，selected 要求 1—1000 个唯一合法对。

`PublishJob{ActorID,ID string;Mode PublishMode;Sequence int64;Total,Pending,Published,Skipped,Failed int64;CreatedAt time.Time}`，JSON 序列从 1 开始，计数非负且总数相等。状态通过计数派生：pending>0 为 pending；否则 failed>0 为 partial；否则 completed，不另存可能矛盾的状态字段。`PublishItem{ID,ExternalID,Title,TitleZH string;State PublishItemState;ErrorCode,ErrorPath string}`，标题最多 128 码点，错误路径最多 512 字节；响应不含正文或 editToken。

`PublishStepInput{ExpectedSequence int64;BatchSize *int}`（省略为 nil，显式 0 拒绝，归一化默认20后计算请求摘要）；`PublishStepReceipt{ActorID,JobID string;Sequence int64;Items []PublishItem;Pending,Published,Skipped,Failed,Total int64}`。序列和计数最多 9007199254740991，条目回执最多 20 项；单项及列表响应最多 1 MiB。

独立 `PublishRepository` 提供 B2/B3 的五个签名，不扩旧 `knowledgeadmin.Repository`。三表按 Spec 名称；jobs 的 `(owner_user_id,key)` 唯一，items 的 `(job_id,knowledge_id)` 唯一；receipt 的 owner 与 job 绑定，动作/键/资源/摘要唯一绑定。清单成员、期望令牌及回执不可变；pending 才能转成终态，成功终态不复活。job 序列与计数同批更新，禁止仅改计数伪造完成。

- [ ] 写 `TestPublishInputExactBounds`：selected=0/1001、重复 ID、错误令牌、all+items 拒绝；1000 个正确条目通过；省略 batchSize 默认20，显式0或21拒绝；不安全序列拒绝。
- [ ] 写 `TestPublishSchema16Constraints`、`TestPublishSchema16KeepsOriginal`：三表、复合所有权、不可变快照/回执、非法状态及终态回退拒绝；原 managed/sharing/review 摘要分别等于既有常量。
- [ ] 先运行 RED，再实现模型、迁移及 `publishSchemaIntegrity(ctx,*sql.Tx)error`；新摘要从规范 search_path 的真实迁移采集后固定，删除约束或禁用触发器必须使新发布能力失败。
- [ ] 写 `TestPublishMigrationMissingKeepsOriginal`、`TestPublishMigrationEmptyDownAndUsedGuard`：无 16 时原读取/单条发布/复习可用；三表均空才允许 Down；存在任何操作或回执时拒绝 Down，常规代码回退保留 16。
- [ ] 将 reviewSchemaDB 及恢复测试需要“准确版本 15”的升级改为既有 Goose `UpToContext(...,15)`。运行原 `TestReviewEmptyMigrationDown` 仍要求 v14、原恢复脚本仍只允许增加迁移 15；新增 `TestPublishMigration16AfterReviewRestore` 验证 15→16，不把原 `upgrade15` 放宽为任意未来版本。
- [ ] 在原 resetFixtureStorage 的同一条 TRUNCATE 中加入三个新表，再跑 `TestFixture` 原重置原子性/故障/轮次测试及新增“先有发布操作再重置”场景；保持原一次存储调用断言，不拆成多个独立清理事务。
- [ ] GREEN 后提交：`feat: 定义知识发布操作与独立迁移16`。

Run：runner 下分组 `go test ./internal/knowledgeadmin -run '^TestPublishInput' -timeout 5m -count=1`、`go test ./internal/store -run '^TestPublish(Schema|Migration)' -timeout 5m -count=1`、原 review schema/down 组；真实恢复单独运行。Expected：无 skip、三套原摘要相同、旧迁移无 diff。

## B2：原子固定清单、本人进度与创建幂等

**Files:** 新增 `backend/internal/store/knowledge_publish_jobs.go`、`knowledge_publish_jobs_test.go`；修改 B1 的 `backend/internal/knowledgeadmin/publish_batch.go`；新增 `backend/internal/store/knowledge_publish_fixture_test.go`，复用现有知识管理员隔离夹具。

**Interfaces:** `CreatePublishJob(ctx context.Context,a knowledgeadmin.Access,in PublishCreateInput)(PublishJob,error)`；`ListPublishJobs(ctx,a,query PublishJobQuery)(pagination.ActorPage[PublishJob],error)`，`PublishJobQuery{CreateKey string;Limit int;Offset int64}`、默认20/最大100；`ReadPublishJob(ctx,a,id string,query PublishItemQuery)(PublishJobDetail,error)`，`PublishItemQuery{State PublishItemState;Limit int;Offset int64}`、默认20/最大100；`PublishJobDetail{Job PublishJob;pagination.Page[PublishItem]}`。这些方法列入 PublishRepository；输入 access 带原身份、CSRF/请求 ID 与写入幂等键。

创建回执以 mode 加排序后的唯一选中 ID/令牌计算摘要。幂等重放在重新读当前知识或序列前发生。all 在原知识写锁内单次 `INSERT ... SELECT` 固定未发布/未删除 ID、令牌和稳定顺序；selected 记录已经不存在的 ID 为 failed，不以知识 FK 静默丢弃。清单不存正文。

- [ ] 写 `TestPublishCreateSelectionAndAllSnapshot`：8 条合成记录中 5 个未发布正常、1 已发布、1 删除、1 不存在选中；all 的 `Total==5`；创建后新增和改排序不改变成员集合，selected 不存在项保留失败结果。
- [ ] 写 `TestPublishCreateReplayAndOwner`：同键同输入返回创建回执，同键换模式/令牌冲突；创建人以外管理员也不能读该操作；角色撤销/强制改密拒绝；all=0 时返回完成及序列1。
- [ ] 写 `TestPublishCreateInterruptedResponseLookup`：创建已提交但客户端无响应后，按原创建键查询唯一同一 job；不能把重新查询时新增的记录吸收进原清单。
- [ ] 运行 RED；实现写锁下的原子清单、回执和本人分页查询，身份/权限提交前复核。计数查询不返回来源原始绑定、正文和令牌；创建超时必须没有半份 job/items。
- [ ] GREEN 后提交：`feat: 固定批量发布清单并持久化操作进度`。

Run：`node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestPublishCreate' -timeout 5m -count=1`。Expected：清单成员精确、幂等/所有权正确，失败事务无半写。

## B3：原子短批次、冲突与提交未知

**Files:** 新增 `backend/internal/store/knowledge_publish_step.go`、`knowledge_publish_step_test.go`；精确修改 `backend/internal/store/knowledge_admin_write.go`；完善 `backend/internal/knowledgeadmin/publish_batch.go` 的错误类型；更新 `backend/internal/store/knowledge_admin_write_test.go`。

**Interfaces:** `StepPublishJob(ctx context.Context,a knowledgeadmin.Access,id string,in PublishStepInput)(PublishStepReceipt,error)`；`PublishRepository` 还包含 `ReadPublishJobReceipt(ctx,a,id,key string)(*PublishStepReceipt,error)`，供恢复确认原批次。`ErrPublishRolledBack` 仅在从未尝试 COMMIT 且确认回滚后成立；提交结果未知继续 `REQUEST_INTERRUPTED`，普通旧写错误不改名。

抽出内部 `publishManagedKnowledgeTx(ctx context.Context,tx *sql.Tx,u auth.User,id,expectedToken,jobID string)(knowledgeadmin.Knowledge,error)`，只承载原单条发布的令牌、内容校验、状态更新及原审计；不自行开启事务、不存整份批量正文回执。原 `SetManagedKnowledgeState` 的 publish 分支调用它并继续存原单条回执，其他 action 保持；jobID 为空时原审计字段保持，不能增加空 jobID 字段。batch 在调用前把已发布记录标为 skipped；不存在/删除/过期令牌/无效内容标为 failed。

- [ ] 写 `TestPublishStepMixedResultsAndOriginalAPI`：41 条中成功、已发布跳过、删除、不存在、令牌变动及格式错误；`Total==Pending+Published+Skipped+Failed`，只有成功产生一次 publish 事件及新令牌；旧单条回执结构和字节含义不变。
- [ ] 写 `TestPublishStepReplayAndTabs`：同键重放返回原结果，即使后续序列已前进；同键不同 batchSize 拒绝；两个标签同 expectedSequence 至多一个推进，另一个刷新；不同 job 重叠项只成功一次，其余 skipped。
- [ ] 写 `TestPublishStepRevocationAndRollback`：身份撤销先提交则拒绝；锁忙、校验期间截止、SQL 错误时知识、条目、计数、序列、回执同进同退。确定的内容校验失败不使整批其他合法项失败。
- [ ] 写 `TestPublishStepCommitUnknown`、`TestPublishStepLargeContent`：COMMIT 已发送后的故障不能返回 ErrPublishRolledBack；原键确认找到回执或执行尚未提交的原批；正文逐项释放、响应≤1048576字节；批量20超时时确认回滚后可用新键+原序列缩至1。
- [ ] 先 RED；按原身份锁→知识锁→job行锁→稳定知识 ID 锁顺序实现。先查回执，再做 expectedSequence；同一事务更新知识、条目、计数、序列和回执。数据库事务中止后立即退出，不继续假装处理下个条目。
- [ ] GREEN 后提交：`feat: 原子执行知识发布批次并支持幂等恢复`。

Run：runner 下分别 `go test ./internal/store -run '^TestPublishStep(Mixed|Replay|Revocation|Commit)' -timeout 5m -count=1`、`go test ./internal/store -run '^TestPublishStepLarge' -timeout 5m -count=1` 及旧单条写入组。Expected：无部分提交、无重复事件、不把未知结果冒充回滚。

## B4：发布路由、契约和可靠客户端

**Files:** 新增 `backend/internal/httpapi/knowledge_publish_routes.go`、`knowledge_publish_test.go`；修改 `backend/internal/httpapi/knowledge_admin_routes.go`；修改 `api/openapi.yaml`、生成 `frontend/src/lib/api/generated.d.ts`；新增 `frontend/src/lib/knowledge-admin/{publish-batch,publish-batch.test,publish-batch-schemas,publish-batch-schemas.test}.ts`；修改 `frontend/src/lib/knowledge-admin/{client,schemas,types}.ts`、`frontend/src/lib/api/knowledge-admin-proxy.ts` 及对应已有测试。

**Interfaces:** PublishRepository 从实际 `o.Service.Repository` type assert 获取，未安装迁移/能力返回新功能不可用，不改旧 Repository 构造与 mock。新增四组路径按 Spec 分发，`publish-jobs` 必须在通用知识 ID 分支前匹配；GET detail 也支持 `receiptKey` 精确确认某批回执，不能同时要求条目页与回执输出；该模式输出 `PublishReceiptLookup{ActorID,JobID,Key string;Receipt *PublishStepReceipt}`。非管理员403；不存在或属于其他管理员的操作404。

前端 `createPublishJob(actorId,input,key,signal):Promise<PublishJob>`、`listPublishJobs(actorId,query,signal):Promise<ActorPage<PublishJob>>`、`readPublishJob(actorId,id,query,signal):Promise<PublishJobDetail>`、`readPublishReceipt(actorId,id,key,signal):Promise<PublishStepReceipt|null>`、`stepPublishJob(actorId,id,input,key,signal):Promise<PublishStepReceipt>`；类型来自生成契约。新增 transport 接受外部 AbortSignal，并以单一10秒截止覆盖鉴权、fetch、流读取与解析；原 managedRequest 原合同保持，不扩大旧错误接受范围。

- [ ] 写 `TestKnowledgePublishHTTPPermissionsAndBounds`：非管理员、跨源、CSRF、重复 header/query、未知字段、262144/262145字节、1000/1001条、非法令牌、batch21和序列溢出按准确状态拒绝；未配置16只影响新路由。
- [ ] 写 `publish batch schemas bind actor job sequence`、`publish transport bounded stream and cancellation`：拒绝 actor/job不符、计数不一致、超过20项回执、未知字段、超过1MiB及慢流；旧路径不能误走新 schema。
- [ ] 先 RED；实现代理新路径及独立大小限制：create≤256KiB、step≤8192字节、响应≤1MiB；`REQUEST_INTERRUPTED` 和 `PUBLISH_BATCH_ROLLED_BACK` 严格区分。回执精确查询只允许创建人读取。
- [ ] 生成类型，运行 GREEN 及旧 proxy/client 测试，提交：`feat: 接入受限批量发布接口与确认客户端`。

Run：runner 下 `go test ./internal/httpapi -run '^TestKnowledgePublish' -timeout 5m -count=1`；前端 `npm run api:generate`、新 publish-batch/schema tests、原 knowledge-admin client/proxy tests、`npm run typecheck`。Expected：大小、身份、未知结果和原接口均通过。

## B5：跨页勾选、串行调度、暂停与恢复

**Files:** 新增 `frontend/src/features/knowledge-admin/{publish-batch.tsx,publish-batch.test.tsx,publish-selection.ts,publish-selection.test.ts}`；修改 `frontend/src/features/knowledge-admin/list.tsx`、`frontend/src/features/knowledge-admin/knowledge-admin.test.tsx`、`frontend/src/app/admin/knowledge/page.tsx`、`frontend/src/lib/i18n/messages/{zh-CN,en}.ts`、`frontend/src/styles/content.module.css`；新增 `frontend/src/lib/knowledge-admin/{publish-runner,publish-runner.test}.ts`。

**Interfaces:** 选择模型 `setSelected(current:ReadonlyMap<string,SelectedKnowledge>,item:SelectedKnowledge,checked:boolean):Map<string,SelectedKnowledge>`，`SelectedKnowledge={id,editToken,title,titleZh}`；当前页全选只处理可选项，筛选提交清空，跨页保留。`PublishBatch` 组件接收 `actorId,selection,onStarted,onSettled`；job progress 单独渲染，不因 router.refresh 卸载未确认命令。

管理员页面将 `contentPageAccess` 的 `access.user.id` 作为 actorId 传入 `KnowledgeAdminList`，不接受 URL 提供的 actor；浏览器 transport 在读取与写入前后核对当前会话身份。创建回执只证明该操作已建立，runner 总是先读取当前 job 再按最新序列推进，不能将重放的序列1当成当前状态。

runner `runPublishJob({actorId,jobId,signal,onProgress,onPending,onConfirmed},transport):Promise<PublishJob>`；仅一个 step 在途；停止发下一批与取消未知请求是不同状态。pending 为冻结的 `{actorId,jobId,input,key}`，禁止更换原输入；每份回执核对 actor、job、序列、计数后再读当前进度。sessionStorage 只保存创建键和 jobID，key 为 `math-master:publish:<actorId>`，不保存正文、令牌或未提交选择；恢复状态由服务端重读。

- [ ] 写 `selection across pages and filter reset`：页1选2、页2选1后 selected.size===3；回页1为部分选中；全选排除发布/删除条目，超过1000提示；新筛选清空且有提示。
- [ ] 写 `runner serial and pending confirmation`：41项按20/20/1执行，最多一请求在途；丢响应后原键确认，再下一批；已确认但读进度失败只重读；创建丢响应按createKey查同job，不重新固定清单。
- [ ] 写 `runner shrinks only confirmed rollback`：仅 `PUBLISH_BATCH_ROLLED_BACK` 以新键依次20/10/5/2/1；未知提交不能缩批；单条超时暂停；临时失败第三次后暂停；成功批次清连续失败计数。
- [ ] 写 `pause refresh account switch and two tabs`：暂停不撤销已发布；刷新GET恢复后等待“继续发布”；账号切换清理恢复提示并取消调度，迟到响应不能更新；另标签冲突只刷新。浏览器禁用存储时仍可本页操作，刷新可从服务端本人操作列表恢复。
- [ ] 写 `full publish ignores current filters`、`failure retry uses explicit current selection`：full输入只有mode=all；失败项修正后先重读展示再明确创建 selected 操作，原成功/跳过项不重发。
- [ ] 运行 RED，再实现文字按钮、复选框、进度/失败明细分页和 runner；busy 覆盖单条/批量/全量重复触发，主上传流程保留。GREEN 后提交：`feat: 提供知识批量发布进度暂停与恢复`。

Run：runner 下执行新 selection/runner/batch tests 及原 knowledge-admin、batch-upload、i18n/final-review 测试。Expected：所有选择/恢复/身份/临时错误状态通过，无自动换令牌或后台常驻任务。

## B6：容量、真实回归、兼容登记与交付证据

**Files:** 新增 `backend/internal/store/knowledge_publish_capacity_test.go`、`backend/internal/e2etest/{knowledge_publish_fixture.go,knowledge_publish_fixture_test.go,knowledge_publish_control.go}`、`tests/e2e/knowledge-batch-publish.spec.ts`、`tools/verify/{knowledge-publish-ci.mjs,knowledge-publish-ci.test.mjs}`、`.github/workflows/knowledge-publish.yml`；修改 `backend/internal/e2etest/harness.go`；修改 `tools/verify/{go-batches.mjs,go-batches.test.mjs,topic-learning-compatibility.mjs}`、`api/topic-learning-compatibility-baseline.json` 及 P6 列明的精确例外登记；新增 `docs/operations/knowledge-batch-publish.md`；更新总计划指定验收记录。

**Interfaces:** 10001 条容量在仓储组验证全部批次；浏览器使用41条覆盖20/20/1，不把10001条慢操作塞进30秒单例。CI 库存集合必须包含全部新测试且不重复，原测试分组和排除名单的语义保持；新增独立 workflow，不放宽原 CI 作业。

- [ ] 写 `TestPublishCapacity10001SnapshotAndSteps`：固定10001条，在创建后新增1条、改变排序、分批处理；原10001条均有结果，新1条保持未发布，sum===10001，事件不重复；容量断言不包含源夹具构造耗时，只为业务事务计时并保留总测试截止。
- [ ] 写 `TestPublishCapacityLargeBodyAndInterrupt`：正文逐项、有界回执、确定回滚缩批和未知结果恢复；10001容量与大正文分别执行，不把fixture缓存误当真实发布吞吐。
- [ ] 浏览器用例 `selected publish keeps pages`、`full publish scopes entire library`、`resume after lost response`、`conflict and revoked admin`：真实 Go/SQL、桌面/手机；全量清单在有筛选和页码时仍为全部；完成后表格及地图统计刷新，失败明细真实。
- [ ] node CI 测试断言测试库存、runner及每组时限；先 RED，再实现合成夹具和独立workflow；精确登记新文件/API摘要/测试清单，保留旧schema摘要及迁移原字节。
- [ ] 分别运行 runner 下容量10001组、大正文组、HTTP/仓储发布组、原知识写/分享/复习组；浏览器命令 `npm exec -- playwright test --config ../tests/e2e/playwright.config.ts knowledge-batch-publish.spec.ts`；前端全组若超过540秒按目录分批，不删测试。
- [ ] 提交：`test: 验证知识全量发布容量与中断恢复`；编写实际使用说明及验收证据，回到总计划执行整分支审查、相关修复回归、Git push 与 MR。

Expected：新能力、原协议、迁移16兼容、合成容量及真实浏览器均有可复查证据；没有数据库配置、浏览器依赖或恢复容器时明确记录缺口，不以 skip 代替验证完成。
