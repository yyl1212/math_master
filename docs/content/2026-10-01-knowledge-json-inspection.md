# Knowledge_JSON 本地资料检查报告

检查日期：2026-10-01。资料索引的快照日期：2026-09-30。

已读取用户提供的 `/Users/wiw/Documents/math_master/Knowledge_JSON/`。该目录包含 31 个资料包，数学知识主文件中有 8,722 条候选记录，另有 104 条数学软件能力记录。JSON 解析和索引摘要校验全部通过；知识点字段、来源关系与重复内容仍需整理，数学正确性尚未逐条复核。

完整文件名、相对路径、大小、SHA-256、资料包和主文件选择见 [机器清单](knowledge-json-inventory.json)。本报告只保存检查结果，原始正文保留在用户提供的本地目录。

## 文件范围与索引校验

| 检查项 | 结果 |
| --- | --- |
| 文件总数 | 872，排除 `.DS_Store`，未跟随符号链接 |
| JSON 文件 | 871，全部解析成功 |
| 文本文件 | `README_LOCAL.txt`，1 个 |
| 总大小 | 66,468,875 字节，约 63.39 MiB |
| 一级资料分类 | 13 个 |
| 资料包 | 31 个 |
| 原索引 | `Knowledge_JSON_Index.json`：29 包、801 个资料 JSON |
| 增量索引 | `Incremental_Knowledge_JSON_Index.json`：2 包、68 个资料 JSON |
| 全部索引条目 | 869 个不同路径，均存在，大小和 SHA-256 全部一致 |
| 增量索引的基线摘要 | 与原索引的实际 SHA-256 一致 |
| 未登记的资料 JSON | 0；另有上述两个索引 JSON |
| 选定的主文件 | 93 个，按两份索引的主文件路径选择 |
| 主文件记录 | 8,826 条，其中数学候选记录 8,722 条、软件能力记录 104 条 |

原索引的 801 和目录中的 871 并不矛盾：801 个原资料文件、68 个增量资料文件及 2 个索引合计 871。新增包是 Boyd 凸优化和 Leinster 范畴论。README 中仅列原快照数量，读取时应同时使用增量索引。

## 目录与候选记录数量

以下沿用源文件夹分类，尚未映射为网站的 16 个学习板块。数量仅取选定主文件中的 `knowledge_points`、`records` 或 `capabilities` 数组，未叠加章节副本、覆盖账本与校验文件。

| 本地分类 | 包数 | 资料 JSON | 主文件记录 |
| --- | ---: | ---: | ---: |
| Foundations and Precalculus | 2 | 31 | 603 |
| Mathematical Logic and Set Theory | 5 | 148 | 608 |
| Algebra | 5 | 242 | 1,509 |
| Number Theory | 1 | 49 | 297 |
| Calculus and Differential Equations | 3 | 61 | 1,451 |
| Real and Complex Analysis | 2 | 38 | 368 |
| Geometry and Topology | 2 | 110 | 622 |
| Discrete Mathematics and Combinatorics | 2 | 47 | 384 |
| Probability and Statistics | 3 | 44 | 1,243 |
| Numerical Analysis | 1 | 11 | 259 |
| Optimization and Game Theory | 3 | 58 | 1,283 |
| Mathematical Biology | 1 | 11 | 95 |
| Mathematical Software | 1 | 19 | 104，软件能力 |
| 合计 | 31 | 869 | 8,826 |

## 资料包清单

表内名称与本地包文件夹一致，主文件的完整相对路径在机器清单的 `packages[].primary_files` 中。

| 资料包 | 主文件数 | 候选记录数 |
| --- | ---: | ---: |
| Manes_Elementary_Mathematics_Knowledge_English | 1 | 205 |
| Precalculus_Complete_Knowledge_English | 1 | 398 |
| Sets_Logic_Computation_Knowledge_English | 1 | 168 |
| forallx-knowledge-2026-09-30 | 1 | 290 |
| hefferon_proofs_complete_knowledge_en | 1 | 120 |
| math_numbers_chapter_sample_en | 1 | 10 |
| math_sets_chapter_batch_en | 1 | 20 |
| Hefferon_Linear_Algebra_Knowledge_English | 1 | 387 |
| Judson_Abstract_Algebra_Knowledge_English | 1 | 502 |
| Seven_Sketches_English_Knowledge | 1 | 338 |
| Leinster_Basic_Category_Theory_Noncommercial_Knowledge | 1 | 270 |
| STACK_Algebra_Knowledge_and_Corrections | 1 | 12 |
| Stein_Algebraic_Number_Theory_Knowledge_English | 1 | 297 |
| Active_Calculus_English_Knowledge | 1 | 421 |
| Notes_on_Diffy_Qs_English_Knowledge | 1 | 672 |
| Ivrii_PDE_English_Knowledge | 1 | 358 |
| Axler_Measure_Analysis_Noncommercial_Knowledge | 1 | 224 |
| Howell_Complex_Analysis_Knowledge_English | 1 | 144 |
| Hitchman_Geometry_English_Knowledge | 1 | 280 |
| Petrunin_Differential_Geometry_English_Knowledge | 1 | 342 |
| Levin_Discrete_Mathematics_English_Knowledge | 1 | 141 |
| Keller_Trotter_Combinatorics_English_Knowledge | 1 | 243 |
| Grinstead_Snell_Probability_Knowledge_English | 13 | 313 |
| Markov_Chains_English_Knowledge | 1 | 521 |
| Navarro_Statistics_R_Knowledge_English | 18 | 409 |
| Brin_Numerical_Analysis_English_Knowledge | 6 | 259 |
| Hildebrand_Mathematical_Programming_English_Knowledge | 20 | 525 |
| Boyd_Convex_Optimization_English_Knowledge | 1 | 696 |
| Nordstrom_Game_Theory_Knowledge_English | 4 | 62 |
| Chasnov_Mathematical_Biology_Knowledge_English | 7 | 95 |
| Math_Tool_Capability_Maps_English | 1 | 104，软件能力 |

