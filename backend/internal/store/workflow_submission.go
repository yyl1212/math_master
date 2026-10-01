package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"time"
)

func workflowStoredAssets(ctx context.Context, tx *sql.Tx, workspace string, p content.Package) (map[string][]byte, error) {
	out := map[string][]byte{}
	total := 0
	rows, err := tx.QueryContext(ctx, `SELECT asset_id,sha256,bytes FROM content_workspace_assets WHERE workspace_id=$1 ORDER BY asset_id`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	expected := map[string]string{}
	for _, a := range p.Assets {
		expected[a.ID] = a.SHA256
	}
	for rows.Next() {
		var id, sha string
		var b []byte
		if err = rows.Scan(&id, &sha, &b); err != nil {
			return nil, err
		}
		total += len(b)
		if total > 4<<20 {
			return nil, publication.ErrContentLimitExceeded
		}
		if expected[id] != sha {
			return nil, publication.ErrContentInvalid
		}
		out[id] = b
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(out) != len(p.Assets) {
		return nil, publication.ErrContentInvalid
	}
	return out, nil
}
func (s *Store) ValidateDraft(ctx context.Context, a publication.Access, id string, input publication.ValidateInput) (publication.GateReport, error) {
	var out publication.GateReport
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowReadTx(ctx, a, publication.ValidateDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		d, _, err := s.readWorkflowDraft(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		if input.ExpectedRevision != d.Revision || d.Status != "editing" {
			return publication.ErrDraftConflict
		}
		c, _, err := workflowCatalogue(ctx, tx, d.CatalogueVersion)
		if err != nil {
			return err
		}
		bytes, err := workflowStoredAssets(ctx, tx, id, d.Package)
		if err != nil {
			return err
		}
		_, report := content.ValidateWorkflow(ctx, c, d.Package, workflowMemoryAssets(bytes))
		out = publication.GateFromReport(report)
		out.Digest = d.Gate.Digest
		return ctx.Err()
	})
	return out, err
}
func workflowAssetVersionChange(ctx context.Context, tx *sql.Tx, p content.Package) error {
	for _, a := range p.Assets {
		var conflict bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM package_members m JOIN imported_packages p ON p.id=m.package_id AND p.version=m.package_version CROSS JOIN LATERAL jsonb_array_elements(p.body->'assets') old WHERE m.kind='asset' AND m.id=$1 AND m.asset_sha256<>$2 AND old->>'id'=m.id AND old->'knowledge'->>'id'=$3 AND (old->'knowledge'->>'version')::integer=$4)`, a.ID, a.SHA256, a.Knowledge.ID, a.Knowledge.Version).Scan(&conflict)
		if err != nil {
			return err
		}
		if conflict {
			return publication.ErrImmutableConflict
		}
	}
	return nil
}
func (s *Store) SubmitDraft(ctx context.Context, a publication.Access, id string, input publication.SubmitInput) (publication.SubmissionView, error) {
	var out publication.SubmissionView
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowTx(ctx, a, publication.SubmitDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		d, _, err := s.readWorkflowDraft(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		prior, found, err := workflowJSONReplay[publication.SubmissionView](s, ctx, tx, u, a, publication.SubmitDraftAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		if d.Status != "editing" || d.Revision != input.ExpectedRevision || d.Gate.Digest != input.ExpectedDigest {
			return publication.ErrDraftConflict
		}
		c, _, err := workflowCatalogue(ctx, tx, d.CatalogueVersion)
		if err != nil {
			return err
		}
		bytes, err := workflowStoredAssets(ctx, tx, id, d.Package)
		if err != nil {
			return err
		}
		sealed, report := content.ValidateWorkflow(ctx, c, d.Package, workflowMemoryAssets(bytes))
		if !report.ReadyToSubmit {
			return publication.ErrContentNotReady
		}
		if !sealed.Verify() {
			return publication.ErrContentInvalid
		}
		if err = workflowAssetVersionChange(ctx, tx, d.Package); err != nil {
			return err
		}
		if _, err = s.importValidatedTx(ctx, tx, sealed); err != nil {
			return err
		}
		frozen := publication.FrozenBody{CatalogueVersion: d.CatalogueVersion, CatalogueSHA256: d.CatalogueSHA256, Package: d.Package, SourceMap: d.SourceMap, AuthorIDs: d.AuthorIDs, LegacyUnattributed: d.LegacyUnattributed, Assets: d.Assets}
		payload, err := publication.FrozenBytes(frozen)
		if err != nil {
			return err
		}
		if len(payload) > 4<<20 {
			return publication.ErrContentLimitExceeded
		}
		frozen.FrozenDigest, err = publication.FrozenDigest(frozen)
		if err != nil {
			return err
		}
		subID, err := workflowID()
		if err != nil {
			return err
		}
		gate := publication.GateFromReport(report)
		gate.Digest = d.Gate.Digest
		_, err = tx.ExecContext(ctx, `INSERT INTO content_submissions(id,workspace_id,owner_user_id,revision,package_id,package_version,package_sha256,catalogue_version,catalogue_sha256,frozen_body,frozen_bytes,frozen_digest,gate,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, subID, id, u.ID, d.Revision, d.Package.ID, d.Package.Version, sealed.SHA256(), d.CatalogueVersion, d.CatalogueSHA256, string(payload), payload, frozen.FrozenDigest, body(gate), now)
		if err != nil {
			return err
		}
		for _, author := range d.AuthorIDs {
			if _, err = tx.ExecContext(ctx, `INSERT INTO content_submission_authors VALUES($1,$2)`, subID, author); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO content_submission_members SELECT $1,m.package_id,m.package_version,m.kind,m.id,m.version,CASE m.kind WHEN 'knowledge' THEN k.sha256 WHEN 'unit' THEN u.sha256 WHEN 'path' THEN p.sha256 ELSE m.asset_sha256 END FROM package_members m LEFT JOIN knowledge_versions k ON m.kind='knowledge' AND k.id=m.id AND k.version=m.version LEFT JOIN unit_versions u ON m.kind='unit' AND u.id=m.id AND u.version=m.version LEFT JOIN path_versions p ON m.kind='path' AND p.id=m.id AND p.version=m.version WHERE m.package_id=$2 AND m.package_version=$3`, subID, d.Package.ID, d.Package.Version)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE content_submissions SET sealed=true WHERE id=$1`, subID); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE content_workspaces SET status='submitted',author_ids=$4,updated_at=$5 WHERE id=$1 AND owner_user_id=$2 AND revision=$3 AND status='editing'`, id, u.ID, d.Revision, body(d.AuthorIDs), now)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return publication.ErrDraftConflict
		}
		out = publication.SubmissionView{ID: subID, WorkspaceID: id, OwnerID: u.ID, Status: "pending", Revision: d.Revision, Frozen: frozen, Gate: gate, CreatedAt: now.UTC().Format(time.RFC3339)}
		if err = workflowEvent(ctx, tx, u, a, publication.SubmitDraftAction, "submission", subID, d.Gate.Digest, frozen.FrozenDigest, fmt.Sprintf("editing:%d", d.Revision), "pending", "", now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.SubmitDraftAction, id, input, out)
	})
	return out, err
}
func (s *Store) readWorkflowSubmission(ctx context.Context, tx *sql.Tx, u auth.User, id string, ownerOnly bool) (publication.SubmissionView, error) {
	var out publication.SubmissionView
	var frozen, gate []byte
	var digest string
	var created time.Time
	err := tx.QueryRowContext(ctx, `SELECT id::text,workspace_id::text,owner_user_id::text,revision,status,frozen_body,frozen_digest,gate,created_at FROM content_submissions WHERE id=$1 AND sealed`, id).Scan(&out.ID, &out.WorkspaceID, &out.OwnerID, &out.Revision, &out.Status, &frozen, &digest, &gate, &created)
	if err != nil {
		return out, workflowRowError(err)
	}
	if out.OwnerID != u.ID && (ownerOnly || !publication.HasRole(u, auth.RoleReviewer) && !publication.HasRole(u, auth.RoleAdmin)) {
		return publication.SubmissionView{}, auth.ErrNotFound
	}
	if len(frozen) > 4<<20 {
		return out, publication.ErrContentLimitExceeded
	}
	if json.Unmarshal(frozen, &out.Frozen) != nil || json.Unmarshal(gate, &out.Gate) != nil {
		return out, auth.ErrUnavailable
	}
	computed, err := publication.FrozenDigest(out.Frozen)
	if err != nil || computed != digest {
		return out, auth.ErrUnavailable
	}
	out.Frozen.FrozenDigest = digest
	out.CreatedAt = created.UTC().Format(time.RFC3339)
	var review publication.ReviewDecision
	var checks []byte
	var reviewTime time.Time
	err = tx.QueryRowContext(ctx, `SELECT id::text,submission_id::text,reviewer_user_id::text,frozen_digest,decision,checks,independence_note,note,created_at FROM content_review_decisions WHERE submission_id=$1`, id).Scan(&review.ID, &review.SubmissionID, &review.ReviewerID, &review.FrozenDigest, &review.Decision, &checks, &review.IndependenceNote, &review.Note, &reviewTime)
	if err == sql.ErrNoRows {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if json.Unmarshal(checks, &review.Checks) != nil {
		return out, auth.ErrUnavailable
	}
	review.CreatedAt = reviewTime.UTC().Format(time.RFC3339)
	out.Review = &review
	return out, nil
}
func (s *Store) ReadSubmission(ctx context.Context, a publication.Access, id string) (publication.SubmissionView, error) {
	var out publication.SubmissionView
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowReadTx(ctx, a, publication.ReadSubmissionAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var err error
		out, err = s.readWorkflowSubmission(ctx, tx, u, id, false)
		return err
	})
	return out, err
}
func (s *Store) ReadSubmissionAsset(ctx context.Context, a publication.Access, id, sha string) ([]byte, error) {
	var out []byte
	if !publication.ValidID(id) || !publication.ValidSHA(sha) {
		return nil, auth.ErrInvalidInput
	}
	err := s.workflowReadTx(ctx, a, publication.ReadSubmissionAssetAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if _, err := s.readWorkflowSubmission(ctx, tx, u, id, false); err != nil {
			return err
		}
		return workflowRowError(tx.QueryRowContext(ctx, `SELECT a.bytes FROM content_submission_members m JOIN assets a ON a.sha256=m.sha256 WHERE m.submission_id=$1 AND m.kind='asset' AND m.sha256=$2 LIMIT 1`, id, sha).Scan(&out))
	})
	return out, err
}
func (s *Store) ReviseSubmission(ctx context.Context, a publication.Access, id string) (publication.DraftView, error) {
	var out publication.DraftView
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowTx(ctx, a, publication.ReviseSubmissionAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		sub, err := s.readWorkflowSubmission(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		prior, found, err := workflowJSONReplay[publication.DraftView](s, ctx, tx, u, a, publication.ReviseSubmissionAction, id, struct{}{})
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		in, err := workflowImportedInput(ctx, tx, sub.Frozen.Package.ID, sub.Frozen.Package.Version)
		if err != nil {
			return err
		}
		in.Package = sub.Frozen.Package
		in.SourceMap = sub.Frozen.SourceMap
		out, err = s.createWorkflowDraft(ctx, tx, u, a, in, id, sub.Frozen.AuthorIDs, sub.Frozen.LegacyUnattributed, now)
		if err != nil {
			return err
		}
		if err = workflowEvent(ctx, tx, u, a, publication.ReviseSubmissionAction, "draft", out.ID, sub.Frozen.FrozenDigest, out.Gate.Digest, sub.Status, "editing:1", "", now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.ReviseSubmissionAction, id, struct{}{}, out)
	})
	return out, err
}
func (s *Store) ListSubmissions(ctx context.Context, a publication.Access, q publication.ListQuery) (publication.Page[publication.SubmissionSummary], error) {
	q, err := publication.ValidateList(q, "pending", "approved", "returned")
	out := publication.Page[publication.SubmissionSummary]{Items: []publication.SubmissionSummary{}, Limit: q.Limit, Offset: q.Offset}
	if err != nil {
		return out, err
	}
	err = s.workflowReadTx(ctx, a, publication.ListSubmissionsAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		all := q.Scope == "all"
		if all && !publication.HasRole(u, auth.RoleAdmin) {
			return auth.ErrForbidden
		}
		if q.Scope == "" && publication.HasRole(u, auth.RoleReviewer) {
			if q.Status != "" && q.Status != "pending" {
				return auth.ErrInvalidInput
			}
			all = true
			q.Status = "pending"
		}
		where := `WHERE sealed AND ($1 OR owner_user_id=$2) AND ($3='' OR status=$3)`
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM content_submissions `+where, all, u.ID, q.Status).Scan(&out.Total); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id::text,workspace_id::text,owner_user_id::text,package_id,package_version,catalogue_version,status,revision,frozen_digest,created_at FROM content_submissions `+where+` ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`, all, u.ID, q.Status, q.Limit, q.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item publication.SubmissionSummary
			var created time.Time
			if err = rows.Scan(&item.ID, &item.WorkspaceID, &item.OwnerID, &item.PackageID, &item.PackageVersion, &item.CatalogueVersion, &item.Status, &item.Revision, &item.FrozenDigest, &created); err != nil {
				return err
			}
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			out.Items = append(out.Items, item)
		}
		return rows.Err()
	})
	return out, err
}
