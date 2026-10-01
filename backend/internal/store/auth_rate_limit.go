package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"math"
	"sort"
	"time"
)

func (s *Store) ConsumeRates(ctx context.Context, input []auth.RateKey) error {
	if len(input) == 0 || len(input) > 10 {
		return auth.ErrInvalidInput
	}
	keys := append([]auth.RateKey(nil), input...)
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if (a.Key == "") != (b.Key == "") {
			return a.Key == ""
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		return a.Window < b.Window
	})
	hasGlobal := false
	for i, key := range keys {
		if key.Scope == "" || len(key.Scope) > 64 || len(key.Key) > 128 || key.Limit < 1 || key.Limit > 1000000 || key.Window < time.Second || key.Window > 24*time.Hour || key.Window%time.Second != 0 {
			return auth.ErrInvalidInput
		}
		if i > 0 && key.Scope == keys[i-1].Scope && key.Key == keys[i-1].Key {
			return auth.ErrInvalidInput
		}
		if key.Key == "" {
			hasGlobal = true
		}
	}
	if !hasGlobal {
		return auth.ErrInvalidInput
	}
	retry := 0
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		take := func(key auth.RateKey) error {
			seconds := int64(key.Window / time.Second)
			start := time.Unix(now.Unix()/seconds*seconds, 0).UTC()
			end := start.Add(key.Window)
			var attempts int
			err := tx.QueryRowContext(ctx, `INSERT INTO auth_rate_limits(scope,key,window_start,window_end,attempts) VALUES($1,$2,$3,$4,1)
    ON CONFLICT(scope,key,window_start) DO UPDATE SET attempts=LEAST(auth_rate_limits.attempts+1,$5) RETURNING attempts`, key.Scope, key.Key, start, end, key.Limit+1).Scan(&attempts)
			if err != nil {
				return err
			}
			if attempts > key.Limit {
				n := int(math.Ceil(end.Sub(now).Seconds()))
				if n < 1 {
					n = 1
				}
				if n > retry {
					retry = n
				}
			}
			return nil
		}
		i := 0
		for i < len(keys) && keys[i].Key == "" {
			if err = take(keys[i]); err != nil {
				return err
			}
			i++
		}
		// Commit saturated global attempts, without creating adversarial per-identity keys.
		if retry > 0 {
			return nil
		}
		for ; i < len(keys); i++ {
			if err = take(keys[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if retry > 0 {
		return &auth.RateLimitError{RetryAfterSeconds: retry}
	}
	return nil
}
func (s *Store) CleanupAuth(ctx context.Context, limit int) (int, error) {
	if limit < 1 {
		return 0, auth.ErrInvalidInput
	}
	if limit > 2000 {
		limit = 2000
	}
	removed := 0
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		cutoff := now.Add(-24 * time.Hour)
		queries := []string{
			`WITH doomed AS (SELECT token_hash FROM auth_sessions WHERE LEAST(revoked_at,absolute_expires_at,last_seen_at+interval '30 minutes')<$1 ORDER BY token_hash LIMIT $2 FOR UPDATE SKIP LOCKED) DELETE FROM auth_sessions WHERE token_hash IN (SELECT token_hash FROM doomed)`,
			`WITH doomed AS (SELECT token_hash FROM auth_preauth WHERE LEAST(consumed_at,expires_at)<$1 ORDER BY token_hash LIMIT $2 FOR UPDATE SKIP LOCKED) DELETE FROM auth_preauth WHERE token_hash IN (SELECT token_hash FROM doomed)`,
			`WITH doomed AS (SELECT scope,key,window_start FROM auth_rate_limits WHERE window_end<$1 ORDER BY scope,key,window_start LIMIT $2 FOR UPDATE SKIP LOCKED) DELETE FROM auth_rate_limits WHERE (scope,key,window_start) IN (SELECT scope,key,window_start FROM doomed)`,
		}
		for _, q := range queries {
			if removed >= limit {
				break
			}
			result, err := tx.ExecContext(ctx, q, cutoff, limit-removed)
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			removed += int(n)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}
