package httpapi

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"testing"
	"time"
)

func TestNotificationHTTPIntegration(t *testing.T) {
	f := correctionRealHTTPFixture(t)
	var original assessment.PracticeView
	// Readiness and seal come from the actual published sources and original API.
	var readingJSON struct {
		State         struct{ Knowledge json.RawMessage }
		KnowledgeHead string
		QuestionHead  *string
	}
	contentHTTPCall(t, f.h, f.learner, "GET", "/api/v1/learning/knowledge/learning-http-root?version=1", nil, &readingJSON, 200)
	input := assessment.PracticeCreateInput{}
	if json.Unmarshal(readingJSON.State.Knowledge, &input.Knowledge) != nil {
		t.Fatal("identity")
	}
	input.ExpectedKnowledgeHead = readingJSON.KnowledgeHead
	input.ExpectedQuestionHead = *readingJSON.QuestionHead
	contentHTTPCall(t, f.h, f.learner, "POST", "/api/v1/learning/practice", input, &original, 201)
	cIn := correction.CaseInput{Kind: correction.GradingRuleCase, Rule: &correction.RuleScope{RuleVersion: 1, Kind: "all"}}
	var created correction.Envelope[correction.Receipt]
	contentHTTPCall(t, f.h, f.manager, "POST", "/api/v1/corrections/cases", cIn, &created, 201)
	lease, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || lease == nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ProcessCorrectionJob(f.ctx, *lease, 50); e != nil {
		t.Fatal(e)
	}
	var page notification.Envelope[notification.Page[notification.Metadata]]
	contentHTTPCall(t, f.h, f.learner, "GET", "/api/v1/notifications", nil, &page, 200)
	if len(page.Data.Items) != 1 || page.Data.Items[0].Type != notification.Checking {
		t.Fatal("static active notice")
	}
	id := page.Data.Items[0].ID
	var count notification.Envelope[notification.UnreadCount]
	contentHTTPCall(t, f.h, f.learner, "GET", "/api/v1/notifications/count", nil, &count, 200)
	if count.Data.Count != 1 {
		t.Fatal("unread")
	}
	contentHTTPCall(t, f.h, f.other, "GET", "/api/v1/notifications/"+id, nil, nil, 404)
	contentHTTPCall(t, f.h, f.other, "POST", "/api/v1/notifications/"+id+"/read", struct{}{}, nil, 404)
	contentHTTPCall(t, f.h, f.learner, "GET", "/api/v1/notifications/"+id, nil, nil, 200)
	var marked notification.Envelope[notification.ReadReceipt]
	contentHTTPCall(t, f.h, f.learner, "POST", "/api/v1/notifications/"+id+"/read", struct{}{}, &marked, 200)
	key := f.learner["Idempotency-Key"]
	first := marked.Data.ReadAt
	w := privateRequest(f.h, "POST", "/api/v1/notifications/"+id+"/read", "{}", f.learner)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &marked) != nil || !marked.Data.ReadAt.Equal(first) {
		t.Fatal("HTTP replay")
	}
	if f.learner["Idempotency-Key"] != key {
		t.Fatal("test key")
	}
	contentHTTPCall(t, f.h, f.learner, "GET", "/api/v1/notifications/count", nil, &count, 200)
	if count.Data.Count != 0 || time.Since(first) > time.Minute {
		t.Fatal("live unread time")
	}
}
