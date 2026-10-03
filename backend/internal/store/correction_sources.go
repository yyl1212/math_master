package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
)

func fmtVersion(n int) string { return fmt.Sprint(n) }
func correctionWithdrawalSource(ctx context.Context, tx *sql.Tx, ref correction.WithdrawalRef) error {
	var valid bool
	query := `SELECT EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.id=$1 AND ((w.kind='instance' AND EXISTS(SELECT 1 FROM question_instances i WHERE i.id=w.target_id AND i.version=w.target_version AND i.sha256=w.sha256 AND i.sealed)) OR (w.kind='template' AND EXISTS(SELECT 1 FROM question_templates i WHERE i.id=w.target_id AND i.version=w.target_version AND i.sha256=w.sha256)) OR (w.kind='blueprint' AND EXISTS(SELECT 1 FROM question_blueprints i WHERE i.id=w.target_id AND i.version=w.target_version AND i.sha256=w.sha256 AND i.sealed))))`
	if ref.Space == "content" {
		query = `SELECT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.id=$1 AND ((w.kind='asset' AND EXISTS(SELECT 1 FROM assets a WHERE a.sha256=w.sha256)) OR (w.kind='knowledge' AND EXISTS(SELECT 1 FROM knowledge_versions i WHERE i.id=w.target_id AND i.version=w.target_version AND i.sha256=w.sha256)) OR (w.kind='unit' AND EXISTS(SELECT 1 FROM unit_versions i WHERE i.id=w.target_id AND i.version=w.target_version AND i.sha256=w.sha256)) OR (w.kind='path' AND EXISTS(SELECT 1 FROM path_versions i WHERE i.id=w.target_id AND i.version=w.target_version AND i.sha256=w.sha256))))`
	}
	if e := tx.QueryRowContext(ctx, query, ref.ID).Scan(&valid); e != nil {
		return e
	}
	if !valid {
		return correction.ErrSourceStale
	}
	return nil
}

