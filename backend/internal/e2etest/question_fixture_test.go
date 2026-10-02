package e2etest

import (
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
