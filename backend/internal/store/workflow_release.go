package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"time"
)

func workflowHead(ctx context.Context, tx *sql.Tx) (*string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT snapshot_id FROM publication_heads WHERE singleton`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &id, err
}
func sameWorkflowHead(a, b *string) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}
func validWorkflowHead(id *string) bool { return id == nil || publication.ValidID(*id) }
func workflowResponseSize(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(raw) > 4<<20 {
		return publication.ErrContentLimitExceeded
	}
	return nil
}
func readWorkflowPublication(ctx context.Context, tx *sql.Tx, id string) (publication.PublicationView, error) {
	var out publication.PublicationView
	var raw, diff []byte
	var created time.Time
	var base sql.NullString
	var version int
	err := tx.QueryRowContext(ctx, `SELECT s.id,s.status,s.catalogue_version,m.base_head,m.manifest_bytes,m.sha256,m.diff,m.created_at FROM publication_snapshots s JOIN content_publication_manifests m ON m.snapshot_id=s.id WHERE s.id=$1`, id).Scan(&out.ID, &out.Status, &version, &base, &raw, &out.ManifestSHA, &diff, &created)
	if err != nil {
		return out, workflowRowError(err)
	}
	if len(raw) > 4<<20 {
		return out, publication.ErrContentLimitExceeded
	}
	if json.Unmarshal(raw, &out.Manifest) != nil || json.Unmarshal(diff, &out.Diff) != nil {
		return out, auth.ErrUnavailable
	}
	sha, err := publication.ManifestDigest(out.Manifest)
	if err != nil || sha != out.ManifestSHA || out.Manifest.CatalogueVersion != version || base.Valid != (out.Manifest.BaseHead != nil) || base.Valid && base.String != *out.Manifest.BaseHead {
		return out, auth.ErrUnavailable
	}
	out.CreatedAt = created.UTC().Format(time.RFC3339)
	return out, workflowResponseSize(out)
}

// The immutable evidence is checked in bulk. Only newly selected approvals need
// a current reviewer role; published historical approvals remain valid evidence.
func workflowManifestEvidence(ctx context.Context, tx *sql.Tx, m publication.Manifest, currentReviewers bool) error {
	const evidenceSQL = `SELECT count(*) FROM jsonb_to_recordset($1::jsonb->'members') e(identity jsonb,evidence jsonb)
 JOIN content_submission_members sm ON sm.submission_id=(e.evidence->>'submissionId')::uuid AND sm.package_id=e.identity->>'packageId' AND sm.package_version=(e.identity->>'packageVersion')::integer AND sm.kind=e.identity->>'kind' AND sm.id=e.identity->>'id' AND sm.version=(e.identity->>'version')::integer AND sm.sha256=e.identity->>'sha256'
 JOIN content_submissions sub ON sub.id=sm.submission_id AND sub.sealed AND sub.status='approved' AND sub.frozen_digest=e.evidence->>'frozenDigest' AND sub.catalogue_version=$2 AND sub.catalogue_sha256=$3
 JOIN content_review_decisions d ON d.submission_id=sub.id AND d.id=(e.evidence->>'decisionId')::uuid AND d.frozen_digest=sub.frozen_digest AND d.decision='approve'
 WHERE d.checks @> '{"mathematics":true,"explanations":true,"relationships":true,"sources":true,"illustrations":true}'
 AND NOT EXISTS(SELECT 1 FROM content_submission_authors a WHERE a.submission_id=sub.id AND a.user_id=d.reviewer_user_id)
 AND (NOT $4 OR e.evidence->>'inheritedFrom' IS NOT NULL OR EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=d.reviewer_user_id AND r.role='reviewer'))`
	var count int
	if err := tx.QueryRowContext(ctx, evidenceSQL, body(m), m.CatalogueVersion, m.CatalogueSHA256, currentReviewers).Scan(&count); err != nil {
		return err
	}
	if count != len(m.Members) {
		return publication.ErrReviewRequired
	}
	inherited := 0
	for _, member := range m.Members {
		if member.Evidence.InheritedFrom != nil {
			inherited++
			if !sameWorkflowHead(member.Evidence.InheritedFrom, m.BaseHead) {
				return publication.ErrReviewRequired
			}
		}
	}
	if inherited == 0 {
		return nil
	}
	// Every inherited member was checked against BaseHead above. Expand that
	// immutable manifest once; expanding it for each incoming member is quadratic.
	const inheritedSQL = `WITH previous AS MATERIALIZED (
 SELECT old.identity,old.evidence-'inheritedFrom' AS evidence
 FROM content_publication_manifests pm
 JOIN publication_snapshots s ON s.id=pm.snapshot_id AND s.status='published'
 CROSS JOIN LATERAL jsonb_to_recordset(pm.manifest->'members') old(identity jsonb,evidence jsonb)
 WHERE pm.snapshot_id=$2
 ) SELECT count(*) FROM jsonb_to_recordset($1::jsonb->'members') e(identity jsonb,evidence jsonb)
 JOIN previous old ON e.identity=old.identity AND (e.evidence-'inheritedFrom')=old.evidence
 WHERE e.evidence->>'inheritedFrom'=$2`
	if err := tx.QueryRowContext(ctx, inheritedSQL, body(m), *m.BaseHead).Scan(&count); err != nil {
		return err
	}
	if count != inherited {
		return publication.ErrReviewRequired
	}
	return nil
}
func workflowBlacklisted(ctx context.Context, tx *sql.Tx, m publication.Manifest) error {
	var found bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jsonb_to_recordset($1::jsonb->'members') e(identity jsonb,evidence jsonb) JOIN content_withdrawals w ON (w.kind='asset' AND e.identity->>'kind'='asset' AND w.sha256=e.identity->>'sha256') OR (w.kind=e.identity->>'kind' AND w.kind<>'asset' AND w.target_id=e.identity->>'id' AND w.target_version=(e.identity->>'version')::integer))`, body(m)).Scan(&found)
	if err != nil {
		return err
	}
	if found {
		return publication.ErrContentInvalid
	}
	return nil
}

