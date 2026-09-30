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

type DomainSummary struct {
	Domain
	ContentStatus           string `json:"contentStatus"`
	PublishedKnowledgeCount int    `json:"publishedKnowledgeCount"`
}
type PathSummary struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Title   string `json:"title"`
	TitleZh string `json:"titleZh"`
}
type DomainDetail struct {
	DomainSummary
	Paths []PathSummary `json:"paths"`
}
