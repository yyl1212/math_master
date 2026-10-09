package store_test

import (
	"bytes"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"os"
	"testing"
)

type knowledgeAdminFixture struct {
	*authFixture
	access map[string]knowledgeadmin.Access
	ids    map[string]string
	doc    knowledgeadmin.SourceDocument
}

func newKnowledgeAdminFixture(t *testing.T) *knowledgeAdminFixture {
	f := &knowledgeAdminFixture{authFixture: newAuthFixture(t), access: map[string]knowledgeadmin.Access{}, ids: map[string]string{}}
	for _, name := range []string{"admin_a", "admin_b", "editor_a", "reviewer_a", "learner_a", "learner_b"} {
		u, c, csrf := f.signup(name)
		f.ids[name] = u.ID
		role := "learner"
		switch name {
		case "admin_a", "admin_b":
			role = "admin"
		case "editor_a":
			role = "editor"
		case "reviewer_a":
			role = "reviewer"
		}
		if role != "learner" {
			f.exec("INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)", u.ID, role)
		}
		p, e := auth.DecodeContentProof(c, csrf, true)
		if e != nil {
			t.Fatal(e)
		}
		f.access[name] = knowledgeadmin.Access{TokenHash: p.TokenHash, CSRF: p.CSRF, IdempotencyKey: "initial-" + name, RequestID: "managed-test"}
	}
	b, e := os.ReadFile("../knowledgeadmin/testdata/valid-source.json")
	if e != nil {
		t.Fatal(e)
	}
	f.doc, e = knowledgeadmin.DecodeSource(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func (f *knowledgeAdminFixture) Access(name, key string) knowledgeadmin.Access {
	a := f.access[name]
	a.IdempotencyKey = key
	return a
}
