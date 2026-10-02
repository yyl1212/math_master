package e2etest

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHarnessQuestionSceneAndPrivateService(t *testing.T) {
	s, _, _, _ := startHarness(t)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", s.ControlURL+"/scene/question", nil)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	response, err := client.Do(req)
	if err != nil {
		t.Fatal("question scene unavailable")
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatalf("question scene status %d", response.StatusCode)
	}
	response, err = client.Get(s.APIURL + "/api/v1/question-bank/drafts")
	if err != nil {
		t.Fatal("question service unavailable")
	}
	response.Body.Close()
	if response.StatusCode != 401 || len(response.Cookies()) != 0 {
		t.Fatalf("private question boundary %d", response.StatusCode)
	}
	response, err = client.Get(s.APIURL + "/api/v1/knowledge/e2e-question-fractions")
	if err != nil {
		t.Fatal("knowledge unavailable")
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("question scene needs real published knowledge")
	}
}

func TestHarnessQuestionControlStateAndCLIStayPrivate(t *testing.T) {
	s, _, _, _ := startHarness(t)
	client := &http.Client{}
	scene := func(name string) {
		req, _ := http.NewRequest("POST", s.ControlURL+"/scene/"+name, nil)
		req.Header.Set("Authorization", "Bearer "+s.Token)
		response, e := client.Do(req)
		if e != nil {
			t.Fatal("question control failed")
		}
		response.Body.Close()
		if response.StatusCode != 204 {
			t.Fatalf("question control %s status %d", name, response.StatusCode)
		}
	}
	scene("question")
	scene("question-cli-import")
	for _, endpoint := range []struct {
		url    string
		status int
	}{{s.ControlURL + "/question/state", 401}, {s.APIURL + "/question/state", 404}} {
		response, e := client.Get(endpoint.url)
		if e != nil {
			t.Fatal("control privacy probe failed")
		}
		response.Body.Close()
		if response.StatusCode != endpoint.status {
			t.Fatal("control state exposed")
		}
	}
	req, _ := http.NewRequest("GET", s.ControlURL+"/question/state", nil)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	response, e := client.Do(req)
	if e != nil {
		t.Fatal("state request failed")
	}
	defer response.Body.Close()
	var state questionDatabaseState
	if response.StatusCode != 200 || json.NewDecoder(response.Body).Decode(&state) != nil {
		t.Fatal("real database state unavailable")
	}
	if state.Head != nil || state.Reviews != 0 || state.Published != 0 || state.Submissions != 0 {
		t.Fatal("CLI generated approval")
	}
	scene("question-hold-slots")
	scene("question-release-slots")
	scene("question-migration-missing")
	scene("question-migration-recover")
	scene("question-advance-knowledge")
	scene("question-revoke-editor")
}
