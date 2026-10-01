package content

// AssetBinding fixes the bytes used by one immutable unit version.
type AssetBinding struct {
	Unit    VersionRef `json:"unit"`
	AssetID string     `json:"assetId"`
	SHA256  string     `json:"sha256"`
}

// Snapshot is a server-assembled set of exact content versions.
type Snapshot struct {
	CatalogueVersion int            `json:"catalogueVersion"`
	Knowledge        []Knowledge    `json:"knowledge"`
	Units            []Unit         `json:"units"`
	Paths            []Path         `json:"paths"`
	Assets           []Asset        `json:"assets"`
	Bindings         []AssetBinding `json:"bindings"`
}
