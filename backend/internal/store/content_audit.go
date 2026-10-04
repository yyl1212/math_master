package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"time"
)

// ReadContentAudit is an operator-only store method. No HTTP handler exposes it.
func (s *Store) ReadContentAudit(ctx context.Context, route content.VersionRef) (contentaudit.PublishedFacts, error) {
	out := contentaudit.PublishedFacts{Approvals: []contentaudit.ApprovalFact{}, EligibleInstances: []question.Identity{}, Excluded: []contentaudit.Exclusion{}}
	if !question.ValidMathID(route.ID) || route.Version < 1 || route.Version > 2147483647 {
		return out, contentaudit.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if e != nil {
		return out, readError(e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `SET LOCAL statement_timeout='8s'`); e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, `SET LOCAL lock_timeout='1s'`); e != nil {
		return out, e
	}
	var kh, qh sql.NullString
	var database string
	e = tx.QueryRowContext(ctx, `SELECT current_database(),(SELECT snapshot_id FROM publication_heads WHERE singleton),(SELECT publication_id::text FROM question_heads WHERE singleton)`).Scan(&database, &kh, &qh)
	if e != nil {
		return out, e
	}
	out.FixtureOnly = strings.HasPrefix(database, "math_master_test_")
	if !kh.Valid {
		return out, ErrNotFound
	}
	khead := kh.String
	k, e := s.loadWorkflowCandidate(ctx, tx, &khead)
	if e != nil {
		return out, e
	}
	if e = workflowBlacklisted(ctx, tx, k.Manifest); e != nil {
		return out, e
	}
	var ksha string
	if e = tx.QueryRowContext(ctx, `SELECT sha256 FROM content_publication_manifests WHERE snapshot_id=$1`, khead).Scan(&ksha); e != nil {
		return out, e
	}
	out.KnowledgeHead = &contentaudit.Head{ID: khead, SHA256: ksha}
	out.CatalogueVersion = k.Manifest.CatalogueVersion
	cat, csha, e := workflowCatalogue(ctx, tx, out.CatalogueVersion)
	if e != nil || csha != k.Manifest.CatalogueSHA256 {
		return out, contentaudit.ErrInvalid
	}
	out.CatalogueSHA = csha
	// Full head validation precedes scoping. A corrupt unrelated object cannot be silently hidden.
	report, e := content.ValidateSnapshot(ctx, cat, k.Snapshot, workflowDatabaseAssets(tx))
	if e != nil || !report.ReadyToSubmit {
		return out, contentaudit.ErrInvalid
	}
	found := false
	for _, p := range k.Snapshot.Paths {
		if p.ID == route.ID && p.Version == route.Version {
			out.Path = p
			out.PathSHA = content.Digest(p)
			found = true
			break
		}
	}
	if !found {
		return out, ErrNotFound
	}
	scope := map[content.VersionRef]bool{}
	for _, r := range out.Path.Nodes {
		scope[r] = true
	}
	out.Content = content.Snapshot{CatalogueVersion: out.CatalogueVersion, Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{out.Path}, Assets: []content.Asset{}, Bindings: []content.AssetBinding{}}
	unitRefs := map[content.VersionRef]bool{}
	for _, x := range k.Snapshot.Knowledge {
		if scope[content.VersionRef{ID: x.ID, Version: x.Version}] {
			out.Content.Knowledge = append(out.Content.Knowledge, x)
		}
	}
	for _, u := range k.Snapshot.Units {
		if scope[u.Knowledge] {
			out.Content.Units = append(out.Content.Units, u)
			unitRefs[content.VersionRef{ID: u.ID, Version: u.Version}] = true
		}
	}
	for _, a := range k.Snapshot.Assets {
		if scope[a.Knowledge] {
			out.Content.Assets = append(out.Content.Assets, a)
		}
	}
	for _, b := range k.Snapshot.Bindings {
		if unitRefs[b.Unit] {
			out.Content.Bindings = append(out.Content.Bindings, b)
		}
	}
	out.Bank = question.Candidate{Templates: []question.Template{}, Blueprints: []question.Blueprint{}, Instances: []question.Instance{}, Manifest: question.Manifest{CatalogueVersion: out.CatalogueVersion, CatalogueSHA256: csha, Members: []question.ManifestMember{}, Resolved: []question.KnownObject{}}}
	var fullBank question.Candidate
	if qh.Valid {
		var published bool
		var metadataBytes int
		var sha string
		e = tx.QueryRowContext(ctx, `SELECT sealed AND status='published',manifest_sha256,octet_length(manifest_bytes) FROM question_publications WHERE id=$1`, qh.String).Scan(&published, &sha, &metadataBytes)
		if e != nil || !published {
			return out, contentaudit.ErrInvalid
		}
		if metadataBytes > question.MaxManifestBytes {
			return out, question.ErrLimitExceeded
		}
		out.QuestionHead = &contentaudit.Head{ID: qh.String, SHA256: sha}
		fullBank, e = questionLoadCandidate(ctx, tx, qh.String)
		if e != nil {
			return out, e
		}
		if e = question.CheckCandidate(ctx, fullBank, question.ReferenceSnapshot{}, false); e != nil {
			return out, e
		}
		if e = questionEvidence(ctx, tx, fullBank.Manifest, false); e != nil {
			return out, e
		}
		if e = questionBlacklisted(ctx, tx, fullBank.Manifest); e != nil {
			return out, e
		}
		out.Bank.Manifest = fullBank.Manifest
		for _, t := range fullBank.Templates {
			if scope[t.Knowledge] {
				out.Bank.Templates = append(out.Bank.Templates, t)
			}
		}
		for _, b := range fullBank.Blueprints {
			if scope[b.Knowledge] {
				out.Bank.Blueprints = append(out.Bank.Blueprints, b)
			}
		}
		for _, i := range fullBank.Instances {
			if scope[i.Body.Knowledge] {
				out.Bank.Instances = append(out.Bank.Instances, i)
			}
		}
	}
	// Verify frozen bytes and normalized authors once per submission, even for a 10000-instance bank.
	allContent, e := auditApprovals(ctx, tx, "content", khead)
	if e != nil {
		return out, e
	}
	allQuestions := []contentaudit.ApprovalFact{}
	if qh.Valid {
		allQuestions, e = auditApprovals(ctx, tx, "question", qh.String)
		if e != nil {
			return out, e
		}
	}
	wanted := map[string]bool{}
	add := func(kind, id string, v int) { wanted[fmt.Sprintf("%s:%s:%d", kind, id, v)] = true }
	for _, x := range out.Content.Knowledge {
		add("knowledge", x.ID, x.Version)
	}
	for _, u := range out.Content.Units {
		add("unit", u.ID, u.Version)
	}
	for _, a := range out.Content.Assets {
		add("asset", a.ID, 1)
	}
	add("path", out.Path.ID, out.Path.Version)
	for _, t := range out.Bank.Templates {
		add("template", t.ID, t.Version)
	}
	for _, b := range out.Bank.Blueprints {
		add("blueprint", b.ID, b.Version)
	}
	for _, i := range out.Bank.Instances {
		add("instance", i.Identity.ID, i.Identity.Version)
	}
	for _, a := range append(allContent, allQuestions...) {
		v := 1
		if a.Object.Version != nil {
			v = *a.Object.Version
		}
		if wanted[fmt.Sprintf("%s:%s:%d", a.Object.Kind, a.Object.ID, v)] {
			out.Approvals = append(out.Approvals, a)
		}
	}
	if qh.Valid {
		rows, e := tx.QueryContext(ctx, auditCandidateSQL(), body(out.Path.Nodes), khead, qh.String)
		if e != nil {
			return out, e
		}
		eligible := map[question.Identity]bool{}
		for rows.Next() {
			var i question.Identity
			if e = rows.Scan(&i.ID, &i.Version, &i.SHA256); e != nil {
				rows.Close()
				return out, e
			}
			eligible[i] = true
			if len(eligible) > question.MaxBankInstances {
				rows.Close()
				return out, question.ErrLimitExceeded
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
		eligibleTemplates := map[question.Identity]bool{}
		for _, i := range out.Bank.Instances {
			if eligible[i.Identity] {
				out.EligibleInstances = append(out.EligibleInstances, i.Identity)
				if i.Template != nil {
					eligibleTemplates[*i.Template] = true
				}
			} else {
				v := i.Identity.Version
				out.Excluded = append(out.Excluded, contentaudit.Exclusion{Object: contentaudit.ObjectIdentity{Kind: "instance", ID: i.Identity.ID, Version: &v, SHA256: i.Identity.SHA256}, Code: "CURRENT_DEPENDENCY_OR_RULE_RESTRICTED"})
			}
		}
		for _, t := range out.Bank.Templates {
			_, sha, e := question.CanonicalTemplate(t)
			if e != nil {
				return out, e
			}
			if !eligibleTemplates[question.Identity{ID: t.ID, Version: t.Version, SHA256: sha}] {
				v := t.Version
				out.Excluded = append(out.Excluded, contentaudit.Exclusion{Object: contentaudit.ObjectIdentity{Kind: "template", ID: t.ID, Version: &v, SHA256: sha}, Code: "TEMPLATE_HAS_NO_CURRENT_ELIGIBLE_INSTANCE"})
			}
		}

	}
	if e = tx.Commit(); e != nil {
		return out, readError(e)
	}
	return out, nil
}
func auditApprovals(ctx context.Context, tx *sql.Tx, space, head string) ([]contentaudit.ApprovalFact, error) {
	out := []contentaudit.ApprovalFact{}
	var query string
	switch space {
	case "content":
		query = `WITH selected AS MATERIALIZED (SELECT DISTINCT (e->'evidence'->>'submissionId')::uuid id FROM content_publication_manifests cm CROSS JOIN LATERAL jsonb_array_elements(cm.manifest->'members') e WHERE cm.snapshot_id=$1), verified AS MATERIALIZED (
 SELECT s.id,s.frozen_digest,s.frozen_body->'authorIds' authors,
 (encode(sha256(s.frozen_bytes),'hex')=s.frozen_digest AND convert_from(s.frozen_bytes,'UTF8')::jsonb=s.frozen_body AND s.frozen_body->>'purpose'='math-master/frozen-submission/v1' AND
 coalesce((SELECT jsonb_agg(a.user_id::text ORDER BY a.user_id) FROM content_submission_authors a WHERE a.submission_id=s.id),'[]')=s.frozen_body->'authorIds') valid
 FROM selected r JOIN content_submissions s ON s.id=r.id AND s.sealed AND s.status='approved')
 SELECT e->'identity',e->'evidence',v.authors,r.reviewer_user_id::text,
 r.checks @> '{"mathematics":true,"explanations":true,"relationships":true,"sources":true,"illustrations":true}',v.valid AND r.frozen_digest=v.frozen_digest
 FROM content_publication_manifests cm CROSS JOIN LATERAL jsonb_array_elements(cm.manifest->'members') e JOIN verified v ON v.id=(e->'evidence'->>'submissionId')::uuid JOIN content_review_decisions r ON r.id=(e->'evidence'->>'decisionId')::uuid WHERE cm.snapshot_id=$1 ORDER BY e->'identity'->>'kind',e->'identity'->>'id'`
	case "question":
		query = `WITH selected AS MATERIALIZED (SELECT DISTINCT submission_id id FROM question_publication_members WHERE publication_id=$1), verified AS MATERIALIZED (
 SELECT s.id,s.frozen_digest,s.frozen_body#>'{body,body,authorIds}' authors,
 (encode(sha256(s.frozen_bytes),'hex')=s.frozen_digest AND convert_from(s.frozen_bytes,'UTF8')::jsonb=s.frozen_body AND s.frozen_body->>'purpose'='question-submission-v1' AND
 coalesce((SELECT jsonb_agg(a.user_id::text ORDER BY a.user_id) FROM question_submission_authors a WHERE a.submission_id=s.id),'[]')=s.frozen_body#>'{body,body,authorIds}') valid
 FROM selected r JOIN question_submissions s ON s.id=r.id AND s.sealed AND s.status='approved')
 SELECT jsonb_build_object('kind',m.kind,'id',m.id,'version',m.version,'sha256',m.sha256),m.evidence,v.authors,r.reviewer_user_id::text,
 r.checks @> '{"mathematics":true,"explanations":true,"objectives":true,"sources":true,"illustrations":true,"generation":true}',v.valid AND r.frozen_digest=v.frozen_digest
 FROM question_publication_members m JOIN verified v ON v.id=m.submission_id JOIN question_review_decisions r ON r.id=m.review_id WHERE m.publication_id=$1 ORDER BY m.kind,m.id`
	default:
		return out, contentaudit.ErrInvalid
	}
	rows, e := tx.QueryContext(ctx, query, head)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		a := contentaudit.ApprovalFact{Space: space}
		var object, proof, authors []byte
		if e = rows.Scan(&object, &proof, &authors, &a.ReviewerID, &a.ChecksComplete, &a.FrozenMatches); e != nil {
			return out, e
		}
		var id struct {
			Kind, ID, SHA256 string
			Version          int
		}
		if json.Unmarshal(object, &id) != nil || json.Unmarshal(proof, &a.Evidence) != nil || json.Unmarshal(authors, &a.AuthorIDs) != nil || !a.FrozenMatches || !a.ChecksComplete || len(a.AuthorIDs) == 0 {
			return out, contentaudit.ErrInvalid
		}
		a.Object = contentaudit.ObjectIdentity{Kind: id.Kind, ID: id.ID, Version: &id.Version, SHA256: id.SHA256}
		if id.Kind == "asset" {
			a.Object.Version = nil
		}
		out = append(out, a)
		if len(out) > 17200 {
			return out, question.ErrLimitExceeded
		}
	}
	return out, rows.Err()
}

// Reuse the existing public eligibility predicates but remove personal joins and blueprint filtering.
// The audit assesses static availability; exposure is tested by the pure worst-case witness instead.
func auditCandidateSQL() string {
	base := learningCandidateSQL
	cteStart := strings.Index(base, "available_units AS MATERIALIZED")
	selectStart := strings.Index(base, " SELECT i.id")
	fromStart := strings.Index(base, " FROM question_instances")
	personalStart := strings.Index(base, " LEFT JOIN learner_question_views")
	whereStart := strings.Index(base, " WHERE i.sealed")
	bpStart := strings.Index(base, " AND ($6::text")
	withdrawStart := strings.Index(base, " AND NOT EXISTS(SELECT 1 FROM question_withdrawals")
	orderStart := strings.LastIndex(base, " ORDER BY i.id")
	if cteStart < 0 || selectStart < 0 || fromStart < 0 || personalStart < 0 || whereStart < 0 || bpStart < 0 || withdrawStart < 0 || orderStart < 0 {
		panic("assessment eligibility contract changed")
	}
	where := base[whereStart:bpStart] + base[withdrawStart:orderStart]
	where = strings.Replace(where, "i.knowledge_id=$2 AND i.knowledge_version=$3", "(i.knowledge_id,i.knowledge_version) IN(SELECT id,version FROM jsonb_to_recordset($1::jsonb) r(id text,version integer))", 1)
	query := "WITH " + base[cteStart:selectStart] + " SELECT i.id,i.version,i.sha256" + base[fromStart:personalStart] + where
	query = strings.NewReplacer("$4", "$2", "$5", "$3").Replace(query)
	return query + ` AND NOT EXISTS(SELECT 1 FROM correction_cases c WHERE c.sealed AND c.kind='grading_rule' AND c.rule_version=1 AND (c.scope_kind='all' OR (c.knowledge_id=i.knowledge_id AND c.knowledge_version=i.knowledge_version AND c.knowledge_sha256=(SELECT k.sha256 FROM knowledge_versions k WHERE k.id=i.knowledge_id AND k.version=i.knowledge_version))) AND NOT EXISTS(SELECT 1 FROM correction_plans p WHERE p.case_id=c.id AND p.status='approved' AND p.sealed AND p.algorithm_version=1 AND EXISTS(SELECT 1 FROM correction_events e WHERE e.case_id=c.id AND e.subject_kind='plan' AND e.subject_id=p.id AND e.subject_version=p.version AND e.sequence=p.sequence AND e.kind='plan_approved'))) ORDER BY i.id,i.version LIMIT 10001`
}
