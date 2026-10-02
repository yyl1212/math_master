package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func questionOfferable(ctx context.Context, tx *sql.Tx, target question.WithdrawalTarget) (bool, error) {
	if err := question.ValidateWithdrawalTarget(target); err != nil {
		return false, err
	}
	c, refs, head, err := questionCoverageState(ctx, tx)
	if err != nil {
		return false, err
	}
	present := false
	for _, m := range c.Manifest.Members {
		if m.Identity.Kind == target.Kind && m.Identity.ID == target.ID && m.Identity.Version == target.Version {
			present = true
			break
		}
	}
	if !present {
		return false, nil
	}
	if target.Kind == "blueprint" {
		report, err := question.ComputeCoverage(ctx, c, refs, head)
		if err != nil {
			return false, err
		}
		for _, node := range report.Nodes.Items {
			if node.Blueprint != nil && node.Blueprint.ID == target.ID && node.Blueprint.Version == target.Version {
				return node.Ready, nil
			}
		}
		return false, nil
	}
	in := question.DraftInput{CatalogueVersion: c.Manifest.CatalogueVersion, QuestionPackage: question.QuestionPackage{Templates: []question.Template{}, FixedQuestions: []question.FixedQuestion{}, Blueprints: []question.Blueprint{}}, SourceMap: []question.SourceLink{}}
	if target.Kind == "template" {
		for _, t := range c.Templates {
			if t.ID == target.ID && t.Version == target.Version {
				in.QuestionPackage.Templates = []question.Template{t}
			}
		}
	}
	if target.Kind == "instance" {
		for _, i := range c.Instances {
			if i.Identity.ID == target.ID && i.Identity.Version == target.Version {
				in.QuestionPackage.FixedQuestions = []question.FixedQuestion{{ID: i.Identity.ID, Version: i.Identity.Version, Body: i.Body}}
				if i.Template != nil {
					for _, t := range c.Templates {
						if t.ID == i.Template.ID && t.Version == i.Template.Version {
							in.QuestionPackage.Templates = append(in.QuestionPackage.Templates, t)
						}
					}
				}
			}
		}
	}
	exact, err := questionReferences(ctx, tx, in)
	if err != nil {
		return false, err
	}
	expectedKnowledge := map[question.Ref]bool{}
	expectedUnits := map[question.Ref]bool{}
	expectedAssets := map[string]string{}
	add := func(k question.Ref, coverage []question.ObjectiveCoverage, units []question.Ref, assets []question.AssetRef) {
		expectedKnowledge[k] = true
		for _, m := range coverage {
			expectedKnowledge[m.Knowledge] = true
		}
		for _, u := range units {
			expectedUnits[u] = true
		}
		for _, a := range assets {
			expectedAssets[a.ID] = a.SHA256
		}
	}
	for _, t := range in.QuestionPackage.Templates {
		add(t.Knowledge, t.Coverage, t.Units, t.Assets)
	}
	for _, i := range in.QuestionPackage.FixedQuestions {
		b := i.Body
		add(b.Knowledge, b.Coverage, b.Units, b.Assets)
	}
	if len(exact.Knowledge) != len(expectedKnowledge) || len(exact.Units) != len(expectedUnits) || len(exact.Assets) != len(expectedAssets) {
		return false, nil
	}
	for _, a := range exact.Assets {
		if expectedAssets[a.ID] != a.SHA256 {
			return false, nil
		}
	}
	return true, nil
}
func questionHistoricalFacts(ctx context.Context, tx *sql.Tx, target question.WithdrawalTarget, publicationID *string) (question.HistoricalFacts, error) {
	out := question.HistoricalFacts{Replacements: []question.ReplacementFact{}, Withdrawals: []question.WithdrawalFact{}}
	if publicationID != nil && !question.ValidID(*publicationID) {
		return out, auth.ErrInvalidInput
	}
	sha, err := questionWithdrawalTarget(ctx, tx, target)
	if err != nil {
		return out, err
	}
	out.Identity = question.Identity{ID: target.ID, Version: target.Version, SHA256: sha}
	var evidence []byte
	err = tx.QueryRowContext(ctx, `SELECT m.evidence FROM question_publication_members m JOIN question_publications p ON p.id=m.publication_id AND p.sealed AND p.status='published' WHERE m.kind=$1 AND m.id=$2 AND m.version=$3 AND m.sha256=$4 AND ($5::uuid IS NULL OR p.id=$5::uuid) ORDER BY p.created_at,p.id LIMIT 1`, target.Kind, target.ID, target.Version, sha, publicationID).Scan(&evidence)
	if errors.Is(err, sql.ErrNoRows) {
		if publicationID != nil {
			return out, auth.ErrNotFound
		}
	} else if err != nil {
		return out, err
	} else {
		var proof question.MemberEvidence
		if json.Unmarshal(evidence, &proof) != nil {
			return out, auth.ErrUnavailable
		}
		out.Approval = &proof
	}
	wanted := []any{map[string]any{"before": map[string]any{"kind": target.Kind, "id": target.ID, "version": target.Version, "sha256": sha}}}
	rows, err := tx.QueryContext(ctx, `SELECT p.id::text,p.created_at,c.v FROM question_publications p CROSS JOIN LATERAL jsonb_array_elements(p.changes) c(v) WHERE p.sealed AND p.status='published' AND p.changes @> $1::jsonb AND c.v->'before'->>'kind'=$2 AND c.v->'before'->>'id'=$3 AND (c.v->'before'->>'version')::integer=$4 AND c.v->'before'->>'sha256'=$5 AND c.v->>'reason'<>'permanent withdrawal closure' ORDER BY p.created_at,p.id`, body(wanted), target.Kind, target.ID, target.Version, sha)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var f question.ReplacementFact
		var created time.Time
		var raw []byte
		if err = rows.Scan(&f.PublicationID, &created, &raw); err != nil {
			rows.Close()
			return out, err
		}
		var change question.Change
		if json.Unmarshal(raw, &change) != nil || change.Before == nil {
			rows.Close()
			return out, auth.ErrUnavailable
		}
		f.From = out.Identity
		if change.After != nil {
			f.To = &question.Identity{ID: change.After.ID, Version: change.After.Version, SHA256: change.After.SHA256}
		}
		f.CreatedAt = created.UTC().Format(time.RFC3339)
		out.Replacements = append(out.Replacements, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT w.id::text,w.kind,w.target_id,w.target_version,w.reason,w.created_at FROM question_withdrawals w WHERE (w.kind=$1 AND w.target_id=$2 AND w.target_version=$3)
 OR ($1='instance' AND w.kind='template' AND EXISTS(SELECT 1 FROM question_instances i WHERE i.id=$2 AND i.version=$3 AND i.template_id=w.target_id AND i.template_version=w.target_version))
 OR ($1='blueprint' AND EXISTS(SELECT 1 FROM question_blueprint_sources s WHERE s.blueprint_id=$2 AND s.blueprint_version=$3 AND s.kind=w.kind AND s.id=w.target_id AND s.version=w.target_version)) ORDER BY w.created_at,w.id`, target.Kind, target.ID, target.Version)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var f question.WithdrawalFact
		var created time.Time
		if err = rows.Scan(&f.EventID, &f.Target.Kind, &f.Target.ID, &f.Target.Version, &f.Reason, &created); err != nil {
			return out, err
		}
		f.CreatedAt = created.UTC().Format(time.RFC3339)
		out.Withdrawals = append(out.Withdrawals, f)
	}
	return out, rows.Err()
}