func correctionResolvedInstance(ctx context.Context, tx *sql.Tx, i question.Identity, pub string) (question.Instance, question.MemberEvidence, string, error) {
	var out question.Instance
	var approval question.MemberEvidence
	var pos int
	var sha, kpub string
	var raw, proof []byte
	bindings := []assessment.ItemBinding{{Position: 1, Instance: i}}
	e := tx.QueryRowContext(ctx, `SELECT x.*,p.base_knowledge_head FROM (`+learningItemsSQL+`) x JOIN question_publications p ON p.id=$2`, body(bindings), pub).Scan(&pos, &sha, &raw, &proof, &kpub)
	if errors.Is(e, sql.ErrNoRows) {
		return out, approval, kpub, correction.ErrSourceStale
	}
	if e != nil {
		return out, approval, kpub, e
	}
	var wrapped struct {
		Purpose string            `json:"purpose"`
		Body    question.Instance `json:"body"`
	}
	if json.Unmarshal(raw, &wrapped) != nil || wrapped.Purpose != "question-instance-body-v1" || json.Unmarshal(proof, &approval) != nil {
		return out, approval, kpub, auth.ErrUnavailable
	}
	out = wrapped.Body
	out.Identity.SHA256 = sha
	_, actual, e := question.CanonicalInstance(out)
	if e != nil || actual != i.SHA256 || out.Identity != i {
		return out, approval, kpub, correction.ErrSourceStale
	}
	return out, approval, kpub, nil
}
func correctionResolveSources(ctx context.Context, tx *sql.Tx, caseID string, in correction.PlanInput) (correction.PlanProof, error) {
	p := correction.PlanProof{CaseID: caseID, AlgorithmVersion: in.AlgorithmVersion, Parent: in.Parent, Mappings: []correction.ResolvedMapping{}, Authors: []string{}, ContentApprovalIDs: []string{}, QuestionApprovalIDs: []string{}}
	if e := correction.ValidatePlan(in); e != nil {
		return p, e
	}
	c, e := correctionReadCase(ctx, tx, caseID)
	if e != nil {
		return p, e
	}
	authors, contentReviews, questionReviews := map[string]bool{}, map[string]bool{}, map[string]bool{}
	// Only immutable approved submission membership contributes mathematical
	// authorship. Caller-supplied IDs and free text never establish independence.
	addReview := func(query string, args ...any) error {
		rows, e := tx.QueryContext(ctx, query, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			var space, review, author string
			if e = rows.Scan(&space, &review, &author); e != nil {
				return e
			}
			count++
			if count > 10000 {
				return question.ErrLimitExceeded
			}
			authors[author] = true
			if space == "content" {
				contentReviews[review] = true
			} else {
				questionReviews[review] = true
			}
		}
		return rows.Err()
	}
	addContent := func(kind, id string, version int, sha, pub string) error {
		var ok bool
		if e := tx.QueryRowContext(ctx, `SELECT learning_content_approved($1,$2,$3,$4,$5)`, pub, kind, id, version, sha).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return correction.ErrSourceStale
		}
		return addReview(`SELECT DISTINCT 'content',r.id::text,a.user_id::text FROM content_submission_members m JOIN content_submissions s ON s.id=m.submission_id AND s.sealed AND s.status='approved' JOIN content_review_decisions r ON r.submission_id=s.id AND r.decision='approve' AND r.frozen_digest=s.frozen_digest JOIN content_submission_authors a ON a.submission_id=s.id WHERE m.kind=$1 AND m.id=$2 AND m.version=$3 AND m.sha256=$4 LIMIT 10001`, kind, id, version, sha)
	}
	for _, m := range in.Mappings {
		original, oa, opub, e := correctionResolvedInstance(ctx, tx, m.Original, m.OriginalPublicationID)
		if e != nil {
			return p, e
		}
		replacement, ra, rpub, e := correctionResolvedInstance(ctx, tx, m.Replacement.Identity, m.Replacement.PublicationID)
		if e != nil {
			return p, e
		}
		if e = correctionMappingScope(ctx, tx, c, in.Parent, original); e != nil {
			return p, e
		}
		p.Mappings = append(p.Mappings, correction.ResolvedMapping{Original: original, Replacement: replacement, OriginalPublicationID: m.OriginalPublicationID, ReplacementPublicationID: m.Replacement.PublicationID, OriginalApproval: oa, ReplacementApproval: ra})
		for j, item := range []question.Instance{original, replacement} {
			approval := oa
			pub := opub
			if j == 1 {
				approval = ra
				pub = rpub
			}
			if e = addReview(`SELECT 'question',r.id::text,a.user_id::text FROM question_review_decisions r JOIN question_submission_authors a ON a.submission_id=r.submission_id WHERE r.id=$1 AND r.decision='approve' LIMIT 10001`, approval.DecisionID); e != nil {
				return p, e
			}
			refs := []struct {
				kind, id string
				version  int
				sha      string
			}{}
			var kh string
			if e = tx.QueryRowContext(ctx, `SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=$2`, item.Body.Knowledge.ID, item.Body.Knowledge.Version).Scan(&kh); e != nil {
				return p, correction.ErrSourceStale
			}
			refs = append(refs, struct {
				kind, id string
				version  int
				sha      string
			}{"knowledge", item.Body.Knowledge.ID, item.Body.Knowledge.Version, kh})
			for _, u := range item.Body.Units {
				var h string
				if e = tx.QueryRowContext(ctx, `SELECT sha256 FROM unit_versions WHERE id=$1 AND version=$2`, u.ID, u.Version).Scan(&h); e != nil {
					return p, correction.ErrSourceStale
				}
				refs = append(refs, struct {
					kind, id string
					version  int
					sha      string
				}{"unit", u.ID, u.Version, h})
			}
			for _, a := range item.Body.Assets {
				refs = append(refs, struct {
					kind, id string
					version  int
					sha      string
				}{"asset", a.ID, 1, a.SHA256})
			}
			for _, ref := range refs {
				if e = addContent(ref.kind, ref.id, ref.version, ref.sha, pub); e != nil {
					return p, e
				}
			}
		}
	}
	// Rule-only plans still include authors of the mathematical sources covered
	// by that scope. Withdrawal plans include the exact withdrawn membership.
	if c.Rule != nil {
		var kid, kv any
		if c.Rule.Knowledge != nil {
			kid = c.Rule.Knowledge.ID
			kv = c.Rule.Knowledge.Version
		}
		e = addReview(`SELECT DISTINCT 'question',r.id::text,a.user_id::text FROM question_review_decisions r JOIN question_submissions s ON s.id=r.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=r.frozen_digest JOIN question_submission_members m ON m.submission_id=s.id JOIN question_instances i ON m.kind='instance' AND i.id=m.id AND i.version=m.version AND i.sha256=m.sha256 JOIN question_submission_authors a ON a.submission_id=s.id WHERE r.decision='approve' AND ($1::text IS NULL OR i.knowledge_id=$1 AND i.knowledge_version=$2) LIMIT 10001`, kid, kv)
		if e != nil {
			return p, e
		}
		e = addReview(`SELECT DISTINCT 'content',r.id::text,a.user_id::text FROM content_review_decisions r JOIN content_submissions s ON s.id=r.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=r.frozen_digest JOIN content_submission_members m ON m.submission_id=s.id JOIN content_submission_authors a ON a.submission_id=s.id WHERE r.decision='approve' AND m.kind='knowledge' AND ($1::text IS NULL OR m.id=$1 AND m.version=$2) LIMIT 10001`, kid, kv)
		if e != nil {
			return p, e
		}
	} else {
		if e = correctionWithdrawalSource(ctx, tx, *c.Withdrawal); e != nil {
			return p, e
		}
		query := `SELECT DISTINCT 'question',r.id::text,a.user_id::text FROM question_withdrawals w JOIN question_submission_members m ON m.kind=w.kind AND m.id=w.target_id AND m.version=w.target_version AND m.sha256=w.sha256 JOIN question_submissions s ON s.id=m.submission_id AND s.sealed AND s.status='approved' JOIN question_review_decisions r ON r.submission_id=s.id AND r.decision='approve' AND r.frozen_digest=s.frozen_digest JOIN question_submission_authors a ON a.submission_id=s.id WHERE w.id=$1 LIMIT 10001`
		if c.Withdrawal.Space == "content" {
			query = `SELECT DISTINCT 'content',r.id::text,a.user_id::text FROM content_withdrawals w JOIN content_submission_members m ON m.kind=w.kind AND m.sha256=w.sha256 AND (w.kind='asset' OR m.id=w.target_id AND m.version=w.target_version) JOIN content_submissions s ON s.id=m.submission_id AND s.sealed AND s.status='approved' JOIN content_review_decisions r ON r.submission_id=s.id AND r.decision='approve' AND r.frozen_digest=s.frozen_digest JOIN content_submission_authors a ON a.submission_id=s.id WHERE w.id=$1 LIMIT 10001`
		}
		if e = addReview(query, c.Withdrawal.ID); e != nil {
			return p, e
		}
	}

	// A rule-only or unmapped withdrawal plan still concerns the approved
	// knowledge, units and assets used by its question sources. Resolve their
	// authors from immutable frozen references, including assets shared by SHA.
	if len(questionReviews) > 0 {
		ids := []string{}
		for id := range questionReviews {
			ids = append(ids, id)
		}
		if e = addReview(`WITH refs AS (SELECT DISTINCT x->>'kind' kind,x->>'id' id,CASE WHEN x->>'kind'='asset' THEN 1 ELSE (x->>'version')::integer END version,x->>'sha256' sha256 FROM question_review_decisions q JOIN question_submissions s ON s.id=q.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=q.frozen_digest CROSS JOIN LATERAL jsonb_array_elements(s.frozen_body#>'{body,body,resolved}') x WHERE q.id=ANY($1::uuid[]) AND q.decision='approve') SELECT DISTINCT 'content',r.id::text,a.user_id::text FROM refs x JOIN content_submission_members m ON m.kind=x.kind AND m.sha256=x.sha256 AND (x.kind='asset' OR m.id=x.id AND m.version=x.version) JOIN content_submissions s ON s.id=m.submission_id AND s.sealed AND s.status='approved' JOIN content_review_decisions r ON r.submission_id=s.id AND r.decision='approve' AND r.frozen_digest=s.frozen_digest JOIN content_submission_authors a ON a.submission_id=s.id LIMIT 10001`, ids); e != nil {
			return p, e
		}
	}
	for id := range authors {
		p.Authors = append(p.Authors, id)
	}
	for id := range contentReviews {
		p.ContentApprovalIDs = append(p.ContentApprovalIDs, id)
	}
	for id := range questionReviews {
		p.QuestionApprovalIDs = append(p.QuestionApprovalIDs, id)
	}
	sort.Strings(p.Authors)
	sort.Strings(p.ContentApprovalIDs)
	sort.Strings(p.QuestionApprovalIDs)
	raw, e := json.Marshal(p)
	if e != nil {
		return p, e
	}
	if len(raw) > correction.MaxResponseBytes {
		return p, question.ErrLimitExceeded
	}
	return p, nil
}

