package httpapi

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"strings"
	"testing"
)

type shareLockedNotes struct{ knowledgeadmin.StudyRepository }

func (*shareLockedNotes) SaveManagedNote(context.Context, knowledgeadmin.Access, string, knowledgeadmin.ManagedNoteInput) (knowledgeadmin.ManagedNote, error) {
	return knowledgeadmin.ManagedNote{}, notesharing.ErrLocked
}
func (*shareLockedNotes) DeleteManagedNote(context.Context, knowledgeadmin.Access, string, knowledgeadmin.ManagedNoteInput) (knowledgeadmin.ManagedNote, error) {
	return knowledgeadmin.ManagedNote{}, notesharing.ErrNotReady
}
func TestSharingLockedOriginalNoteHTTP(t *testing.T) {
	h := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Knowledge: &KnowledgeOptions{Study: &shareLockedNotes{}, PublicOrigin: privateOrigin}, PublicOrigin: privateOrigin})
	id := knowledgeadmin.KnowledgeID("lock-fixture")
	in := knowledgeadmin.ManagedNoteInput{Knowledge: knowledgeadmin.Ref{ID: id, ContentSHA256: strings.Repeat("a", 64), SourceKind: "managed"}, ExpectedRevision: 1, Body: "新正文"}
	b, _ := json.Marshal(in)
	w := privateRequest(h, "PUT", "/api/v3/study/knowledge/"+id+"/note", string(b), contentHeaders())
	if w.Code != 409 || !strings.Contains(w.Body.String(), "NOTE_SHARING_LOCKED") {
		t.Fatal("sharing lock mapping", w.Code, w.Body.String())
	}
	in.Body = ""
	b, _ = json.Marshal(in)
	w = privateRequest(h, "DELETE", "/api/v3/study/knowledge/"+id+"/note", string(b), contentHeaders())
	if w.Code != 503 || !strings.Contains(w.Body.String(), "NOTE_SHARING_NOT_READY") {
		t.Fatal("sharing schema mapping", w.Code, w.Body.String())
	}
}
