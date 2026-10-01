package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

func questionRequestSHA(id string, input any) string { return workflowRequestSHA(id, input) }
func questionJSONReplay[T any](s *Store, ctx context.Context, tx *sql.Tx, u auth.User, a question.Access, action question.Action, id string, input any) (T, bool, error) {
	var out T
	raw, found, err := s.questionReplay(ctx, tx, u.ID, string(action), a.IdempotencyKey, questionRequestSHA(id, input))
	if err != nil || !found {
		return out, found, err
	}
	if json.Unmarshal(raw, &out) != nil {
		return out, false, auth.ErrUnavailable
	}
	return out, true, nil
}
func questionJSONRemember[T any](s *Store, ctx context.Context, tx *sql.Tx, u auth.User, a question.Access, action question.Action, id string, input any, out T) error {
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return s.questionRemember(ctx, tx, u.ID, string(action), a.IdempotencyKey, questionRequestSHA(id, input), raw)
}
func questionEvent(ctx context.Context, tx *sql.Tx, u auth.User, a question.Access, action question.Action, kind, id, before, after, beforeState, afterState, reason string, now time.Time) error {
	event, err := workflowID()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO question_events(id,actor_user_id,action,object_kind,object_id,before_digest,after_digest,before_state,after_state,reason,request_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, event, u.ID, string(action), kind, id, before, after, beforeState, afterState, reason, a.RequestID, now)
	return err
}
func questionValidAdoption(reason string) bool {
	n := utf8.RuneCountInString(reason)
	return utf8.ValidString(reason) && n >= 10 && n <= 2000 && len(reason) <= 6000 && strings.TrimSpace(reason) != "" && !strings.ContainsRune(reason, 0)
}
func questionDraftStructure(input question.DraftInput) (question.ValidationReport, error) {
	report := question.ValidationReport{StructuralErrors: []question.Issue{}, CompletenessErrors: []question.Issue{{Code: "VALIDATION_REQUIRED", Path: "/questionPackage"}}, HumanReviewRequirements: []question.Issue{}, CompletenessTotal: 1, Generation: []question.GenerationReport{}, Coverage: []question.CoverageNode{}}
	raw, err := json.Marshal(input)
	if err != nil {
		return report, question.ErrInvalid
	}
	if _, err = question.DecodeDraft(bytes.NewReader(raw)); err != nil {
		return report, err
	}
	p := input.QuestionPackage
	pkgRaw, _ := json.Marshal(p)
	report.PackageBytes = len(pkgRaw)
	if len(p.Templates) > 50 || len(p.FixedQuestions) > 200 || len(p.Blueprints) > 100 {
		return report, question.ErrLimitExceeded
	}
	sourceRaw, _ := json.Marshal(input.SourceMap)
	if len(sourceRaw) > question.MaxSourceMapBytes {
		return report, question.ErrLimitExceeded
	}
	for _, s := range input.SourceMap {
		if !question.ValidSHA(s.SHA256) || !question.ValidSHA(s.BatchSHA256) || !publication.RelativeSourcePath(s.RelativePath) {
			return report, question.ErrInvalid
		}
	}
	for _, t := range p.Templates {
		if len(t.Parameters) > 4 || len(t.Coverage) > 4 || len(t.Assets) > 8 || len(t.PromptTemplate)+len(t.ExplanationTemplate) > question.MaxQuestionTextBytes {
			return report, question.ErrLimitExceeded
		}
		for _, param := range t.Parameters {
			if len(param.Values) > 32 {
				return report, question.ErrLimitExceeded
			}
		}
	}
	for _, i := range p.FixedQuestions {
		if len(i.Body.Coverage) > 4 || len(i.Body.Assets) > 8 || len(i.Body.Choices) > 6 || len(i.Body.Prompt)+len(i.Body.Explanation) > question.MaxQuestionTextBytes {
			return report, question.ErrLimitExceeded
		}
	}
	report.Digest = questionRequestSHA("", input)
	return report, nil
}

