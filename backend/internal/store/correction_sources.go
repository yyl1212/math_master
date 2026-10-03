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
