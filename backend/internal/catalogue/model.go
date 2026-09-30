package catalogue

type Catalogue struct {
	SchemaVersion int      `json:"schemaVersion"`
	Version       int      `json:"version"`
	Domains       []Domain `json:"domains"`
}
type Domain struct {
	ID               string   `json:"id"`
	Order            int      `json:"order"`
	Name             string   `json:"name"`
	NameZh           string   `json:"nameZh"`
	Topics           []Topic  `json:"topics"`
	RelatedDomainIDs []string `json:"relatedDomainIds"`
}
type Topic struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	NameZh string `json:"nameZh"`
}
