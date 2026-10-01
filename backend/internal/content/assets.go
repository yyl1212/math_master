package content

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var svgElements = words("svg g rect line path circle ellipse polygon polyline text tspan title desc")
var svgAttrs = words("xmlns width height viewBox x y x1 y1 x2 y2 cx cy r rx ry d points transform fill stroke stroke-width font-size font-family text-anchor role aria-label")

func words(s string) map[string]bool {
	m := map[string]bool{}
	for _, v := range strings.Fields(s) {
		m[v] = true
	}
	return m
}

var colorPattern = regexp.MustCompile(`^(?:none|black|white|red|green|blue|gray|grey|orange|purple|yellow|transparent|#[a-fA-F0-9]{3}|#[a-fA-F0-9]{6}|#[a-fA-F0-9]{8})$`)

func readAsset(root string, a Asset) ([]byte, error) {
	bad := errors.New("invalid asset")
	if !ValidAssetPath(a.Path) {
		return nil, bad
	}
	realRoot, e := filepath.EvalSymlinks(root)
	if e != nil {
		return nil, bad
	}
	current := realRoot
	for _, part := range strings.Split(a.Path, "/") {
		current = filepath.Join(current, part)
		s, e := os.Lstat(current)
		if e != nil || s.Mode()&os.ModeSymlink != 0 {
			return nil, bad
		}
	}
	f, e := os.Open(current)
	if e != nil {
		return nil, bad
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if e != nil || len(b) > 1024*1024 || fmt.Sprintf("%x", sha256.Sum256(b)) != a.SHA256 {
		return nil, bad
	}
	if e = validateSVG(b); e != nil {
		return nil, bad
	}
	return b, nil
}
func validateSVG(b []byte) error {
	bad := errors.New("invalid SVG")
	if !validSVGLexical(b) {
		return bad
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	depth, count, roots := 0, 0, 0
	for {
		token, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return bad
		}
		switch n := token.(type) {
		case xml.Directive, xml.ProcInst:
			return bad
		case xml.StartElement:
			count++
			if count > 10000 || depth > 64 || !svgElements[n.Name.Local] || (n.Name.Space != "" && n.Name.Space != "http://www.w3.org/2000/svg") {
				return bad
			}
			if depth == 0 {
				roots++
				if n.Name.Local != "svg" || n.Name.Space != "http://www.w3.org/2000/svg" {
					return bad
				}
			}
			depth++
			seen := map[string]bool{}
			for _, a := range n.Attr {
				if a.Name.Space != "" || !svgAttrs[a.Name.Local] || seen[a.Name.Local] {
					return bad
				}
				seen[a.Name.Local] = true
				if a.Name.Local == "xmlns" && a.Value != "http://www.w3.org/2000/svg" {
					return bad
				}
				if (a.Name.Local == "fill" || a.Name.Local == "stroke") && !colorPattern.MatchString(a.Value) {
					return bad
				}
				if strings.Contains(strings.ToLower(a.Value), "url(") {
					return bad
				}
			}
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(n)) != "" {
				return bad
			}
		}
	}
	if roots != 1 || depth != 0 {
		return bad
	}
	return nil
}

// encoding/xml substitutes invalid numeric surrogates with U+FFFD and resolves
// prefixes. Validate their original spelling before accepting decoded tokens.
func validSVGLexical(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	for {
		start := d.InputOffset()
		token, err := d.RawToken()
		if err == io.EOF {
			return true
		}
		if err != nil {
			return false
		}
		raw := b[start:d.InputOffset()]
		switch n := token.(type) {
		case xml.StartElement:
			if n.Name.Space != "" {
				return false
			}
			for _, a := range n.Attr {
				if a.Name.Space != "" {
					return false
				}
			}
		case xml.EndElement:
			if n.Name.Space != "" {
				return false
			}
			continue
		case xml.CharData:
			if bytes.HasPrefix(raw, []byte("<![CDATA[")) {
				continue
			}
		default:
			continue
		}
		for {
			at := bytes.Index(raw, []byte("&#"))
			if at < 0 {
				break
			}
			raw = raw[at+2:]
			end := bytes.IndexByte(raw, ';')
			if end < 0 {
				return false
			}
			digits := string(raw[:end])
			base := 10
			if strings.HasPrefix(digits, "x") {
				base = 16
				digits = digits[1:]
			}
			v, e := strconv.ParseUint(digits, base, 32)
			if e != nil || !(v == 9 || v == 10 || v == 13 || v >= 0x20 && v <= 0xd7ff || v >= 0xe000 && v <= 0xfffd || v >= 0x10000 && v <= 0x10ffff) {
				return false
			}
			raw = raw[end+1:]
		}
	}
}
