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
- [x] Task 12: complete

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

Task 12: 完整17批124项浏览器PASS（原100+新24），零skip/retry。最终两视口筛选/back/实际序号冲突、owner及handler active原template阻断通过，24张脱敏截图已核对，证据导出到docs/operations/evidence/p5a。完整273前端、24Node、11基础Go包、新旧store/三容量/构建/vet均PASS；一次fresh整分支审查与交付继续执行。

Final: review range 20a68fa6f087e5e78561bf07c0229e072642e257..4a5ced873a5b76601bac373057b51a7ce80cef0c；一位fresh reviewer只读审查，无再委派；Critical无，Important五项，Minor一项。
Final: grade Important/P1 SSR换账户复用私有子树；旧账户草稿可实际作为新账户POST，必须修复。
Final: grade Important/P2 成功POST后GET失败丢失原命令；实际重复创建及扣额度，必须修复。
Final: grade Important/P2 临时身份失败无法恢复；五页恢复入口不可用，必须修复。
Final: grade Important/P2 过期来源没有保留草稿刷新入口；用户只能持续失败或丢4000字输入，必须修复。
Final: grade Important/P2 状态刷新清空未提交草稿及处理依据；冲突恢复丢输入，必须修复。
Final: minor (deferred): 检测重合提示缺少返回检测入口；主导航仍可到达，暂缓增加专用链接。
Final: Ruling: P5b重算、影响任务和通知保留后续阶段—已批准分段，本轮保证反馈不改变原资格—若错误，用户需要的重算通知须另行实现，不能以P5a宣称完成。
Final: Ruling: revision_published仅证明现有独立发布事实，不承诺完整供题恢复或资格恢复—不复制P4b/P5b算法—若错误，恢复资格预期与实际不符，需另审契约。
Final: Ruling: 进入表单前的历史公共快照按批准ContextQuery解析当前ID；表单加载后的过期来源必须在本轮修复—没有批准历史快照定位协议—若错误，历史页面反馈会定位当前版本，需新协议及兼容性审查。
Final: Ruling: 正式数学质量及生产部署/灾备不以技术夹具验收替代—P6/P7独立门槛—若错误，未验收内容与恢复能力可能被误用于上线。
Final: Ruling: 未创建MR及精确HEAD的远端CI不视为产品缺实现，仍列本轮交付门槛—先完成修复再交付并核对四项—若错误，可能交付未被远端验证的版本；最终必须查实全部四项。

Final: 修复回归已先看到9项确切RED（原命令丢失error/deadline两类、SSR账户、显式/聚焦/SSR恢复、目标刷新、owner和handler草稿）；随后附加总截止后元数据晚到导航实际RED。当前42项反馈UI/钩子GREEN。五页共同的Provider按actor原子重建；同actor新SSR及显式/聚焦重验清暂时错误；成功命令保留pending/原key并标记已确认，后续读取继承原截止；目标刷新保留4000字、更新同来源并等主动新提交；状态刷新不清草稿，只有提交成功且最新元数据确认后清理。等待完整复验后记录Final fixed及最终SHA。

Final: 新增双视口联调首批4/6PASS；身份恢复两例的单次注入被公共导航先消耗，未形成Provider故障，真实失败日志保留。改为明确整个故障时段全部身份请求失败，服务恢复后才按Reload验证；不改产品逻辑或重试次数。

Final: minor (deferred): 视觉核对发现反馈身份服务恢复后，共用顶部账户栏可能仍暂显Accounts unavailable；已有AuthStatus只在挂载/身份通知时重读，反馈表单已由新鲜身份核验恢复。此项为旧共用栏状态同步，暂缓统一恢复入口；不涉及账户越权。

Final: fixed Important/P1 SSR账户草稿串用—atomically removes the old private draft before a silent SSR account change is verified RED→GREEN，suite 前端283/283、浏览器130/130、Node24/24及全部Go/容量通过。
Final: fixed Important/P2 成功POST后的读失败丢原命令—keeps the confirmed original command when latest-view refresh fails by error/deadline及ignores a latest-metadata response arriving after the confirmed command deadline RED→GREEN，suite 前端283/283、浏览器130/130及全部矩阵通过。
Final: fixed Important/P2 临时身份错误无法恢复—Reload action/focus/same-account SSR三个实际组件RED→GREEN，suite 前端283/283、浏览器130/130及全部矩阵通过。
Final: fixed Important/P2 过期来源没有保留输入刷新路径—refreshes a stale target while retaining the original text and waiting for explicit new submission RED→GREEN，真实新发布v2两视口通过，suite 前端283/283、浏览器130/130及全部矩阵通过。
Final: fixed Important/P2 刷新工单清草稿—manual owner status refresh及conflict recovery preserves handler draft and references while an old-key retry stays frozen RED→GREEN，suite 前端283/283、浏览器130/130及全部矩阵通过。
Final: product repair commit 175c61f41df6c1c1f974c0ce3e9a023c54361951；只执行一次重要修复，不再派独立审查。完整复验4前端阶段/11后端阶段/17浏览器阶段均exit0，17批130项/零skip/retry，最长浏览器74.814s；三容量11.187s/232.357s/121.352s，反馈最大182.861792ms、607731-byte正文页。Go原5m/浏览器8m/包装器9m/CI job30m保持。Task12本机与审查Step4完成，Step5 SSH MR和最新四项CI待远端。
Task 12: complete (commits 640658e..6c550d9, tests: node tools/verify/run.mjs -- node --test tools/verify/run.test.mjs tools/verify/learning-compatibility.test.mjs tools/verify/feedback-compatibility.test.mjs tools/verify/feedback-ci.test.mjs tools/content-ingest/snapshot.test.mjs → ℹ duration_ms 5538.483042)

Final: draft MR #21已创建并attach，首轮交付head 6c550d9aa68bc4a5bc2a15cd35b8168cc5ab2d61 的Go/前端push/PR四项CI全部SUCCESS；以下最终文档提交仅保存交付事实，仍再次核对该最新HEAD四项CI，结果保存到MR描述，不以旧head代替。
