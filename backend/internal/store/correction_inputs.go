package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
)

type correctionBasisFactsKey struct{}
type correctionBasisFacts struct {
	tx     *sql.Tx
	proofs map[correction.PlanRef]correction.PlanProof
}

func correctionBasisFactsFor(ctx context.Context, tx *sql.Tx) *correctionBasisFacts {
	facts, _ := ctx.Value(correctionBasisFactsKey{}).(*correctionBasisFacts)
	if facts == nil || facts.tx != tx {
		return nil
	}
	return facts
}

type correctionPrior struct {
	id string
	n  int
}
type correctionInputItem struct {
	position   int
	sha        string
	raw, proof []byte
}
type correctionInputPlan struct {
	inputRaw, raw []byte
	err           error
}

func (p correctionInputPlan) decode() (correction.PlanProof, error) {
	if p.err != nil {
		return correction.PlanProof{}, p.err
	}
	var input correction.PlanInput
	if json.Unmarshal(p.inputRaw, &input) != nil {
		return correction.PlanProof{}, auth.ErrUnavailable
	}
	return correctionDecodeFrozenPlan(p.raw, input)
}

type correctionPracticeInputs struct {
	practice               practiceRecord
	items                  []correctionInputItem
	deps                   []correction.Dependency
	proofs                 []correctionProofRecord
	planInputs             map[correction.PlanRef]correctionInputPlan
	old                    []correctionPrior
	parentRaw              map[string][]byte
	parentErrors           map[string]error
	planBytes, parentBytes int
}

