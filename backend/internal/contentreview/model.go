// Package contentreview prepares private review materials without publication or database access.
package contentreview

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/question"
)

var ErrInvalid = errors.New("invalid content review input")
var ErrLimit = errors.New("content review capacity exceeded")

const MaxManifestBytes = 64 << 10

type InputManifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	CataloguePath string   `json:"cataloguePath"`
	ContentPath   string   `json:"contentPath"`
	QuestionPaths []string `json:"questionPaths"`
	AssetsRoot    string   `json:"assetsRoot"`
}

func (m InputManifest) Selection() contentaudit.DraftSelection {
	return contentaudit.DraftSelection{CataloguePath: m.CataloguePath, ContentPath: m.ContentPath, QuestionPaths: m.QuestionPaths, AssetsRoot: m.AssetsRoot}
}

type PrepareInput struct {
	CodeSHA         string
	Route           content.VersionRef
	Manifest        InputManifest
	ManifestRaw     []byte
	Selected        contentaudit.SelectedDraft
	Sources         contentaudit.SourceBundle
	SourceReportRaw []byte
	SourceMapRaw    []byte
	FixtureOnly     bool
}
type DerivedInstance struct {
	Identity   question.Identity         `json:"identity"`
	Template   question.Identity         `json:"template"`
	Parameters []question.ParameterValue `json:"parameters"`
	SourceIDs  []string                  `json:"sourceIds"`
}
type SourceIdentity struct {
	Path       string `json:"path"`
	FileSHA256 string `json:"fileSHA256"`
	DatasetID  string `json:"datasetId"`
	RecordID   string `json:"recordId"`
}
type SourceRecord struct {
	SourceIdentity
	MappingIDs []string                      `json:"mappingIds"`
	Mappings   []contentaudit.MappedSource   `json:"mappings"`
	UsedBy     []contentaudit.ObjectIdentity `json:"usedBy"`
}
type ReviewScope struct {
	Objects          []contentaudit.ObjectIdentity
	MappedObjects    []contentaudit.ObjectIdentity
	DerivedInstances []DerivedInstance
	Sources          []SourceRecord
	RequiredChecks   map[string][]string
	facts            contentaudit.DraftFacts
}

func ObjectKey(o contentaudit.ObjectIdentity) string { return content.Digest(o) }
func SourceKey(s SourceIdentity) string              { return content.Digest(s) }
