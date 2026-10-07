package study

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"time"
)

var ErrCutoverNotReady = errors.New("topic cutover not ready")
var ErrCutoverConflict = errors.New("topic cutover state changed")

type CutoverInput struct {
	ExpectedPair             taxonomy.PairRef `json:"expectedPair"`
	CodeSHA                  string           `json:"codeSHA"`
	ExpectedMigrationBatchID string           `json:"expectedMigrationBatchId"`
	Reason                   string           `json:"reason"`
	BackupRecord             string           `json:"backupRecord"`
}
type CutoverReport struct {
	Mode                 taxonomy.ExperienceMode `json:"mode"`
	Pair                 *taxonomy.PairRef       `json:"pair"`
	SchemaReady          bool                    `json:"schemaReady"`
	MigrationDone        bool                    `json:"migrationDone"`
	UnmappedLegacyEvents int                     `json:"unmappedLegacyEvents"`
	Conflicts            int                     `json:"conflicts"`
	HistoryReady         bool                    `json:"historyReady"`
	CodeCompatible       bool                    `json:"codeCompatible"`
	MigrationBatchID     *string                 `json:"migrationBatchId"`
	CutoverID            *string                 `json:"cutoverId"`
	RecordedAt           *time.Time              `json:"recordedAt"`
	Activated            bool                    `json:"activated"`
}
