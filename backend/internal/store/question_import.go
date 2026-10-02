package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func questionCanonical(purpose string, value any) ([]byte, string, error) {
	raw, err := json.Marshal(struct {
		Purpose string `json:"purpose"`
		Body    any    `json:"body"`
	}{purpose, value})
	if err != nil {
		return nil, "", err
	}
	return raw, fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}
func (s *Store) ImportQuestionDraft(ctx context.Context, archive question.Archive) (question.QuestionImportResult, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var result question.QuestionImportResult
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, questionError(err)
	}
	defer tx.Rollback()
	if err = questionConfigured(ctx, tx); err != nil {
		return result, questionError(err)
	}
	if _, err = tx.ExecContext(ctx, `SET LOCAL lock_timeout='1s'`); err != nil {
		return result, questionError(err)
	}
	// Offline tools take only the established content lock, never the admin lock after it.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1296127048)`); err != nil {
		return result, questionError(err)
	}
	refs, err := questionReferences(ctx, tx, archive.Envelope)
	if err != nil {
		return result, questionError(err)
	}
	if refs.KnowledgeHead == nil {
		return result, question.ErrNotReady
	}
	sealed, _, err := question.ValidateArchive(ctx, archive, refs)
	if err != nil {
		return result, questionError(err)
	}
	responsible, err := questionLocalResponsibility(ctx, tx, sealed.Package)
	if err != nil {
		return result, questionError(err)
	}
	// Claimed UUIDs in an external file are not verifiable platform authorship.
	var local bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM question_packages WHERE id=$1 AND version=$2 AND sha256=$3 AND sealed)`, sealed.Package.ID, sealed.Package.Version, sealed.PackageSHA).Scan(&local); err != nil {
		return result, questionError(err)
	}
	if !local {
		responsible.LegacyUnattributed = true
	}
	result, err = questionStoreSealedTx(ctx, tx, sealed, archive.Envelope.SourceMap, responsible)
	if err != nil {
		return result, questionError(err)
	}
	return result, questionError(tx.Commit())
}

// Only server-verified callers may reach this helper; all public inputs are regenerated first.
func questionStoreSealedTx(ctx context.Context, tx *sql.Tx, sealed question.SealedPackage, sources []question.SourceLink, responsible question.SourceResponsibility) (question.QuestionImportResult, error) {
	p := sealed.Package
	result := question.QuestionImportResult{PackageID: p.ID, PackageVersion: p.Version, PackageSHA: sealed.PackageSHA, Status: "draft"}
	raw, hash, err := question.CanonicalPackage(p)
	if err != nil || hash != sealed.PackageSHA {
		return result, question.ErrInvalid
	}
	var prior string
	err = tx.QueryRowContext(ctx, `SELECT sha256 FROM question_packages WHERE id=$1 AND version=$2`, p.ID, p.Version).Scan(&prior)
	if err == nil {
		if prior != hash {
			return result, question.ErrImmutableConflict
		}
		result.DuplicateInstances = len(sealed.Instances)
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	insertFixed := func(table, id string, version int, raw []byte, sha string, k question.Ref) (bool, error) {
		var previous string
		err := tx.QueryRowContext(ctx, `SELECT sha256 FROM `+table+` WHERE id=$1 AND version=$2`, id, version).Scan(&previous)
		if err == nil {
			if previous != sha {
				return false, question.ErrVersionConflict
			}
			return false, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO `+table+`(id,version,sha256,body,body_bytes,knowledge_id,knowledge_version) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, version, sha, string(raw), raw, k.ID, k.Version)
		return err == nil, err
	}
	for _, t := range p.Templates {
		traw, tsha, err := questionCanonical("question-template-v1", t)
		if err != nil {
			return result, err
		}
		if _, err = insertFixed("question_templates", t.ID, t.Version, traw, tsha, t.Knowledge); err != nil {
			return result, err
		}
	}
	identities := []question.Identity{}
	for _, i := range sealed.Instances {
		identities = append(identities, i.Identity)
		iraw, isha, err := question.CanonicalInstance(i)
		if err != nil || isha != i.Identity.SHA256 {
			return result, question.ErrInvalid
		}
		var previous string
		err = tx.QueryRowContext(ctx, `SELECT sha256 FROM question_instances WHERE id=$1 AND version=$2`, i.Identity.ID, i.Identity.Version).Scan(&previous)
		if err == nil {
			if previous != isha {
				return result, question.ErrVersionConflict
			}
			result.DuplicateInstances++
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return result, err
		}
		var templateID, templateVersion, templateSHA, gv, vv any
		if i.Template != nil {
			templateID = i.Template.ID
			templateVersion = i.Template.Version
			templateSHA = i.Template.SHA256
			gv = *i.GeneratorVersion
			vv = *i.VerifierVersion
		}
		params, _ := json.Marshal(i.Parameters)
		paramSHA := fmt.Sprintf("%x", sha256.Sum256(params))
		if _, err = tx.ExecContext(ctx, `INSERT INTO question_instances(id,version,sha256,body,body_bytes,origin,knowledge_id,knowledge_version,template_id,template_version,template_sha256,generator_version,verifier_version,parameters,parameter_bytes,parameter_sha256) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, i.Identity.ID, i.Identity.Version, isha, string(iraw), iraw, i.Origin, i.Body.Knowledge.ID, i.Body.Knowledge.Version, templateID, templateVersion, templateSHA, gv, vv, string(params), params, paramSHA); err != nil {
			return result, err
		}
		for _, c := range i.Body.Coverage {
			for _, index := range c.ObjectiveIndices {
				if _, err = tx.ExecContext(ctx, `INSERT INTO question_instance_coverage VALUES($1,$2,$3,$4,$5)`, i.Identity.ID, i.Identity.Version, c.Knowledge.ID, c.Knowledge.Version, index); err != nil {
					return result, err
				}
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE question_instances SET sealed=true WHERE id=$1 AND version=$2`, i.Identity.ID, i.Identity.Version); err != nil {
			return result, err
		}
		result.ImportedInstances++
	}
	for _, b := range p.Blueprints {
		braw, bsha, err := questionCanonical("question-blueprint-v1", b)
		if err != nil {
			return result, err
		}
		fresh, err := insertFixed("question_blueprints", b.ID, b.Version, braw, bsha, b.Knowledge)
		if err != nil {
			return result, err
		}
		if !fresh {
			continue
		}
		for _, source := range b.Sources {
			var tid, iid any
			if source.Kind == "template" {
				tid = source.Ref.ID
			} else {
				iid = source.Ref.ID
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO question_blueprint_sources(blueprint_id,blueprint_version,kind,id,version,template_id,instance_id) VALUES($1,$2,$3,$4,$5,$6,$7)`, b.ID, b.Version, source.Kind, source.Ref.ID, source.Ref.Version, tid, iid); err != nil {
				return result, err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE question_blueprints SET sealed=true WHERE id=$1 AND version=$2`, b.ID, b.Version); err != nil {
			return result, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO question_packages(id,version,sha256,body,body_bytes,catalogue_version,catalogue_sha256,instance_identities,source_map,author_ids,legacy_unattributed) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, p.ID, p.Version, hash, string(raw), raw, sealed.ReferenceSnapshot.CatalogueVersion, sealed.ReferenceSnapshot.CatalogueSHA256, body(identities), body(sources), body(responsible.AuthorIDs), responsible.LegacyUnattributed); err != nil {
		return result, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE question_packages SET sealed=true WHERE id=$1 AND version=$2`, p.ID, p.Version)
	return result, err
}