// Authorship comes from immutable local records. A caller's author list is never a proof.
func questionLocalResponsibility(ctx context.Context, tx *sql.Tx, p question.QuestionPackage) (question.SourceResponsibility, error) {
	out := question.SourceResponsibility{AuthorIDs: []string{}}
	rows, err := tx.QueryContext(ctx, `WITH wanted AS (
 SELECT 'template'::text kind,v obj FROM jsonb_array_elements($1::jsonb->'templates') v
 UNION ALL SELECT 'instance',v FROM jsonb_array_elements($1::jsonb->'fixedQuestions') v
 UNION ALL SELECT 'blueprint',v FROM jsonb_array_elements($1::jsonb->'blueprints') v
 ), matching AS (
 SELECT DISTINCT p.id package_id,p.version package_version,p.author_ids,p.legacy_unattributed FROM wanted w JOIN question_packages p ON p.sealed AND (p.body->'body') @> jsonb_build_object(CASE w.kind WHEN 'template' THEN 'templates' WHEN 'instance' THEN 'fixedQuestions' ELSE 'blueprints' END,jsonb_build_array(w.obj)) CROSS JOIN LATERAL jsonb_array_elements(p.body->'body'->CASE w.kind WHEN 'template' THEN 'templates' WHEN 'instance' THEN 'fixedQuestions' ELSE 'blueprints' END) v WHERE v=w.obj
 UNION SELECT id,version,author_ids,legacy_unattributed FROM question_packages WHERE id=$2 AND version=$3 AND sealed AND body->'body'=$1::jsonb)
 SELECT author_ids,legacy_unattributed FROM matching
 UNION SELECT coalesce(jsonb_agg(a.user_id::text),'[]'),bool_or((s.frozen_body#>>'{body,body,legacyUnattributed}')::boolean) FROM matching m JOIN question_submissions s ON s.package_id=m.package_id AND s.package_version=m.package_version AND s.sealed JOIN question_submission_authors a ON a.submission_id=s.id HAVING count(*)>0`, body(p), p.ID, p.Version)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	ids := map[string]bool{}
	for rows.Next() {
		var raw []byte
		var legacy bool
		if err = rows.Scan(&raw, &legacy); err != nil {
			return out, err
		}
		var authors []string
		if json.Unmarshal(raw, &authors) != nil {
			return out, auth.ErrUnavailable
		}
		for _, id := range authors {
			if !question.ValidID(id) {
				return out, auth.ErrUnavailable
			}
			ids[id] = true
		}
		out.LegacyUnattributed = out.LegacyUnattributed || legacy
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	out.AuthorIDs = sortedQuestionAuthors(ids)
	return out, nil
}
func sortedQuestionAuthors(ids map[string]bool) []string {
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
func questionAuthors(ctx context.Context, tx *sql.Tx, owner string, p question.QuestionPackage, inherited []string, legacy bool) (question.SourceResponsibility, error) {
	out, err := questionLocalResponsibility(ctx, tx, p)
	if err != nil {
		return out, err
	}
	ids := map[string]bool{owner: true}
	for _, id := range inherited {
		ids[id] = true
	}
	for _, id := range out.AuthorIDs {
		ids[id] = true
	}
	out.AuthorIDs = sortedQuestionAuthors(ids)
	out.LegacyUnattributed = out.LegacyUnattributed || legacy
	return out, nil
}
func (s *Store) questionRelated(ctx context.Context, p question.QuestionPackage, workspace string) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, questionError(err)
	}
	defer tx.Rollback()
	if err = questionConfigured(ctx, tx); err != nil {
		return nil, questionError(err)
	}
	responsible, err := questionLocalResponsibility(ctx, tx, p)
	if err != nil {
		return nil, questionError(err)
	}
	ids := map[string]bool{}
	for _, id := range responsible.AuthorIDs {
		ids[id] = true
	}
	if workspace != "" {
		rows, err := tx.QueryContext(ctx, `SELECT user_id::text FROM question_workspace_authors WHERE workspace_id=$1`, workspace)
		if err != nil {
			return nil, questionError(err)
		}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return nil, questionError(err)
			}
			ids[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, questionError(err)
		}
	}
	return sortedQuestionAuthors(ids), questionError(tx.Commit())
}
func questionAuthorsLocked(ids []string, related []string, actor string) bool {
	set := map[string]bool{actor: true}
	for _, id := range related {
		set[id] = true
	}
	for _, id := range ids {
		if !set[id] {
			return false
		}
	}
	return true
}
func (s *Store) readQuestionDraft(ctx context.Context, tx *sql.Tx, u auth.User, id string, own bool) (question.DraftView, string, error) {
	var d question.DraftView
	var pkg, sources, gate []byte
	var base string
	var created, updated time.Time
	err := tx.QueryRowContext(ctx, `SELECT w.id::text,w.owner_user_id::text,w.catalogue_version,c.sha256,w.revision,w.status,w.package,w.source_map,w.legacy_unattributed,w.gate,coalesce(w.base_submission_id::text,''),w.created_at,w.updated_at FROM question_workspaces w JOIN catalogue_versions c ON c.version=w.catalogue_version WHERE w.id=$1`, id).Scan(&d.ID, &d.OwnerID, &d.CatalogueVersion, &d.CatalogueSHA256, &d.Revision, &d.Status, &pkg, &sources, &d.LegacyUnattributed, &gate, &base, &created, &updated)
	if err != nil {
		return d, base, workflowRowError(err)
	}
	if d.OwnerID != u.ID && (own || !publication.HasRole(u, auth.RoleAdmin)) {
		return question.DraftView{}, "", auth.ErrNotFound
	}
	for _, pair := range []struct {
		raw []byte
		out any
	}{{pkg, &d.QuestionPackage}, {sources, &d.SourceMap}, {gate, &d.Gate}} {
		if json.Unmarshal(pair.raw, pair.out) != nil {
			return d, base, auth.ErrUnavailable
		}
	}
	d.AuthorIDs = []string{}
	rows, err := tx.QueryContext(ctx, `SELECT user_id::text FROM question_workspace_authors WHERE workspace_id=$1 ORDER BY user_id`, id)
	if err != nil {
		return d, base, err
	}
	defer rows.Close()
	for rows.Next() {
		var author string
		if err = rows.Scan(&author); err != nil {
			return d, base, err
		}
		d.AuthorIDs = append(d.AuthorIDs, author)
	}
	d.CreatedAt = created.UTC().Format(time.RFC3339)
	d.UpdatedAt = updated.UTC().Format(time.RFC3339)
	return d, base, rows.Err()
}
func (s *Store) createQuestionDraft(ctx context.Context, tx *sql.Tx, u auth.User, input question.DraftInput, base string, inherited []string, legacy bool, related []string, now time.Time) (question.DraftView, error) {
	var d question.DraftView
	gate, err := questionDraftStructure(input)
	if err != nil {
		return d, err
	}
	_, sha, err := workflowCatalogue(ctx, tx, input.CatalogueVersion)
	if err != nil {
		return d, err
	}
	responsible, err := questionAuthors(ctx, tx, u.ID, input.QuestionPackage, inherited, legacy)
	if err != nil {
		return d, err
	}
	if !questionAuthorsLocked(responsible.AuthorIDs, related, u.ID) {
		return d, question.ErrDraftConflict
	}
	id, err := workflowID()
	if err != nil {
		return d, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO question_workspaces(id,owner_user_id,catalogue_version,package,source_map,legacy_unattributed,base_submission_id,revision,gate,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$9,$9)`, id, u.ID, input.CatalogueVersion, body(input.QuestionPackage), body(input.SourceMap), responsible.LegacyUnattributed, workflowNullableID(base), body(gate), now)
	if err != nil {
		return d, err
	}
	for _, author := range responsible.AuthorIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO question_workspace_authors VALUES($1,$2)`, id, author); err != nil {
			return d, err
		}
	}
	d = question.DraftView{ID: id, OwnerID: u.ID, CatalogueVersion: input.CatalogueVersion, CatalogueSHA256: sha, Revision: 1, Status: "editing", QuestionPackage: input.QuestionPackage, SourceMap: input.SourceMap, AuthorIDs: responsible.AuthorIDs, LegacyUnattributed: responsible.LegacyUnattributed, Gate: gate, CreatedAt: now.UTC().Format(time.RFC3339), UpdatedAt: now.UTC().Format(time.RFC3339)}
	return d, nil
}
func (s *Store) CreateQuestionDraft(ctx context.Context, a question.Access, input question.DraftInput) (question.DraftView, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var out question.DraftView
	if _, err := questionDraftStructure(input); err != nil {
		return out, err
	}
	related, err := s.questionRelated(ctx, input.QuestionPackage, "")
	if err != nil {
		return out, err
	}
	err = s.questionTx(ctx, a, question.CreateDraftAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, found, err := questionJSONReplay[question.DraftView](s, ctx, tx, u, a, question.CreateDraftAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		out, err = s.createQuestionDraft(ctx, tx, u, input, "", nil, false, related, now)
		if err != nil {
			return err
		}
		if err = questionEvent(ctx, tx, u, a, question.CreateDraftAction, "draft", out.ID, "", out.Gate.Digest, "", "editing:1", "", now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.CreateDraftAction, "", input, out)
	})
	return out, err
}
func (s *Store) ReadQuestionDraft(ctx context.Context, a question.Access, id string) (question.DraftView, error) {
	var out question.DraftView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.questionReadTx(ctx, a, question.ReadDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var err error
		out, _, err = s.readQuestionDraft(ctx, tx, u, id, false)
		return err
	})
	return out, err
}
func (s *Store) SaveQuestionDraft(ctx context.Context, a question.Access, id string, input question.SaveDraftInput) (question.DraftView, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var out question.DraftView
	if !question.ValidID(id) || input.ExpectedRevision < 1 {
		return out, auth.ErrInvalidInput
	}
	gate, err := questionDraftStructure(input.DraftInput)
	if err != nil {
		return out, err
	}
	related, err := s.questionRelated(ctx, input.QuestionPackage, id)
	if err != nil {
		return out, err
	}
	err = s.questionTx(ctx, a, question.SaveDraftAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, _, err := s.readQuestionDraft(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		replay, found, err := questionJSONReplay[question.DraftView](s, ctx, tx, u, a, question.SaveDraftAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = replay
			return nil
		}
		if prior.Status != "editing" || prior.Revision != input.ExpectedRevision {
			return question.ErrDraftConflict
		}
		_, sha, err := workflowCatalogue(ctx, tx, input.CatalogueVersion)
		if err != nil {
			return err
		}
		responsible, err := questionAuthors(ctx, tx, u.ID, input.QuestionPackage, prior.AuthorIDs, prior.LegacyUnattributed)
		if err != nil {
			return err
		}
		if !questionAuthorsLocked(responsible.AuthorIDs, related, u.ID) {
			return question.ErrDraftConflict
		}
		out = prior
		out.QuestionPackage = input.QuestionPackage
		out.CatalogueVersion = input.CatalogueVersion
		out.CatalogueSHA256 = sha
		out.SourceMap = input.SourceMap
		out.AuthorIDs = responsible.AuthorIDs
		out.LegacyUnattributed = responsible.LegacyUnattributed
		out.Gate = gate
		out.Revision++
		out.UpdatedAt = now.UTC().Format(time.RFC3339)
		if _, err = tx.ExecContext(ctx, `UPDATE question_workspaces SET catalogue_version=$2,package=$3,source_map=$4,legacy_unattributed=$5,gate=$6,revision=revision+1,updated_at=$7 WHERE id=$1`, id, out.CatalogueVersion, body(out.QuestionPackage), body(out.SourceMap), out.LegacyUnattributed, body(gate), now); err != nil {
			return err
		}
		for _, author := range out.AuthorIDs {
			if _, err = tx.ExecContext(ctx, `INSERT INTO question_workspace_authors VALUES($1,$2) ON CONFLICT DO NOTHING`, id, author); err != nil {
				return err
			}
		}
		if err = questionEvent(ctx, tx, u, a, question.SaveDraftAction, "draft", id, prior.Gate.Digest, out.Gate.Digest, "editing", "editing", "", now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.SaveDraftAction, id, input, out)
	})
	return out, err
}
func (s *Store) ListQuestionDrafts(ctx context.Context, a question.Access, q question.ListQuery) (question.Page[question.DraftSummary], error) {
	pq, err := publication.ValidateList(publication.ListQuery{Scope: q.Scope, Status: q.Status, Limit: q.Limit, Offset: q.Offset}, "editing", "submitted")
	out := question.Page[question.DraftSummary]{Items: []question.DraftSummary{}, Limit: pq.Limit, Offset: pq.Offset}
	if err != nil {
		return out, err
	}
	err = s.questionReadTx(ctx, a, question.ListDraftsAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		all := pq.Scope == "all" || pq.Scope == "" && publication.HasRole(u, auth.RoleAdmin)
		if all && !publication.HasRole(u, auth.RoleAdmin) {
			return auth.ErrForbidden
		}
		where := `WHERE ($1 OR owner_user_id=$2) AND ($3='' OR status=$3)`
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM question_workspaces `+where, all, u.ID, pq.Status).Scan(&out.Total); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id::text,owner_user_id::text,package->>'id',(package->>'version')::integer,status,catalogue_version,revision,coalesce((gate->>'structuralTotal')::integer,0),coalesce((gate->>'completenessTotal')::integer,0),created_at,updated_at FROM question_workspaces `+where+` ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`, all, u.ID, pq.Status, pq.Limit, pq.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item question.DraftSummary
			var created, updated time.Time
			if err = rows.Scan(&item.ID, &item.OwnerID, &item.PackageID, &item.PackageVersion, &item.Status, &item.CatalogueVersion, &item.Revision, &item.StructuralTotal, &item.CompletenessTotal, &created, &updated); err != nil {
				return err
			}
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			item.UpdatedAt = updated.UTC().Format(time.RFC3339)
			out.Items = append(out.Items, item)
		}
		return rows.Err()
	})
	return out, err
}
func (s *Store) ValidateQuestionDraft(ctx context.Context, a question.Access, id string, input question.ValidateInput) (question.ValidationReport, error) {
	var out question.ValidationReport
	if !question.ValidID(id) || input.ExpectedRevision < 1 {
		return out, auth.ErrInvalidInput
	}
	err := s.questionReadTx(ctx, a, question.ValidateDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		d, _, err := s.readQuestionDraft(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		if d.Status != "editing" || d.Revision != input.ExpectedRevision {
			return question.ErrDraftConflict
		}
		in := question.DraftInput{CatalogueVersion: d.CatalogueVersion, QuestionPackage: d.QuestionPackage, SourceMap: d.SourceMap}
		refs, err := questionReferences(ctx, tx, in)
		if err != nil {
			return err
		}
		out, err = question.ValidateEditable(ctx, in, refs)
		return err
	})
	return out, err
}
func (s *Store) AdoptQuestionDraft(ctx context.Context, a question.Access, input question.AdoptInput) (question.DraftView, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var out question.DraftView
	if !question.ValidMathID(input.PackageID) || input.PackageVersion < 1 || input.PackageVersion > 2147483647 || !questionValidAdoption(input.Reason) {
		return out, auth.ErrInvalidInput
	}
	archive, err := s.ExportQuestionArchive(ctx, input.PackageID, input.PackageVersion)
	if err != nil {
		return out, err
	}
	related, err := s.questionRelated(ctx, archive.Envelope.QuestionPackage, "")
	if err != nil {
		return out, err
	}
	err = s.questionTx(ctx, a, question.AdoptDraftAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, found, err := questionJSONReplay[question.DraftView](s, ctx, tx, u, a, question.AdoptDraftAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		arch, err := questionExportTx(ctx, tx, input.PackageID, input.PackageVersion)
		if err != nil {
			return err
		}
		out, err = s.createQuestionDraft(ctx, tx, u, arch.Envelope, "", arch.SourceResponsibility.AuthorIDs, arch.SourceResponsibility.LegacyUnattributed, related, now)
		if err != nil {
			return err
		}
		if err = questionEvent(ctx, tx, u, a, question.AdoptDraftAction, "draft", out.ID, "", out.Gate.Digest, "", "editing:1", input.Reason, now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.AdoptDraftAction, "", input, out)
	})
	return out, err
}
