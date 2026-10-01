package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
)

type CapacityFixture struct {
	Catalogue catalogue.Catalogue
	Snapshot  content.Snapshot
	Inputs    []publication.DraftInput
	Reader    content.AssetReader
	SVGBytes  int
}

// CapacityContent constructs original technical data, never mathematical approval.
// Every individual batch stays within the existing submission limits while the
// combined snapshot reaches all count and byte maxima simultaneously.
func CapacityContent(seedDir, catalogueFile string) (CapacityFixture, error) {
	fail := func() (CapacityFixture, error) { return CapacityFixture{}, errors.New("capacity fixture unavailable") }
	file, err := os.Open(catalogueFile)
	if err != nil {
		return fail()
	}
	c, err := content.DecodeCatalogue(file)
	file.Close()
	if err != nil {
		return fail()
	}
	raw, err := os.ReadFile(filepath.Join(seedDir, "workflow-ready.json"))
	if err != nil {
		return fail()
	}
	var seed content.Package
	if json.Unmarshal(raw, &seed) != nil {
		return fail()
	}
	svg, err := os.ReadFile(filepath.Join(seedDir, "workflow-ready.svg"))
	if err != nil {
		return fail()
	}
	out := CapacityFixture{Catalogue: c, Snapshot: content.Snapshot{CatalogueVersion: c.Version, Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{}, Assets: []content.Asset{}, Bindings: []content.AssetBinding{}}, Inputs: []publication.DraftInput{}}
	assets := map[string][]byte{}
	for i := 0; i < 1000; i++ {
		k := seed.Knowledge[0]
		k.ID = fmt.Sprintf("capacity-k-%04d", i)
		k.Title = fmt.Sprintf("Original capacity fixture %04d", i)
		k.Relations = []content.Relation{}
		group := i / 16
		count := 16
		if group == 62 {
			count = 8
		}
		local := i % 16
		for j := 1; j < count; j++ {
			k.Relations = append(k.Relations, content.Relation{Kind: "related", Target: content.VersionRef{ID: fmt.Sprintf("capacity-k-%04d", group*16+(local+j)%count), Version: 1}})
		}
		extra := 1
		if group == 0 {
			extra = 2
		}
		if group == 62 {
			extra = 7
		}
		for j := 1; j <= extra; j++ {
			k.Relations = append(k.Relations, content.Relation{Kind: "derivation", Target: content.VersionRef{ID: fmt.Sprintf("capacity-k-%04d", group*16+(local+j)%count), Version: 1}})
		}
		out.Snapshot.Knowledge = append(out.Snapshot.Knowledge, k)
		a := seed.Assets[0]
		a.ID = fmt.Sprintf("capacity-a-%04d", i)
		a.Path = a.ID + ".svg"
		a.Knowledge = content.VersionRef{ID: k.ID, Version: 1}
		size := (10 << 20) / 1000
		if i < (10<<20)%1000 {
			size++
		}
		prefix := strings.TrimSuffix(strings.TrimSpace(string(svg)), "</svg>") + fmt.Sprintf("<desc>Original capacity bytes %04d</desc><!--", i)
		suffix := "--></svg>"
		data := []byte(prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix)
		a.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		assets[a.SHA256] = data
		out.SVGBytes += len(data)
		out.Snapshot.Assets = append(out.Snapshot.Assets, a)
		for j := 0; j < 4; j++ {
			u := seed.Units[0]
			u.ID = fmt.Sprintf("capacity-u-%04d-%d", i, j)
			u.Knowledge = a.Knowledge
			u.AssetIDs = []string{a.ID}
			u.Angles = append([]content.Angle{}, u.Angles...)
			for n := range u.Angles {
				u.Angles[n].Body = strings.ReplaceAll(u.Angles[n].Body, "asset:halves", "asset:"+a.ID)
			}
			out.Snapshot.Units = append(out.Snapshot.Units, u)
			out.Snapshot.Bindings = append(out.Snapshot.Bindings, content.AssetBinding{Unit: content.VersionRef{ID: u.ID, Version: 1}, AssetID: a.ID, SHA256: a.SHA256})
		}
	}
	pathIndex := 0
	for group := 0; group < 63; group++ {
		count := 3
		if group < 11 {
			count = 4
		}
		nodes := []content.VersionRef{}
		for i := group * 16; i < (group+1)*16 && i < 1000; i++ {
			nodes = append(nodes, content.VersionRef{ID: out.Snapshot.Knowledge[i].ID, Version: 1})
		}
		for range count {
			p := content.Path{ID: fmt.Sprintf("capacity-p-%03d", pathIndex), Version: 1, DomainIDs: seed.Knowledge[0].DomainIDs, Title: "Original capacity fixture route", TitleZh: "原创容量夹具路线", Nodes: nodes}
			out.Snapshot.Paths = append(out.Snapshot.Paths, p)
			pathIndex++
		}
	}
	raw, err = json.Marshal(out.Snapshot)
	if err != nil || len(raw) >= 32<<20 {
		return fail()
	}
	remaining := (32 << 20) - len(raw)
	for i := range out.Snapshot.Knowledge {
		n := remaining / 1000
		if i < remaining%1000 {
			n++
		}
		out.Snapshot.Knowledge[i].Statement += strings.Repeat("x", n)
	}
	pathIndex = 0
	for group := 0; group < 63; group++ {
		start := group * 16
		end := start + 16
		if end > 1000 {
			end = 1000
		}
		pathCount := 3
		if group < 11 {
			pathCount = 4
		}
		p := content.Package{SchemaVersion: 1, ID: fmt.Sprintf("capacity-batch-%02d", group), Version: 1, Knowledge: out.Snapshot.Knowledge[start:end], Units: out.Snapshot.Units[start*4 : end*4], Paths: out.Snapshot.Paths[pathIndex : pathIndex+pathCount], Assets: out.Snapshot.Assets[start:end]}
		pathIndex += pathCount
		input := publication.DraftInput{CatalogueVersion: c.Version, Package: p, AssetBytes: []publication.AssetInput{}, SourceMap: []publication.SourceLink{}}
		for _, a := range p.Assets {
			input.AssetBytes = append(input.AssetBytes, publication.AssetInput{ID: a.ID, Base64: base64.StdEncoding.EncodeToString(assets[a.SHA256])})
		}
		if _, err := publication.DraftAssets(input); err != nil {
			return fail()
		}
		out.Inputs = append(out.Inputs, input)
	}
	out.Reader = func(ctx context.Context, a content.Asset) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, ok := assets[a.SHA256]
		if !ok {
			return nil, errors.New("unknown capacity asset")
		}
		return data, nil
	}
	return out, nil
}
