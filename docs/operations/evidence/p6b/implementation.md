# P6b 准备段实施与交接

本轮已确认范围是 Native 准备段 Task1—10；正式 R1—R4 未执行，整体为 `awaiting_review`，正式知识/模板/有效实例数量均为零。开发前通过 SSH 获取 master `b78108d3dd363eae237dae59f76a7be774b9de85`，从干净隔离工作树新建 `codex/p6b-content-review-preparation`。文档 PR #26 未合并。

## 工具结果

新增 `content-review prepare` 与 `verify-evidence`，只读显式本地输入并写全新私有目录，不读取数据库配置或连接网络。混合 v1/v2 由清单明确选择；正文、SVG 和来源都捕获同一份字节。复用既有校验、生成/封存及 canonical 身份，生成958对象/205来源、1163个未检查登记、完整834实例/五证明/九图材料及六个正常工作区导入文件。

固定 P6a 私有快照的[实际烟测](preparation.json)在 `a6df3d7424766ea29a758fa3c126d3507d8b1cbc` 构建运行：prepare=0，缺真实审批/实际head的 verify=3，无验收文件。该 G 保留实际运行时点，不重标后续文档提交。真实人员/许可/构建证明由负责人核验，工具仅检查文件、固定送审、全量登记和最终八文件的一致性；原 content-audit 才核验数据库当前批准、资格、撤回、双head及数量。

## 回归事实

[Go矩阵](matrix-go.json)16/16批、1376个 Test pass 结果事件（含父测试及具名子测试）、fail0/skip0；原宽入口包含新 Store/CLI 测试且 skip未扩充。六项容量均实际通过：反馈、学习题源、最大路线、最大内容、纠错影响、通知。旧题源准备超时本轮未复现，本地本次89.819秒通过；具体慢查询根因仍未证实，[旧诊断](../p6b-design/ci-diagnosis.md)保留，不宣称通过即修复。

[Node矩阵](matrix-node.json)原72项+新9项=81/81；[前端矩阵](matrix-frontend.json)350/350、类型生成/检查、构建及依赖审计通过，generated.d.ts无diff。[当前harness构建](matrix-harness.json)通过；[浏览器矩阵](matrix-browser.json)原22批/162双视口全部通过，failed0/skipped0；该原矩阵完成后进行了唯一整分支审查；两项Important已一次修复并完成相关全套回归，准确产品head远端CI已成功，归档后的最终head按下方交付规则再核验。

单批Go5m、浏览器8m、包装9m、CI job30m，原8秒操作与四分钟准备限制、数据量、workers=1、retries=0全部保留。新纯层99个结果事件包含真实256MiB读取、预算+1、64附件、取消/不覆盖/失败清理及证据缺项；不以小夹具替代原最大容量。

## 证据定位

矩阵记录 actual command 与 workflowCommand、UTC起止、exit、耗时、实际结果、运行时HEAD和实际源文件集合SHA。Go额外 `-json` 只改变日志表示。Task10仅增加检查与CI配置、三处新测试键控字段及文档；本地矩阵基于parent HEAD与该工作树，因此不把它们重标成将来的交付HEAD。私有烟测G、源集合SHA及远端准确HEAD分别记录。

[284项兼容基线](compatibility-baseline.json)来自上述实际master，保护既有运行时/数学/公开API/生成类型/00001—00008/原CLI/schema与双head参数。允许的既有文件差异仅是共用私有读取/校验主体提取及新离线8MiB严格入口；HTTP原4MiB不变。CI移除新增块/尾部两检查后必须逐字节等于原工作流，前端工作流完全不改；删除/改名旧组、取消容量、扩skip、去CGO或加预算的负例均拒绝。

日志位于 logs/*.gz；[索引](log-index.json)同时记录原始、脱敏后解压及压缩文件SHA。仅替换绝对工作树前缀和私有测试环境辅助脚本名；完整原始字节另保存在0700/0600私有目录。公开库不保存原corpus、私有复核包、个人/环境/答案材料。可用Python标准库gzip解压，校验 `storedUncompressedSHA256`；没有脱敏时它与 `originalLogSHA256` 相同。

## 正式交接

[操作手册](../../content-review.md)和[全量复核清单](../../../content/p6b-review-checklist.md)用于实际负责人落实R1环境/人员/范围，R2全量独立数学与出处许可复核，R3正常批准/双head发布与知情非生产纠错，R4最终八检查和实际一致只读accepted。已有旧 `elementary-foundations/v1` 路线时追加准确版本/清单/来源映射并重新复核，不能覆盖不可变路线。

[正式状态](formal-status.json)记录所有门槛。当前没有真实库初始化、角色授予、数学批准、正式发布/accepted或P7部署；技术夹具不能替代真实审批。MR保持draft，合并由用户后续指令决定。

## 审查修复后的独立回归记录

[唯一审查与裁决](review.md)保留原范围、原No意见、两项Important、零新增Minor和八项无法代替实际人员/执行者判断的边界。I1增加同一捕获字节的离线校验入口，原字节SHA贯穿release/register/全部学习上下文及报告；I2任何合法failed优先not_ready，完整性单独决定是否生成既有验收证据。CLI及纯层均有真实RED→GREEN。

[后审查Go矩阵](matrix-go-post-review.json)16/16批、1403个pass事件、fail0/skip0；[后审查Node矩阵](matrix-node-post-review.json)81/81。六容量未放宽，学习题源本次89.909s、最大路线121.574s、最大内容67.461s、纠错影响135.310s、通知42.153s，具体根因未知的历史超时仍不称作已修复。前端/浏览器原记录保留其实际来源，离线修复没有修改其生产代码。

后审查矩阵起始HEAD为 `fc0bd30c6789de70733ba467efe729dc8c4abe64`，包含未提交的修复，实际源集合SHA为 `6b1517bf472867ef6f7909f5a1c9e22df5b6c50821b0eee0b9290c5457d102cc`；记录均准确保留该时点，不以未来产品/归档HEAD改标签。日志与原矩阵分别命名，并独立复算压缩/解压摘要。

[审查修复后的固定快照烟测](preparation-post-review.json)在真实修复代码提交 `07c79b37275f85c87f21090c14170e8d303fc072` 重新构建并运行，实际源集合与后审查矩阵完全一致：prepare0/verify3、无验收文件、正式计数0。原a6df3d7烟测和此记录各保留实际G，后续归档提交不改变标签。

## 准备段完成与远端交付规则

Native Task1—10/50步骤已闭合，契约测试再执行81项通过；唯一整分支审查及一次修复相关全套回归已完成。SSH推送并创建[草稿PR #27](https://github.com/yyl1212/math_master/pull/27)。[远端CI原始结构化事实](ci-product-head.json)准确绑定产品head `0f8aa1ab72c857a800a4da44328f4cd4824fe6b1`，push与pull_request各运行Go和前端，合计四workflow/六job全completed/success。

本证据归档会产生新的提交。交付时再读取PR最新完整head，并核验新head对应四workflow/六job全部success；新SHA/运行链接保存在动态PR说明和交接答复。公共已存报告保持其实际G，不引用自身commit以免无限归档循环。PR保持draft；正式R1—R4与P7未执行，整体仍awaiting_review。
