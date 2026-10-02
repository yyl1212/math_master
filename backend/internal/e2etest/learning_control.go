package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type learningDatabaseState struct {
	PublishedCounts    map[string]int      `json:"publishedCounts"`
	MaxPathNodes       int                 `json:"maxPathNodes"`
	KnowledgeHead      string              `json:"knowledgeHead"`
	QuestionHead       string              `json:"questionHead"`
	Knowledge          []question.Identity `json:"knowledge"`
	Path               question.Identity   `json:"path"`
	Blueprints         []question.Identity `json:"blueprints"`
	LearningEvents     int                 `json:"learningEvents"`
	PracticeAttempts   int                 `json:"practiceAttempts"`
	AssessmentAttempts int                 `json:"assessmentAttempts"`
	Answers            int                 `json:"answers"`
	Results            int                 `json:"results"`
	Qualifications     int                 `json:"qualifications"`
	Unlocks            int                 `json:"unlocks"`
	Exposures          int                 `json:"exposures"`
	Views              int                 `json:"views"`
}

func learningState(ctx context.Context, db *sql.DB) (learningDatabaseState, error) {
	s := learningDatabaseState{Knowledge: []question.Identity{}, Blueprints: []question.Identity{}, PublishedCounts: map[string]int{}}
	e := db.QueryRowContext(ctx, `SELECT (SELECT snapshot_id::text FROM publication_heads),(SELECT publication_id::text FROM question_heads),(SELECT count(*) FROM learning_events),(SELECT count(*) FROM practice_attempts),(SELECT count(*) FROM assessment_attempts),(SELECT count(*) FROM assessment_answers),(SELECT count(*) FROM assessment_results),(SELECT count(*) FROM learning_qualification_events),(SELECT count(*) FROM learning_unlocks),(SELECT count(*) FROM learner_answer_exposures),(SELECT count(*) FROM learner_question_views)`).Scan(&s.KnowledgeHead, &s.QuestionHead, &s.LearningEvents, &s.PracticeAttempts, &s.AssessmentAttempts, &s.Answers, &s.Results, &s.Qualifications, &s.Unlocks, &s.Exposures, &s.Views)
	if e != nil {
		return s, e
	}
	counts, e := db.QueryContext(ctx, `SELECT kind,count(*) FROM publication_members WHERE snapshot_id=$1 AND availability='active' GROUP BY kind UNION ALL SELECT kind,count(*) FROM question_publication_members WHERE publication_id=$2 GROUP BY kind`, s.KnowledgeHead, s.QuestionHead)
	if e != nil {
		return s, e
	}
	for counts.Next() {
		var kind string
		var n int
		if e = counts.Scan(&kind, &n); e != nil {
			counts.Close()
			return s, e
		}
		s.PublishedCounts[kind] = n
	}
	e = counts.Err()
	counts.Close()
	if e != nil {
		return s, e
	}
	if e = db.QueryRowContext(ctx, `SELECT coalesce(max(n),0) FROM (SELECT count(*) n FROM publication_members m JOIN path_nodes pn ON pn.path_id=m.id AND pn.path_version=m.version WHERE m.snapshot_id=$1 AND m.kind='path' AND m.availability='active' GROUP BY m.id) sizes`, s.KnowledgeHead).Scan(&s.MaxPathNodes); e != nil {
		return s, e
	}
	rows, e := db.QueryContext(ctx, `SELECT k.id,k.version,k.sha256 FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.availability='active' ORDER BY k.id`, s.KnowledgeHead)
	if e != nil {
		return s, e
	}
	for rows.Next() {
		var v question.Identity
		if e = rows.Scan(&v.ID, &v.Version, &v.SHA256); e != nil {
			rows.Close()
			return s, e
		}
		s.Knowledge = append(s.Knowledge, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return s, e
	}
	e = db.QueryRowContext(ctx, `SELECT p.id,p.version,p.sha256 FROM publication_members m JOIN path_versions p ON p.id=m.id AND p.version=m.version WHERE m.snapshot_id=$1 AND m.kind='path' AND m.availability='active' ORDER BY p.id LIMIT 1`, s.KnowledgeHead).Scan(&s.Path.ID, &s.Path.Version, &s.Path.SHA256)
	if e != nil {
		return s, e
	}
	rows, e = db.QueryContext(ctx, `SELECT id,version,sha256 FROM question_publication_members WHERE publication_id=$1 AND kind='blueprint' ORDER BY id`, s.QuestionHead)
	if e != nil {
		return s, e
	}
	defer rows.Close()
	for rows.Next() {
		var v question.Identity
		if e = rows.Scan(&v.ID, &v.Version, &v.SHA256); e != nil {
			return s, e
		}
		s.Blueprints = append(s.Blueprints, v)
	}
	return s, rows.Err()
}
