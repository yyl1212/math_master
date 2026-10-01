package content

type VersionRef struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}
type Relation struct {
	Kind   string     `json:"kind"`
	Target VersionRef `json:"target"`
}
type Source struct {
	Kind        string `json:"kind"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	AccessedAt  string `json:"accessedAt"`
	License     string `json:"license"`
	Attribution string `json:"attribution"`
}
type Knowledge struct {
	ID         string     `json:"id"`
	Version    int        `json:"version"`
	DomainIDs  []string   `json:"domainIds"`
	TopicIDs   []string   `json:"topicIds"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	TitleZh    string     `json:"titleZh"`
	Statement  string     `json:"statement"`
	Scope      string     `json:"scope"`
	Objectives []string   `json:"objectives"`
	Conditions []string   `json:"conditions"`
	System     string     `json:"system"`
	Proof      string     `json:"proof"`
	Sources    []Source   `json:"sources"`
	Relations  []Relation `json:"relations"`
}
type Angle struct {
	Kind string `json:"kind"`
	Body string `json:"body"`
}
type Unit struct {
	ID              string     `json:"id"`
	Version         int        `json:"version"`
	Knowledge       VersionRef `json:"knowledge"`
	Angles          []Angle    `json:"angles"`
	Examples        []string   `json:"examples"`
	Counterexamples []string   `json:"counterexamples"`
	AssetIDs        []string   `json:"assetIds"`
}
type Path struct {
	ID        string       `json:"id"`
	Version   int          `json:"version"`
	DomainIDs []string     `json:"domainIds"`
	Title     string       `json:"title"`
	TitleZh   string       `json:"titleZh"`
	Nodes     []VersionRef `json:"nodes"`
}
type Asset struct {
	ID          string     `json:"id"`
	Path        string     `json:"path"`
	SHA256      string     `json:"sha256"`
	Author      string     `json:"author"`
	License     string     `json:"license"`
	Attribution string     `json:"attribution"`
	Knowledge   VersionRef `json:"knowledge"`
}
type Package struct {
	SchemaVersion int         `json:"schemaVersion"`
	ID            string      `json:"id"`
	Version       int         `json:"version"`
	Knowledge     []Knowledge `json:"knowledge"`
	Units         []Unit      `json:"units"`
	Paths         []Path      `json:"paths"`
	Assets        []Asset     `json:"assets"`
}

type AssetView struct {
	ID          string     `json:"id"`
	SHA256      string     `json:"sha256"`
	Author      string     `json:"author"`
	License     string     `json:"license"`
	Attribution string     `json:"attribution"`
	Knowledge   VersionRef `json:"knowledge"`
}
type KnowledgeView struct {
	Knowledge Knowledge   `json:"knowledge"`
	Units     []Unit      `json:"units"`
	Assets    []AssetView `json:"assets"`
}
type PathView struct {
	Path      Path            `json:"path"`
	Knowledge []KnowledgeView `json:"knowledge"`
}
