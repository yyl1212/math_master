package question

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"regexp"
	"strings"
)

var placeholders = regexp.MustCompile(`\{\{([^{}]*)\}\}`)
var unsafeMath = regexp.MustCompile(`(?i)\\(?:href|url|html[a-z]*|input|include[a-z]*|def|gdef|edef|xdef|let|newcommand|renewcommand|providecommand|newenvironment|renewenvironment|usepackage|require|write|openout|read|catcode|csname)\b`)

func safeQuestionMarkdown(s string, assets []AssetRef) bool {
	if strings.ContainsRune(s, 0) || unsafeMath.MatchString(s) {
		return false
	}
	allowed := map[string]bool{}
	for _, a := range assets {
		allowed[a.ID] = true
	}
	root := goldmark.New().Parser().Parse(text.NewReader([]byte(s)))
	safe := true
	_ = ast.Walk(root, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.RawHTML, *ast.HTMLBlock, *ast.Link, *ast.AutoLink, *ast.CodeSpan, *ast.FencedCodeBlock, *ast.CodeBlock:
			safe = false
		case *ast.Image:
			dest := string(node.Destination)
			if !strings.HasPrefix(dest, "asset:") || !allowed[strings.TrimPrefix(dest, "asset:")] {
				safe = false
			}
		}
		return ast.WalkContinue, nil
	})
	return safe
}
func renderQuestion(s string, values map[string]string, answer string, explanation bool, assets []AssetRef) (string, error) {
	if !safeQuestionMarkdown(s, assets) {
		return "", ErrInvalid
	}
	invalid := false
	rendered := placeholders.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-2]
		if name == "answer" {
			if explanation {
				return answer
			}
			invalid = true
			return ""
		}
		if v, ok := values[name]; ok {
			return v
		}
		invalid = true
		return ""
	})
	if invalid || strings.Contains(rendered, "{{") || strings.Contains(rendered, "}}") || !safeQuestionMarkdown(rendered, assets) {
		return "", ErrInvalid
	}
	return rendered, nil
}
func displayRational(r Rational) string {
	if r.Denominator == "1" {
		return r.Numerator
	}
	return r.Numerator + "/" + r.Denominator
}
