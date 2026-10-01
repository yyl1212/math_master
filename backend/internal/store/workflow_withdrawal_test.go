package store_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"testing"
)

func workflowWithdrawalInput(f *workflowFixture) publication.DraftInput {
	input := f.Input()
	raw, _ := json.Marshal(input.Package.Knowledge[0])
	var child, related content.Knowledge
	_ = json.Unmarshal(raw, &child)
	_ = json.Unmarshal(raw, &related)
	ref := content.VersionRef{ID: input.Package.Knowledge[0].ID, Version: 1}
	child.ID = "workflow-dependent"
	child.Relations = []content.Relation{{Kind: "prerequisite", Target: ref}}
	related.ID = "workflow-related"
	related.Relations = []content.Relation{{Kind: "related", Target: ref}}
	input.Package.Knowledge = append(input.Package.Knowledge, child, related)
	for _, id := range []string{child.ID, related.ID} {
		u := content.Unit{ID: id + "-unit", Version: 1, Knowledge: content.VersionRef{ID: id, Version: 1}, Angles: []content.Angle{{Kind: "formal", Body: "An original formal technical explanation."}, {Kind: "intuitive", Body: "An original intuitive technical explanation."}}, Examples: []string{"A technical example."}, Counterexamples: []string{}, AssetIDs: []string{}}
		input.Package.Units = append(input.Package.Units, u)
	}
	input.Package.Paths = []content.Path{{ID: "workflow-path", Version: 1, DomainIDs: child.DomainIDs, Title: "Technical root to dependent", TitleZh: "路线", Nodes: []content.VersionRef{ref, {ID: child.ID, Version: 1}}}}
	return input
}
func workflowWithdraw(f *workflowFixture, target publication.WithdrawalTarget, head *string) publication.WithdrawalResult {
	f.t.Helper()
	out, err := f.repo.WithdrawVersion(f.ctx, f.Access("admin_a", true), publication.WithdrawalInput{Target: target, ExpectedHead: head, Reason: "Withdraw an isolated technical fixed-version fixture only."})
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}
func TestWithdrawalVersionLedger(t *testing.T) {
	t.Run("closure and blacklist", func(t *testing.T) {
		f := newWorkflowFixture(t)
		sub := f.ApprovedInput(workflowWithdrawalInput(f))
		p := f.Prepare(sub, nil)
		f.Activate(p, nil)
		candidate := f.Prepare(sub, &p.ID)
		target := publication.WithdrawalTarget{Kind: "knowledge", ID: "workflow-fractions", Version: 1}
		events := f.count(`SELECT count(*) FROM content_workflow_events`)
		preview, err := f.repo.PreviewWithdrawal(f.ctx, f.Access("admin_a", false), publication.WithdrawalPreviewInput{Target: target})
		if err != nil || preview.Diff.Removed != 6 || preview.CurrentHead == nil || *preview.CurrentHead != p.ID {
			t.Fatal(preview, err)
		}
		if f.count(`SELECT count(*) FROM content_workflow_events`) != events || f.count(`SELECT count(*) FROM content_withdrawals`) != 0 {
			t.Fatal("preview mutated state")
		}
		result := workflowWithdraw(f, target, &p.ID)
		if result.Publication.Diff.Removed != 6 || result.PreviousHead == nil || *result.PreviousHead != p.ID {
			t.Fatal("closure missing from derived diff")
		}
		if f.count(`SELECT count(*) FROM content_withdrawals`) != 1 || f.count(`SELECT count(*) FROM content_withdrawals WHERE target_id='workflow-dependent'`) != 0 {
			t.Fatal("dependent versions permanently blacklisted")
		}
		if _, err = f.repo.GetPublishedKnowledge(f.ctx, "workflow-dependent"); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("dependent still public", err)
		}
		v, err := f.repo.GetPublishedKnowledge(f.ctx, "workflow-related")
		if err != nil || len(v.Knowledge.Relations) != 0 {
			t.Fatal("related edge not filtered or node incorrectly removed", err)
		}
		if _, err = f.repo.GetPublishedAsset(f.ctx, sub.Frozen.Assets[0].SHA256); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("withdrawn SVG still public", err)
		}
		if _, err = f.repo.GetPublishedPath(f.ctx, "workflow-path"); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("impacted path still public", err)
		}
		if _, err = f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), candidate.ID, publication.ActivateInput{ExpectedHead: &p.ID, ExpectedManifestSHA: candidate.ManifestSHA, Reason: "Old prepared snapshots must not restore withdrawn content."}); !errors.Is(err, publication.ErrPublicationStale) {
			t.Fatal("stale candidate restored withdrawal", err)
		}
		if _, err = f.repo.PrepareRelease(f.ctx, f.Access("admin_a", false), publication.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedHead: &result.Publication.ID, Reason: "Permanently blacklisted fixed content cannot be prepared again."}); !errors.Is(err, publication.ErrContentInvalid) {
			t.Fatal("blacklisted fixed version restored", err)
		}
		if _, err = f.repo.WithdrawVersion(f.ctx, f.Access("admin_a", true), publication.WithdrawalInput{Target: target, ExpectedHead: &result.Publication.ID, Reason: "Duplicate fixed-target withdrawals must not create another event."}); !errors.Is(err, publication.ErrImmutableConflict) {
			t.Fatal("duplicate withdrawal accepted", err)
		}
		for _, q := range []string{`UPDATE content_withdrawals SET reason='altered permanent withdrawal reason'`, `DELETE FROM content_withdrawals`} {
			if _, err = f.db.ExecContext(f.ctx, q); err == nil {
				t.Fatal("withdrawal ledger mutable")
			}
		}
	})
	t.Run("unpublished approved target and no head", func(t *testing.T) {
		f := newWorkflowFixture(t)
		sub := f.Approved("author_a", "reviewer_a")
		p := f.Prepare(sub, nil)
		result := workflowWithdraw(f, publication.WithdrawalTarget{Kind: "knowledge", ID: "workflow-fractions", Version: 1}, nil)
		if result.Publication.Status != "published" || len(result.Publication.Manifest.Members) != 0 || result.Publication.Manifest.CatalogueVersion != 1 {
			t.Fatal("no-head withdrawal did not create an empty catalogue snapshot")
		}
		if _, err := f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "Unpublished withdrawal must invalidate an already prepared snapshot."}); !errors.Is(err, publication.ErrPublicationStale) {
			t.Fatal(err)
		}
	})
}
func TestWithdrawalRollbackAndHistory(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Approved("author_a", "reviewer_a")
	p := f.Prepare(sub, nil)
	f.Activate(p, nil)
	f.exec(`CREATE FUNCTION isolated_withdrawal_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='withdrawVersion' THEN RAISE EXCEPTION 'isolated forced withdrawal failure'; END IF;RETURN NEW;END $$`)
	f.exec(`CREATE TRIGGER isolated_withdrawal_failure BEFORE INSERT ON content_workflow_events FOR EACH ROW EXECUTE FUNCTION isolated_withdrawal_failure()`)
	target := publication.WithdrawalTarget{Kind: "unit", ID: "workflow-fractions-unit", Version: 1}
	if _, err := f.repo.WithdrawVersion(f.ctx, f.Access("admin_a", true), publication.WithdrawalInput{Target: target, ExpectedHead: &p.ID, Reason: "Late audit failure must roll back all withdrawal state."}); err == nil {
		t.Fatal("late failure committed")
	}
	if f.count(`SELECT count(*) FROM content_withdrawals`) != 0 || f.count(`SELECT count(*) FROM publication_heads WHERE snapshot_id=$1`, p.ID) != 1 || f.count(`SELECT count(*) FROM content_publication_manifests`) != 1 {
		t.Fatal("withdrawal left partial state")
	}
	f.exec(`DROP TRIGGER isolated_withdrawal_failure ON content_workflow_events`)
	result := workflowWithdraw(f, target, &p.ID)
	if len(result.Publication.Manifest.Members) != 0 || f.count(`SELECT count(*) FROM knowledge_versions WHERE id='workflow-fractions'`) != 1 || f.count(`SELECT count(*) FROM content_review_decisions WHERE submission_id=$1`, sub.ID) != 1 || f.count(`SELECT count(*) FROM publication_members WHERE snapshot_id=$1`, p.ID) != 3 {
		t.Fatal("withdrawal erased historical versions, evidence or members")
	}
	if _, err := f.repo.GetPublishedKnowledge(f.ctx, "workflow-fractions"); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("paused knowledge still available")
	}
}
func TestPublicationWithdrawalRace(t *testing.T) {
	for _, withdrawFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("withdrawalFirst=%v", withdrawFirst), func(t *testing.T) {
			f := newWorkflowFixture(t)
			first := f.Prepare(f.Approved("author_a", "reviewer_a"), nil)
			f.Activate(first, nil)
			second := f.Prepare(f.ApprovedInput(newMaterial(f.Input())), &first.ID)
			activateAccess, withdrawAccess := f.Access("admin_a", true), f.Access("admin_a", true)
			expected := first.ID
			if !withdrawFirst {
				expected = second.ID
			}
			command := publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "knowledge", ID: "workflow-fractions", Version: 1}, ExpectedHead: &expected, Reason: "Race fixed-version withdrawal against an actual publication transaction."}
			gate := holdAdminLock(t, f.db)
			activation := make(chan error, 1)
			withdrawal := make(chan error, 1)
			results := make(chan publication.WithdrawalResult, 1)
			activate := func() {
				_, err := f.repo.ActivateRelease(f.ctx, activateAccess, second.ID, publication.ActivateInput{ExpectedHead: &first.ID, ExpectedManifestSHA: second.ManifestSHA, Reason: "Activate a candidate competing with the withdrawal transaction."})
				activation <- err
			}
			withdraw := func() {
				out, err := f.repo.WithdrawVersion(f.ctx, withdrawAccess, command)
				results <- out
				withdrawal <- err
			}
			if withdrawFirst {
				go withdraw()
			} else {
				go activate()
			}
			waitForAdminLockWaiters(t, f.db, 1)
			if withdrawFirst {
				go activate()
			} else {
				go withdraw()
			}
			waitForAdminLockWaiters(t, f.db, 2)
			if err := gate.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-withdrawal; err != nil {
				t.Fatal(err)
			}
			result := <-results
			err := <-activation
			if withdrawFirst {
				if !errors.Is(err, publication.ErrPublicationStale) {
					t.Fatal("old activation restored problem version", err)
				}
			} else {
				if err != nil || result.PreviousHead == nil || *result.PreviousHead != second.ID {
					t.Fatal("withdrawal did not derive from winning publication", err)
				}
			}
			newerSub := f.ApprovedInput(newMaterial(newMaterial(f.Input())))
			newer := f.Prepare(newerSub, &result.Publication.ID)
			f.Activate(newer, &result.Publication.ID)
			replay, err := f.repo.WithdrawVersion(f.ctx, withdrawAccess, command)
			if err != nil || replay.EventID != result.EventID || f.count(`SELECT count(*) FROM publication_heads WHERE snapshot_id=$1`, newer.ID) != 1 {
				t.Fatal("historical withdrawal replay restored previous head", err)
			}
		})
	}
}

