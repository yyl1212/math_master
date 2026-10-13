package studyreview

import (
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"time"
)

// NextSchedule uses elapsed seconds at the database completion time, never local midnight.
func NextSchedule(previous *Schedule, current Ref, rating Rating, now time.Time) (Schedule, error) {
	if !knowledgeadmin.ValidManagedRef(current) || !validRating(rating) || !validTime(now) {
		return Schedule{}, ErrInvalid
	}
	result := Schedule{LastRef: current, LastRating: rating, Sequence: 1, GradedAt: now.UTC()}
	same := false
	if previous != nil {
		if previous.Stage < 0 || previous.Stage > 5 || previous.Sequence < 0 || previous.Sequence >= MaxSequence {
			return Schedule{}, ErrInvalid
		}
		result.Sequence = previous.Sequence + 1
		same = previous.LastRef == current
		if same {
			result.Stage = previous.Stage
		}
	}
	seconds := int64(0)
	switch rating {
	case Forgotten:
		result.Stage = 0
		seconds = 600
	case Uncertain:
		result.Stage = max(0, result.Stage-1)
		seconds = 86400
	case Familiar:
		if same && now.Before(previous.DueAt) {
			if !validTime(previous.DueAt) {
				return Schedule{}, ErrInvalid
			}
			result.DueAt = previous.DueAt.UTC()
			return result, nil
		}
		result.Stage = min(5, result.Stage+1)
		seconds = []int64{0, 86400, 259200, 604800, 1209600, 2592000}[result.Stage]
	}
	result.DueAt = now.UTC().Add(time.Duration(seconds) * time.Second)
	if !validTime(result.DueAt) || !result.DueAt.After(now) {
		return Schedule{}, ErrInvalid
	}
	return result, nil
}
func validTime(t time.Time) bool { return !t.IsZero() && t.UTC().Year() >= 1 && t.UTC().Year() <= 9999 }
