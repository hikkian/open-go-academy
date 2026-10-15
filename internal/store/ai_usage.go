package store

import (
	"database/sql"
	"errors"
	"math"
	"time"
)

var (
	ErrAIBudget        = errors.New("daily AI token budget exceeded")
	ErrAIUsageConflict = errors.New("AI usage reservation already finished")
	ErrAIUsageNotFound = errors.New("AI usage reservation not found")
)

type AIUsage struct {
	Day               string `json:"day"`
	UsedTokens        int    `json:"used_tokens"`
	ReservedTokens    int    `json:"reserved_tokens"`
	Requests          int    `json:"requests"`
	FailedRequests    int    `json:"failed_requests"`
	EstimatedRequests int    `json:"estimated_requests"`
}

const maxAITokens = 1_000_000

func validUsageDay(day string) bool {
	t, err := time.Parse("2006-01-02", day)
	return err == nil && t.Format("2006-01-02") == day
}

// AIUsage returns the durable token ledger summary for one local calendar day.
func (s *Store) AIUsage(day string) (AIUsage, error) {
	if !validUsageDay(day) {
		return AIUsage{}, errors.New("day must be YYYY-MM-DD")
	}
	u := AIUsage{Day: day}
	err := s.db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN status='finished' THEN charged_tokens ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status='reserved' THEN reserved_tokens ELSE 0 END),0),
		COUNT(*),
		COALESCE(SUM(CASE WHEN status='finished' AND success=0 THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status='finished' AND estimated=1 THEN 1 ELSE 0 END),0)
		FROM ai_usage_requests WHERE day=?`, day).Scan(&u.UsedTokens, &u.ReservedTokens, &u.Requests, &u.FailedRequests, &u.EstimatedRequests)
	return u, err
}

// ReserveAI atomically reserves capacity before a provider request is made.
func (s *Store) ReserveAI(day string, tokens, budget int) (int64, error) {
	if !validUsageDay(day) {
		return 0, errors.New("day must be YYYY-MM-DD")
	}
	if tokens <= 0 || budget <= 0 || tokens > maxAITokens || budget > maxAITokens {
		return 0, errors.New("tokens and budget must be between 1 and 1000000")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var used, reserved int64
	err = tx.QueryRow(`SELECT COALESCE(SUM(CASE WHEN status='finished' THEN charged_tokens ELSE 0 END),0),COALESCE(SUM(CASE WHEN status='reserved' THEN reserved_tokens ELSE 0 END),0) FROM ai_usage_requests WHERE day=?`, day).Scan(&used, &reserved)
	if err != nil {
		return 0, err
	}
	if used+reserved+int64(tokens) > int64(budget) {
		return 0, ErrAIBudget
	}
	r, err := tx.Exec(`INSERT INTO ai_usage_requests(day,reserved_tokens,created_at) VALUES(?,?,?)`, day, tokens, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// FinishAI releases a reservation and records only numeric provider usage metadata.
// Failed or estimated calls consume the full reservation conservatively.
func (s *Store) FinishAI(id int64, inputTokens, outputTokens int, estimated, success bool) error {
	if id <= 0 || inputTokens < 0 || outputTokens < 0 || inputTokens > maxAITokens || outputTokens > maxAITokens || int64(inputTokens)+int64(outputTokens) > math.MaxInt32 {
		return errors.New("invalid AI usage result")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var reserved int
	var status string
	err = tx.QueryRow(`SELECT reserved_tokens,status FROM ai_usage_requests WHERE id=?`, id).Scan(&reserved, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAIUsageNotFound
	}
	if err != nil {
		return err
	}
	if status != "reserved" {
		return ErrAIUsageConflict
	}
	charged := inputTokens + outputTokens
	conservative := estimated || !success
	if conservative {
		charged = reserved
	}
	res, err := tx.Exec(`UPDATE ai_usage_requests SET charged_tokens=?,input_tokens=?,output_tokens=?,estimated=?,success=?,status='finished',finished_at=? WHERE id=? AND status='reserved'`, charged, inputTokens, outputTokens, boolInt(conservative), boolInt(success), time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrAIUsageConflict
	}
	return tx.Commit()
}
