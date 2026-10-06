package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"testing"
)

func (f *topicWorkflowFixture) topicApproved(topic string) publication.SubmissionView {
	f.t.Helper()
	d := f.draft()
	m := f.member
	m.TopicIDs = []string{topic}
	if _, e := f.repo.SaveDraftTopics(f.ctx, f.Access("author_a", false), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: 1, TaxonomyVersionID: f.v.ID, Member: m}); e != nil {
		f.t.Fatal(e)
	}
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		f.t.Fatal(e)
	}
	sub, e = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedReviewInput())
	if e != nil {
		f.t.Fatal(e)
	}
	return sub
}
func (f *topicWorkflowFixture) preparePair(pair taxonomy.PairRef, ids ...string) taxonomy.ReleaseView {
	f.t.Helper()
	p, e := f.repo.PrepareTopicRelease(f.ctx, f.Access("admin_a", false), taxonomy.PrepareInput{SubmissionIDs: ids, ExpectedPair: pair, Reason: "Prepare only an isolated reviewed topic fixture."})
	if e != nil {
		f.t.Fatal(e)
	}
	return p
}
func (f *topicWorkflowFixture) activatePair(p taxonomy.ReleaseView) taxonomy.ReleaseView {
	f.t.Helper()
	v, e := f.repo.ActivateTopicRelease(f.ctx, f.Access("admin_a", true), p.ID, taxonomy.ActivateInput{ExpectedPair: p.Pair, ManifestSHA: p.ManifestSHA, Reason: "Activate only this isolated topic fixture."})
	if e != nil {
		f.t.Fatal(e)
	}
	return v
}
func (f *topicWorkflowFixture) pairHeads() (string, string) {
	var kh, th sql.NullString
	if e := f.db.QueryRow("SELECT (SELECT snapshot_id FROM publication_heads),(SELECT release_id::text FROM taxonomy_heads)").Scan(&kh, &th); e != nil {
		f.t.Fatal(e)
	}
	return kh.String, th.String
}
func (f *topicWorkflowFixture) initialPair() taxonomy.PairRef {
	return taxonomy.PairRef{TaxonomyVersionID: f.v.ID}
}
func TestTopicPairAtomicFailure(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	kh, th := f.pairHeads()
	f.exec(`CREATE FUNCTION reject_pair_head() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated fault'; END $$; CREATE TRIGGER reject_pair_head BEFORE INSERT ON taxonomy_heads FOR EACH ROW EXECUTE FUNCTION reject_pair_head()`)
	_, e := f.repo.ActivateTopicRelease(f.ctx, f.Access("admin_a", true), p.ID, taxonomy.ActivateInput{ExpectedPair: p.Pair, ManifestSHA: p.ManifestSHA, Reason: "Inject a topic head fault after the knowledge head write."})
	akh, ath := f.pairHeads()
	if e == nil || kh != akh || th != ath || f.count("SELECT count(*) FROM publication_snapshots WHERE status='published'") != 0 {
		t.Fatal("partial publication", e)
	}
}
func TestTopicPairEmptyCatalogue(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair())
	f.activatePair(p)
	kh, th := f.pairHeads()
	if kh != "" || th != p.ID {
		t.Fatal("empty catalogue manufactured knowledge")
	}
	page, e := f.repo.ListTopics(f.ctx, taxonomy.Query{Limit: 100})
	if e != nil || page.Total != 63 || len(page.Items) != 63 {
		t.Fatal(page, e)
	}
}
func TestTopicPairStaleHead(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	sub := f.topicApproved("msc-00a00")
	p, q := f.preparePair(f.initialPair(), sub.ID), f.preparePair(f.initialPair(), sub.ID)
	f.activatePair(p)
	_, e := f.repo.ActivateTopicRelease(f.ctx, f.Access("admin_a", true), q.ID, taxonomy.ActivateInput{ExpectedPair: q.Pair, ManifestSHA: q.ManifestSHA, Reason: "A competing publication invalidates both expected heads."})
	if !errors.Is(e, publication.ErrPublicationStale) {
		t.Fatal("stale pair activated", e)
	}
	kh, th := f.pairHeads()
	if kh != *p.KnowledgePublicationID || th != p.ID {
		t.Fatal("stale pair changed heads")
	}
}
func TestTopicLegacyActivationCannotBypassPair(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	f.activatePair(p)
	old := f.Prepare(f.Approved("author_a", "reviewer_a"), p.KnowledgePublicationID)
	_, e := f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), old.ID, publication.ActivateInput{ExpectedHead: p.KnowledgePublicationID, ExpectedManifestSHA: old.ManifestSHA, Reason: "Legacy activation cannot move the knowledge pointer independently."})
	if !errors.Is(e, publication.ErrPublicationStale) {
		t.Fatal("legacy activation bypass", e)
	}
}
func TestTopicLegacyActivationBeforePairedPublication(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	// The study capability is installed before the experience is switched.
	// It cannot masquerade as a previously published taxonomy head.
	f.exec("UPDATE topic_learning_state SET study_enabled=true WHERE singleton")
	old := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
	_, e := f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), old.ID, publication.ActivateInput{ExpectedHead: nil, ExpectedManifestSHA: old.ManifestSHA, Reason: "Installed study capability keeps healthy legacy publication available until a pair is published."})
	if e != nil {
		t.Fatal("study capability alone retired legacy activation", e)
	}
}
func TestTopicLegacyActivationMissingHeadFailsClosed(t *testing.T) {
	for _, missing := range []string{"table", "row"} {
		t.Run(missing, func(t *testing.T) {
			f := newTopicWorkflowFixture(t)
			p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
			f.activatePair(p)
			old := f.Prepare(f.Approved("author_a", "reviewer_a"), p.KnowledgePublicationID)
			if missing == "table" {
				f.exec("ALTER TABLE taxonomy_heads RENAME TO hidden_taxonomy_heads")
			} else {
				f.exec("DELETE FROM taxonomy_heads")
			}
			_, e := f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), old.ID, publication.ActivateInput{ExpectedHead: p.KnowledgePublicationID, ExpectedManifestSHA: old.ManifestSHA, Reason: "Damaged taxonomy heads must not permit independent activation."})
			var kh string
			if scan := f.db.QueryRow("SELECT snapshot_id FROM publication_heads WHERE singleton").Scan(&kh); scan != nil {
				t.Fatal(scan)
			}
			if !errors.Is(e, taxonomy.ErrNotConfigured) || kh != *p.KnowledgePublicationID {
				t.Fatal("damaged taxonomy allowed independent activation", e, kh)
			}
		})
	}
}
func TestTopicWithdrawalKeepsHistory(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	sub := f.topicApproved("msc-00a00")
	p := f.preparePair(f.initialPair(), sub.ID)
	f.activatePair(p)
	result := workflowWithdraw(f.workflowFixture, publication.WithdrawalTarget{Kind: "knowledge", ID: f.member.Knowledge.ID, Version: 1}, p.KnowledgePublicationID)
	kh, th := f.pairHeads()
	if kh != result.Publication.ID || th == p.ID {
		t.Fatal("withdrawal did not pair")
	}
	if f.count("SELECT count(*) FROM taxonomy_release_assignments WHERE release_id='"+p.ID+"'") != 1 || f.count("SELECT count(*) FROM taxonomy_release_assignments WHERE release_id='"+th+"'") != 0 {
		t.Fatal("history erased or current member retained")
	}
	if _, e := f.repo.GetPublishedKnowledge(f.ctx, f.member.Knowledge.ID); !errors.Is(e, store.ErrNotFound) {
		t.Fatal(e)
	}
	if _, e := f.repo.ReadSubmissionTopics(f.ctx, f.Access("reviewer_a", false), sub.ID); e != nil {
		t.Fatal("history lost", e)
	}
}
func TestTopicPairClassificationKeepsKnowledgeSHA(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	f.activatePair(p)
	before, e := f.repo.ListTopicKnowledge(f.ctx, "msc-00a00", taxonomy.Query{})
	if e != nil {
		t.Fatal(e)
	}
	pair := taxonomy.PairRef{KnowledgeHead: p.KnowledgePublicationID, TaxonomyHead: &p.ID, TaxonomyVersionID: f.v.ID}
	q := f.preparePair(pair, f.topicApproved("msc-00a01").ID)
	f.activatePair(q)
	after, e := f.repo.ListTopicKnowledge(f.ctx, "msc-00a01", taxonomy.Query{})
	if e != nil || len(after.Items) != 1 || after.Items[0].SHA256 != before.Items[0].SHA256 {
		t.Fatal("classification rewrote mathematics", e)
	}
	if q.KnowledgePublicationID == nil || *q.KnowledgePublicationID != *p.KnowledgePublicationID || len(q.Diff.ChangedTopicMemberships) != 1 || len(q.Diff.Added) != 0 || len(q.Diff.Removed) != 0 {
		t.Fatal("classification-only publication changed knowledge", q)
	}
}
func TestTopicPairRevocationRace(t *testing.T) {
	for _, first := range []bool{true, false} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			f := newTopicWorkflowFixture(t)
			p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
			a, r := f.Access("admin_a", true), f.Access("admin_a", true)
			gate := holdAdminLock(t, f.db)
			ac, rc := make(chan error, 1), make(chan error, 1)
			activate := func() {
				_, e := f.repo.ActivateTopicRelease(f.ctx, a, p.ID, taxonomy.ActivateInput{ExpectedPair: p.Pair, ManifestSHA: p.ManifestSHA, Reason: "Race paired publication against real reviewer role revocation."})
				ac <- e
			}
			revoke := func() {
				_, e := f.repo.ReplaceAccountRoles(f.ctx, auth.SessionProof{TokenHash: r.TokenHash, CSRF: r.CSRF}, f.ids["reviewer_a"], auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "topic-reviewer-race")
				rc <- e
			}
			if first {
				go revoke()
			} else {
				go activate()
			}
			waitForAdminLockWaiters(t, f.db, 1)
			if first {
				go activate()
			} else {
				go revoke()
			}
			waitForAdminLockWaiters(t, f.db, 2)
			if e := gate.Commit(); e != nil {
				t.Fatal(e)
			}
			if e := <-rc; e != nil {
				t.Fatal(e)
			}
			e := <-ac
			kh, th := f.pairHeads()
			if first {
				if !errors.Is(e, publication.ErrReviewRequired) || kh != "" || th != "" {
					t.Fatal("revocation did not block pair", e)
				}
			} else if e != nil || kh != *p.KnowledgePublicationID || th != p.ID {
				t.Fatal("activation winner lost", e)
			}
		})
	}
}
func TestTopicWithdrawalMissingTaxonomyFailsClosed(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	f.activatePair(p)
	kh, th := f.pairHeads()
	f.exec("DROP TABLE taxonomy_nodes CASCADE")
	_, e := f.repo.WithdrawVersion(f.ctx, f.Access("admin_a", true), publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "knowledge", ID: f.member.Knowledge.ID, Version: 1}, ExpectedHead: p.KnowledgePublicationID, Reason: "A paired publication must fail closed if its new schema is damaged."})
	akh, ath := f.pairHeads()
	if !errors.Is(e, taxonomy.ErrNotConfigured) || kh != akh || th != ath {
		t.Fatal("damaged taxonomy allowed partial withdrawal", e)
	}
}
func TestTopicPairReplayDoesNotRestoreOldHead(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	a := f.Access("admin_a", true)
	in := taxonomy.ActivateInput{ExpectedPair: p.Pair, ManifestSHA: p.ManifestSHA, Reason: "An idempotent historical receipt must never move current heads."}
	if _, e := f.repo.ActivateTopicRelease(f.ctx, a, p.ID, in); e != nil {
		t.Fatal(e)
	}
	workflowWithdraw(f.workflowFixture, publication.WithdrawalTarget{Kind: "knowledge", ID: f.member.Knowledge.ID, Version: 1}, p.KnowledgePublicationID)
	kh, th := f.pairHeads()
	if _, e := f.repo.ActivateTopicRelease(f.ctx, a, p.ID, in); e != nil {
		t.Fatal(e)
	}
	akh, ath := f.pairHeads()
	if kh != akh || th != ath {
		t.Fatal("replay restored old pair")
	}
}
func TestTopicPairCurrentSelfReviewLosesAdmin(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	f.activatePair(p)
	f.exec("INSERT INTO auth_user_roles(user_id,role) VALUES($1,'admin'),($1,'reviewer')", f.ids["author_a"])
	d := f.draft()
	member := f.member
	member.TopicIDs = []string{"msc-00a01"}
	if _, e := f.repo.SaveDraftTopics(f.ctx, f.Access("author_a", false), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: 1, TaxonomyVersionID: f.v.ID, Member: member}); e != nil {
		t.Fatal(e)
	}
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal(e)
	}
	sub, e = f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, approvedReviewInput())
	if e != nil {
		t.Fatal(e)
	}
	q := f.preparePair(taxonomy.PairRef{KnowledgeHead: p.KnowledgePublicationID, TaxonomyHead: &p.ID, TaxonomyVersionID: f.v.ID}, sub.ID)
	f.exec("DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'", f.ids["author_a"])
	if _, e = f.repo.ActivateTopicRelease(f.ctx, f.Access("admin_a", true), q.ID, taxonomy.ActivateInput{ExpectedPair: q.Pair, ManifestSHA: q.ManifestSHA, Reason: "Classification-only self-review must retain current administrator eligibility."}); !errors.Is(e, publication.ErrReviewRequired) {
		t.Fatal("self-review bypassed current admin check", e)
	}
}
func TestTopicReleaseHistoryPaged(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair())
	f.activatePair(p)
	q := f.preparePair(taxonomy.PairRef{KnowledgeHead: p.KnowledgePublicationID, TaxonomyHead: &p.ID, TaxonomyVersionID: f.v.ID})
	f.activatePair(q)
	page, e := f.repo.ListTopicReleases(f.ctx, f.Access("admin_a", false), taxonomy.Query{Limit: 1})
	if e != nil || page.Total != 2 || len(page.Items) != 1 || page.Limit != 1 || page.Pair.TaxonomyHead == nil || *page.Pair.TaxonomyHead != q.ID {
		t.Fatal(page, e)
	}
}
func TestTaxonomySearchKnowledgeTitle(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	f.activatePair(p)
	page, e := f.repo.ListTopics(f.ctx, taxonomy.Query{Q: f.input.Package.Knowledge[0].TitleZh, Level: 3})
	if e != nil || len(page.Items) != 1 || page.Items[0].ID != "msc-00a00" {
		t.Fatal("knowledge title missing from topic search", page, e)
	}
}
func TestTaxonomyNonPrimaryDefaultLevel(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair())
	f.activatePair(p)
	for _, item := range []struct {
		kind  string
		total int
	}{{"auxiliary", 503}, {"other", 534}} {
		page, e := f.repo.ListTopics(f.ctx, taxonomy.Query{Kind: item.kind})
		if e != nil || page.Total != item.total {
			t.Fatal("wrong category default", page, e)
		}
	}
}
