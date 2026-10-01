package store

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
)

func TestWorkflowPublicationEnvelopeReservesPublishedStatus(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	identity := publication.MemberIdentity{Kind: "knowledge", ID: "root", Version: 1, PackageID: "original", PackageVersion: 1, SHA256: strings.Repeat("a", 64)}
	v := publication.PublicationView{ID: id, Status: "draft", ManifestSHA: strings.Repeat("b", 64), CreatedAt: "2026-10-01T00:00:00Z", Manifest: publication.Manifest{CatalogueVersion: 1, CatalogueSHA256: strings.Repeat("c", 64), Members: []publication.ManifestMember{{Identity: identity, Evidence: publication.MemberEvidence{SubmissionID: id, DecisionID: id, FrozenDigest: strings.Repeat("d", 64)}}}, Bindings: []content.AssetBinding{}}, Diff: publication.Diff{Added: 1, Changes: []publication.Change{{Kind: "knowledge", ID: "root", After: &identity, Reason: ""}}}}
	page := publication.PublicationPage{Items: []publication.PublicationView{v}, Head: &id, Total: 2147483647, Limit: 100, Offset: 100000}
	raw, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	v.Diff.Changes[0].Reason = strings.Repeat("x", (4<<20)-len(raw))
	if err := workflowPublicationResponseSize(v); !errors.Is(err, publication.ErrContentLimitExceeded) {
		t.Fatal("draft page left no bytes for its future published status", err)
	}
	v.Diff.Changes[0].Reason = v.Diff.Changes[0].Reason[4:]
	if err := workflowPublicationResponseSize(v); err != nil {
		t.Fatal("exact future page rejected", err)
	}
	v.Status = "published"
	if err := workflowPublicationResponseSize(v); err != nil {
		t.Fatal("accepted publication became unreadable on activation", err)
	}
	v.Diff.Changes[0].Reason += "x"
	if err := workflowPublicationResponseSize(v); !errors.Is(err, publication.ErrContentLimitExceeded) {
		t.Fatal("published page +1 accepted", err)
	}
}
