package e2etest

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"strings"
)

func studyReviewChange(ctx context.Context, s *store.Store, accounts *auth.Service, operation string) error {
	proof, e := fixtureAccess(ctx, accounts, "auth_admin", false)
	if e != nil {
		return e
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: proof.IdempotencyKey, RequestID: "study-review-control"}
	external := "demo-rational-fraction"
	if strings.HasSuffix(operation, "-proof") {
		external = "demo-equivalent-fractions"
		operation = strings.TrimSuffix(operation, "-proof")
	}
	id := knowledgeadmin.KnowledgeID(external)
	k, e := s.ReadManagedKnowledge(ctx, a, id)
	if e != nil {
		return e
	}
	switch operation {
	case "unpublish", "publish", "delete", "restore":
		_, e = s.SetManagedKnowledgeState(ctx, a, id, k.EditToken, operation)
		return e
	case "update", "boundary-raw", "boundary-wire", "boundary-oversize":
		in := knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources, TopicKeys: k.TopicKeys}
		if operation == "update" {
			in.Point.Statement += " 已更新本测试知识。"
		} else {
			in.Point.Title = strings.Repeat("😀", 512)
			in.Point.TitleZH = strings.Repeat("😀", 512)
			in.Point.Type = "mathematical-thinking"
			// Preserve every other required source field and its legal schema values.
			in.Point.Statement = "x"
			in.Point.Conditions = []string{}
			base, err := studyreview.BuildReference(in.Point, studyreview.Recall, studyreview.Statement)
			if err != nil {
				return err
			}
			other := -1
			for _, part := range base.Expected {
				for _, item := range part.Items {
					other += len(item)
				}
			}
			length := studyreview.MaxMaterialBytes - other
			setText := func(text string) {
				in.Point.Statement = text[:65536]
				in.Point.Conditions = []string{}
				for i := 65536; i < len(text); i += 65536 {
					end := i + 65536
					if end > len(text) {
						end = len(text)
					}
					in.Point.Conditions = append(in.Point.Conditions, string(rune('A'+len(in.Point.Conditions)))+text[i+1:end])
				}
			}
			setText(strings.Repeat("x", length))
			if operation != "boundary-raw" {
				m, err := studyreview.BuildReference(in.Point, studyreview.Recall, studyreview.Statement)
				if err != nil {
					return err
				}
				raw, _ := json.Marshal(studyreview.ExpectedReference{Sections: m.Expected, SHA256: m.SHA256})
				extra := studyreview.MaxMaterialWireBytes - len(raw) - 2
				escaped, quotes := extra/5, extra%5
				if operation == "boundary-oversize" {
					quotes++
				}
				setText(strings.Repeat("<", escaped) + strings.Repeat("\"", quotes) + strings.Repeat("x", length-escaped-quotes))
			}
		}
		_, e = s.UpdateManagedKnowledge(ctx, a, id, k.EditToken, in)
		return e
	default:
		return knowledgeadmin.ErrInvalid
	}
}
