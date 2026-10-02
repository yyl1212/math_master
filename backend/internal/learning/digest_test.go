package learning

import (
	"crypto/sha256"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
	"time"
)

func TestLearningSealEventImmutableCanonical(t *testing.T) {
	e := EventSeal{ID: "11111111-1111-4111-8111-111111111111", ActorID: "22222222-2222-4222-8222-222222222222", Knowledge: question.Identity{ID: "fractions", Version: 1, SHA256: strings.Repeat("a", 64)}, KnowledgePublicationID: "snapshot", Kind: "completed", RecordedAt: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}
	raw, h, err := CanonicalLearningEvent(e)
	if err != nil || h != fmt.Sprintf("%x", sha256.Sum256(raw)) || !strings.Contains(string(raw), `"purpose":"learning-event-v1"`) || !strings.Contains(string(raw), `"units":[]`) {
		t.Fatal(string(raw), h, err)
	}
	e.Kind = "started"
	_, h2, _ := CanonicalLearningEvent(e)
	if h == h2 {
		t.Fatal("different learning event collided")
	}
	e.Kind = "completed"
	e.Units = []question.Identity{{ID: "new-material", Version: 2, SHA256: strings.Repeat("b", 64)}}
	_, h3, _ := CanonicalLearningEvent(e)
	if h == h3 {
		t.Fatal("material not covered")
	}
}
