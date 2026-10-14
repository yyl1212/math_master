# 第二阶段方案可行性与兼容性审查记录

日期：2026-10-14。审查基线：afff5aad42f395d7a1e488796133de74d3a82946。范围为本阶段详细方案、实施计划及对应仓库实现；不开发新产品代码、不迁移数据库、不访问生产用户数据。

## 1. 已确认需求与方案边界

用户优先个人持续数学学习；默写答完后自行比对参考；用户明确选择忘记／不熟／熟悉自评自动安排。具体10分钟及1/3/7/14/30天参数、陈述/证明模式待书面方案整体确认。

本次同时交付方案和计划遵循用户直接要求；开发阶段继续保留用户确认。计划按既有Native偏好编排，开发前重新拉master、新建开发分支，再真实TDD、代码及兼容审查、回归与MR交付。

## 2. 静态可行性证据

| 证据 | 核查结果 |
| --- | --- |
| backend/internal/knowledgeadmin/model.go SourcePoint | 已存在statement/conditions/scope/system/equations/proof/proof_scope，参考无需生成或改变源格式 |
| backend/internal/knowledgeadmin/study.go及store/knowledge_admin_study.go | 已存在四状态、sequence、activeReviewId/lastReviewId及准确Ref，支持新事务衔接合法原周期 |
| backend/internal/study/validate.go | MaxSequence=9007199254740991；ValidateNote提供字符/字节及安全规则可复用 |
| store/knowledge_admin_tx.go | 原8秒事务、1秒锁、身份与提交复核；新store文件可复用同包辅助函数，旧ApplyManagedStudy可保持字节 |
| store/knowledge_admin_maintenance.go managedSchemaDigestSQL | 约束按conrelid归属筛选，触发器排除tgisinternal；新FK只放新表、不改旧函数，可保持原摘要 |
| store/note_sharing_schema.go | 原sharing摘要固定2179de9c9ef5b78ec31debb6d5f4fb6c，可独立验证，不需要合并新摘要 |
| httpapi/application.go 与Next v3 catchall | 当前通用study分发/代理不认识新族；计划已列专门分发、代理与依赖注入 |
| 原阅读和笔记组件 | SafeMarkdown/工具栏/预览/NoteLeaveGuard可复用，独立默写数据不进入原笔记或分享 |
| ops/deploy.py | 当前最大迁移14且要求noteSharing；计划明确增加受审15及Go→Next→公网studyReview能力 |
| api/openapi.yaml | 基线SHA256 a91e2a8af81ac48131746fc2198c49edcdc0740b6b007d328b8c0db59bdd1514；本轮不修改API |

可行性不依赖新供应商、付费服务、评分模型或后台提醒。新的真实约束、竞争、响应上限、10000项目容量和恢复性能必须在开发阶段实测，本轮不冒充已实现或已通过。

## 3. 自审发现与落盘修订

1. 新自评直接改旧事件/DTO会破坏严格解析；采用独立五表/新API，原四状态与六事件保持。
2. 新会话重复调用旧HTTP可能多次提交；改为同knowledgeTx内原周期与新事实，旧finish已完成同周期时保留，不重复旧事件。
3. 隐藏UI但仍返回参考会提前暴露目标；改为API/SSR/React数据不交付目标，陈述与证明题干边界分别明确。
4. 当前知识无可依赖的公开历史版本接口；内部冻结准确参考，但改版/下架不交付失效参考；保留本人答案与元数据。
5. 旧无自评记录批量设逾期或补质量会造历史；改为“待安排”，不迁入虚构事件或评价。
6. 连续提前熟悉可把阶段快速刷满；同内容提前熟悉不推进，改版从有效阶段0重新安排。
7. 比对输入未显式表达空答；统一CompareInput.unanswered，read=false，recall空答=true且只能forgotten。
8. 新FK会使原场景TRUNCATE失败；计划包含新五表加入原单Exec名单，并保留#58真实原子性与认证验证。
9. 把新浏览器测试塞入原近预算上限作业有超时风险；追加独立新作业，原命令、规模和预算保持。

## 4. 兼容性检查矩阵

