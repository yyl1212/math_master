package notesharing

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"strings"
)

func Excerpt(body string) string {
	source := []byte(body)
	node := goldmark.New().Parser().Parse(text.NewReader(source))
	var out strings.Builder
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if v, ok := n.(*ast.Text); ok {
				out.Write(v.Segment.Value(source))
				if v.SoftLineBreak() || v.HardLineBreak() {
					out.WriteByte(' ')
				}
			}
		}
		if !entering && (n.Kind() == ast.KindParagraph || n.Kind() == ast.KindHeading) {
			out.WriteByte(' ')
		}
		return ast.WalkContinue, nil
	})
	plain := strings.Join(strings.Fields(strings.ReplaceAll(out.String(), "$", "")), " ")
	r := []rune(plain)
	if len(r) > 160 {
		plain = string(r[:159]) + "…"
	}
	return plain
}
