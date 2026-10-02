package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

// Only page-selected publication ids enter the grouped counts. No historical frozen payload is expanded.
const questionSummarySelect = `SELECT p.id::text,p.status,p.manifest_sha256,p.created_at,p.base_knowledge_head,p.base_question_head::text,p.catalogue_version,p.catalogue_sha256,p.diff,coalesce(c.templates,0),coalesce(c.instances,0),coalesce(c.blueprints,0) FROM selected p LEFT JOIN counts c ON c.publication_id=p.id ORDER BY p.created_at DESC,p.id DESC`
const questionCounts = `counts AS (SELECT m.publication_id,count(*) FILTER(WHERE kind='template') templates,count(*) FILTER(WHERE kind='instance') instances,count(*) FILTER(WHERE kind='blueprint') blueprints FROM question_publication_members m JOIN selected p ON p.id=m.publication_id GROUP BY m.publication_id)`

func questionScanSummary(row interface{ Scan(...any) error }) (question.PublicationSummary, error) {
	var p question.PublicationSummary
	var created time.Time
	var khead, qhead sql.NullString
	var diff []byte
	err := row.Scan(&p.ID, &p.Status, &p.ManifestSHA, &created, &khead, &qhead, &p.CatalogueVersion, &p.CatalogueSHA256, &diff, &p.TemplateCount, &p.InstanceCount, &p.BlueprintCount)
	if err != nil {
		return p, workflowRowError(err)
	}
	if json.Unmarshal(diff, &p.Diff) != nil {
		return p, auth.ErrUnavailable
	}
	p.CreatedAt = created.UTC().Format(time.RFC3339)
	if khead.Valid {
		p.BaseKnowledgeHead = &khead.String
	}
	if qhead.Valid {
		p.BaseQuestionHead = &qhead.String
	}
	return p, nil
}
func questionReadSummary(ctx context.Context, tx *sql.Tx, id string) (question.PublicationSummary, error) {
	return questionScanSummary(tx.QueryRowContext(ctx, `WITH selected AS MATERIALIZED(SELECT id,status,manifest_sha256,created_at,base_knowledge_head,base_question_head,catalogue_version,catalogue_sha256,diff FROM question_publications WHERE id=$1 AND sealed),`+questionCounts+` `+questionSummarySelect, id))
}
func (s *Store) ReadQuestionPublication(ctx context.Context, a question.Access, id string) (question.PublicationSummary, error) {
	var out question.PublicationSummary
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.questionReadTx(ctx, a, question.ReadPublicationAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		var e error
		out, e = questionReadSummary(ctx, tx, id)
		return e
	})
	return out, err
}
func (s *Store) ListQuestionPublications(ctx context.Context, a question.Access, q question.ListQuery) (question.PublicationPage, error) {
	out := question.PublicationPage{Page: question.Page[question.PublicationSummary]{Items: []question.PublicationSummary{}}}
	pq, err := publication.ValidateList(publication.ListQuery{Scope: q.Scope, Status: q.Status, Limit: q.Limit, Offset: q.Offset}, "prepared", "published")
	if err != nil || q.Scope != "" && q.Scope != "all" {
		return out, auth.ErrInvalidInput
	}
	out.Limit = pq.Limit
	out.Offset = pq.Offset
	err = s.questionReadTx(ctx, a, question.ListPublicationsAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		var err error
		out.Head, err = questionHead(ctx, tx)
		if err != nil {
			return err
		}
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM question_publications WHERE sealed AND ($1='' OR status=$1)`, pq.Status).Scan(&out.Total); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `WITH selected AS MATERIALIZED(SELECT id,status,manifest_sha256,created_at,base_knowledge_head,base_question_head,catalogue_version,catalogue_sha256,diff FROM question_publications WHERE sealed AND ($1='' OR status=$1) ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3),`+questionCounts+` `+questionSummarySelect, pq.Status, pq.Limit, pq.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			p, err := questionScanSummary(rows)
			if err != nil {
				return err
			}
			out.Items = append(out.Items, p)
		}
		if err = rows.Err(); err != nil {
			return err
		}
		return questionFitPage(&out.Page, func() any { return out })
	})
	return out, err
}
func questionDetailQuery(id string, q question.ListQuery) (publication.ListQuery, error) {
	if !question.ValidID(id) || q.Scope != "" || q.Status != "" {
		return publication.ListQuery{}, auth.ErrInvalidInput
	}
	return publication.ValidateList(publication.ListQuery{Limit: q.Limit, Offset: q.Offset})
}
func (s *Store) ListQuestionMembers(ctx context.Context, a question.Access, id string, q question.ListQuery) (question.MemberPage, error) {
	out := question.MemberPage{Page: question.Page[question.ManifestMember]{Items: []question.ManifestMember{}}, PublicationID: id}
	pq, err := questionDetailQuery(id, q)
	if err != nil {
		return out, err
	}
	out.Limit = pq.Limit
	out.Offset = pq.Offset
	err = s.questionReadTx(ctx, a, question.ListMembersAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		summary, err := questionReadSummary(ctx, tx, id)
		if err != nil {
			return err
		}
		out.ManifestSHA = summary.ManifestSHA
		out.Total = summary.TemplateCount + summary.InstanceCount + summary.BlueprintCount
		rows, err := tx.QueryContext(ctx, `SELECT kind,id,version,sha256,package_id,package_version,evidence FROM question_publication_members WHERE publication_id=$1 ORDER BY kind,id LIMIT $2 OFFSET $3`, id, pq.Limit, pq.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m question.ManifestMember
			var evidence []byte
			if err = rows.Scan(&m.Identity.Kind, &m.Identity.ID, &m.Identity.Version, &m.Identity.SHA256, &m.Identity.PackageID, &m.Identity.PackageVersion, &evidence); err != nil {
				return err
			}
			if json.Unmarshal(evidence, &m.Evidence) != nil {
				return auth.ErrUnavailable
			}
			out.Items = append(out.Items, m)
		}
		if err = rows.Err(); err != nil {
			return err
		}
		return questionFitPage(&out.Page, func() any { return out })
	})
	return out, err
}
func (s *Store) ListQuestionChanges(ctx context.Context, a question.Access, id string, q question.ListQuery) (question.ChangePage, error) {
	out := question.ChangePage{Page: question.Page[question.Change]{Items: []question.Change{}}, PublicationID: id}
	pq, err := questionDetailQuery(id, q)
	if err != nil {
		return out, err
	}
	out.Limit = pq.Limit
	out.Offset = pq.Offset
	err = s.questionReadTx(ctx, a, question.ListChangesAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		if err := tx.QueryRowContext(ctx, `SELECT manifest_sha256,jsonb_array_length(changes) FROM question_publications WHERE id=$1 AND sealed`, id).Scan(&out.ManifestSHA, &out.Total); err != nil {
			return workflowRowError(err)
		}
		rows, err := tx.QueryContext(ctx, `SELECT v FROM question_publications p CROSS JOIN LATERAL jsonb_array_elements(p.changes) WITH ORDINALITY q(v,n) WHERE p.id=$1 ORDER BY n LIMIT $2 OFFSET $3`, id, pq.Limit, pq.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				return err
			}
			var change question.Change
			if json.Unmarshal(raw, &change) != nil {
				return auth.ErrUnavailable
			}
			out.Items = append(out.Items, change)
		}
		if err = rows.Err(); err != nil {
			return err
		}
		return questionFitPage(&out.Page, func() any { return out })
	})
	return out, err
}
