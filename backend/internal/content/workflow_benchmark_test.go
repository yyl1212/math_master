package content_test

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func BenchmarkWorkflowValidation(b *testing.B) {
	fixture, err := testutil.CapacityContent("testdata", "../../../content/catalogue/domains.json")
	if err != nil {
		b.Fatal(err)
	}
	input := fixture.Inputs[0]
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, report := content.ValidateWorkflow(context.Background(), fixture.Catalogue, input.Package, fixture.Reader)
		if !report.ReadyToSubmit {
			b.Fatal("capacity batch invalid")
		}
	}
}
func BenchmarkWorkflowSnapshot(b *testing.B) {
	fixture, err := testutil.CapacityContent("testdata", "../../../content/catalogue/domains.json")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		report, err := content.ValidateSnapshot(context.Background(), fixture.Catalogue, fixture.Snapshot, fixture.Reader)
		if err != nil || !report.ReadyToSubmit {
			b.Fatal("capacity snapshot invalid", err)
		}
	}
}
