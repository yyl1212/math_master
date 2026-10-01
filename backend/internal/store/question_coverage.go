package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func questionEmptyCandidate(ctx context.Context, tx *sql.Tx) (question.Candidate, error) {
	c := question.Candidate{Manifest: question.Manifest{Members: []question.ManifestMember{}, Resolved: []question.KnownObject{}}, Templates: []question.Template{}, Instances: []question.Instance{}, Blueprints: []question.Blueprint{}, Changes: []question.Change{}}
	var version int
	err := tx.QueryRowContext(ctx, `SELECT s.catalogue_version FROM publication_heads h JOIN publication_snapshots s ON s.id=h.snapshot_id AND s.status='published' WHERE h.singleton`).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `SELECT coalesce(max(version),0) FROM catalogue_versions`).Scan(&version)
	}
	if err != nil {
		return c, err
	}
	if version == 0 {
		return c, question.ErrNotConfigured
	}
	_, sha, err := workflowCatalogue(ctx, tx, version)
	c.Manifest.CatalogueVersion = version
	c.Manifest.CatalogueSHA256 = sha
	return c, err
}

// Load only identities and prerequisite edges for unbound published nodes. Full goals are already resolved for bank references.
func questionPublishedKnowledge(ctx context.Context, tx *sql.Tx, head *string) ([]question.FixedKnowledge, error) {
	out := []question.FixedKnowledge{}
	if head == nil {
		return out, nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT k.id,k.version,k.sha256,k.body->'relations' FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.availability='active' AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='knowledge' AND w.target_id=k.id AND w.target_version=k.version) ORDER BY k.id,k.version`, *head)
	if err != nil {
		return out, err
	}
	known := map[question.Ref]question.Identity{}
	edges := map[question.Ref][]question.Ref{}
	for rows.Next() {
		var i question.Identity
		var raw []byte
		if err = rows.Scan(&i.ID, &i.Version, &i.SHA256, &raw); err != nil {
			rows.Close()
			return out, err
		}
		var relations []content.Relation
		if json.Unmarshal(raw, &relations) != nil {
			rows.Close()
			return out, auth.ErrUnavailable
		}
		ref := question.Ref{ID: i.ID, Version: i.Version}
		known[ref] = i
		for _, r := range relations {
			if r.Kind == "prerequisite" {
				edges[ref] = append(edges[ref], r.Target)
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	color := map[question.Ref]int{}
	eligible := map[question.Ref]bool{}
	var check func(question.Ref) bool
	check = func(ref question.Ref) bool {
		if color[ref] == 1 {
			return false
		}
		if color[ref] == 2 {
			return eligible[ref]
		}
		if _, exists := known[ref]; !exists {
			return false
		}
		color[ref] = 1
		valid := true
		for _, target := range edges[ref] {
			if !check(target) {
				valid = false
			}
		}
		color[ref] = 2
		eligible[ref] = valid
		return valid
	}
	for ref, i := range known {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if check(ref) {
			out = append(out, question.FixedKnowledge{Identity: i, Objectives: []string{}})
		}
	}
	return out, nil
}
func questionCoverageState(ctx context.Context, tx *sql.Tx) (question.Candidate, question.ReferenceSnapshot, *string, error) {
	var c question.Candidate
	var refs question.ReferenceSnapshot
	head, err := questionHead(ctx, tx)
	if err != nil {
		return c, refs, head, err
	}
	if head == nil {
		c, err = questionEmptyCandidate(ctx, tx)
	} else {
		c, err = questionLoadCandidate(ctx, tx, *head)
	}
	if err != nil {
		return c, refs, head, err
	}
	refs, err = questionReferences(ctx, tx, question.CandidateDraft(c))
	if err != nil {
		return c, refs, head, err
	}
	if err = question.CheckCandidate(ctx, c, refs, false); err != nil {
		return c, refs, head, err
	}
	if err = questionEvidence(ctx, tx, c.Manifest, false); err != nil {
		return c, refs, head, err
	}
	if err = questionBlacklisted(ctx, tx, c.Manifest); err != nil {
		return c, refs, head, err
	}
	nodes, err := questionPublishedKnowledge(ctx, tx, refs.KnowledgeHead)
	if err != nil {
		return c, refs, head, err
	}
	have := map[question.Ref]bool{}
	for _, k := range refs.Knowledge {
		have[question.Ref{ID: k.Identity.ID, Version: k.Identity.Version}] = true
	}
	for _, k := range nodes {
		if !have[question.Ref{ID: k.Identity.ID, Version: k.Identity.Version}] {
			refs.Knowledge = append(refs.Knowledge, k)
		}
	}
	return c, refs, head, nil
}
func (s *Store) ReadQuestionCoverage(ctx context.Context, a question.Access, q question.CoverageQuery) (question.CoverageReport, error) {
	var out question.CoverageReport
	pq, err := publication.ValidateList(publication.ListQuery{Limit: q.Limit, Offset: q.Offset})
	if err != nil || q.KnowledgeID != "" && !question.ValidMathID(q.KnowledgeID) {
		return out, auth.ErrInvalidInput
	}
	err = s.questionReadTx(ctx, a, question.ReadCoverageAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		c, refs, head, err := questionCoverageState(ctx, tx)
		if err != nil {
			return err
		}
		out, err = question.ComputeCoverage(ctx, c, refs, head)
		if err != nil {
			return err
		}
		selected := []question.CoverageNode{}
		for _, node := range out.Nodes.Items {
			if q.KnowledgeID == "" || node.Knowledge.ID == q.KnowledgeID {
				selected = append(selected, node)
			}
		}
		out.Nodes.Total = len(selected)
		out.Nodes.Limit = pq.Limit
		out.Nodes.Offset = pq.Offset
		start := pq.Offset
		if start > len(selected) {
			start = len(selected)
		}
		end := start + pq.Limit
		if end > len(selected) {
			end = len(selected)
		}
		out.Nodes.Items = selected[start:end]
		return questionFitPage(&out.Nodes, func() any { return out })
	})
	return out, err
}