const workflowMemberBodies = `SELECT m.kind,m.id,m.version,m.package_id,m.package_version,
 CASE m.kind WHEN 'knowledge' THEN k.sha256 WHEN 'unit' THEN u.sha256 WHEN 'path' THEN p.sha256 ELSE pm.asset_sha256 END,
 CASE m.kind WHEN 'knowledge' THEN k.body WHEN 'unit' THEN u.body WHEN 'path' THEN p.body ELSE a.value END
 FROM publication_members m
 JOIN package_members pm ON pm.package_id=m.package_id AND pm.package_version=m.package_version AND pm.kind=m.kind AND pm.id=m.id AND pm.version=m.version
 LEFT JOIN knowledge_versions k ON m.kind='knowledge' AND k.id=m.id AND k.version=m.version
 LEFT JOIN unit_versions u ON m.kind='unit' AND u.id=m.id AND u.version=m.version
 LEFT JOIN path_versions p ON m.kind='path' AND p.id=m.id AND p.version=m.version
 LEFT JOIN imported_packages ip ON m.kind='asset' AND ip.id=m.package_id AND ip.version=m.package_version
 LEFT JOIN LATERAL (SELECT value FROM jsonb_array_elements(ip.body->'assets') WHERE value->>'id'=m.id) a ON m.kind='asset'
 WHERE m.snapshot_id=$1 AND m.availability='active'`

