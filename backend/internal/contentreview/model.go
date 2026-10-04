// Package contentreview prepares private review materials without publication or database access.
package contentreview

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
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

type Imports struct {
	Knowledge publication.DraftInput
	Questions []question.DraftInput
}

type ExportFile struct {
	Path  string
	Bytes []byte
}
type FileEntry struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}
type RouteIdentity struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	SHA256  string `json:"sha256"`
}
type SubjectChecks struct {
	Key    string   `json:"key"`
	Checks []string `json:"checks"`
}
type ReviewManifest struct {
	SchemaVersion       int                           `json:"schemaVersion"`
	CodeSHA             string                        `json:"codeSHA"`
	FixtureOnly         bool                          `json:"fixtureOnly"`
	CatalogueVersion    int                           `json:"catalogueVersion"`
	CatalogueSHA256     string                        `json:"catalogueSHA256"`
	Route               RouteIdentity                 `json:"route"`
	SnapshotID          string                        `json:"snapshotId"`
	SourceReportSHA256  string                        `json:"sourceReportSHA256"`
	SourceMapSHA256     string                        `json:"sourceMapSHA256"`
	InputManifestSHA256 string                        `json:"inputManifestSHA256"`
	Inputs              []FileEntry                   `json:"inputs"`
	Objects             []contentaudit.ObjectIdentity `json:"objects"`
	DerivedInstances    []DerivedInstance             `json:"derivedInstances"`
	Sources             []SourceRecord                `json:"sources"`
	RequiredChecks      []SubjectChecks               `json:"requiredChecks"`
	Files               []FileEntry                   `json:"files"`
}
type Bundle struct {
	Manifest ReviewManifest
	Files    []ExportFile
}
type FileRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type ReviewCheck struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Basis       string `json:"basis"`
	Issue       string `json:"issue"`
	ReviewerRef string `json:"reviewerRef"`
}
type ReviewRow struct {
	Key    string                       `json:"key"`
	Object *contentaudit.ObjectIdentity `json:"object"`
	Source *SourceIdentity              `json:"source"`
	Checks []ReviewCheck                `json:"checks"`
}
type ReviewRegister struct {
	SchemaVersion  int       `json:"schemaVersion"`
	ManifestSHA256 string    `json:"manifestSHA256"`
	Parts          []FileRef `json:"parts"`
}
type RegisterPart struct {
	SchemaVersion  int         `json:"schemaVersion"`
	ManifestSHA256 string      `json:"manifestSHA256"`
	Rows           []ReviewRow `json:"rows"`
}

type Verification struct {
	ReviewComplete bool                             `json:"reviewComplete"`
	SchemaVersion  int                              `json:"schemaVersion"`
	Conclusion     string                           `json:"conclusion"`
	FixtureOnly    bool                             `json:"fixtureOnly"`
	ManifestSHA256 string                           `json:"manifestSHA256"`
	Reasons        []string                         `json:"reasons"`
	Evidence       *contentaudit.AcceptanceEvidence `json:"-"`
	Files          []ExportFile                     `json:"-"`
}

type ReleaseIdentity struct {
	CodeSHA          string            `json:"codeSHA"`
	CatalogueVersion int               `json:"catalogueVersion"`
	CatalogueSHA256  string            `json:"catalogueSHA256"`
	Route            RouteIdentity     `json:"route"`
	KnowledgeHead    contentaudit.Head `json:"knowledgeHead"`
	QuestionHead     contentaudit.Head `json:"questionHead"`
	ManifestSHA256   string            `json:"manifestSHA256"`
	FixtureOnly      bool              `json:"fixtureOnly"`
}
type ReleaseContext struct {
	SchemaVersion int `json:"schemaVersion"`
	ReleaseIdentity
	BuildRecord FileRef `json:"buildRecord"`
	AttestedBy  string  `json:"attestedBy"`
	Attestation FileRef `json:"attestation"`
}
type FrozenBinding struct {
	Space                string   `json:"space"`
	Submission           FileRef  `json:"submission"`
	Archive              *FileRef `json:"archive"`
	IndependenceVerified bool     `json:"independenceVerified"`
	Attestation          FileRef  `json:"attestation"`
}
type LearningFile struct {
	Name string  `json:"name"`
	File FileRef `json:"file"`
}
type EvidenceInput struct {
	SchemaVersion  int             `json:"schemaVersion"`
	ReleaseContext FileRef         `json:"releaseContext"`
	ReviewRegister FileRef         `json:"reviewRegister"`
	Bindings       []FrozenBinding `json:"bindings"`
	LearningChecks []LearningFile  `json:"learningChecks"`
}
type LearningStep struct {
	Action   string `json:"action"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}
type LearningRecord struct {
	SchemaVersion int             `json:"schemaVersion"`
	Name          string          `json:"name"`
	Result        string          `json:"result"`
	Context       ReleaseIdentity `json:"context"`
	ExecutedAt    string          `json:"executedAt"`
	Steps         []LearningStep  `json:"steps"`
	Attachments   []FileRef       `json:"attachments"`
	AttestedBy    string          `json:"attestedBy"`
	Attestation   FileRef         `json:"attestation"`
}
