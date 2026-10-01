package store_test

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"strings"
	"testing"
)

func (f *workflowFixture) Prepare(sub publication.SubmissionView, head *string) publication.PublicationView {
	f.t.Helper()
	p, err := f.repo.PrepareRelease(f.ctx, f.Access("admin_a", false), publication.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedHead: head, Reason: "Prepare an isolated technically approved content snapshot."})
	if err != nil {
		f.t.Fatal(err)
	}
	return p
}
func (f *workflowFixture) Activate(p publication.PublicationView, head *string) publication.PublicationView {
	f.t.Helper()
	out, err := f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), p.ID, publication.ActivateInput{ExpectedHead: head, ExpectedManifestSHA: p.ManifestSHA, Reason: "Activate this isolated technical content fixture only."})
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}
func (f *workflowFixture) ApprovedInput(v publication.DraftInput) publication.SubmissionView {
	f.t.Helper()
	d, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), v)
	if err != nil {
		f.t.Fatal(err)
	}
	sub, err := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if err != nil {
		f.t.Fatal(err)
	}
	sub, err = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedReviewInput())
	if err != nil {
		f.t.Fatal(err)
	}
	return sub
}
func TestReleaseMergePreparedIsPrivate(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Approved("author_a", "reviewer_a")
	p := f.Prepare(sub, nil)
	if p.Status != "draft" || p.Manifest.BaseHead != nil || p.ManifestSHA == "" || len(p.Manifest.Members) != 3 || f.count(`SELECT count(*) FROM publication_heads`) != 0 {
		t.Fatal("prepare changed public head")
	}
	if _, err := f.repo.GetPublishedKnowledge(f.ctx, sub.Frozen.Package.Knowledge[0].ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("prepared content public")
	}
	read, err := f.repo.ReadPublication(f.ctx, f.Access("admin_a", false), p.ID)
	if err != nil || read.ManifestSHA != p.ManifestSHA {
		t.Fatal("prepared manifest changed")
	}
	page, err := f.repo.ListPublications(f.ctx, f.Access("admin_a", false), publication.ListQuery{})
	if err != nil || len(page.Items) != 1 || page.Head != nil {
		t.Fatal(page, err)
	}
	if _, err = f.repo.ActivateRelease(f.ctx, f.Access("admin_a", false), p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "No recent administrator password proof."}); !errors.Is(err, auth.ErrReauthRequired) {
		t.Fatal("activation ignored recent proof", err)
	}
	active := f.Activate(p, nil)
	if active.Status != "published" {
		t.Fatal("activation not published")
	}
	view, err := f.repo.GetPublishedKnowledge(f.ctx, sub.Frozen.Package.Knowledge[0].ID)
	if err != nil || view.Knowledge.Version != 1 || len(view.Units) != 1 {
		t.Fatal("published fixed content unavailable", err)
	}
	if _, err = f.repo.GetPublishedAsset(f.ctx, sub.Frozen.Assets[0].SHA256); err != nil {
		t.Fatal("published bound asset unavailable", err)
	}
}
func TestManifestFrozenAndLegacyEvidence(t *testing.T) {
	f := newWorkflowFixture(t)
	p := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
	for _, q := range []string{`UPDATE publication_members SET availability='withdrawn' WHERE snapshot_id=$1`, `DELETE FROM publication_members WHERE snapshot_id=$1`, `INSERT INTO publication_members SELECT * FROM publication_members WHERE snapshot_id=$1`} {
		if _, err := f.db.ExecContext(f.ctx, q, p.ID); err == nil {
			t.Fatal("manifest member mutable")
		}
	}
	f2 := newWorkflowFixture(t)
	sub := f2.Approved("author_a", "reviewer_a")
	legacy := "66666666-6666-4666-8666-666666666666"
	f2.exec(`INSERT INTO publication_snapshots VALUES($1,1,'published')`, legacy)
	f2.exec(`INSERT INTO publication_members SELECT $1,package_id,package_version,kind,id,version,'active' FROM package_members WHERE package_id='elementary-fractions'`, legacy)
	f2.exec(`INSERT INTO publication_heads VALUES(true,$1)`, legacy)
	if _, err := f2.repo.GetPublishedKnowledge(f2.ctx, "numbers"); err != nil {
		t.Fatal("legacy public fixture unreadable", err)
	}
	if _, err := f2.repo.PrepareRelease(f2.ctx, f2.Access("admin_a", false), publication.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedHead: &legacy, Reason: "Legacy unreviewed members cannot be inherited."}); !errors.Is(err, publication.ErrReviewRequired) {
		t.Fatal("legacy unreviewed members inherited", err)
	}
}
func TestReleaseRejectsOldUnitBindingInStore(t *testing.T) {
	f := newWorkflowFixture(t)
	f.Approved("author_a", "reviewer_a")
	v := f.Input()
	v.Package.Version = 2
	v.Package.Knowledge[0].Version = 2
	v.Package.Units[0].Knowledge.Version = 2
	v.Package.Assets[0].Knowledge.Version = 2
	v.SourceMap[0].Knowledge.Version = 2
	bytes, err := base64.StdEncoding.DecodeString(v.AssetBytes[0].Base64)
	if err != nil {
		t.Fatal(err)
	}
	bytes = append(bytes, '\n')
	v.AssetBytes[0].Base64 = base64.StdEncoding.EncodeToString(bytes)
	v.Package.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(bytes))
	d, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest}); !errors.Is(err, publication.ErrImmutableConflict) {
		t.Fatal("old unit rebound to new bytes", err)
	}
	pending := f.Submitted("author_a")
	if _, err = f.repo.PrepareRelease(f.ctx, f.Access("admin_a", false), publication.PrepareInput{SubmissionIDs: []string{pending.ID}, Reason: "Pending content has no independent approval."}); !errors.Is(err, publication.ErrReviewRequired) {
		t.Fatal("pending content prepared", err)
	}
}
func TestActivationReplayDoesNotRestoreOldHead(t *testing.T) {
	f := newWorkflowFixture(t)
	first := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
	a := f.Access("admin_a", true)
	command := publication.ActivateInput{ExpectedManifestSHA: first.ManifestSHA, Reason: "Activate a technical fixture with a stable operation key."}
	original, err := f.repo.ActivateRelease(f.ctx, a, first.ID, command)
	if err != nil {
		t.Fatal(err)
	}
	second := f.Prepare(f.ApprovedInput(newMaterial(f.Input())), &first.ID)
	f.Activate(second, &first.ID)
	replay, err := f.repo.ActivateRelease(f.ctx, a, first.ID, command)
	if err != nil || replay.ID != original.ID {
		t.Fatal("historical activation result unavailable", err)
	}
	var head string
	if err = f.db.QueryRowContext(f.ctx, `SELECT snapshot_id FROM publication_heads`).Scan(&head); err != nil || head != second.ID {
		t.Fatal("activation replay restored old head")
	}
	command.Reason = "A changed payload must conflict with the historical operation key."
	if _, err = f.repo.ActivateRelease(f.ctx, a, first.ID, command); !errors.Is(err, publication.ErrIdempotencyConflict) {
		t.Fatal("changed activation replay accepted", err)
	}
	if f.count(`SELECT count(*) FROM publication_snapshots WHERE status='published'`) != 2 {
		t.Fatal("historical published snapshot overwritten")
	}
}
func TestActivationRacesRevocationAndHead(t *testing.T) {
	t.Run("reviewer revoked", func(t *testing.T) {
		f := newWorkflowFixture(t)
		p := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
		admin := f.Access("admin_a", true)
		if _, err := f.repo.ReplaceAccountRoles(f.ctx, auth.SessionProof{TokenHash: admin.TokenHash, CSRF: admin.CSRF}, f.ids["reviewer_a"], auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "revoke-before-activation"); err != nil {
			t.Fatal(err)
		}
		if _, err := f.repo.ActivateRelease(f.ctx, admin, p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "The reviewer no longer has current approval eligibility."}); !errors.Is(err, publication.ErrReviewRequired) {
			t.Fatal("revoked reviewer activated", err)
		}
		if f.count(`SELECT count(*) FROM publication_heads`) != 0 {
			t.Fatal("revoked approval changed head")
		}
	})
	t.Run("admin revoked", func(t *testing.T) {
		f := newWorkflowFixture(t)
		p := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
		a := f.Access("admin_a", true)
		f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'`, f.ids["admin_a"])
		if _, err := f.repo.ActivateRelease(f.ctx, a, p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "A revoked actor must not activate this snapshot."}); !errors.Is(err, auth.ErrForbidden) {
			t.Fatal("revoked administrator activated", err)
		}
	})
	t.Run("competing releases", func(t *testing.T) {
		f := newWorkflowFixture(t)
		sub := f.Approved("author_a", "reviewer_a")
		one, two := f.Prepare(sub, nil), f.Prepare(sub, nil)
		gate := holdAdminLock(t, f.db)
		results := make(chan error, 2)
		for _, p := range []publication.PublicationView{one, two} {
			a := f.Access("admin_a", true)
			go func() {
				_, err := f.repo.ActivateRelease(f.ctx, a, p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "Race two candidates prepared from the same empty head."})
				results <- err
			}()
		}
		waitForAdminLockWaiters(t, f.db, 2)
		if err := gate.Commit(); err != nil {
			t.Fatal(err)
		}
		success, stale := 0, 0
		for range 2 {
			err := <-results
			if err == nil {
				success++
			} else if errors.Is(err, publication.ErrPublicationStale) {
				stale++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || stale != 1 || f.count(`SELECT count(*) FROM content_workflow_events WHERE action='activateRelease'`) != 1 {
			t.Fatal("competing activation changed head twice")
		}
	})
	t.Run("failed audit", func(t *testing.T) {
		f := newWorkflowFixture(t)
		p := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
		f.exec(`CREATE FUNCTION isolated_activation_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='activateRelease' THEN RAISE EXCEPTION 'isolated forced audit failure'; END IF;RETURN NEW;END $$`)
		f.exec(`CREATE TRIGGER isolated_activation_audit BEFORE INSERT ON content_workflow_events FOR EACH ROW EXECUTE FUNCTION isolated_activation_audit_failure()`)
		if _, err := f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "Atomic activation must roll back a failed audit."}); err == nil {
			t.Fatal("audit failure activated")
		}
		if f.count(`SELECT count(*) FROM publication_heads`) != 0 || f.count(`SELECT count(*) FROM publication_snapshots WHERE id=$1 AND status='draft'`, p.ID) != 1 {
			t.Fatal("failed activation left a partial head")
		}
	})
}

