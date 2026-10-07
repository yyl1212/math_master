package study

import "time"

type LegacyCursor struct {
	RecordedAt time.Time `json:"recordedAt"`
	EventID    string    `json:"eventId"`
}
type MigrationReport struct {
	BatchID       string        `json:"batchId"`
	Processed     int           `json:"processed"`
	CreatedEvents int           `json:"createdEvents"`
	LinkedEvents  int           `json:"linkedEvents"`
	Conflicts     int           `json:"conflicts"`
	Cursor        *LegacyCursor `json:"cursor"`
	Done          bool          `json:"done"`
}
type MigrationInspection struct {
	SchemaReady          bool    `json:"schemaReady"`
	UnmappedLegacyEvents int     `json:"unmappedLegacyEvents"`
	InvalidLinks         int     `json:"invalidLinks"`
	MigrationBatchID     *string `json:"migrationBatchId"`
	MigrationDone        bool    `json:"migrationDone"`
	Conflicts            int     `json:"conflicts"`
}
