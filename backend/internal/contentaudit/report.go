package contentaudit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func WriteReport(jsonOut, markdownOut io.Writer, r Report) error {
	b, e := json.MarshalIndent(r, "", "  ")
	if e != nil {
		return e
	}
	if len(b)+1 > MaxReportBytes {
		return ErrLimit
	}
	var m bytes.Buffer
	fmt.Fprintf(&m, "# 首批内容验收报告\n\n结论：%s；模式：%s；技术夹具：%t。\n\n代码：%s\n路线：%s / %d；摘要：%s\n目录版本：%d；摘要：%s\n\n来源政策：%d；快照摘要：%s；报告摘要：%s；完整：%t；未解决问题：%d。\n\n| 口径 | 知识 | 模板 | 固定实例 | 生成实例 | 有效实例 | 重复 | 排除 |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n", r.Conclusion, r.Mode, r.FixtureOnly, r.CodeSHA, r.Context.Route.ID, r.Context.Route.Version, r.Context.RouteSHA, r.Context.CatalogueVersion, r.Context.CatalogueSHA, r.Source.PolicyVersion, r.Source.SnapshotID, r.Source.ReportSHA, r.Source.Complete, r.Source.UnresolvedCount)
	for _, row := range []struct {
		name string
		c    Counts
	}{{"草稿技术", r.DraftCounts}, {"正式", r.FormalCounts}} {
		c := row.c
		fmt.Fprintf(&m, "| %s | %d | %d | %d | %d | %d | %d | %d |\n", row.name, c.Knowledge, c.Templates, c.FixedInstances, c.GeneratedInstances, c.EffectiveInstances, c.Duplicates, c.Excluded)
	}
	fmt.Fprint(&m, "\n| 知识/版本 | 有效实例 | 检测实例 | 五题证据数 | 曝光后可覆盖 | 就绪 |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, n := range r.Nodes {
		fmt.Fprintf(&m, "| %s / %d | %d | %d | %d | %t | %t |\n", n.Knowledge.ID, n.Knowledge.Version, n.EffectiveInstances, n.AssessmentInstances, len(n.FiveWitness), n.AfterPracticeWitness, n.Ready)
	}
	fmt.Fprint(&m, "\n原因：\n")
	for _, reason := range r.Reasons {
		fmt.Fprintf(&m, "- %s（%s）\n", reason.Code, reason.Path)
	}
	if m.Len() > MaxReportBytes {
		return ErrLimit
	}
	if _, e = jsonOut.Write(append(b, '\n')); e != nil {
		return e
	}
	_, e = markdownOut.Write(m.Bytes())
	return e
}