func TestActivationInheritedApprovalSurvivesReviewerRoleLoss(t *testing.T) {
	f := newWorkflowFixture(t)
	first := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
	f.Activate(first, nil)
	admin := f.Access("admin_a", true)
	if _, err := f.repo.ReplaceAccountRoles(f.ctx, auth.SessionProof{TokenHash: admin.TokenHash, CSRF: admin.CSRF}, f.ids["reviewer_a"], auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "revoke-historical-reviewer"); err != nil {
		t.Fatal(err)
	}
	input := f.Input()
	input.Package.ID = "second-package"
	input.Package.Knowledge[0].ID = "second-knowledge"
	input.Package.Units[0].ID = "second-unit"
	input.Package.Units[0].Knowledge.ID = "second-knowledge"
	input.Package.Units[0].AssetIDs = []string{"second-asset"}
	for i := range input.Package.Units[0].Angles {
		input.Package.Units[0].Angles[i].Body = strings.ReplaceAll(input.Package.Units[0].Angles[i].Body, "asset:halves", "asset:second-asset")
	}
	input.Package.Assets[0].ID = "second-asset"
	input.Package.Assets[0].Knowledge.ID = "second-knowledge"
	input.AssetBytes[0].ID = "second-asset"
	input.SourceMap[0].Knowledge.ID = "second-knowledge"
	d, err := f.repo.CreateDraft(f.ctx, f.Access("author_b", false), input)
	if err != nil {
		t.Fatal(err)
	}
	sub, err := f.repo.SubmitDraft(f.ctx, f.Access("author_b", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if err != nil {
		t.Fatal(err)
	}
	sub, err = f.repo.DecideReview(f.ctx, f.Access("reviewer_b", false), sub.ID, approvedReviewInput())
	if err != nil {
		t.Fatal(err)
	}
	second := f.Prepare(sub, &first.ID)
	inherited := 0
	for _, m := range second.Manifest.Members {
		if m.Evidence.InheritedFrom != nil {
			inherited++
			if *m.Evidence.InheritedFrom != first.ID {
				t.Fatal("inheritance points to ancestor instead of current head")
			}
		}
	}
	if inherited != 3 {
		t.Fatal("published evidence not retained", inherited)
	}
	f.Activate(second, &first.ID)
	if _, err = f.repo.GetPublishedKnowledge(f.ctx, "workflow-fractions"); err != nil {
		t.Fatal("historically approved member removed", err)
	}
}

func TestActivationReviewerRevocationAtDatabaseBarrier(t *testing.T) {
	for _, revocationFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("revocationFirst=%v", revocationFirst), func(t *testing.T) {
			f := newWorkflowFixture(t)
			p := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
			activateAccess, revokeAccess := f.Access("admin_a", true), f.Access("admin_a", true)
			gate := holdAdminLock(t, f.db)
			activation, revocation := make(chan error, 1), make(chan error, 1)
			activate := func() {
				_, err := f.repo.ActivateRelease(f.ctx, activateAccess, p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "Race activation against an actual administrator role revocation."})
				activation <- err
			}
			revoke := func() {
				_, err := f.repo.ReplaceAccountRoles(f.ctx, auth.SessionProof{TokenHash: revokeAccess.TokenHash, CSRF: revokeAccess.CSRF}, f.ids["reviewer_a"], auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "reviewer-activation-race")
				revocation <- err
			}
			if revocationFirst {
				go revoke()
			} else {
				go activate()
			}
			waitForAdminLockWaiters(t, f.db, 1)
			if revocationFirst {
				go activate()
			} else {
				go revoke()
			}
			waitForAdminLockWaiters(t, f.db, 2)
			if err := gate.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-revocation; err != nil {
				t.Fatal(err)
			}
			err := <-activation
			if revocationFirst {
				if !errors.Is(err, publication.ErrReviewRequired) || f.count(`SELECT count(*) FROM publication_heads`) != 0 {
					t.Fatal("revocation winner did not prevent activation", err)
				}
			} else {
				if err != nil || f.count(`SELECT count(*) FROM publication_heads WHERE snapshot_id=$1`, p.ID) != 1 {
					t.Fatal("activation winner lost historical publication", err)
				}
			}
		})
	}
}

