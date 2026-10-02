package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func questionExportTx(ctx context.Context, tx *sql.Tx, id string, version int) (question.Archive, error) {
	var archive question.Archive
	var raw, sourceMap, authors, identities []byte
	var size int
	err := tx.QueryRowContext(ctx, `SELECT catalogue_version,sha256,body,source_map,author_ids,legacy_unattributed,instance_identities,octet_length(body_bytes) FROM question_packages WHERE id=$1 AND version=$2 AND sealed`, id, version).Scan(&archive.Envelope.CatalogueVersion, &archive.PackageSHA, &raw, &sourceMap, &authors, &archive.SourceResponsibility.LegacyUnattributed, &identities, &size)
	if err != nil {
		return archive, workflowRowError(err)
	}
	if size > question.MaxCanonicalPackageBytes {
		return archive, question.ErrLimitExceeded
	}
	var wrapped struct {
		Purpose string                   `json:"purpose"`
		Body    question.QuestionPackage `json:"body"`
	}
	if json.Unmarshal(raw, &wrapped) != nil || json.Unmarshal(sourceMap, &archive.Envelope.SourceMap) != nil || json.Unmarshal(authors, &archive.SourceResponsibility.AuthorIDs) != nil {
		return archive, auth.ErrUnavailable
	}
	archive.Envelope.QuestionPackage = wrapped.Body
	var expected []question.Identity
	if json.Unmarshal(identities, &expected) != nil {
		return archive, auth.ErrUnavailable
	}
	archive.Instances = []question.Instance{}
	rows, err := tx.QueryContext(ctx, `SELECT i.body,i.sha256 FROM jsonb_to_recordset($1::jsonb) r(id text,version integer,sha256 text) JOIN question_instances i ON i.id=r.id AND i.version=r.version AND i.sha256=r.sha256 AND i.sealed ORDER BY i.id,i.version`, string(identities))
	if err != nil {
		return archive, err
	}
	total := size + len(sourceMap) + len(authors)
	for rows.Next() {
		var b []byte
		var sha string
		if err = rows.Scan(&b, &sha); err != nil {
			rows.Close()
			return archive, err
		}
		total += len(b)
		if total > question.MaxEnvelopeBytes {
			rows.Close()
			return archive, question.ErrLimitExceeded
		}
		var wrapped struct {
			Purpose string            `json:"purpose"`
			Body    question.Instance `json:"body"`
		}
		if json.Unmarshal(b, &wrapped) != nil {
			rows.Close()
			return archive, auth.ErrUnavailable
		}
		wrapped.Body.Identity.SHA256 = sha
		archive.Instances = append(archive.Instances, wrapped.Body)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return archive, err
	}
	if len(archive.Instances) != len(expected) {
		return archive, auth.ErrUnavailable
	}
	archive.GeneratorVersions, archive.VerifierVersions = question.UsedEngineVersions(archive.Envelope.QuestionPackage, archive.Instances)
	responsibility, err := questionLocalResponsibility(ctx, tx, archive.Envelope.QuestionPackage)
	if err != nil {
		return archive, err
	}
	archive.SourceResponsibility = responsibility
	bytes, err := json.Marshal(archive)
	if err != nil {
		return archive, err
	}
	if len(bytes) > question.MaxEnvelopeBytes {
		return archive, question.ErrLimitExceeded
	}
	_, sha, err := question.CanonicalPackage(archive.Envelope.QuestionPackage)
	if err != nil || sha != archive.PackageSHA {
		return archive, auth.ErrUnavailable
	}
	return archive, nil
}
func (s *Store) ExportQuestionArchive(ctx context.Context, id string, version int) (question.Archive, error) {
	var archive question.Archive
	if !question.ValidMathID(id) || version < 1 || version > 2147483647 {
		return archive, auth.ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return archive, questionError(err)
	}
	defer tx.Rollback()
	if err = questionConfigured(ctx, tx); err != nil {
		return archive, questionError(err)
	}
	archive, err = questionExportTx(ctx, tx, id, version)
	if err != nil {
		return archive, questionError(err)
	}
	return archive, questionError(tx.Commit())
}
