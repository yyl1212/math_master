package assessment

import "github.com/yyl1212/math_master/backend/internal/question"

// ProjectQuestion deliberately reads only the presentation whitelist. The store must
// bind Knowledge.SHA256 from Seal.Knowledge before returning it to a learner.
func ProjectQuestion(position int, i question.Instance) (SafeQuestion, error) {
	if !question.ValidMathID(i.Body.Knowledge.ID) || i.Body.Knowledge.Version < 1 || position < 1 || position > 5 || !question.ValidInstanceID(i.Identity.ID) || i.Identity.Version < 1 || !question.ValidSHA(i.Identity.SHA256) {
		return SafeQuestion{}, question.ErrInvalid
	}
	out := SafeQuestion{Position: position, Instance: i.Identity, Knowledge: question.Identity{ID: i.Body.Knowledge.ID, Version: i.Body.Knowledge.Version}, Prompt: i.Body.Prompt, Choices: append([]question.Choice{}, i.Body.Choices...), Assets: append([]question.AssetRef{}, i.Body.Assets...)}
	switch i.Body.Type {
	case "single_choice":
		out.Type = "choice"
	case "numeric":
		out.Type = "numeric"
		if i.Body.AnswerFormat == nil || (*i.Body.AnswerFormat != "rational" && *i.Body.AnswerFormat != "percentage") {
			return SafeQuestion{}, question.ErrInvalid
		}
		format := *i.Body.AnswerFormat
		out.AnswerFormat = &format
	default:
		return SafeQuestion{}, question.ErrInvalid
	}
	return out, nil
}
