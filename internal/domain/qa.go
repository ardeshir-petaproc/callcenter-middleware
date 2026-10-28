package domain

// QACriterion is one question on a scorecard.
type QACriterion struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Weight   int    `json:"weight"`    // relative importance
	MaxScore int    `json:"max_score"` // scores range 0..max_score
	Critical bool   `json:"critical"`  // scoring 0 fails the whole evaluation
}

// QAScorecard defines how calls are evaluated.
type QAScorecard struct {
	ID          int64         `json:"id"`
	TenantID    int64         `json:"tenant_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Criteria    []QACriterion `json:"criteria"`
	PassScore   float64       `json:"pass_score"` // percentage
	Enabled     bool          `json:"enabled"`
}

// QAEvaluation is a reviewer's assessment of one call.
type QAEvaluation struct {
	ID           int64             `json:"id"`
	TenantID     int64             `json:"tenant_id"`
	CallID       string            `json:"call_id"`
	AgentID      int64             `json:"agent_id"`
	ScorecardID  int64             `json:"scorecard_id"`
	EvaluatorID  int64             `json:"evaluator_id"`
	Scores       map[string]int    `json:"scores"`
	Notes        map[string]string `json:"notes"`
	Comment      string            `json:"comment"`
	Score        float64           `json:"score"` // weighted percentage
	AutoFailed   bool              `json:"auto_failed"`
	Passed       bool              `json:"passed"`
	Status       string            `json:"status"` // draft or final (final is visible to the agent)
	Acknowledged bool              `json:"acknowledged"`
	AgentComment string            `json:"agent_comment"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

// Grade computes the weighted score of an evaluation against a scorecard.
func (e *QAEvaluation) Grade(sc *QAScorecard) {
	var got, total float64
	e.AutoFailed = false
	for _, c := range sc.Criteria {
		w, top := float64(max(c.Weight, 1)), max(c.MaxScore, 1)
		s := min(max(e.Scores[c.Key], 0), top)
		got += w * float64(s) / float64(top)
		total += w
		if c.Critical && s == 0 {
			e.AutoFailed = true
		}
	}
	e.Score = 0
	if total > 0 {
		e.Score = float64(int(got/total*1000+0.5)) / 10
	}
	if e.AutoFailed {
		e.Score = 0
	}
	e.Passed = !e.AutoFailed && e.Score >= sc.PassScore
}
