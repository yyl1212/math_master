package httpapi

import (
	"bytes"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"strings"
	"testing"
)

func TestPrivateJSONIsStrict(t *testing.T) {
	bad := []string{`{"username":"abc","username":"def","password":"a long enough test passphrase"}`, `{"Username":"abc","password":"a long enough test passphrase"}`, `null`, `[]`, `{"username":"abc","password":"a long enough test passphrase"} {}`, `{"username":null,"password":"a long enough test passphrase"}`, `{"username":"abc","password":"\ud800"}`, `{"username":"abc","password":"\udc00"}`, `{"username":"abc","password":"\ud800x"}`, `{"username":"abc","password":"valid passphrase", "role":"admin"}`, strings.Repeat(" ", 8193), `{"username":"abc","password":"` + string([]byte{0xff}) + `"}`, `{"username":{"nested":[[[[[[[[1]]]]]]]]},"password":"test"}`}
	for i, input := range bad {
		var value auth.RegisterInput
		if decodePrivateJSON(bytes.NewReader([]byte(input)), &value) == nil {
			t.Fatalf("invalid JSON boundary %d accepted", i)
		}
	}
	var good auth.RegisterInput
	input := `{"username":"Math_User","password":"合法中文密码搭配空格保留 \ud83d\ude00 "}`
	if decodePrivateJSON(strings.NewReader(input), &good) != nil || good.Username != "Math_User" || good.Password != "合法中文密码搭配空格保留 😀 " {
		t.Fatal("Unicode password was rewritten")
	}
	var empty struct{}
	if decodePrivateJSON(strings.NewReader(`{}`), &empty) != nil {
		t.Fatal("empty logout object rejected")
	}
	if decodePrivateJSON(strings.NewReader(`{"extra":true}`), &empty) == nil {
		t.Fatal("extra logout field accepted")
	}
}
