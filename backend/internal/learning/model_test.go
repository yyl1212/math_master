package learning

import (
	"testing"
	"time"
)

func TestLearningSealRejectsUnknownEventKind(t *testing.T) {
	if _, _, err := CanonicalLearningEvent(EventSeal{Kind: "viewed", RecordedAt: time.Now()}); err == nil {
		t.Fatal("passive view became a learning event")
	}
}