| 项目 | 本轮 | 实施/验收门禁 |
| --- | --- | --- |
| 产品Go/Next、API、迁移1—14、依赖、CI | 全部不改 | 原接口/DTO与文件精确保护、全部原回归 |
| 原首次/已完成事实及进度 | 明确不能被忘记/改版改成未学 | 原字段/六事件，真实交叉入口/回滚测试 |
| 笔记和审批分享 | 不复用答案为笔记，不改变冻结与最新10条 | 原全部笔记/分享/browser安全回归 |
| 新内容/草稿隐私 | 仅本人；管理员无越权；时效投影 | 已准备参考的身份撤销、迟到响应及SSR实际数据测试 |
| 新迁移 | 只新增五表，原摘要和旧DDL保持 | 14备份Up15指纹、15全库真实恢复、序列 |
| 发布与回退 | 15需要独立能力，不能退旧14 | 缺Go/Next/公网能力及未来16必须拒绝 |
| 现有基线政策 | 只追加三份文档的精确路径/摘要与政策pin | 2184历史文件及原政策/快照字节对照，不先给代码豁免 |

## 5. 独立审查与最终判定

已完成一次新鲜独立只读审查，覆盖afff5aa..9b6ae4f三份文档和对应源码。初稿Critical 0、Important 7、Minor 0；Declined to judge为空。七项意见核对源码后均接受，本席集中修订文档，尚未修改产品。由同一审查者继续核对本次修订，不派第二审查者、不重复旧阶段完整审查。

| 初稿问题 | 具体修订与后续测试 | 对应任务 |
| --- | --- | --- |
| 原只读RR不能证明撤销竞争下的新鲜身份 | 新私有读局部READ COMMITTED + 原共享保护锁 +账户/会话/知识锁；明确两种先后顺序，旧knowledgeTx保持 | Task2/3/5/6 |
| 改版不熟继承旧高阶段 | 三评分统一使用有效阶段，改版为0；A5→B不熟→B到期熟悉断言0/1天→1/1天 | Task1/2 |
| 15架空旧14迁移Down用例 | 两旧测试文件列入清单，UpTo14/缺14明确13，保留原精确断言，另测15上分享 | Task2/6/7 |
| raw512KiB不保证JSON≤1MiB且Prompt重复 | 消除重复，以实际json.Marshal wire589824 +答案最坏393216 +最大元数据65536合计预算，发起前422不落库；真实组合边界 | Task1/2/3/6 |
| Session写响应与元数据回执冲突 | 全部五写返回无正文CommandReceipt，后GET当前态；身份/资源/摘要后先重放，再检查新请求当前状态 | Task2/3/4/5 |
| 证明Prompt在改版/下架仍泄漏原正文 | 任意状态Prompt与Reference同时按当前可用Ref投影；API/SSR/迟到响应测试 | Task2/3/5/6 |
| equations中的命题未进入证明题干 | 证明Prompt加全部非空equations，不生成分类或证明；“下式成立”独立公式用例 | Task1/5/6 |

这些是方案与计划修订，未来测试名不是已完成测试证据。新表/功能/容量/恢复仍须开发后实测。原87项兼容工具初稿实跑全通过，5.626秒；19个API/迁移/依赖文件逐字节相同；已有policy全部字段、旧快照及旧例外在移除三文档登记后与master完全相等，校验器除policy pin外相同。修订门禁87/87通过，5.622秒；同席核对及最终判定如下。

## 6. 当前结论

同席增量核对9b6ae4f..c1f776c确认七项Important均已在方案层面化解，Critical 0、Important 0、Minor 4，Declined to judge为空；无剩余设计阻塞。四处Minor是回执后GET措辞、冻结标题/类型的显式元数据、材料raw与整个快照的字节定义、16000码点最多64000字节合法答案向量。本席已统一四处文字与类型，65536继续作为原保护及保守wire预算，不当成合法成功向量；同一审查者只核对四处增量，不增加审查席位。最终审查c1f776c..b865a10确认四项Minor全部关闭，剩余Critical 0、Important 0、Minor 0，Declined to judge为空，无延期项。

最终结论：当前方案和计划无设计阻塞，可交用户确认并按已选Native偏好开发。产品实现、真实身份/内容竞争、容量、迁移恢复和浏览器验证尚未进行；最终书面成果确认后才开发。
