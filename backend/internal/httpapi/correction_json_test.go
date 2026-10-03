package httpapi

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestCorrectionJSONSharedBoundaries(t *testing.T) {
	raw, e := os.ReadFile("../../../api/correction-boundary-cases.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Version  int `json:"version"`
		Requests []struct {
			Name, Domain, Action, Raw string
			Valid                     bool
		} `json:"requests"`
	}
	if json.Unmarshal(raw, &f) != nil || f.Version != 1 || len(f.Requests) < 30 {
		t.Fatal("shared fixture")
	}
	for _, c := range f.Requests {
		t.Run(c.Name, func(t *testing.T) {
			var e error
			if c.Domain == "notification" {
				_, e = DecodeNotificationInput([]byte(c.Raw))
			} else {
				_, e = DecodeCorrectionInput([]byte(c.Raw), correction.Action(c.Action))
			}
			if (e == nil) != c.Valid {
				t.Fatal("boundary mismatch", e)
			}
		})
	}
}

func TestCorrectionHTTPNamedContracts(t *testing.T) {
	raw, e := os.ReadFile("../../../api/openapi.yaml")
	if e != nil {
		t.Fatal(e)
	}
	var api struct {
		Paths      map[string]map[string]json.RawMessage
		Components struct{ Schemas map[string]json.RawMessage }
	}
	if json.Unmarshal(raw, &api) != nil {
		t.Fatal("OpenAPI")
	}
	paths, operations := 0, 0
	for p, ops := range api.Paths {
		if strings.HasPrefix(p, "/api/v1/corrections/") || strings.HasPrefix(p, "/api/v1/notifications") {
			paths++
			operations += len(ops)
		}
	}
	if paths != 16 || operations != 19 {
		t.Fatal("fixed routes", paths, operations)
	}
	dtos := map[string]any{"CorrectionCaseInput": correction.CaseInput{}, "CorrectionCaseMetadata": correction.CaseMetadata{}, "CorrectionWithdrawalRef": correction.WithdrawalRef{}, "CorrectionPlanInput": correction.PlanInput{}, "CorrectionPlanMetadata": correction.PlanMetadata{}, "CorrectionPlanMetadataView": correction.PlanDetail{}, "CorrectionPlanDetail": correction.PlanDetail{}, "CorrectionMapping": correction.Mapping{}, "CorrectionPublishedInstance": correction.PublishedInstance{}, "CorrectionRuleScope": correction.RuleScope{}, "CorrectionEvidenceRef": correction.EvidenceRef{}, "CorrectionPlanRef": correction.PlanRef{}, "CorrectionResultMetadata": correction.ResultMetadata{}, "CorrectionResultMetadataView": correction.ResultDetail{}, "CorrectionResultDetail": correction.ResultDetail{}, "CorrectionCorrectedItem": correction.CorrectedItem{}, "CorrectionSubmitInput": correction.SubmitInput{}, "CorrectionRetryInput": correction.RetryInput{}, "CorrectionDecisionInput": correction.DecisionInput{}, "CorrectionJobMetadata": correction.JobMetadata{}, "CorrectionReceipt": correction.Receipt{}, "NotificationMetadata": notification.Metadata{}, "NotificationUnreadCount": notification.UnreadCount{}, "NotificationReadReceipt": notification.ReadReceipt{}}
	var assertShape func(string, json.RawMessage, map[string]bool)
	assertShape = func(name string, raw json.RawMessage, fields map[string]bool) {
		var schema struct {
			Type                 string
			AdditionalProperties any
			Required             []string
			Properties           map[string]json.RawMessage
			OneOf                []json.RawMessage
		}
		if json.Unmarshal(raw, &schema) != nil {
			t.Fatal(name)
		}
		if len(schema.OneOf) > 0 {
			for _, branch := range schema.OneOf {
				assertShape(name, branch, fields)
			}
			return
		}
		if schema.Type != "object" || schema.AdditionalProperties != false || len(schema.Properties) != len(fields) || len(schema.Required) != len(fields) {
			t.Fatal("named closed DTO", name)
		}
		for _, k := range schema.Required {
			if !fields[k] || schema.Properties[k] == nil {
				t.Fatal("unmapped named property", name, k)
			}
		}
	}
	for name, v := range dtos {
		ty := reflect.TypeOf(v)
		fields := map[string]bool{}
		for n := 0; n < ty.NumField(); n++ {
			k := ty.Field(n).Tag.Get("json")
			if k == "" || strings.Contains(k, ",") {
				t.Fatal("nullable key must be explicit", name, k)
			}
			fields[k] = true
		}
		assertShape(name, api.Components.Schemas[name], fields)
	}
}
func TestCorrectionJSONInvalidUTF8(t *testing.T) {
	for _, raw := range [][]byte{{'{', '"', 'x', '"', ':', '"', 255, '"', '}'}, []byte(`{"kind":"grading_rule","withdrawal":null,"rule":{"ruleVersion":1,"kind":"all"}}`), []byte(`{"expectedSequence":null,"parent":null,"algorithmVersion":1,"mappings":[],"reason":"\u0000"}`)} {
		if _, e := DecodeCorrectionInput(raw, correction.CreatePlanAction); e == nil {
			t.Fatal("malformed JSON input accepted")
		}
	}
}
