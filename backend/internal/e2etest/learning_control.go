package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type learningDatabaseState struct {
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
	s := learningDatabaseState{Knowledge: []question.Identity{}, Blueprints: []question.Identity{}}
	e := db.QueryRowContext(ctx, `SELECT (SELECT snapshot_id::text FROM publication_heads),(SELECT publication_id::text FROM question_heads),(SELECT count(*) FROM learning_events),(SELECT count(*) FROM practice_attempts),(SELECT count(*) FROM assessment_attempts),(SELECT count(*) FROM assessment_answers),(SELECT count(*) FROM assessment_results),(SELECT count(*) FROM learning_qualification_events),(SELECT count(*) FROM learning_unlocks),(SELECT count(*) FROM learner_answer_exposures),(SELECT count(*) FROM learner_question_views)`).Scan(&s.KnowledgeHead, &s.QuestionHead, &s.LearningEvents, &s.PracticeAttempts, &s.AssessmentAttempts, &s.Answers, &s.Results, &s.Qualifications, &s.Unlocks, &s.Exposures, &s.Views)
	if e != nil {
		return s, e
	}
	rows, e := db.QueryContext(ctx, `SELECT id,version,sha256 FROM knowledge_versions WHERE id LIKE 'learning-%' ORDER BY id`)
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
	e = db.QueryRowContext(ctx, `SELECT id,version,sha256 FROM path_versions WHERE id='learning-route' ORDER BY version DESC LIMIT 1`).Scan(&s.Path.ID, &s.Path.Version, &s.Path.SHA256)
	if e != nil {
		return s, e
	}
	rows, e = db.QueryContext(ctx, `SELECT id,version,sha256 FROM question_blueprints WHERE id LIKE 'learning-%' ORDER BY id`)
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
