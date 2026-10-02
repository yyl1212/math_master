# SDD ledger — plan: docs/superpowers/plans/2026-10-03-feedback-workflow.md

用户已确认12项/60步计划，保留Native，授权CI通过后合并文档PR20。PR20完整head四项CI全部SUCCESS，已合并；产品分支codex/p5a-feedback-workflow从最新master建立。

## Pre-flight
| 任务接口 | 消费/产出核对 |
| --- | --- |
| 1→2/3/4/5/6 | DTO/状态/权限/摘要统一；Binding仅内部，不传播自由文本 |
| 2/3→4 | 四表/准确来源/当前身份用于原子创建补充；replay先于当前head和序号 |
| 4→5 | 共用原回执/成功配额，处理只校验已有发布撤回事实 |
| 3/4/5→6 | 元数据与discussion分开；原instance/template固定，提交前DBclock曝光 |
| 3—6→7 | 七个完整仓储方法后才声明Repository/编译断言；Preflight不扣额度 |
| 7→8 | 14操作/具名DTO/raw bytes共享，合法形状不伪称HTTP成功 |
| 8→9 | 固定actor/同键手动retry/10s包含identity，无持久化私有草稿 |
| 3/5/7—9→10 | 真Go/DB审核发布/练习测评来源，老harness前提保持 |
| 1—10→11/12 | baseline固定master8400ee5；原14批次/两最大容量完整保留 |

## Tasks
- [x] Task 1: complete
- [x] Task 2: complete
- [x] Task 3: complete
- [x] Task 4: complete
- [x] Task 5: complete
- [x] Task 6: complete
- [x] Task 7: complete
- [x] Task 8: complete
- [x] Task 9: complete
- [x] Task 10: complete
- [x] Task 11: complete
- [ ] Task 12: in_progress

Baseline commit: 20a68fa6f087e5e78561bf07c0229e072642e257
Task 1: complete (commits 20a68fa..a9419ac, tests: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/feedback -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/feedback	0.318s)

Baseline: 135 frontend unit tests, 13 node/compat/snapshot tests, 10 Go foundation packages, store+CLI PASS (213.458s/1.172s). Task1: RED undefined new contracts → GREEN 9 tests; compilation corrected to local RateError (auth uses concrete RateLimitError).

Task 2: Ruling: “各表单独非空”按合法关系状态验证（ticket+创建事件、追加事件、回执，以及quota单独非空）—事件和回执必须依赖真实工单，不能为测试关闭约束；Down仍显式检查全部四表—若未来改变外键关系，需要补充独立非空用例。
Task 2: complete (commits a9419ac..dc67879, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedback(Schema|Migration)' -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/store	6.064s)
Task 3: complete (commits dc67879..4f3a14e, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedback(Targets|CommitIdentity|Configuration)' -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/store	3.687s)
Task 4: complete (commits 4f3a14e..6095ff1, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedback(Commands|Replay|Sliding|Rates)' -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/store	3.980s)
Task 5: complete (commits 6095ff1..cb293d8, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedback(Transitions|GeneratedReplacement|DuplicatePrivacy)' -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/store	5.767s)
Task 6: complete (commits cb293d8..1c74f33, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestFeedback(Metadata|OriginalTemplate|Exposure|Cursor|Read)' -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/store	9.270s)

Task 7: 依赖修正：Metadata.targetValidity严格对齐已批准四枚举current/replaced/withdrawn/not_applicable。
Task 6: Ruling: 游标采用计划明确的version=1、512-byte边界格式，不携带账户/路由—“跨账户游标”负例验证权限WHERE不扩大，不额外要求拒绝合法边界—若未来需要账户绑定游标，应另审新版本协议。
Task 7: complete (commits 1c74f33..64e7102, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/feedback ./internal/httpapi -run Feedback -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/httpapi	10.102s)
Task 8: complete (commits 64e7102..17bc09a, tests: node tools/verify/run.mjs --cwd frontend -- npm test -- src/lib/feedback src/lib/api/feedback-proxy.test.ts →    Duration  412ms (environment 63%, setup 12%, transform 11%, import 7%, tests 5%, worker 1%))
Task 9: complete (commits 17bc09a..415f159, tests: node tools/verify/run.mjs --cwd frontend -- npm test -- src/features/feedback →    Duration  504ms (environment 53%, transform 15%, tests 14%, setup 10%, import 8%))

Task 10: Ruling: 文件表补充auth_fixture.go四表TRUNCATE—新外键使旧账户重置即使空表也失败，必须显式同时重置四张反馈表以保留所有原harness场景—仅隔离随机测试库，不更改产品账户逻辑；若未来新增关联表未被列入重置，原harness会再次被外键阻断。
Task 10: complete (commits 415f159..6731569, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/e2etest -run Feedback -timeout 5m -count=1 → ok  	github.com/yyl1212/math_master/backend/internal/e2etest	3.453s)

Task 11: 容量首跑真实PASS（不伪造RED）；1000工单/10000事件、50条分页、607731-byte最大正文页、同微秒连续事件、并发1成功、105条长期结案、原回执重放不退状态/不加额度，首次最大操作173.074792ms。补齐原Go5m剩余截止、实际SQL EXPLAIN、缺迁移旧学习可用及恢复不改事实后10.516s全PASS，非空原学习/答案/成绩/资格/解锁逐字节保持。兼容RED缺基线→从指定8400ee5生成→75/192/32/3及两类负测GREEN；新旧兼容9项均PASS。原store/CLI138.103s、学习78.575s、原题源90.940s、原路线119.659s、迁移/schema/配置6.854s均PASS；产品无性能改动。
Task 11: complete (commits 6731569..640658e, tests: python3 .superpowers/sdd/2026-10-03-feedback-workflow/test-env.py node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run ^TestFeedbackCapacity$ -timeout 5m -count=1 -v → ok  	github.com/yyl1212/math_master/backend/internal/store	10.712s)

Task 12: CI RED缺3浏览器及独立反馈批次→扩充全部真实命令并修正-count=1解析→6项GREEN（删除旧批次、增加skip/retry、删容量、扩大旧skip均能检出）。先提交Step3 CI/运维，便于最终整分支审查包含完整提交diff；Step4全矩阵及Step5最终验收仍进行中。npm ci/生成无漂移/typecheck/271项前端/构建均PASS。首次audit为TLS连接建立前中断（非漏洞），保留原代理环境并显式npm_config_https_proxy后0漏洞PASS；不改变依赖或CI网络配置。

Task 12: 补齐Task9/10筛选依赖回归。两项真实组件RED：新SSR页数据仍显示旧行，route改动后defaultValue仍为旧状态/分类。TicketList同步新初始页与路由，筛选表单按route重建，SSR列表按actor+route重新挂载以终止旧分页响应；32项反馈UI GREEN，273项全前端/生成无漂移/typecheck/生产构建GREEN。新增两视口真实筛选、category及浏览器back场景，等待完整17批验证；无DTO、数学行为或依赖变化。