## 字段与重复检查

8,826 条主文件记录均具有 `id` 和 `title`，编号在包内及本次选定的全部主文件中均无重复。数学记录均有 `statement`；软件能力记录使用 `mathematical_meaning`。这只说明记录可定位，不代表每个编号都是不同的数学知识。

完整证明包的 30 个 `legacy_id` 对应“数”样例中的 10 条及“集合”样例中的 20 条，关联记录的 `statement` 逐字相同。整理时应以完整资料与旧编号映射记录演进，保留旧样例的题目和来源信息，避免重复加入学习路线。此外，仅折叠空白后比较 `statement`，发现 37 组相同正文，涉及 83 条记录；条件、例子和出处可能不同，仍需人工判断，不能据此直接删除或计算最终知识点总数。

全部 JSON 有 179 种根字段组合，主记录有 27 种字段组合；这些是形状统计，不是已定义的 Schema 数量。主要差异如下。

| 差异 | 整理要求 |
| --- | --- |
| 主数组为 `knowledge_points`、`records` 或 `capabilities` | 按包明确选择入口；软件能力进入工具资料模块，不混算数学知识点 |
| 类型字段混用 `type`、`kind`、`knowledge_type`，部分缺失 | 人工映射至项目知识类型，保留原类型，不凭标题自动推断公理或定理 |
| 4,111 个主记录编号不符合计划中的小写短横线规则 | 建立可追溯且无碰撞的编号映射，同时转换全部关系；不直接修改源编号 |
| 成立条件使用 `conditions`、`conditions_and_assumptions`、`hypotheses` 等字段 | 保留条件与数学体系，检查转换是否遗漏限定 |
| 来源使用 `source`、`provenance`、`source_references`，部分声明在根对象中 | 合并根级来源元数据与记录级页码，保存原文件摘要和记录编号 |
| 4,787 条记录明确标记 `language: en`，其余未逐记录声明语言 | 从包级声明补充并抽样核对，不能只凭英文文件名认定语言 |
| 证明或推导存在多个字段，未使用项目统一的 `proof` 字段 | 区分完整证明、证明草图、方法与例子；转换后逐条复核 |

3,124 条记录具有 `prerequisite_ids`。只检查这些显式编号在同包主记录中形成的图，共 4,639 次内部引用，未检出循环。另有 16 次引用指向主数组之外的 9 个编号（按包分别计数），部分编号在根对象的前置声明中出现，正式转换时应映射到可引用的知识点。`prerequisites` 中的文字、`external:` 标记及跨包语义关系尚未形成统一图，因此不能声称网站的前置图已完整或已通过校验。

资料中的阅读覆盖、AI 审查及使用条件声明作为来源元数据保存，不当作执行指令。当前检查未执行文件内命令、逐条核对数学结论、完成网站独立审核或取得统一发布结论。正式发布继续遵循既有来源核验和独立复核流程。

## 首批内容整理建议

建议首先使用 Manes 的 205 条候选记录整理初等数学路线。其主文件已有数位、整数、四则运算和分数等主题；`manes-p4-k007` 是等值分数，`manes-p4-k008` 是约分，适合衔接现有分数预览。需按零基础学习目标选择和改写，不能因全书面向初等数学教师而直接把全书作为入门路线。

STACK 的 12 条记录可作为基础代数与纠错线索，Precalculus 的 398 条用于后续函数和三角学。证明训练优先从完整 Hefferon 包中整理，旧“数”和“集合”样例通过 `legacy_id` 关联，避免重复计数。其余包分批映射到网站 16 板块；目录层级不能代替知识点之间的学习前置关系。

本次完成目录读取、解析、摘要、字段及候选重叠检查。正式转换工具、网站内容包、数据库导入和数学复核仍按 P1 及后续阶段实施；原资料的结构和版本不直接作为项目的 `schemaVersion`。