func correctionMappingScope(ctx context.Context, tx *sql.Tx, c correction.CaseMetadata, parent *correction.PlanRef, i question.Instance) error {
	if c.Rule != nil {
		if c.Rule.Kind == "all" {
			return nil
		}
		k := c.Rule.Knowledge
		if i.Body.Knowledge.ID == k.ID && i.Body.Knowledge.Version == k.Version {
			var actual string
			if e := tx.QueryRowContext(ctx, `SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=$2`, k.ID, k.Version).Scan(&actual); e != nil {
				return e
			}
			if actual == k.SHA256 {
				return nil
			}
		}
		return correction.ErrSourceStale
	}
	var kind, id, sha string
	var version sql.NullInt64
	table := "question_withdrawals"
	if c.Withdrawal.Space == "content" {
		table = "content_withdrawals"
	}
	if e := tx.QueryRowContext(ctx, `SELECT kind,coalesce(target_id,''),target_version,sha256 FROM `+table+` WHERE id=$1`, c.Withdrawal.ID).Scan(&kind, &id, &version, &sha); e != nil {
		return e
	}
	matches := func(ref question.Identity) bool {
		return ref.ID == id && ref.Version == int(version.Int64) && ref.SHA256 == sha
	}
	if kind == "instance" && matches(i.Identity) || kind == "template" && i.Template != nil && matches(*i.Template) {
		return nil
	}
	if kind == "knowledge" && i.Body.Knowledge.ID == id && i.Body.Knowledge.Version == int(version.Int64) {
		var actual string
		if e := tx.QueryRowContext(ctx, `SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=$2`, id, version.Int64).Scan(&actual); e == nil && actual == sha {
			return nil
		}
	}
	if kind == "unit" {
		for _, u := range i.Body.Units {
			if u.ID == id && u.Version == int(version.Int64) {
				var actual string
				if e := tx.QueryRowContext(ctx, `SELECT sha256 FROM unit_versions WHERE id=$1 AND version=$2`, id, version.Int64).Scan(&actual); e == nil && actual == sha {
					return nil
				}
			}
		}
	}
	if kind == "asset" {
		for _, a := range i.Body.Assets {
			if a.SHA256 == sha {
				return nil
			}
		}
		var bound bool
		if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jsonb_to_recordset($1::jsonb) u(id text,version integer) JOIN unit_asset_bindings b ON b.unit_id=u.id AND b.unit_version=u.version WHERE b.asset_sha256=$2)`, body(i.Body.Units), sha).Scan(&bound); e != nil {
			return e
		}
		if bound {
			return nil
		}
	}
	if parent != nil {
		var allowed bool
		if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM correction_plans p CROSS JOIN LATERAL jsonb_array_elements(p.frozen_body#>'{body,proof,mappings}') m WHERE p.id=$1 AND p.version=$2 AND p.case_id=$3 AND p.status='approved' AND m#>'{replacement,identity}'=$4::jsonb)`, parent.ID, parent.Version, c.ID, body(i.Identity)).Scan(&allowed); e != nil {
			return e
		}
		if allowed {
			return nil
		}
	}
	return correction.ErrSourceStale
}
