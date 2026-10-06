package study

import "regexp"

var noteHTML = regexp.MustCompile(`(?is)<\s*[/!?]?[a-z][^>]*>`)
var noteImage = regexp.MustCompile(`!\s*\[`)
var noteUnsafeMacro = regexp.MustCompile(`(?i)\\(?:html[a-z]+|href|url|includegraphics|def|gdef|edef|xdef|let|newcommand|renewcommand|providecommand|csname|catcode|input|write|openout|read|special)\b`)
var noteUnsafeURL = regexp.MustCompile(`(?i)(?:javascript|vbscript|data|file)\s*:`)

func validateNoteMarkup(body string) error {
	if noteHTML.MatchString(body) || noteImage.MatchString(body) || noteUnsafeMacro.MatchString(body) || noteUnsafeURL.MatchString(body) {
		return ErrInvalid
	}
	return nil
}
