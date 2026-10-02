package testutil

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
)

// LearningCapacityFixture contains only original source inputs. Both the store
// capacity tests and private harness must publish these through real workflows.
// It contains no approved flags, generated answer bodies, scores, or user state.
type LearningCapacityFixture struct {
	Root       content.VersionRef
	Content    []publication.DraftInput
	LongRoutes publication.DraftInput
	Questions  []question.DraftInput
	Paths      []content.Path
}

func LearningCapacity(seedDir, catalogueFile, questionFile string) (LearningCapacityFixture, error) {
	capacity, e := CapacityContent(seedDir, catalogueFile)
	if e != nil {
		return LearningCapacityFixture{}, e
	}
	root := content.VersionRef{ID: "capacity-k-0000", Version: 1}
	out := LearningCapacityFixture{Root: root, Content: []publication.DraftInput{}, Questions: []question.DraftInput{}, Paths: []content.Path{}}
	pathIndex := 0
	for start := 1; start < 1000; start += 15 {
		end := start + 15
		if end > 1000 {
			end = 1000
		}
		indices := []int{0}
		for i := start; i < end; i++ {
			indices = append(indices, i)
		}
		in := publication.DraftInput{CatalogueVersion: 1, Package: content.Package{SchemaVersion: 1, ID: fmt.Sprintf("lc-content-%03d", start), Version: 1, Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{}, Assets: []content.Asset{}}, AssetBytes: []publication.AssetInput{}, SourceMap: []publication.SourceLink{}}
		nodes := []content.VersionRef{}
		for _, i := range indices {
			k := capacity.Snapshot.Knowledge[i]
			k.Statement = "Original technical rational addition capacity fixture."
			k.Relations = []content.Relation{}
			k.Objectives = []string{"Add rational numbers exactly."}
			if i > 0 {
				k.Relations = []content.Relation{{Kind: "prerequisite", Target: root}}
			} else {
				for n := 1; n < 8; n++ {
					k.Objectives = append(k.Objectives, fmt.Sprintf("Interpret original rational addition goal %d.", n))
				}
			}
			in.Package.Knowledge = append(in.Package.Knowledge, k)
			nodes = append(nodes, content.VersionRef{ID: k.ID, Version: 1})
			in.Package.Units = append(in.Package.Units, capacity.Snapshot.Units[i*4:i*4+4]...)
			a := capacity.Snapshot.Assets[i]
			in.Package.Assets = append(in.Package.Assets, a)
			raw, e := capacity.Reader(context.Background(), a)
			if e != nil {
				return LearningCapacityFixture{}, e
			}
			in.AssetBytes = append(in.AssetBytes, publication.AssetInput{ID: a.ID, Base64: base64.StdEncoding.EncodeToString(raw)})
		}
		count := 3
		if pathIndex+count > 200 {
			count = 200 - pathIndex
		}
		for range count {
			in.Package.Paths = append(in.Package.Paths, content.Path{ID: fmt.Sprintf("capacity-p-%03d", pathIndex), Version: 1, DomainIDs: []string{"elementary-mathematics"}, Title: "Original capacity route", TitleZh: "原创容量路线", Nodes: nodes})
			pathIndex++
		}
		out.Content = append(out.Content, in)
		out.Paths = append(out.Paths, in.Package.Paths...)
	}

	// The old authoring contract requires all path nodes and their units to be
	// in the same package. One root asset and 99 asset-free replacement units
	// keep this genuine 100-node package within every old submission limit.
	out.LongRoutes = publication.DraftInput{CatalogueVersion: 1,
		Package: content.Package{SchemaVersion: 1, ID: "capacity-long-routes", Version: 1,
			Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{},
			Assets: []content.Asset{capacity.Snapshot.Assets[0]}},
		AssetBytes: []publication.AssetInput{}, SourceMap: []publication.SourceLink{}}
	nodes := []content.VersionRef{}
	for i := 0; i < 100; i++ {
		// Reuse exactly the previously published immutable knowledge identity.
		k := out.Content[0].Package.Knowledge[0]
		if i > 0 {
			k = out.Content[(i-1)/15].Package.Knowledge[(i-1)%15+1]
		}
		out.LongRoutes.Package.Knowledge = append(out.LongRoutes.Package.Knowledge, k)
		nodes = append(nodes, content.VersionRef{ID: k.ID, Version: k.Version})
		u := capacity.Snapshot.Units[i*4]
		if i > 0 {
			u = capacity.Snapshot.Units[i*4+3]
			u.Version = 2
			u.AssetIDs = []string{}
			u.Angles = []content.Angle{{Kind: "formal", Body: "For rational numbers, use a common denominator to add."},
				{Kind: "intuitive", Body: "One half plus one half forms one whole."}}
		}
		out.LongRoutes.Package.Units = append(out.LongRoutes.Package.Units, u)
	}
	asset := capacity.Snapshot.Assets[0]
	raw, e := capacity.Reader(context.Background(), asset)
	if e != nil {
		return LearningCapacityFixture{}, e
	}
	out.LongRoutes.AssetBytes = append(out.LongRoutes.AssetBytes, publication.AssetInput{ID: asset.ID, Base64: base64.StdEncoding.EncodeToString(raw)})
	for i := 0; i < 20; i++ {
		p := content.Path{ID: fmt.Sprintf("capacity-p-%03d", i), Version: 2, DomainIDs: []string{"elementary-mathematics"},
			Title: fmt.Sprintf("Original maximum approved route %d", i), TitleZh: "原创最大审核路线", Nodes: nodes}
		out.LongRoutes.Package.Paths = append(out.LongRoutes.Package.Paths, p)
		out.Paths[i] = p
	}
	// Maximum legal page: twenty reviewed 100-node routes plus eighty existing
	// reviewed 16-node routes. All 200 paths remain in the publication.
	out.Paths = out.Paths[:100]
	raw, e = os.ReadFile(questionFile)
	if e != nil {
		return LearningCapacityFixture{}, e
	}
	seed, e := question.DecodePackage(bytes.NewReader(raw))
	if e != nil || len(seed.Templates) == 0 {
		return LearningCapacityFixture{}, fmt.Errorf("capacity question seed unavailable")
	}
	base := seed.Templates[0]
	base.Constraints = []question.Constraint{}
	base.Assets = []question.AssetRef{}
	rootSources := []question.BlueprintSource{}
	for n := 0; n < 20; n++ {
		rootSources = append(rootSources, question.BlueprintSource{Kind: "template", Ref: question.Ref{ID: fmt.Sprintf("lc-template-0-%d", n), Version: 1}})
	}
	// A blueprint may refer only to templates in its own frozen package.
	// Ten packages share the exact same twenty root templates, with one hundred
	// distinct blueprints each; nine template-only packages add the other 9000
	// instances. The final nineteen real approvals fit one existing release.
	for batch := 0; batch < 19; batch++ {
		templateBatch := 0
		if batch >= 10 {
			templateBatch = batch - 9
		}
		p := question.QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: fmt.Sprintf("learning-capacity-bank-%d", batch), Version: 1, Templates: []question.Template{}, FixedQuestions: []question.FixedQuestion{}, Blueprints: []question.Blueprint{}}
		for n := 0; n < 20; n++ {
			tpl := base
			tpl.ID = fmt.Sprintf("lc-template-%d-%d", templateBatch, n)
			tpl.Knowledge = question.Ref{ID: fmt.Sprintf("capacity-k-%04d", templateBatch), Version: 1}
			coverage := []int{0}
			if templateBatch == 0 && n < 4 {
				coverage = [][]int{{0, 1, 2}, {0, 3, 4}, {0, 5, 6}, {0, 7}}[n]
			}
			tpl.Coverage = []question.ObjectiveCoverage{{Knowledge: tpl.Knowledge, ObjectiveIndices: coverage}}
			tpl.Units = []question.Ref{{ID: fmt.Sprintf("capacity-u-%04d-0", templateBatch), Version: 1}}
			tpl.Parameters = []question.Parameter{{Name: "left", Values: []string{}}, {Name: "right", Values: []string{"1", "2"}}}
			for v := 1; v <= 25; v++ {
				tpl.Parameters[0].Values = append(tpl.Parameters[0].Values, fmt.Sprint(v))
			}
			if templateBatch == 0 {
				asset := capacity.Snapshot.Assets[0]
				tpl.Assets = []question.AssetRef{{ID: asset.ID, SHA256: asset.SHA256}}
			}
			p.Templates = append(p.Templates, tpl)
		}
		for n := 0; batch < 10 && n < 100; n++ {
			p.Blueprints = append(p.Blueprints, question.Blueprint{ID: fmt.Sprintf("lc-blueprint-%d-%d", batch, n), Version: 1, Knowledge: question.Ref{ID: root.ID, Version: 1}, CoreObjectiveIndices: []int{0, 1, 2, 3, 4, 5, 6, 7}, Sources: rootSources, CoverageNote: "Original sparse finite sources cover all eight declared capacity goals.", RuleVersion: 1, QuestionCount: 5, PassCount: 4})
		}
		out.Questions = append(out.Questions, question.DraftInput{CatalogueVersion: 1, QuestionPackage: p, SourceMap: []question.SourceLink{}})
	}
	return out, nil
}
