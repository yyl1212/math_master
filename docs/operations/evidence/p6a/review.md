# P6a 独立整分支审查记录

审查者：唯一 fresh-context `gpt-6-astra`，`p6a_final_whole_branch_review`。范围：master基线8bf3c95至9d1cd05，加Task10全部未提交补丁/新增文件。全程只读，没有修改工作树、索引、HEAD或Git状态，没有再派子审查。本记录保留完整发现/严重性/范围，原会话审查报告另有逐项说明；以下行号是修复前代码。

结论：With fixes，Critical0、Important3、Minor2。审查时没有宣称未完成远端CI通过。

## 实际核验的优点

三索引保留全部声明和冲突摘要；旧快照身份算法、采集中改写/删除屏障保持。Store单RepeatableRead/ReadOnly事务先核验完整双head、正文和冻结审核，再精确过滤路线，复用选题资格SQL。随机库自动fixtureOnly、正式数零，真实报告为draft_ready与awaiting_review。

数学抽查全部24模板题意、各节点固定01/06/11/15题；独立Fraction枚举384参数组合，未发现除零或非负约束违例。抽查分数面积、约分反例、终止小数条件、余数及百分数整体，十五固定题分组支持一次曝光后的双目标五题。此次不是全量真实人员数学批准。独立执行22项CI/兼容保护PASS；读取时整Go矩阵尚未结束，浏览器已全PASS。

## 三项Important与处理

1. coverage.go:369/394：ReviewAttestation同decisionId最后值覆盖false，learningChecks以passed布尔值判重漏掉not_run→passed。真实--evidence可达；规范要求按身份集合拒绝重复与未知名称，且保留failed的失败事实。TestContentAuditEvidenceConflicts完整834实例正向、两顺序/同值/冲突值/未知名负测RED→GREEN，完整验收套件PASS。
2. decode.go:18/53/169：非普通文件/FIFO及无ctx的摘要、第二次JSON打开可逃出8秒截止，定位可能来自未核验字节。修复为普通文件、声明大小/独立64MiB、可取消读取及同份字节摘要/解析，草稿SVG和外部证据沿用相同受控读取。SourceReadBudget/SpecialFiles/DeclaredBytes/CorpusCapacity/DraftSpecialAsset/DraftCancellation负测RED→GREEN，完整验收套件及容量PASS。没有以4MiB限制原始corpus。
3. coverage.go:311与store/content_audit.go:216：Store只排除instance，零当前eligible实例模板仍算20门槛。修复准确身份的模板有效集合；纯层五模板失效但450固定覆盖保持时24→19并not_ready，原先accepted；数据库正常工作流将模板专属unit v1替换为v2，15固定题仍有效，原先缺模板排除记录。两负测RED→GREEN，完整套件/容量PASS。

执行者按实际用户影响保留三项Important，并在一次修复批解决，没有再派复审。修复提交4d27dc8f1564bd0ed5b5fa0d4ac1a5a8c504c8ab。

## 两项Minor（保留）

1. coverage.go:133：published纯层在缺manifest时从SourceMap回退blueprintSHA。真实Store提供核验manifest，未发现经当前CLI绕过；后续收紧纯层契约。
2. report.go:20/27：中文Markdown缺逐节点原因/双head；JSON保留这些信息，后续完善摘要。

## 兼容性判断

局部.panel select宽度约束修复手机真实目标溢出，无组件契约变化。裸spec参数采用准确basename glob消除误选，旧预算/视口/workers/retries保持。CI原20批/五容量保留，两新浏览器与ContentAudit独立批分类合理；旧保护/负测未删除。三项兼容改动可接受。

## Declined to judge（全部六项，执行者逐项裁决）

1. 真实人员数学批准/自然人独立性：交P6b真实复核，P6a正式计数零。
2. 205来源出处/许可/条件逐条匹配：只有映射与校验机制在本次范围，P6b逐条确认。
3. 全450固定题、全部正文证明、九图的全面数学认证：参数全量与抽查已做，全量人员复核留P6b；视觉审查只看面积和路线页，不能冒充九图批准。
4. 同节点多已发布blueprint选择：本批唯一蓝图，多蓝图失败关闭；未来另设计选择，可能拒绝未来合法多蓝图报告。
5. 最终完整Go、manifest、准确head远端CI：执行者已补完整本地矩阵，最终远端另核验，不以旧PR替代。
6. 生产部署/迁移/真实发布/MR合并：不在本次授权，后续分阶段。

每项完整裁决、若错误的代价见实施账本；所有Minor一并保留，不静默删除审查意见。
