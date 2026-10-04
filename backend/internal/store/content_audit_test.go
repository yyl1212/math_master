package store_test

import (
	"context"
	"database/sql"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func auditFixture(t *testing.T) *questionFixture {
	t.Helper()
	f := newQuestionFixture(t)
	in := f.Input()
	in.Package.ID = "audit-route-package"
	in.Package.Paths = []content.Path{{ID: "audit-route", Version: 1, DomainIDs: []string{"elementary-mathematics"}, Title: "Original audit technical route", TitleZh: "验收技术夹具", Nodes: []content.VersionRef{{ID: in.Package.Knowledge[0].ID, Version: 1}}}}
	d, e := f.repo.CreateDraft(f.ctx, f.Access("author_b", false), in)
	if e != nil {
		t.Fatal(e)
	}
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_b", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal(e)
	}
	sub, e = f.repo.DecideReview(f.ctx, f.Access("reviewer_b", false), sub.ID, approvedReviewInput())
	if e != nil {
		t.Fatal(e)
	}
	f.Activate(f.Prepare(sub, f.KHead()), f.KHead())
	f.QActivate(f.QPrepare(f.QApproved("author_a", "reviewer_a").ID))
	return f
}

var auditRoute = content.VersionRef{ID: "audit-route", Version: 1}

