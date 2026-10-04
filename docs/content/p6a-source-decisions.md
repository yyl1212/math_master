# P6a 来源选择与隔离处置

本轮使用固定原字节快照，政策版本 1。用户资料仍可继续更新；不会改写资料或把后续字节混入本批。

- snapshotId：eebca0e9f8cbe8ce9afbed5f1f872f382f54e42b8004fcb57d23d05b99ff879a
- source-report SHA256：7bd7c12118b915bc0a0ae4398b8018f1ba76e305736703a0617802677c7874ad
- 快照文件数：2214；索引包数：127；主入口数：325。这些是本轮观测，不是未来数量约束。
- 选中：Foundations and Precalculus/Manes_Elementary_Mathematics_Knowledge_English/manes_elementary_complete__knowledge_corpus.json；datasetId：manes-elementary-teachers-full-book-knowledge-en；记录数 205，每条保留原 ID，不按标题或 legacy ID 合并。
- selected ready：true；publicationApproved：false。AI 标记和 human_reviewed=false 均不构成独立人工数学审查。

| 问题 | 文件 | 处置 |
| --- | --- | --- |
| INDEX_DIGEST_MISMATCH | Geometry and Topology/Gathmann_Algebraic_Geometry_Original_Knowledge/chapters/06.json | 隔离；不纳入本批 |
| INDEX_DIGEST_MISMATCH | Geometry and Topology/Gathmann_Algebraic_Geometry_Original_Knowledge/coverage/statement_review.json | 隔离；不纳入本批 |
| INDEX_DIGEST_MISMATCH | Geometry and Topology/Gathmann_Algebraic_Geometry_Original_Knowledge/knowledge.json | 隔离；不纳入本批 |
| INDEX_DIGEST_MISMATCH | Geometry and Topology/Gathmann_Algebraic_Geometry_Original_Knowledge/manifest.json | 隔离；不纳入本批 |
| INDEX_DIGEST_MISMATCH | Real and Complex Analysis/Howell_Complex_Analysis_Knowledge_English/Howell_Complex_Analysis__artifact_manifest.json | 隔离；不纳入本批 |
| INDEX_DIGEST_MISMATCH | Real and Complex Analysis/Howell_Complex_Analysis_Knowledge_English/Howell_Complex_Analysis__knowledge_points.json | 隔离；不纳入本批 |
| INDEX_DIGEST_MISMATCH | Real and Complex Analysis/Howell_Complex_Analysis_Knowledge_English/Howell_Complex_Analysis__normalized_chapters_01_03.json | 隔离；不纳入本批 |
| INDEX_DIGEST_MISMATCH | Real and Complex Analysis/Howell_Complex_Analysis_Knowledge_English/Howell_Complex_Analysis__normalized_embedded_subsections.json | 隔离；不纳入本批 |

所有索引声明及双方摘要保存在私有报告中；原 provenance、review_status、条件、章节和 legacy 字段保存在原字节快照中。高级材料的差异没有落在唯一所选文件上，因此隔离后不阻止本批草稿准备。任何所选摘要冲突、缺失或记录身份冲突均阻止 ready。原始文件不可直接公开或自动批准。

Manes 资料仅用于事实核对与背景检索。课程说明、题干、例题与 SVG 均原创；来源映射逐对象注明 fact_check、background 或 original_derivation。概念相似不能证明条件相同；条件不明须待核对。第 15、17、30 节点等原创推导或背景映射不得伪装成书中直接命题。正式发布须在 P6b 完成实际独立人工审查、许可核对与学习链验收。