func TestWithdrawalFixedUnitAssetAndPathPublicViews(t *testing.T) {
	for _, kind := range []string{"unit", "asset", "path"} {
		t.Run(kind, func(t *testing.T) {
			f := newWorkflowFixture(t)
			sub := f.ApprovedInput(workflowWithdrawalInput(f))
			p := f.Prepare(sub, nil)
			f.Activate(p, nil)
			target := publication.WithdrawalTarget{Kind: kind, ID: "workflow-fractions-unit", Version: 1}
			if kind == "path" {
				target.ID = "workflow-path"
			}
			if kind == "asset" {
				target = publication.WithdrawalTarget{Kind: "asset", SHA256: sub.Frozen.Assets[0].SHA256}
			}
			result := workflowWithdraw(f, target, &p.ID)
			_, err := f.repo.GetPublishedKnowledge(f.ctx, "workflow-fractions")
			if kind == "path" {
				if err != nil || result.Publication.Diff.Removed != 1 {
					t.Fatal("path-only withdrawal paused knowledge", err)
				}
			} else {
				if !errors.Is(err, store.ErrNotFound) || result.Publication.Diff.Removed != 6 {
					t.Fatal("unit or SVG withdrawal did not pause closure", err)
				}
			}
			if _, err = f.repo.GetPublishedPath(f.ctx, "workflow-path"); !errors.Is(err, store.ErrNotFound) {
				t.Fatal("withdrawn path available", err)
			}
		})
	}
}