func (s *Store) loadWorkflowCandidate(ctx context.Context, tx *sql.Tx, head *string) (publication.Candidate, error) {
	var out publication.Candidate
	if head == nil {
		return out, nil
	}
	var manifestExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM content_publication_manifests WHERE snapshot_id=$1)`, *head).Scan(&manifestExists); err != nil {
		return out, err
	}
	if !manifestExists {
		return out, publication.ErrReviewRequired
	}
	view, err := readWorkflowPublication(ctx, tx, *head)
	if err != nil {
		return out, err
	}
	out.PublicationID = view.ID
	out.Manifest = view.Manifest
	out.Diff = view.Diff
	var knowledge, units, paths, assets int
	var svgBytes, jsonBytes int64
	err = tx.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE kind='knowledge'),count(*) FILTER(WHERE kind='unit'),count(*) FILTER(WHERE kind='path'),count(*) FILTER(WHERE kind='asset') FROM publication_members WHERE snapshot_id=$1 AND availability='active'`, *head).Scan(&knowledge, &units, &paths, &assets)
	if err != nil {
		return out, err
	}
	if knowledge > 1000 || units > 4000 || paths > 200 || assets > 1000 {
		return out, publication.ErrContentLimitExceeded
	}
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(sum(octet_length(bytes)),0) FROM assets WHERE sha256 IN(SELECT pm.asset_sha256 FROM publication_members m JOIN package_members pm ON pm.package_id=m.package_id AND pm.package_version=m.package_version AND pm.kind=m.kind AND pm.id=m.id WHERE m.snapshot_id=$1 AND m.kind='asset' AND m.availability='active')`, *head).Scan(&svgBytes)
	if err != nil {
		return out, err
	}
	if svgBytes > 10<<20 {
		return out, publication.ErrContentLimitExceeded
	}
	// PostgreSQL adds separator spaces to JSONB text. A 64 MiB physical
	// preflight avoids rejecting valid 32 MiB canonical payloads; below, each
	// streamed body is re-encoded and the exact 32 MiB bound is enforced.
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(sum(octet_length(body::text)),0) FROM (`+workflowMemberBodies+`) records(kind,id,version,package_id,package_version,sha256,body)`, *head).Scan(&jsonBytes)
	if err != nil {
		return out, err
	}
	if jsonBytes > 64<<20 {
		return out, publication.ErrContentLimitExceeded
	}
	out.Snapshot = content.Snapshot{CatalogueVersion: out.Manifest.CatalogueVersion, Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{}, Assets: []content.Asset{}, Bindings: []content.AssetBinding{}}
	expected := map[string]publication.MemberIdentity{}
	for _, item := range out.Manifest.Members {
		expected[item.Identity.Kind+"/"+item.Identity.ID] = item.Identity
	}
	rows, err := tx.QueryContext(ctx, workflowMemberBodies+` ORDER BY m.kind,m.id`, *head)
	if err != nil {
		return out, err
	}
	total, seen := 0, 0
	for rows.Next() {
		var identity publication.MemberIdentity
		var raw []byte
		if err = rows.Scan(&identity.Kind, &identity.ID, &identity.Version, &identity.PackageID, &identity.PackageVersion, &identity.SHA256, &raw); err != nil {
			rows.Close()
			return out, err
		}
		if expected[identity.Kind+"/"+identity.ID] != identity {
			rows.Close()
			return out, publication.ErrContentInvalid
		}
		var v any
		switch identity.Kind {
		case "knowledge":
			var k content.Knowledge
			err = json.Unmarshal(raw, &k)
			v = k
			out.Snapshot.Knowledge = append(out.Snapshot.Knowledge, k)
		case "unit":
			var u content.Unit
			err = json.Unmarshal(raw, &u)
			v = u
			out.Snapshot.Units = append(out.Snapshot.Units, u)
		case "path":
			var p content.Path
			err = json.Unmarshal(raw, &p)
			v = p
			out.Snapshot.Paths = append(out.Snapshot.Paths, p)
		case "asset":
			var a content.Asset
			err = json.Unmarshal(raw, &a)
			v = a
			out.Snapshot.Assets = append(out.Snapshot.Assets, a)
			if a.SHA256 != identity.SHA256 {
				err = publication.ErrContentInvalid
			}
		default:
			err = publication.ErrContentInvalid
		}
		if err != nil {
			rows.Close()
			return out, err
		}
		canonical, e := json.Marshal(v)
		if e != nil {
			rows.Close()
			return out, e
		}
		total += len(canonical)
		if total > 32<<20 {
			rows.Close()
			return out, publication.ErrContentLimitExceeded
		}
		if identity.Kind != "asset" && content.Digest(v) != identity.SHA256 {
			rows.Close()
			return out, publication.ErrContentInvalid
		}
		seen++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if seen != len(expected) {
		return out, publication.ErrContentInvalid
	}
	rows, err = tx.QueryContext(ctx, `SELECT b.unit_id,b.unit_version,b.asset_id,b.asset_sha256 FROM unit_asset_bindings b JOIN publication_members m ON m.kind='unit' AND m.id=b.unit_id AND m.version=b.unit_version WHERE m.snapshot_id=$1 AND m.availability='active' ORDER BY b.unit_id,b.unit_version,b.asset_id`, *head)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var b content.AssetBinding
		if err = rows.Scan(&b.Unit.ID, &b.Unit.Version, &b.AssetID, &b.SHA256); err != nil {
			rows.Close()
			return out, err
		}
		out.Snapshot.Bindings = append(out.Snapshot.Bindings, b)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if content.Digest(out.Snapshot.Bindings) != content.Digest(out.Manifest.Bindings) {
		return out, publication.ErrContentInvalid
	}
	if err = publication.CandidateLimits(out.Snapshot); err != nil {
		return out, err
	}
	if err = workflowManifestEvidence(ctx, tx, out.Manifest, false); err != nil {
		return out, err
	}
	return out, nil
}
func workflowDatabaseAssets(tx *sql.Tx) content.AssetReader {
	return func(ctx context.Context, a content.Asset) ([]byte, error) {
		var b []byte
		err := tx.QueryRowContext(ctx, `SELECT bytes FROM assets WHERE sha256=$1`, a.SHA256).Scan(&b)
		return b, err
	}
}
func workflowValidateCandidate(ctx context.Context, tx *sql.Tx, candidate publication.Candidate, currentReviewers bool) error {
	if err := workflowManifestEvidence(ctx, tx, candidate.Manifest, currentReviewers); err != nil {
		return err
	}
	if err := workflowBlacklisted(ctx, tx, candidate.Manifest); err != nil {
		return err
	}
	if err := publication.ValidateCandidateGraph(candidate.Snapshot); err != nil {
		return err
	}
	c, sha, err := workflowCatalogue(ctx, tx, candidate.Manifest.CatalogueVersion)
	if err != nil {
		return err
	}
	if sha != candidate.Manifest.CatalogueSHA256 {
		return publication.ErrContentInvalid
	}
	_, err = content.ValidateSnapshot(ctx, c, candidate.Snapshot, workflowDatabaseAssets(tx))
	return err
}
func (s *Store) workflowReviewedBatch(ctx context.Context, tx *sql.Tx, u auth.User, id string) (publication.ReviewedBatch, error) {
	var out publication.ReviewedBatch
	sub, err := s.readWorkflowSubmission(ctx, tx, u, id, false)
	if err != nil {
		return out, err
	}
	if sub.Status != "approved" || sub.Review == nil || sub.Review.Decision != "approve" {
		return out, publication.ErrReviewRequired
	}
	out.Submission = sub
	out.Members = []publication.MemberIdentity{}
	rows, err := tx.QueryContext(ctx, `SELECT kind,id,version,package_id,package_version,sha256 FROM content_submission_members WHERE submission_id=$1 ORDER BY kind,id`, id)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var m publication.MemberIdentity
		if err = rows.Scan(&m.Kind, &m.ID, &m.Version, &m.PackageID, &m.PackageVersion, &m.SHA256); err != nil {
			rows.Close()
			return out, err
		}
		out.Members = append(out.Members, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	out.Bindings = publication.SnapshotBindings(sub.Frozen.Package)
	return out, nil
}
func (s *Store) insertWorkflowPublication(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time, candidate publication.Candidate, status string) (publication.PublicationView, error) {
	var out publication.PublicationView
	id, err := workflowID()
	if err != nil {
		return out, err
	}
	sha, err := publication.ManifestDigest(candidate.Manifest)
	if err != nil {
		return out, err
	}
	out = publication.PublicationView{ID: id, Status: status, ManifestSHA: sha, CreatedAt: now.UTC().Format(time.RFC3339), Manifest: candidate.Manifest, Diff: candidate.Diff}
	if err = workflowResponseSize(out); err != nil {
		return out, err
	}
	raw, err := publication.ManifestBytes(candidate.Manifest)
	if err != nil {
		return out, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO publication_snapshots VALUES($1,$2,$3)`, id, candidate.Manifest.CatalogueVersion, status); err != nil {
		return out, err
	}
	for _, member := range candidate.Manifest.Members {
		m := member.Identity
		if _, err = tx.ExecContext(ctx, `INSERT INTO publication_members VALUES($1,$2,$3,$4,$5,$6,'active')`, id, m.PackageID, m.PackageVersion, m.Kind, m.ID, m.Version); err != nil {
			return out, err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO content_publication_manifests(snapshot_id,base_head,manifest,manifest_bytes,sha256,diff,creator_user_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, id, candidate.Manifest.BaseHead, string(raw), raw, sha, body(candidate.Diff), u.ID, now)
	return out, err
}

// Related reviewer rows are discovered from immutable submissions/manifests
// under authenticated reads, then locked in UUID order by the mutation guard.
func (s *Store) workflowReleaseReviewers(ctx context.Context, a publication.Access, ids []string, publicationID string) ([]string, error) {
	out := []string{}
	err := s.workflowReadTx(ctx, a, publication.ReadPublicationAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if publicationID != "" {
			view, err := readWorkflowPublication(ctx, tx, publicationID)
			if err != nil {
				return err
			}
			for _, m := range view.Manifest.Members {
				if m.Evidence.InheritedFrom == nil {
					ids = append(ids, m.Evidence.SubmissionID)
				}
			}
		}
		for _, id := range ids {
			var reviewer string
			err := tx.QueryRowContext(ctx, `SELECT reviewer_user_id::text FROM content_review_decisions WHERE submission_id=$1 AND decision='approve'`, id).Scan(&reviewer)
			if err == sql.ErrNoRows {
				continue
			}
			if err != nil {
				return err
			}
			out = append(out, reviewer)
		}
		return nil
	})
	return out, err
}
func (s *Store) PrepareRelease(ctx context.Context, a publication.Access, input publication.PrepareInput) (publication.PublicationView, error) {
	var out publication.PublicationView
	if len(input.SubmissionIDs) < 1 || len(input.SubmissionIDs) > 20 || !validWorkflowHead(input.ExpectedHead) || !publication.ValidNote(input.Reason) {
		return out, auth.ErrInvalidInput
	}
	seen := map[string]bool{}
	for _, id := range input.SubmissionIDs {
		if !publication.ValidID(id) || seen[id] {
			return out, auth.ErrInvalidInput
		}
		seen[id] = true
	}
	ctx, cancel := context.WithTimeout(ctx, workflowTimeout)
	defer cancel()
	related, err := s.workflowReleaseReviewers(ctx, a, append([]string{}, input.SubmissionIDs...), "")
	if err != nil {
		return out, err
	}
	err = s.workflowTx(ctx, a, publication.PrepareReleaseAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, found, err := workflowJSONReplay[publication.PublicationView](s, ctx, tx, u, a, publication.PrepareReleaseAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		head, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		if !sameWorkflowHead(head, input.ExpectedHead) {
			return publication.ErrPublicationStale
		}
		base, err := s.loadWorkflowCandidate(ctx, tx, head)
		if err != nil {
			return err
		}
		batches := []publication.ReviewedBatch{}
		for _, id := range input.SubmissionIDs {
			batch, err := s.workflowReviewedBatch(ctx, tx, u, id)
			if err != nil {
				return err
			}
			batches = append(batches, batch)
		}
		candidate, err := publication.BuildCandidate(base, batches)
		if err != nil {
			return err
		}
		if err = workflowValidateCandidate(ctx, tx, candidate, true); err != nil {
			return err
		}
		out, err = s.insertWorkflowPublication(ctx, tx, u, now, candidate, "draft")
		if err != nil {
			return err
		}
		if err = workflowEvent(ctx, tx, u, a, publication.PrepareReleaseAction, "publication", out.ID, "", out.ManifestSHA, "", "draft", input.Reason, now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.PrepareReleaseAction, "", input, out)
	})
	return out, err
}
func (s *Store) ActivateRelease(ctx context.Context, a publication.Access, id string, input publication.ActivateInput) (publication.PublicationView, error) {
	var out publication.PublicationView
	if !publication.ValidID(id) || !validWorkflowHead(input.ExpectedHead) || !publication.ValidSHA(input.ExpectedManifestSHA) || !publication.ValidNote(input.Reason) {
		return out, auth.ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, workflowTimeout)
	defer cancel()
	related, err := s.workflowReleaseReviewers(ctx, a, nil, id)
	if err != nil {
		return out, err
	}
	err = s.workflowTx(ctx, a, publication.ActivateReleaseAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, found, err := workflowJSONReplay[publication.PublicationView](s, ctx, tx, u, a, publication.ActivateReleaseAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		view, err := readWorkflowPublication(ctx, tx, id)
		if err != nil {
			return err
		}
		head, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		if view.Status != "draft" || !sameWorkflowHead(view.Manifest.BaseHead, head) || !sameWorkflowHead(head, input.ExpectedHead) || view.ManifestSHA != input.ExpectedManifestSHA {
			return publication.ErrPublicationStale
		}
		candidate, err := s.loadWorkflowCandidate(ctx, tx, &id)
		if err != nil {
			return err
		}
		if err = workflowValidateCandidate(ctx, tx, candidate, true); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE publication_snapshots SET status='published' WHERE id=$1 AND status='draft'`, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO publication_heads VALUES(true,$1) ON CONFLICT(singleton) DO UPDATE SET snapshot_id=EXCLUDED.snapshot_id`, id); err != nil {
			return err
		}
		out = view
		out.Status = "published"
		if err = workflowEvent(ctx, tx, u, a, publication.ActivateReleaseAction, "publication", id, "", out.ManifestSHA, "draft", "published", input.Reason, now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.ActivateReleaseAction, id, input, out)
	})
	return out, err
}
func (s *Store) ReadPublication(ctx context.Context, a publication.Access, id string) (publication.PublicationView, error) {
	var out publication.PublicationView
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowReadTx(ctx, a, publication.ReadPublicationAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var err error
		out, err = readWorkflowPublication(ctx, tx, id)
		return err
	})
	return out, err
}
func (s *Store) ListPublications(ctx context.Context, a publication.Access, q publication.ListQuery) (publication.PublicationPage, error) {
	var out publication.PublicationPage
	q, err := publication.ValidateList(q, "draft", "published")
	if err != nil || q.Scope != "" && q.Scope != "all" {
		return out, auth.ErrInvalidInput
	}
	out.Items = []publication.PublicationView{}
	out.Limit = q.Limit
	out.Offset = q.Offset
	err = s.workflowReadTx(ctx, a, publication.ListPublicationsAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var err error
		out.Head, err = workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM content_publication_manifests m JOIN publication_snapshots s ON s.id=m.snapshot_id WHERE $1='' OR s.status=$1`, q.Status).Scan(&out.Total); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT m.snapshot_id FROM content_publication_manifests m JOIN publication_snapshots s ON s.id=m.snapshot_id WHERE $1='' OR s.status=$1 ORDER BY m.created_at DESC,m.snapshot_id LIMIT $2 OFFSET $3`, q.Status, q.Limit, q.Offset)
		if err != nil {
			return err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, id := range ids {
			view, err := readWorkflowPublication(ctx, tx, id)
			if err != nil {
				return err
			}
			out.Items = append(out.Items, view)
			if err = workflowResponseSize(out); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}
