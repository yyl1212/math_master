package knowledgeadmin

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestDirectoryTitlesTranslateOnlyMatchingCanonicalNames(t *testing.T) {
	for _, c := range []struct{ code, en, zh, kind, wantZh, wantEn string }{
		{"15Axx", "Basic linear algebra", "", "secondary", "基本线性代数", "Basic linear algebra"},
		{"15A03", "Vector spaces, linear dependence, rank, lineability", "", "specific", "向量空间、线性相关、秩、可线化性", "Vector spaces, linear dependence, rank, lineability"},
		{"15B99", "Special matrices", "", "other", "特殊矩阵（其他）", "Special matrices — other topics"},
		{"15-XX", "Linear and multilinear algebra; matrix theory", "现有中文一级名", "primary", "现有中文一级名", "Linear and multilinear algebra; matrix theory"},
		{"15Axx", "Original fixture subtheme 15Axx", "原创二级主题", "secondary", "原创二级主题", "Original fixture subtheme 15Axx"},
		{"15A03", "A different canonical topic", "", "specific", "A different canonical topic", "A different canonical topic"},
	} {
		t.Run(c.code+"/"+c.en, func(t *testing.T) {
			zh, en, e := DirectoryTitles(c.code, c.en, c.zh, c.kind)
			if e != nil || zh != c.wantZh || en != c.wantEn {
				t.Fatalf("zh=%q en=%q err=%v", zh, en, e)
			}
		})
	}
}
func TestDirectoryChineseSearchReturnsCodes(t *testing.T) {
	codes, e := DirectoryChineseMatches("线性方程")
	if e != nil || !slices.Contains(codes, "15A06") {
		t.Fatalf("codes=%v err=%v", codes, e)
	}
	codes, e = DirectoryChineseMatches("不存在的原创检索串")
	if e != nil || len(codes) != 0 {
		t.Fatal(codes, e)
	}
}
func TestDirectoryLabelsCompleteAndReadable(t *testing.T) {
	rows, e := loadDirectoryLabels()
	if e != nil {
		t.Fatal(e)
	}
	counts := map[string]int{}
	for code, row := range rows {
		counts[row.Kind]++
		if strings.ContainsAny(row.Chinese, "\\$") {
			t.Errorf("literal markup: %s %q", code, row.Chinese)
		}
	}
	want := map[string]int{"primary": 63, "secondary": 534, "specific": 4969, "other": 534, "auxiliary": 503}
	if !reflect.DeepEqual(counts, want) {
		t.Fatalf("counts=%v", counts)
	}
}
