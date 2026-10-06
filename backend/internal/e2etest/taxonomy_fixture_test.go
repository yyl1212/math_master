package e2etest

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"net/http"
	"testing"
)

func TestTopicCatalogueRealScene(t *testing.T) {
	s, _, _, _ := startHarness(t)
	req, _ := http.NewRequest("POST", s.ControlURL+"/scene/topic-catalogue", nil)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	response, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal("scene request failed")
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatal("topic scene missing", response.StatusCode)
	}
	response, e = http.Get(s.APIURL + "/api/v2/topics?limit=100")
	if e != nil {
		t.Fatal("public topics failed")
	}
	defer response.Body.Close()
	var page taxonomy.Page[taxonomy.TopicSummary]
	if e = json.NewDecoder(response.Body).Decode(&page); e != nil || response.StatusCode != 200 || page.Total != 63 || len(page.Items) != 63 || page.Pair.KnowledgeHead == nil || page.Pair.TaxonomyHead == nil {
		t.Fatal("scene is not a real published pair", e, response.StatusCode)
	}
}