func TestContentAuditReadOnly(t *testing.T) {
	f := auditFixture(t)
	before := auditRows(t, f.db)
	role := "p6a_ro_" + strings.ReplaceAll(f.ID(), "-", "")
	f.exec(`CREATE ROLE ` + role + ` NOLOGIN`)
	t.Cleanup(func() { f.exec(`DROP OWNED BY ` + role); f.exec(`DROP ROLE ` + role) })
	f.exec(`GRANT USAGE ON SCHEMA public TO ` + role)
	f.exec(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO ` + role)
	var database string
	if e := f.db.QueryRow(`SELECT current_database()`).Scan(&database); e != nil {
		t.Fatal(e)
	}
	_, cfg, e := testutil.IsolatedConfigs(os.Getenv("TEST_DATABASE_URL"), database)
	if e != nil {
		t.Fatal(e)
	}
	cfg.RuntimeParams["role"] = role
	ro := testutil.OpenVerified(cfg)
	defer ro.Close()
	facts, e := store.New(ro).ReadContentAudit(f.ctx, auditRoute)
	if e != nil || !facts.FixtureOnly || len(facts.Content.Knowledge) != 1 || len(facts.Bank.Instances) != 28 || len(facts.EligibleInstances) != 28 || len(facts.Approvals) == 0 {
		t.Fatal(facts, e)
	}
	after := auditRows(t, f.db)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("audit mutated business rows")
	}
}
func auditRows(t *testing.T, db *sql.DB) map[string]int {
	t.Helper()
	rows, e := db.Query(`SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	if e != nil {
		t.Fatal(e)
	}
	names := []string{}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		names = append(names, n)
	}
	rows.Close()
	out := map[string]int{}
	for _, n := range names {
		var count int
		if e := db.QueryRow(`SELECT count(*) FROM ` + `"` + n + `"`).Scan(&count); e != nil {
			t.Fatal(e)
		}
		out[n] = count
	}
	return out
}
func TestContentAuditConsistentHeads(t *testing.T) {
	f := auditFixture(t)
	oldK, oldQ := *f.KHead(), *f.QHead()
	in := f.Input()
	in.Package.ID = "audit-route-next"
	in.Package.Paths = []content.Path{{ID: "audit-other-route", Version: 1, DomainIDs: []string{"elementary-mathematics"}, Title: "Another audit route", TitleZh: "另一路线", Nodes: auditFixturePathNodes()}}
	d, e := f.repo.CreateDraft(f.ctx, f.Access("author_b", false), in)
	if e != nil {
		t.Fatal(e)
	}
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_b", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal(e)
	}
	sub, e = f.repo.DecideReview(f.ctx, f.Access("reviewer_b", false), sub.ID, approvedReviewInput())
	if e != nil {
		t.Fatal(e)
	}
	f.Activate(f.Prepare(sub, f.KHead()), f.KHead())
	newK := *f.KHead()
	f.QActivate(f.QPrepare(f.QApproved("author_b", "reviewer_b").ID))
	newQ := *f.QHead()
	f.exec(`UPDATE publication_heads SET snapshot_id=$1 WHERE singleton`, oldK)
	f.exec(`UPDATE question_heads SET publication_id=$1 WHERE singleton`, oldQ)
	barrier, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer barrier.Rollback()
	if _, e = barrier.Exec(`LOCK TABLE question_publications IN ACCESS EXCLUSIVE MODE`); e != nil {
		t.Fatal(e)
	}
	type result struct {
		f contentaudit.PublishedFacts
		e error
	}
	ch := make(chan result, 1)
	go func() { facts, e := f.repo.ReadContentAudit(context.Background(), auditRoute); ch <- result{facts, e} }()
	deadline := time.Now().Add(700 * time.Millisecond)
	blocked := false
	for time.Now().Before(deadline) {
		var n int
		e = f.db.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'SELECT sealed AND status%'`).Scan(&n)
		if e != nil {
			t.Fatal(e)
		}
		if n > 0 {
			blocked = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("audit did not reach transaction barrier")
	}
	if _, e = barrier.Exec(`UPDATE publication_heads SET snapshot_id=$1 WHERE singleton`, newK); e != nil {
		t.Fatal(e)
	}
	if _, e = barrier.Exec(`UPDATE question_heads SET publication_id=$1 WHERE singleton`, newQ); e != nil {
		t.Fatal(e)
	}
	if e = barrier.Commit(); e != nil {
		t.Fatal(e)
	}
	r := <-ch
	if r.e != nil {
		t.Fatal(r.e)
	}
	if r.f.KnowledgeHead.ID != oldK || r.f.QuestionHead.ID != oldQ {
		t.Fatal("mixed snapshots", r.f.KnowledgeHead, r.f.QuestionHead)
	}
}
func auditFixturePathNodes() []content.VersionRef {
	return []content.VersionRef{{ID: "workflow-fractions", Version: 1}}
}
func TestContentAuditWithdrawal(t *testing.T) {
	for _, kind := range []string{"instance", "template", "blueprint", "unit", "asset", "knowledge"} {
		t.Run(kind, func(t *testing.T) {
			f := auditFixture(t)
			before, e := f.repo.ReadContentAudit(f.ctx, auditRoute)
			if e != nil {
				t.Fatal(e)
			}
			if kind == "instance" {
				f.QWithdraw(question.WithdrawalTarget{Kind: kind, ID: before.Bank.Instances[0].Identity.ID, Version: 1})
			} else if kind == "template" {
				f.QWithdraw(question.WithdrawalTarget{Kind: kind, ID: before.Bank.Templates[0].ID, Version: 1})
			} else if kind == "blueprint" {
				f.QWithdraw(question.WithdrawalTarget{Kind: kind, ID: before.Bank.Blueprints[0].ID, Version: 1})
			} else {
				target := publication.WithdrawalTarget{Kind: kind, ID: "workflow-fractions", Version: 1}
				if kind == "unit" {
					target.ID = "workflow-fractions-unit"
				}
				if kind == "asset" {
					target.ID = ""
					target.Version = 0
					target.SHA256 = before.Content.Assets[0].SHA256
				}
				_, e = f.repo.WithdrawVersion(f.ctx, f.Access("admin_a", true), publication.WithdrawalInput{Target: target, ExpectedHead: f.KHead(), Reason: "Isolated audit withdrawal regression."})
				if e != nil {
					t.Fatal(e)
				}
			}
			after, e := f.repo.ReadContentAudit(f.ctx, auditRoute)
			if e != nil {
				return
			}
			if kind == "blueprint" && len(after.Bank.Blueprints) != 0 {
				t.Fatal("withdrawn blueprint counted")
			}
			if kind != "blueprint" && len(after.EligibleInstances) >= len(before.EligibleInstances) {
				t.Fatal("withdrawal did not restrict accurate pool", kind, len(after.EligibleInstances))
			}
		})
	}
}
func TestContentAuditEvidenceMismatch(t *testing.T) {
	for _, kind := range []string{"body", "authors", "approval", "frozen_digest"} {
		t.Run(kind, func(t *testing.T) {
			f := auditFixture(t)
			facts, e := f.repo.ReadContentAudit(f.ctx, auditRoute)
			if e != nil {
				t.Fatal(e)
			}
			tx, e := f.db.Begin()
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback()
			if _, e = tx.Exec(`SET LOCAL session_replication_role=replica`); e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "body":
				in := facts.Bank.Instances[0]
				in.Body.Explanation += " Changed after approval."
				raw, sha, e := question.CanonicalInstance(in)
				if e != nil {
					t.Fatal(e)
				}
				_, e = tx.Exec(`UPDATE question_instances SET body=$1::jsonb,body_bytes=$2,sha256=$3 WHERE id=$4 AND version=$5`, string(raw), raw, sha, in.Identity.ID, in.Identity.Version)
				if e != nil {
					t.Fatal(e)
				}
			case "authors":
				_, e = tx.Exec(`INSERT INTO question_submission_authors(submission_id,user_id) SELECT submission_id,$1 FROM question_publication_members WHERE publication_id=$2 LIMIT 1 ON CONFLICT DO NOTHING`, f.ids["author_b"], facts.QuestionHead.ID)
				if e != nil {
					t.Fatal(e)
				}
			case "frozen_digest":
				_, e = tx.Exec(`UPDATE question_review_decisions SET frozen_digest=repeat('0',64) WHERE id IN(SELECT review_id FROM question_publication_members WHERE publication_id=$1)`, facts.QuestionHead.ID)
				if e != nil {
					t.Fatal(e)
				}
			case "approval":
				_, e = tx.Exec(`UPDATE question_review_decisions SET decision='return' WHERE id IN(SELECT review_id FROM question_publication_members WHERE publication_id=$1)`, facts.QuestionHead.ID)
				if e != nil {
					t.Fatal(e)
				}
			}
			if e = tx.Commit(); e != nil {
				t.Fatal(e)
			}
			if _, e = f.repo.ReadContentAudit(f.ctx, auditRoute); e == nil {
				t.Fatal("corrupt proof accepted", kind)
			}
		})
	}
}

func TestContentAuditRuleRestriction(t *testing.T) {
	f := auditFixture(t)
	if _, e := f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), ruleCaseInput(nil, 1)); e != nil {
		t.Fatal(e)
	}
	facts, e := f.repo.ReadContentAudit(f.ctx, auditRoute)
	if e != nil || len(facts.EligibleInstances) != 0 || len(facts.Excluded) != 28 {
		t.Fatal("grading restriction omitted", e, len(facts.EligibleInstances), len(facts.Excluded))
	}
}
func TestContentAuditLockBudget(t *testing.T) {
	f := auditFixture(t)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`LOCK TABLE question_publications IN ACCESS EXCLUSIVE MODE`); e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	_, e = f.repo.ReadContentAudit(context.Background(), auditRoute)
	if e == nil || time.Since(start) > 2*time.Second {
		t.Fatal("lock wait did not fail closed within budget", e, time.Since(start))
	}
}