func TestReleaseReplacesSVGInStore(t *testing.T) {
	f := newWorkflowFixture(t)
	first := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
	f.Activate(first, nil)
	input := f.Input()
	input.Package.Version = 2
	input.Package.Knowledge[0].Version = 2
	input.Package.Units[0].Version = 2
	input.Package.Units[0].Knowledge.Version = 2
	input.Package.Assets[0].Knowledge.Version = 2
	input.SourceMap[0].Knowledge.Version = 2
	b, err := base64.StdEncoding.DecodeString(input.AssetBytes[0].Base64)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	input.AssetBytes[0].Base64 = base64.StdEncoding.EncodeToString(b)
	input.Package.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(b))
	next := f.Prepare(f.ApprovedInput(input), &first.ID)
	f.Activate(next, &first.ID)
	view, err := f.repo.GetPublishedKnowledge(f.ctx, input.Package.Knowledge[0].ID)
	if err != nil || view.Knowledge.Version != 2 || view.Units[0].Version != 2 || view.Assets[0].SHA256 != input.Package.Assets[0].SHA256 {
		t.Fatal("same-ID illustration revision not published", err)
	}
	old, err := f.repo.ReadPublication(f.ctx, f.Access("admin_a", false), first.ID)
	if err != nil || old.ManifestSHA != first.ManifestSHA {
		t.Fatal("previous publication changed", err)
	}
}
