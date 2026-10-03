package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
)

type correctionProcessingKey struct{}
type correctionEvidenceMetadata struct {
	Ref       correction.EvidenceRef
	Owner     string
	Knowledge *question.Identity
	Terminal  bool
}

func correctionReadEvidenceMetadata(ctx context.Context, tx *sql.Tx, ref correction.EvidenceRef) (correctionEvidenceMetadata, error) {
	out := correctionEvidenceMetadata{Ref: ref}
	var kid, kh *string
	var kv *int
	e := tx.QueryRowContext(ctx, `WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT owner::text,kid,kv,kh,terminal FROM evidence WHERE kind=$1 AND id=$2`, ref.Kind, ref.ID).Scan(&out.Owner, &kid, &kv, &kh, &out.Terminal)
	if kid != nil && kv != nil && kh != nil {
		out.Knowledge = &question.Identity{ID: *kid, Version: *kv, SHA256: *kh}
	}
	return out, workflowRowError(e)
}
func correctionEmptyBasis() correction.Basis {
	return correction.Basis{OriginalAnswers: []assessment.Answer{}, OriginalItems: []question.Instance{}, EffectiveItems: []question.Instance{}, HandledCaseIDs: []string{}, ParentResultIDs: []string{}, PlanRefs: []correction.PlanRef{}, AuditDeps: []correction.Dependency{}, EffectiveDeps: []correction.Dependency{}}
}
func correctionOriginalBasis(ctx context.Context, tx *sql.Tx, meta correctionEvidenceMetadata) (correction.Basis, error) {
	b := correctionEmptyBasis()
	deps, e := learningEvidenceDependencies(ctx, tx, string(meta.Ref.Kind), meta.Ref.ID)
	if e != nil {
		return b, e
	}
	for _, d := range deps {
		b.AuditDeps = append(b.AuditDeps, correction.Dependency{Kind: d.Kind, ID: d.ID, Version: d.Version, SHA256: d.SHA256})
	}
	b.EffectiveDeps = append(b.EffectiveDeps, b.AuditDeps...)
	switch meta.Ref.Kind {
	case correction.AssessmentEvidence:
		record, e := learningReadAssessment(ctx, tx, meta.Owner, meta.Ref.ID, true)
		if e != nil {
			return b, e
		}
		if record.Summary.State != "submitted" {
			return b, correction.ErrSourceStale
		}
		b.OriginalSeal = record.Seal
		rows, e := tx.QueryContext(ctx, `SELECT answer FROM assessment_answers WHERE attempt_id=$1 AND owner_user_id=$2 ORDER BY position`, meta.Ref.ID, meta.Owner)
		if e != nil {
			return b, e
		}
		for rows.Next() {
			var raw []byte
			var a assessment.Answer
			if e = rows.Scan(&raw); e != nil {
				rows.Close()
				return b, e
			}
			if e = json.Unmarshal(raw, &a); e != nil {
				rows.Close()
				return b, auth.ErrUnavailable
			}
			b.OriginalAnswers = append(b.OriginalAnswers, a)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return b, e
		}
	case correction.PracticeEvidence:
		record, e := learningReadPractice(ctx, tx, meta.Owner, meta.Ref.ID, true)
		if e != nil {
			return b, e
		}
		if record.Summary.State != "answered" || record.Answer == nil {
			return b, correction.ErrSourceStale
		}
		b.OriginalSeal = record.Seal
		b.OriginalAnswers = append(b.OriginalAnswers, *record.Answer)
	default:
		return b, nil
	}
	b.OriginalItems, e = learningLoadItems(ctx, tx, b.OriginalSeal)
	if e != nil {
		return b, e
	}
	b.EffectiveItems = append(b.EffectiveItems, b.OriginalItems...)
	return b, nil
}
func correctionUniqueDeps(in []correction.Dependency) []correction.Dependency {
	m := map[string]correction.Dependency{}
	for _, d := range in {
		m[body(d)] = d
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []correction.Dependency{}
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}
func correctionSortedIDs(in []string) []string {
	m := map[string]bool{}
	for _, id := range in {
		m[id] = true
	}
	out := []string{}
	for id := range m {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
func correctionInstanceDeps(ctx context.Context, tx *sql.Tx, i question.Instance) ([]correction.Dependency, error) {
	v := func(n int) *int { return &n }
	out := []correction.Dependency{{Kind: "instance", ID: i.Identity.ID, Version: v(i.Identity.Version), SHA256: i.Identity.SHA256}}
	if i.Template != nil {
		out = append(out, correction.Dependency{Kind: "template", ID: i.Template.ID, Version: v(i.Template.Version), SHA256: i.Template.SHA256})
	}
	var h string
	if e := tx.QueryRowContext(ctx, `SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=$2`, i.Body.Knowledge.ID, i.Body.Knowledge.Version).Scan(&h); e != nil {
		return out, e
	}
	out = append(out, correction.Dependency{Kind: "knowledge", ID: i.Body.Knowledge.ID, Version: v(i.Body.Knowledge.Version), SHA256: h})
	for _, u := range i.Body.Units {
		if e := tx.QueryRowContext(ctx, `SELECT sha256 FROM unit_versions WHERE id=$1 AND version=$2`, u.ID, u.Version).Scan(&h); e != nil {
			return out, e
		}
		out = append(out, correction.Dependency{Kind: "unit", ID: u.ID, Version: v(u.Version), SHA256: h})
		rows, e := tx.QueryContext(ctx, `SELECT asset_id,asset_sha256 FROM unit_asset_bindings WHERE unit_id=$1 AND unit_version=$2`, u.ID, u.Version)
		if e != nil {
			return out, e
		}
		for rows.Next() {
			d := correction.Dependency{Kind: "asset"}
			if e = rows.Scan(&d.ID, &d.SHA256); e != nil {
				rows.Close()
				return out, e
			}
			out = append(out, d)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
	}
	for _, a := range i.Body.Assets {
		out = append(out, correction.Dependency{Kind: "asset", ID: a.ID, SHA256: a.SHA256})
	}
	return correctionUniqueDeps(out), nil
}

type correctionProofRecord struct {
	Ref    correction.PlanRef
	CaseID string
	Parent *correction.PlanRef
	Proof  correction.PlanProof
	Bytes  int
}

func correctionBuildBasis(ctx context.Context, tx *sql.Tx, caseID string, plan *correction.PlanRef, ref correction.EvidenceRef) (correction.Basis, error) {
	meta, e := correctionReadEvidenceMetadata(ctx, tx, ref)
	if e != nil {
		return correctionEmptyBasis(), e
	}
	base, e := correctionOriginalBasis(ctx, tx, meta)
	if e != nil {
		return base, e
	}
	if plan == nil || ref.Kind != correction.AssessmentEvidence && ref.Kind != correction.PracticeEvidence {
		return base, nil
	}
	// Select only applicable approved sources, returning sizes/identities first.
	rows, e := tx.QueryContext(ctx, `WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT p.id::text,p.version,p.case_id::text,p.parent_id::text,p.parent_version,octet_length(p.frozen_bytes) FROM correction_plans p JOIN correction_cases c ON c.id=p.case_id CROSS JOIN evidence e WHERE e.kind=$1 AND e.id=$2 AND e.owner=$3 AND p.sealed AND p.status='approved' AND p.algorithm_version=1 AND (c.id=$4 OR `+correctionCaseAffectsSQL+`) AND EXISTS(SELECT 1 FROM correction_events v WHERE v.subject_kind='plan' AND v.subject_id=p.id AND v.subject_version=p.version AND v.kind='plan_approved') ORDER BY p.id,p.version LIMIT 101`, ref.Kind, ref.ID, meta.Owner, caseID)
	if e != nil {
		return base, e
	}
	proofs := []correctionProofRecord{}
	total := 0
	for rows.Next() {
		var r correctionProofRecord
		var pid *string
		var pv *int
		if e = rows.Scan(&r.Ref.ID, &r.Ref.Version, &r.CaseID, &pid, &pv, &r.Bytes); e != nil {
			rows.Close()
			return base, e
		}
		if pid != nil && pv != nil {
			r.Parent = &correction.PlanRef{ID: *pid, Version: *pv}
		}
		total += r.Bytes
		proofs = append(proofs, r)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return base, e
	}
	if len(proofs) > 100 || total > correction.MaxResponseBytes {
		return base, correction.ErrConflict
	}
	mainFound := false
	for idx := range proofs {
		r := &proofs[idx]
		if r.Ref == *plan {
			mainFound = true
		}
		p, e := correctionReadPlan(ctx, tx, r.Ref, false)
		if e != nil {
			return base, e
		}
		r.Proof = p.Proof
	}
	if !mainFound {
		return base, correction.ErrSourceStale
	}
	job, _ := ctx.Value(correctionProcessingKey{}).(correctionJobRecord)
	parents := []correction.Basis{}
	parentIDs := []string{}
	audit := append([]correction.Dependency{}, base.AuditDeps...)
	rows, e = tx.QueryContext(ctx, `SELECT r.id::text,octet_length(r.basis_bytes) FROM correction_results r WHERE r.owner_user_id=$1 AND r.evidence_kind=$2 AND r.evidence_id=$3 AND r.source_key<>$4 AND `+correctionApprovedResultSQL("r")+` AND `+correctionLeafSQL("r")+` ORDER BY r.id LIMIT 101`, meta.Owner, ref.Kind, ref.ID, job.Source)
	if e != nil {
		return base, e
	}
	type prior struct {
		id string
		n  int
	}
	old := []prior{}
	for rows.Next() {
		var r prior
		if e = rows.Scan(&r.id, &r.n); e != nil {
			rows.Close()
			return base, e
		}
		total += r.n
		old = append(old, r)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return base, e
	}
	if len(old) > 100 || total > correction.MaxResponseBytes {
		return base, correction.ErrConflict
	}
	for _, r := range old {
		var raw []byte
		var wrapped struct {
			Body correction.Basis `json:"body"`
		}
		if e = tx.QueryRowContext(ctx, `SELECT basis_bytes FROM correction_results WHERE id=$1`, r.id).Scan(&raw); e != nil {
			return base, e
		}
		if json.Unmarshal(raw, &wrapped) != nil {
			return base, auth.ErrUnavailable
		}
		parents = append(parents, wrapped.Body)
		parentIDs = append(parentIDs, r.id)
		parentIDs = append(parentIDs, wrapped.Body.ParentResultIDs...)
		audit = append(audit, wrapped.Body.AuditDeps...)
	}
	positions := map[question.Identity][]int{}
	for pos, i := range base.OriginalItems {
		positions[i.Identity] = append(positions[i.Identity], pos)
	}
	for _, b := range parents {
		for pos, i := range b.EffectiveItems {
			if pos < len(base.OriginalItems) {
				positions[i.Identity] = append(positions[i.Identity], pos)
			}
		}
	}
	// Discover intermediate approved identities without altering any original input.
	for depth := 0; depth < 100; depth++ {
		changed := false
		for _, r := range proofs {
			for _, m := range r.Proof.Mappings {
				if pos, ok := positions[m.Original.Identity]; ok {
					if _, known := positions[m.Replacement.Identity]; !known {
						positions[m.Replacement.Identity] = append([]int{}, pos...)
						changed = true
					}
				}
			}
		}
		if !changed {
			break
		}
	}
	type edge struct {
		mapping correction.ResolvedMapping
		proof   correctionProofRecord
	}
	edges := map[string]edge{}
	conflict := false
	refs := []correction.PlanRef{}
	handled := []string{}
	descends := func(child, parent correction.PlanRef) bool {
		for step := 0; step < 100; step++ {
			var next *correction.PlanRef
			for _, r := range proofs {
				if r.Ref == child {
					next = r.Parent
					break
				}
			}
			if next == nil {
				return false
			}
			if *next == parent {
				return true
			}
			child = *next
		}
		return false
	}
	for _, r := range proofs {
		refs = append(refs, r.Ref)
		handled = append(handled, r.CaseID)
		for _, m := range r.Proof.Mappings {
			for _, i := range []question.Instance{m.Original, m.Replacement} {
				deps, e := correctionInstanceDeps(ctx, tx, i)
				if e != nil {
					return base, e
				}
				audit = append(audit, deps...)
			}
			if _, ok := positions[m.Original.Identity]; !ok {
				continue
			}
			key := body(m.Original.Identity)
			if prior, ok := edges[key]; ok && prior.mapping.Replacement.Identity != m.Replacement.Identity {
				if descends(r.Ref, prior.proof.Ref) {
					edges[key] = edge{m, r}
				} else if !descends(prior.proof.Ref, r.Ref) {
					conflict = true
				}
				continue
			}
			edges[key] = edge{m, r}
		}
	}
	complete := base
	complete.PlanRefs = refs
	complete.ParentResultIDs = correctionSortedIDs(parentIDs)
	if len(old) == 1 {
		v := old[0].id
		complete.ParentResultID = &v
	}
	complete.AuditDeps = correctionUniqueDeps(audit)
	if conflict {
		complete.HandledCaseIDs = []string{}
		return complete, correction.ErrConflict
	}
	approved := []correction.Basis{}
	for _, r := range proofs {
		b := base
		b.PlanRefs = []correction.PlanRef{r.Ref}
		b.HandledCaseIDs = []string{r.CaseID}
		b.OriginalItems = append([]question.Instance{}, base.OriginalItems...)
		b.EffectiveItems = append([]question.Instance{}, base.OriginalItems...)
		b.AuditDeps = complete.AuditDeps
		b.EffectiveDeps = complete.AuditDeps
		for _, entry := range edges {
			if entry.proof.Ref != r.Ref {
				continue
			}
			for _, pos := range positions[entry.mapping.Original.Identity] {
				if pos < len(b.OriginalItems) {
					b.OriginalItems[pos] = entry.mapping.Original
					b.EffectiveItems[pos] = entry.mapping.Replacement
				}
			}
		}
		approved = append(approved, b)
	}
	composed, e := correction.ComposeBasis(base, nil, approved)
	if e != nil {
		return complete, e
	}
	composed.ParentResultIDs = complete.ParentResultIDs
	composed.ParentResultID = complete.ParentResultID
	composed.AuditDeps = complete.AuditDeps
	// Unit-bound assets remain effective, even when absent from question artwork.
	for _, i := range composed.EffectiveItems {
		deps, e := correctionInstanceDeps(ctx, tx, i)
		if e != nil {
			return complete, e
		}
		composed.EffectiveDeps = append(composed.EffectiveDeps, deps...)
	}
	composed.EffectiveDeps = correctionUniqueDeps(composed.EffectiveDeps)
	composed.AuditDeps = correctionUniqueDeps(append(composed.AuditDeps, composed.EffectiveDeps...))
	composed.HandledCaseIDs = correctionSortedIDs(handled)
	return composed, nil
}
func correctionWriteResult(ctx context.Context, tx *sql.Tx, j correctionJobRecord, meta correctionEvidenceMetadata, plan *correction.PlanRef, b correction.Basis, v correction.Evaluation) (string, error) {
	raw, h, e := correction.Canonical("correction-result-v1", b)
	if e != nil {
		return "", e
	}
	if len(raw) > correction.MaxResponseBytes {
		return "", question.ErrLimitExceeded
	}
	var pid any
	var pv any
	if plan != nil {
		pid = plan.ID
		pv = plan.Version
	}
	var kid, kv, kh any
	if meta.Knowledge != nil {
		kid = meta.Knowledge.ID
		kv = meta.Knowledge.Version
		kh = meta.Knowledge.SHA256
	}
	var id string
	// Rechecking an identical approved basis through a terminal task does not add
	// another result/grant/notice merely because the task source differs.
	e = tx.QueryRowContext(ctx, `SELECT id::text FROM correction_results WHERE owner_user_id=$1 AND case_id=$2 AND evidence_kind=$3 AND evidence_id=$4 AND plan_id IS NOT DISTINCT FROM $5::uuid AND plan_version IS NOT DISTINCT FROM $6::integer AND sealed AND status=$7 AND reason=$8 AND correctness=$9::jsonb AND basis#>'{body,effectiveItems}'=$10::jsonb AND handled_case_ids=$11::jsonb AND basis#>'{body,planRefs}'=$12::jsonb AND basis#>'{body,auditDeps}'=$13::jsonb ORDER BY id LIMIT 1`, meta.Owner, j.Meta.CaseID, meta.Ref.Kind, meta.Ref.ID, pid, pv, v.Status, v.Reason, body(v.Correct), body(b.EffectiveItems), body(b.HandledCaseIDs), body(b.PlanRefs), body(b.AuditDeps)).Scan(&id)
	if e == nil {
		return id, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return "", e
	}
	id, e = workflowID()
	if e != nil {
		return "", e
	}
	e = tx.QueryRowContext(ctx, `INSERT INTO correction_results(id,owner_user_id,case_id,plan_id,plan_version,parent_result_id,evidence_kind,evidence_id,knowledge_id,knowledge_version,knowledge_sha256,status,reason,score,passed,correctness,handled_case_ids,basis,basis_bytes,basis_digest,source_key) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21) ON CONFLICT(source_key,evidence_kind,evidence_id,basis_digest) DO NOTHING RETURNING id::text`, id, meta.Owner, j.Meta.CaseID, pid, pv, b.ParentResultID, meta.Ref.Kind, meta.Ref.ID, kid, kv, kh, v.Status, v.Reason, v.Score, v.Passed, body(v.Correct), body(b.HandledCaseIDs), string(raw), raw, h, j.Source).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		e = tx.QueryRowContext(ctx, `SELECT id::text FROM correction_results WHERE source_key=$1 AND evidence_kind=$2 AND evidence_id=$3 AND basis_digest=$4 AND sealed`, j.Source, meta.Ref.Kind, meta.Ref.ID, h).Scan(&id)
		return id, e
	}
	if e != nil {
		return "", e
	}
	for _, role := range []string{"audit", "effective"} {
		deps := b.AuditDeps
		if role == "effective" {
			deps = b.EffectiveDeps
		}
		positionMap, e := correctionPositionMap(ctx, tx, b, role)
		if e != nil {
			return "", e
		}
		for _, d := range deps {
			positions := positionMap[body(d)]
			if positions == nil {
				positions = []int{}
			}
			if _, e = tx.ExecContext(ctx, `INSERT INTO correction_dependencies(result_id,role,kind,id,version,sha256,positions) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, role, d.Kind, d.ID, d.Version, d.SHA256, positions); e != nil {
				return "", e
			}
		}
	}
	if e = correctionEvent(ctx, tx, "result", id, "result_sealed", "", j.Meta.CaseID, nil, 1, map[string]any{"digest": h}); e != nil {
		return "", e
	}
	_, e = tx.ExecContext(ctx, `UPDATE correction_results SET sealed=true WHERE id=$1`, id)
	return id, e
}

func correctionPositionMap(ctx context.Context, tx *sql.Tx, b correction.Basis, role string) (map[string][]int, error) {
	out := map[string][]int{}
	identities := map[question.Identity][]int{}
	add := func(d correction.Dependency, pos int) {
		key := body(d)
		for _, n := range out[key] {
			if n == pos {
				return
			}
		}
		out[key] = append(out[key], pos)
	}
	addIdentity := func(kind string, i question.Identity, positions []int) {
		v := i.Version
		d := correction.Dependency{Kind: kind, ID: i.ID, Version: &v, SHA256: i.SHA256}
		for _, pos := range positions {
			add(d, pos)
		}
	}
	groups := [][]question.Instance{b.EffectiveItems}
	if role == "audit" {
		groups = append(groups, b.OriginalItems)
	}
	for _, items := range groups {
		for pos, i := range items {
			identities[i.Identity] = append(identities[i.Identity], pos+1)
			addIdentity("instance", i.Identity, []int{pos + 1})
			if i.Template != nil {
				addIdentity("template", *i.Template, []int{pos + 1})
			}
			for _, d := range append(append([]correction.Dependency{}, b.AuditDeps...), b.EffectiveDeps...) {
				matched := d.Kind == "knowledge" && d.ID == b.OriginalSeal.Knowledge.ID && d.SHA256 == b.OriginalSeal.Knowledge.SHA256 || d.Kind == "blueprint" && b.OriginalSeal.Blueprint != nil && d.ID == b.OriginalSeal.Blueprint.ID && d.SHA256 == b.OriginalSeal.Blueprint.SHA256
				if d.Kind == "unit" {
					for _, u := range i.Body.Units {
						matched = matched || d.ID == u.ID && d.Version != nil && *d.Version == u.Version
					}
				}
				if d.Kind == "asset" {
					for _, a := range i.Body.Assets {
						matched = matched || d.SHA256 == a.SHA256
					}
					if pos < len(b.OriginalSeal.Items) {
						for _, a := range b.OriginalSeal.Items[pos].Assets {
							matched = matched || d.SHA256 == a.SHA256
						}
					}
				}
				if matched {
					add(d, pos+1)
				}
			}
		}
	}
	if role == "audit" && len(b.ParentResultIDs) > 0 {
		rows, e := tx.QueryContext(ctx, `SELECT kind,id,version,sha256,to_jsonb(positions) FROM correction_dependencies WHERE result_id=ANY($1::uuid[])`, b.ParentResultIDs)
		if e != nil {
			return out, e
		}
		for rows.Next() {
			var d correction.Dependency
			var positions []int
			var raw []byte
			if e = rows.Scan(&d.Kind, &d.ID, &d.Version, &d.SHA256, &raw); e != nil {
				rows.Close()
				return out, e
			}
			if json.Unmarshal(raw, &positions) != nil {
				rows.Close()
				return out, auth.ErrUnavailable
			}
			for _, pos := range positions {
				add(d, pos)
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
	}
	if role == "audit" && len(b.PlanRefs) > 0 {
		type link struct {
			o, r   question.Identity
			ot, rt *question.Identity
		}
		links := []link{}
		rows, e := tx.QueryContext(ctx, `SELECT m#>'{original,identity}',m#>'{replacement,identity}',m#>'{original,template}',m#>'{replacement,template}' FROM jsonb_to_recordset($1::jsonb) refs(id uuid,version integer) JOIN correction_plans p ON p.id=refs.id AND p.version=refs.version AND p.status='approved' CROSS JOIN LATERAL jsonb_array_elements(p.frozen_body#>'{body,proof,mappings}') m`, body(b.PlanRefs))
		if e != nil {
			return out, e
		}
		for rows.Next() {
			var l link
			var a, c, d, f []byte
			if e = rows.Scan(&a, &c, &d, &f); e != nil {
				rows.Close()
				return out, e
			}
			if json.Unmarshal(a, &l.o) != nil || json.Unmarshal(c, &l.r) != nil || json.Unmarshal(d, &l.ot) != nil || json.Unmarshal(f, &l.rt) != nil {
				rows.Close()
				return out, auth.ErrUnavailable
			}
			links = append(links, l)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
		for depth := 0; depth < 100; depth++ {
			changed := false
			for _, l := range links {
				positions, known := identities[l.o]
				if !known {
					continue
				}
				if _, ok := identities[l.r]; !ok {
					identities[l.r] = append([]int{}, positions...)
					changed = true
				}
				addIdentity("instance", l.r, positions)
				if l.ot != nil {
					addIdentity("template", *l.ot, positions)
				}
				if l.rt != nil {
					addIdentity("template", *l.rt, positions)
				}
			}
			if !changed {
				break
			}
		}
	}
	for key, positions := range out {
		sort.Ints(positions)
		out[key] = positions
	}
	return out, nil
}
