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

一次新鲜独立只读审查待进行。重点包括新旧复习周期、阶段/来源一致性、参考生命周期、所有权/快照、排期与预算、迁移/恢复及文件范围。

最终结论在取得审查结果和文档门禁证据后更新；目前不宣称已无阻塞，不启动产品开发。
