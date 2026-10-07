package study

// Only capability and schema/mode booleans are public; no user or source data.
type SchemaHealth struct {
	Taxonomy    bool `json:"taxonomy"`
	Study       bool `json:"study"`
	Retirement  bool `json:"retirement"`
	SchemaReady bool `json:"schemaReady"`
	TopicsMode  bool `json:"topicsMode"`
}
