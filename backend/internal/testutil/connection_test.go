package testutil

import (
	"strings"
	"testing"
)

func TestIsolatedConfigsRejectDatabaseOverrides(t *testing.T) {
	raw := "postgres://local:do-not-echo@127.0.0.1/math_master_test_ci"
	name := "math_master_test_1234567890abcdef"
	for _, query := range []string{"dbname=math_master", "database=math_master", "%64bname=math_master", "%20dbname%20=math_master", "sslmode=disable&database=math_master_test_other", "sslmode=disable&sslmode=require", "options=-c%20search_path%3Dpublic", "invalid=%ZZ"} {
		_, _, e := IsolatedConfigs(raw+"?"+query, name)
		if e == nil {
			t.Error("database override or ambiguous query accepted")
		}
		if e != nil && strings.Contains(e.Error(), "do-not-echo") {
			t.Error("credentials in parse error")
		}
	}
	for _, input := range []string{"postgres://local/production", "postgres://local/math_master_test_ci#other"} {
		if _, _, e := IsolatedConfigs(input, name); e == nil {
			t.Error("unsafe input accepted")
		}
	}
	if _, _, e := IsolatedConfigs(raw, "math_master"); e == nil {
		t.Error("non-random target accepted")
	}
	admin, work, e := IsolatedConfigs(raw+"?sslmode=disable&connect_timeout=3", name)
	if e != nil || admin.Database != "postgres" || work.Database != name {
		t.Fatal("safe configuration did not preserve ownership")
	}
}
