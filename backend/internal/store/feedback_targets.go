package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func feedbackLabel(t feedback.Target) string {
	if t.Kind == "site" {
		return "Site · " + string(*t.Area)
	}
	s := fmt.Sprintf("%s %s · v%d", t.Kind, t.Identity.ID, t.Identity.Version)
	if t.Part != nil {
		if t.Part.Unit != nil {
			s += fmt.Sprintf(" · unit %s v%d", t.Part.Unit.ID, t.Part.Unit.Version)
		} else {
			s += " · asset " + t.Part.Asset.ID
		}
	}
	return s
}
func feedbackFixedItem(ctx context.Context, tx *sql.Tx, owner string, s feedback.Source) (assessment.ItemBinding, question.Identity, string, string, error) {
	var raw, knowledge []byte
	var kp, qp string
	var e error
	if s.Kind == "practice" {
		e = tx.QueryRowContext(ctx, `SELECT seal#>'{body,items,0}',seal#>'{body,knowledge}',knowledge_publication_id,question_publication_id FROM practice_attempts WHERE id=$1 AND owner_user_id=$2`, *s.AttemptID, owner).Scan(&raw, &knowledge, &kp, &qp)
	} else {
		e = tx.QueryRowContext(ctx, `SELECT i.binding,a.seal#>'{body,knowledge}',a.knowledge_publication_id,a.question_publication_id FROM assessment_attempts a JOIN assessment_items i ON i.attempt_id=a.id AND i.position=$3 WHERE a.id=$1 AND a.owner_user_id=$2 AND a.sealed`, *s.AttemptID, owner, *s.Position).Scan(&raw, &knowledge, &kp, &qp)
	}
	var item assessment.ItemBinding
	var k question.Identity
	if errors.Is(e, sql.ErrNoRows) {
		return item, k, kp, qp, auth.ErrNotFound
	}
	if e != nil {
		return item, k, kp, qp, e
	}
	if json.Unmarshal(raw, &item) != nil || json.Unmarshal(knowledge, &k) != nil {
		return item, k, kp, qp, auth.ErrUnavailable
	}
	return item, k, kp, qp, nil
}
func feedbackResolveTarget(ctx context.Context, tx *sql.Tx, actor string, t feedback.Target, s feedback.Source, creating bool) (feedback.Binding, error) {
	b := feedback.Binding{Target: t, Source: s, Units: []question.Identity{}, Assets: []feedback.AssetRef{}, ContentApprovalIDs: []string{}, QuestionApprovalIDs: []string{}}
	if e := feedback.ValidateTargetSource(t, s); e != nil {
		return b, e
	}
	if t.Kind == "site" {
		return b, nil
	}
	if t.Kind == "instance" {
		item, k, kp, qp, e := feedbackFixedItem(ctx, tx, actor, s)
		if e != nil {
			return b, e
		}
		b.OwnerID = &actor
		b.Knowledge = &k
		b.Instance = &item.Instance
		b.Template = item.Template
		b.Units = item.Units
		b.Assets = item.Assets
		b.KnowledgePublicationID = &kp
		b.QuestionPublicationID = &qp
	} else {
		b.KnowledgePublicationID = s.PublicationID
		if t.Kind == "knowledge" {
			b.Knowledge = t.Identity
		}
		if creating {
			var current bool
			if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM publication_heads WHERE snapshot_id=$1)`, *s.PublicationID).Scan(&current); e != nil {
				return b, e
			}
			if !current {
				return b, feedback.ErrTargetStale
			}
		}
	}
	var valid bool
	if e := tx.QueryRowContext(ctx, `SELECT feedback_target_proof($1::jsonb,$2::jsonb,$3::uuid,$4)`, body(t), body(s), actor, creating).Scan(&valid); e != nil {
		return b, e
	}
	if !valid {
		return b, feedback.ErrTargetStale
	}
	if t.Kind == "knowledge" {
		rows, e := tx.QueryContext(ctx, `SELECT u.id,u.version,u.sha256 FROM unit_versions u JOIN publication_members m ON m.snapshot_id=$1 AND m.kind='unit' AND m.id=u.id AND m.version=u.version AND m.availability='active' WHERE u.knowledge_id=$2 AND u.knowledge_version=$3 AND learning_content_approved($1,'unit',u.id,u.version,u.sha256) ORDER BY u.id`, *s.PublicationID, t.Identity.ID, t.Identity.Version)
		if e != nil {
			return b, e
		}
		for rows.Next() {
			var i question.Identity
			if e = rows.Scan(&i.ID, &i.Version, &i.SHA256); e != nil {
				rows.Close()
				return b, e
			}
			b.Units = append(b.Units, i)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return b, e
		}
		rows, e = tx.QueryContext(ctx, `SELECT DISTINCT a->>'id',a->>'sha256' FROM publication_members m JOIN imported_packages p ON p.id=m.package_id AND p.version=m.package_version CROSS JOIN LATERAL jsonb_array_elements(p.body->'assets') a WHERE m.snapshot_id=$1 AND m.kind='asset' AND m.id=a->>'id' AND m.availability='active' AND a#>>'{knowledge,id}'=$2 AND a#>>'{knowledge,version}'=$3::integer::text AND learning_content_approved($1,'asset',m.id,1,a->>'sha256') ORDER BY 1,2`, *s.PublicationID, t.Identity.ID, t.Identity.Version)
		if e != nil {
			return b, e
		}
		for rows.Next() {
			var a feedback.AssetRef
			if e = rows.Scan(&a.ID, &a.SHA256); e != nil {
				rows.Close()
				return b, e
			}
			b.Assets = append(b.Assets, a)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return b, e
		}
	}
	// Approval identifiers remain internal and are read from immutable release facts.
	rows, e := tx.QueryContext(ctx, `SELECT DISTINCT r.id::text FROM content_review_decisions r JOIN content_submission_members m ON m.submission_id=r.submission_id WHERE r.decision='approve' AND ((m.kind=$1 AND m.id=$2 AND m.version=$3 AND m.sha256=$4) OR (m.kind='knowledge' AND m.id=$5 AND m.version=$6 AND m.sha256=$7)) ORDER BY 1`, t.Kind, t.Identity.ID, t.Identity.Version, t.Identity.SHA256, feedbackKnowledgeID(b), feedbackKnowledgeVersion(b), feedbackKnowledgeSHA(b))
	if e != nil {
		return b, e
	}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return b, e
		}
		b.ContentApprovalIDs = append(b.ContentApprovalIDs, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return b, e
	}
	if b.Instance != nil {
		rows, e = tx.QueryContext(ctx, `SELECT review_id::text FROM question_publication_members WHERE publication_id=$1 AND kind='instance' AND id=$2 AND version=$3 AND sha256=$4`, *b.QuestionPublicationID, b.Instance.ID, b.Instance.Version, b.Instance.SHA256)
		if e != nil {
			return b, e
		}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return b, e
			}
			b.QuestionApprovalIDs = append(b.QuestionApprovalIDs, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return b, e
		}
	}
	return b, nil
}
func feedbackKnowledgeID(b feedback.Binding) string {
	if b.Knowledge == nil {
		return ""
	}
	return b.Knowledge.ID
}
func feedbackKnowledgeVersion(b feedback.Binding) int {
	if b.Knowledge == nil {
		return 0
	}
	return b.Knowledge.Version
}
func feedbackKnowledgeSHA(b feedback.Binding) string {
	if b.Knowledge == nil {
		return ""
	}
	return b.Knowledge.SHA256
}
func (s *Store) ReadFeedbackContext(ctx context.Context, a question.Access, q feedback.ContextQuery) (feedback.Envelope[feedback.Context], error) {
	var out feedback.Envelope[feedback.Context]
	e := s.feedbackTx(ctx, a, feedback.ReadContextAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		t := feedback.Target{}
		src := feedback.Source{}
		creating := true
		switch q.Kind {
		case "site":
			if q.ID != "" || q.Position != 0 || q.PartKind != "" || q.PartID != "" || !feedback.ValidArea(q.Area) {
				return auth.ErrInvalidInput
			}
			t.Kind = "site"
			t.Area = &q.Area
			src.Kind = "site"
		case "knowledge", "path":
			if !question.ValidMathID(q.ID) || q.Position != 0 || q.Area != "" || (q.Kind == "path" && q.PartKind != "") {
				return auth.ErrInvalidInput
			}
			t.Kind = q.Kind
			var head string
			var id question.Identity
			var e error
			// Table identifiers are selected exclusively by this closed switch.
			table := "knowledge_versions"
			if q.Kind == "path" {
				table = "path_versions"
			}
			e = tx.QueryRowContext(ctx, `SELECT h.snapshot_id,v.id,v.version,v.sha256 FROM publication_heads h JOIN publication_members m ON m.snapshot_id=h.snapshot_id AND m.kind=$1 AND m.id=$2 AND m.availability='active' JOIN `+table+` v ON v.id=m.id AND v.version=m.version`, q.Kind, q.ID).Scan(&head, &id.ID, &id.Version, &id.SHA256)
			if errors.Is(e, sql.ErrNoRows) {
				return auth.ErrNotFound
			}
			if e != nil {
				return e
			}
			t.Identity = &id
			src = feedback.Source{Kind: "publication", PublicationID: &head}
		case "practice", "assessment":
			if !question.ValidID(q.ID) || q.Area != "" || (q.Kind == "practice" && q.Position != 0) || (q.Kind == "assessment" && (q.Position < 1 || q.Position > 5)) {
				return auth.ErrInvalidInput
			}
			pos := q.Position
			if q.Kind == "practice" {
				pos = 1
			}
			src = feedback.Source{Kind: q.Kind, AttemptID: &q.ID, Position: &pos}
			item, _, _, _, e := feedbackFixedItem(ctx, tx, u.ID, src)
			if e != nil {
				return e
			}
			t.Kind = "instance"
			t.Identity = &item.Instance
			creating = false
		default:
			return auth.ErrInvalidInput
		}
		if q.PartKind != "" {
			if !question.ValidMathID(q.PartID) {
				return auth.ErrInvalidInput
			}
			part := feedback.Part{Kind: q.PartKind}
			if q.PartKind == "unit" && t.Kind == "knowledge" {
				var i question.Identity
				e := tx.QueryRowContext(ctx, `SELECT u.id,u.version,u.sha256 FROM unit_versions u JOIN publication_members m ON m.snapshot_id=$1 AND m.kind='unit' AND m.id=u.id AND m.version=u.version AND m.availability='active' WHERE u.id=$2 AND u.knowledge_id=$3 AND u.knowledge_version=$4`, *src.PublicationID, q.PartID, t.Identity.ID, t.Identity.Version).Scan(&i.ID, &i.Version, &i.SHA256)
				if errors.Is(e, sql.ErrNoRows) {
					return auth.ErrNotFound
				}
				if e != nil {
					return e
				}
				part.Unit = &i
			} else if q.PartKind == "asset" && (t.Kind == "knowledge" || t.Kind == "instance") {
				var asset feedback.AssetRef
				if t.Kind == "instance" {
					item, _, _, _, e := feedbackFixedItem(ctx, tx, u.ID, src)
					if e != nil {
						return e
					}
					for _, v := range item.Assets {
						if v.ID == q.PartID {
							asset = v
						}
					}
					if asset.ID == "" {
						return auth.ErrNotFound
					}
				} else {
					e := tx.QueryRowContext(ctx, `SELECT m.id,pm.asset_sha256 FROM publication_members m JOIN package_members pm ON pm.package_id=m.package_id AND pm.package_version=m.package_version AND pm.kind='asset' AND pm.id=m.id WHERE m.snapshot_id=$1 AND m.kind='asset' AND m.id=$2 AND m.availability='active'`, *src.PublicationID, q.PartID).Scan(&asset.ID, &asset.SHA256)
					if errors.Is(e, sql.ErrNoRows) {
						return auth.ErrNotFound
					}
					if e != nil {
						return e
					}
				}
				part.Asset = &asset
			} else {
				return auth.ErrInvalidInput
			}
			t.Part = &part
		} else if q.PartID != "" {
			return auth.ErrInvalidInput
		}
		if _, e := feedbackResolveTarget(ctx, tx, u.ID, t, src, creating); e != nil {
			return e
		}
		out = feedback.Envelope[feedback.Context]{ActorID: u.ID, Data: feedback.Context{Target: t, Source: src, Label: feedbackLabel(t)}}
		return nil
	})
	if e != nil {
		return feedback.Envelope[feedback.Context]{}, e
	}
	return out, nil
}
