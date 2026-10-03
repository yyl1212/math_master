package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"net/http/httptest"
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
	if paths != 17 || operations != 20 {
		t.Fatal("fixed routes", paths, operations)
	}
	// The original 16 paths/19 operations are retained; the sole addition is
	// a session-owned read with the same private boundary and bounded SVG bytes.
	assetPath := "/api/v1/corrections/results/{id}/assets/{sha256}"
	if len(api.Paths[assetPath]) != 1 {
		t.Fatal("only the read-only correction image operation may be added")
	}
	var asset struct {
		OperationID string                `json:"operationId"`
		Security    []map[string][]string `json:"security"`
		Parameters  []struct {
			Name     string
			In       string
			Required bool
			Schema   struct{ Type, Pattern string }
		}
		Responses map[string]struct {
			Headers map[string]struct{ Schema map[string]any }
			Content map[string]struct{ Schema struct{ Type, Format string } }
		}
	}
	if json.Unmarshal(api.Paths[assetPath]["get"], &asset) != nil || asset.OperationID != "readOwnCorrectionAsset" || len(asset.Security) != 1 || asset.Security[0]["SessionCookie"] == nil || len(asset.Parameters) != 2 {
		t.Fatal("private owned correction SVG contract")
	}
	for _, parameter := range asset.Parameters {
		want := "^[0-9a-f]{64}$"
		if parameter.Name == "id" {
			want = "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
		} else if parameter.Name != "sha256" {
			t.Fatal("unexpected asset selector")
		}
		if parameter.In != "path" || !parameter.Required || parameter.Schema.Type != "string" || parameter.Schema.Pattern != want {
			t.Fatal("exact owned result and SHA required")
		}
	}
	success := asset.Responses["200"]
	svg := success.Content["image/svg+xml"].Schema
	if len(success.Content) != 1 || svg.Type != "string" || svg.Format != "binary" || success.Headers["Cache-Control"].Schema["const"] != "private, no-store" || success.Headers["X-Content-Type-Options"].Schema["const"] != "nosniff" || success.Headers["Content-Security-Policy"].Schema["const"] != "sandbox; default-src 'none'" || success.Headers["Content-Length"].Schema["maximum"] != float64(1048576) {
		t.Fatal("private bounded SVG response contract")
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

func TestCorrectionHTTPAuthErrorContract(t *testing.T) {
	raw, e := os.ReadFile("../../../api/openapi.yaml")
	if e != nil {
		t.Fatal(e)
	}
	var api struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Properties map[string]struct{ Enum []string }
				}
			}
		}
	}
	if json.Unmarshal(raw, &api) != nil {
		t.Fatal("OpenAPI")
	}
	for _, c := range []struct {
		err    error
		code   string
		status int
	}{{auth.ErrReauthRequired, "REAUTHENTICATION_REQUIRED", 428}, {auth.ErrPasswordChangeRequired, "PASSWORD_CHANGE_REQUIRED", 403}} {
		for _, name := range []string{"CorrectionError", "NotificationError"} {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/v1/corrections/cases", nil)
			r.Header.Set("X-Request-ID", strings.Repeat("a", 32))
			if name == "CorrectionError" {
				correctionError(w, r, c.err)
			} else {
				notificationError(w, r, c.err)
			}
			var body struct{ Error struct{ Code string } }
			if json.Unmarshal(w.Body.Bytes(), &body) != nil || w.Code != c.status || body.Error.Code != c.code {
				t.Fatal("auth contract", name, c.code, w.Code)
			}
			found := false
			for _, code := range api.Components.Schemas[name].Properties["error"].Properties["code"].Enum {
				if code == c.code {
					found = true
				}
			}
			if !found {
				t.Fatal(errors.New("authentication code missing from new contract"), name, c.code)
			}
		}
	}
}