// 所有元数据和字节来自准确事务内的原查询。原题解码、批准与父结果的
// 分阶段检查留在调用者；超过原2MiB/100项上限时不读取对应正文。
func correctionReadPracticeInputs(ctx context.Context, tx *sql.Tx, meta correctionEvidenceMetadata, caseID string, plan *correction.PlanRef, source string) (*correctionPracticeInputs, error) {
	inputs := &correctionPracticeInputs{deps: []correction.Dependency{}, proofs: []correctionProofRecord{}, planInputs: map[correction.PlanRef]correctionInputPlan{}, old: []correctionPrior{}, parentRaw: map[string][]byte{}, parentErrors: map[string]error{}}
	batch := &pgx.Batch{}
	batch.Queue(learningReadPracticeSQL+" FOR UPDATE", meta.Ref.ID, meta.Owner)
	batch.Queue(learningEvidenceDependenciesSQL, meta.Ref.Kind, meta.Ref.ID)
	if plan != nil {
		batch.Queue(correctionApprovedPlanMetadataSQL, meta.Ref.Kind, meta.Ref.ID, meta.Owner, caseID)
		batch.Queue(correctionPriorMetadataSQL, meta.Owner, meta.Ref.Kind, meta.Ref.ID, source)
	}
	var raw, answer []byte
	var terminal sql.NullTime
	supported, err := correctionReadBatch(ctx, tx, batch, func(results pgx.BatchResults) error {
		p := &inputs.practice
		if e := results.QueryRow().Scan(&p.Summary.ID, &p.Summary.Knowledge.ID, &p.Summary.Knowledge.Version, &p.Summary.Knowledge.SHA256, &p.Summary.State, &p.Summary.CreatedAt, &p.Summary.ExpiresAt, &terminal, &raw, &answer, &p.Correct); e != nil {
			return workflowRowError(e)
		}
		rows, e := results.Query()
		if e != nil {
			return e
		}
		for rows.Next() {
			var d correction.Dependency
			if e = rows.Scan(&d.Kind, &d.ID, &d.Version, &d.SHA256); e != nil {
				rows.Close()
				return e
			}
			inputs.deps = append(inputs.deps, d)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if plan == nil {
			return nil
		}
		rows, e = results.Query()
		if e != nil {
			return e
		}
		for rows.Next() {
			var p correctionProofRecord
			var id *string
			var version *int
			if e = rows.Scan(&p.Ref.ID, &p.Ref.Version, &p.CaseID, &id, &version, &p.Bytes); e != nil {
				rows.Close()
				return e
			}
			if id != nil && version != nil {
				p.Parent = &correction.PlanRef{ID: *id, Version: *version}
			}
			inputs.proofs = append(inputs.proofs, p)
			inputs.planBytes += p.Bytes
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		rows, e = results.Query()
		if e != nil {
			return e
		}
		for rows.Next() {
			var p correctionPrior
			if e = rows.Scan(&p.id, &p.n); e != nil {
				rows.Close()
				return e
			}
			inputs.old = append(inputs.old, p)
			inputs.parentBytes += p.n
		}
		e = rows.Err()
		rows.Close()
		return e
	})
	if !supported && err == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	inputs.practice, err = learningDecodePracticeRecord(inputs.practice, raw, answer, terminal)
	if err != nil {
		return nil, err
	}
	p := inputs.practice
	if p.Summary.State != "answered" || p.Answer == nil || len(p.Seal.Items) != 1 && len(p.Seal.Items) != 5 {
		return inputs, nil
	}
	details := &pgx.Batch{}
	details.Queue(learningItemsSQL, body(p.Seal.Items), p.Seal.QuestionPublicationID)
	plansWithinLimit := plan != nil && len(inputs.proofs) <= 100 && inputs.planBytes <= correction.MaxResponseBytes
	if plansWithinLimit {
		for _, p := range inputs.proofs {
			details.Queue(`SELECT body,frozen_bytes FROM correction_plans WHERE id=$1 AND version=$2`, p.Ref.ID, p.Ref.Version)
		}
	}
	parentsWithinLimit := plansWithinLimit && len(inputs.old) <= 100 && inputs.planBytes+inputs.parentBytes <= correction.MaxResponseBytes
	if parentsWithinLimit {
		for _, p := range inputs.old {
			details.Queue(`SELECT basis_bytes FROM correction_results WHERE id=$1`, p.id)
		}
	}
	supported, err = correctionReadBatch(ctx, tx, details, func(results pgx.BatchResults) error {
		rows, e := results.Query()
		if e != nil {
			return e
		}
		for rows.Next() {
			var i correctionInputItem
			if e = rows.Scan(&i.position, &i.sha, &i.raw, &i.proof); e != nil {
				rows.Close()
				return e
			}
			inputs.items = append(inputs.items, i)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if plansWithinLimit {
			for _, p := range inputs.proofs {
				var input correctionInputPlan
				input.err = workflowRowError(results.QueryRow().Scan(&input.inputRaw, &input.raw))
				inputs.planInputs[p.Ref] = input
			}
		}
		if parentsWithinLimit {
			for _, p := range inputs.old {
				var raw []byte
				e = results.QueryRow().Scan(&raw)
				if e != nil {
					inputs.parentErrors[p.id] = e
				} else {
					inputs.parentRaw[p.id] = raw
				}
			}
		}
		return nil
	})
	if !supported && err == nil {
		return nil, auth.ErrUnavailable
	}
	return inputs, err
}

func (inputs *correctionPracticeInputs) base() (correction.Basis, error) {
	b := correctionEmptyBasis()
	b.AuditDeps = append(b.AuditDeps, inputs.deps...)
	b.EffectiveDeps = append(b.EffectiveDeps, b.AuditDeps...)
	p := inputs.practice
	if p.Summary.State != "answered" || p.Answer == nil {
		return b, correction.ErrSourceStale
	}
	b.OriginalSeal = p.Seal
	b.OriginalAnswers = append(b.OriginalAnswers, *p.Answer)
	if len(p.Seal.Items) != 1 && len(p.Seal.Items) != 5 {
		return b, auth.ErrInvalidInput
	}
	for _, item := range inputs.items {
		q, e := learningDecodeBoundItem(p.Seal, item.position, item.sha, item.raw, item.proof)
		if e != nil {
			return b, e
		}
		b.OriginalItems = append(b.OriginalItems, q)
	}
	if len(b.OriginalItems) != len(p.Seal.Items) {
		return b, auth.ErrNotFound
	}
	b.EffectiveItems = append(b.EffectiveItems, b.OriginalItems...)
	return b, nil
}
