package feedback

import (
	"strings"
	"testing"
)

func TestFeedbackValidationUnicodeAndLimits(t *testing.T) {
	for _, c := range []struct {
		name  string
		n     int
		valid bool
	}{{"body", 4000, true}, {"body", 4001, false}, {"title", 120, true}, {"title", 121, false}, {"location", 400, true}, {"location", 401, false}} {
		for _, r := range []string{"中", "😀"} {
			in := siteInput()
			text := strings.Repeat(r, c.n)
			switch c.name {
			case "body":
				in.Message = text
			case "title":
				in.Title = text
			case "location":
				in.Location = text
			}
			if e := ValidateCreate(in); (e == nil) != c.valid {
				t.Fatal(c.name, c.n, r, e)
			}
		}
	}
	for _, s := range []string{"", " \n\t", "x\x00y", string([]byte{0xff}), string([]byte{0xed, 0xa0, 0x80})} {
		in := siteInput()
		in.Message = s
		if ValidateCreate(in) == nil {
			t.Fatal("invalid text")
		}
	}
	for _, n := range []int64{1, 9007199254740991} {
		if e := ValidateReply(ReplyInput{ExpectedSequence: n, Message: strings.Repeat("中", 4000)}); e != nil {
			t.Fatal(e)
		}
	}
	for _, n := range []int64{0, -1, 9007199254740992} {
		if ValidateReply(ReplyInput{ExpectedSequence: n, Message: "Reply"}) == nil {
			t.Fatal("sequence", n)
		}
	}
	if ValidateReply(ReplyInput{ExpectedSequence: 1, Message: strings.Repeat("😀", 4001)}) == nil {
		t.Fatal("reply maximum")
	}
}
func TestFeedbackValidationTargetAndSource(t *testing.T) {
	valid := []CreateInput{siteInput()}
	k := siteInput()
	k.Target = Target{Kind: "knowledge", Identity: testIdentity("fractions")}
	k.Source = Source{Kind: "publication", PublicationID: testPtr(testUUID)}
	k.Category = "math_error"
	valid = append(valid, k)
	unit := k
	unit.Target.Part = &Part{Kind: "unit", Unit: testIdentity("fractions-unit")}
	valid = append(valid, unit)
	asset := k
	asset.Target.Part = &Part{Kind: "asset", Asset: &AssetRef{ID: "fractions-art", SHA256: strings.Repeat("b", 64)}}
	valid = append(valid, asset)
	path := k
	path.Target.Kind = "path"
	valid = append(valid, path)
	for _, kind := range []string{"practice", "assessment"} {
		qi := k
		qi.Target = Target{Kind: "instance", Identity: testIdentity("qi-" + strings.Repeat("a", 64))}
		qi.Source = Source{Kind: kind, AttemptID: testPtr(testUUID), Position: testPtr(1)}
		valid = append(valid, qi)
	}
	for _, in := range valid {
		if e := ValidateCreate(in); e != nil {
			t.Fatal(in, e)
		}
	}
	bad := []CreateInput{}
	for _, category := range []Category{"math_error", "unclear_explanation", "unknown"} {
		in := siteInput()
		in.Category = category
		bad = append(bad, in)
	}
	x := k
	x.Target.Identity = testIdentity("bad")
	x.Target.Identity.Version = 0
	bad = append(bad, x)
	x = k
	x.Source.AttemptID = testPtr(otherUUID)
	bad = append(bad, x)
	x = asset
	x.Target.Part = &Part{Kind: "asset", Unit: testIdentity("fractions-unit"), Asset: asset.Target.Part.Asset}
	bad = append(bad, x)
	x = path
	x.Target.Part = unit.Target.Part
	bad = append(bad, x)
	x = valid[len(valid)-1]
	x.Source.Position = testPtr(6)
	bad = append(bad, x)
	x = valid[len(valid)-2]
	x.Source.Position = testPtr(2)
	bad = append(bad, x)
	x = valid[len(valid)-1]
	x.Target.Part = unit.Target.Part
	bad = append(bad, x)
	for _, in := range bad {
		if ValidateCreate(in) == nil {
			t.Fatal("bad target/source accepted", in)
		}
	}
}
