package rbac

import (
	"database/sql"
	"fmt"
	"time"
)

type tokenRepository struct {
	db *sql.DB
}

func newTokenRepository(db *sql.DB) TokenRepository {
	if db == nil {
		return nil
	}
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(token string, userID int, expiresAt time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("token repository 未初始化")
	}
	nowMs := time.Now().UnixMilli()
	_, err := r.db.Exec(`
		INSERT INTO user_tokens (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, token, userID, expiresAt.UTC().Format(time.RFC3339), nowMs)
	return err
}

func (r *tokenRepository) Lookup(token string) (*TokenRecord, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("token repository 未初始化")
	}
	rec := &TokenRecord{}
	var expiresAt string
	var createdAtMs int64
	err := r.db.QueryRow(`
		SELECT t.user_id, t.expires_at, t.created_at, u.username, u.role_id, r.name
		FROM user_tokens t
		JOIN users u ON t.user_id = u.id
		JOIN roles r ON u.role_id = r.id
		WHERE t.token = ?
	`, token).Scan(&rec.UserID, &expiresAt, &createdAtMs, &rec.Username, &rec.RoleID, &rec.RoleName)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	rec.ExpiresAt, err = time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, err
	}
	if createdAtMs > 0 {
		rec.CreatedAt = time.UnixMilli(createdAtMs)
	}
	return rec, nil
}

func (r *tokenRepository) ListByUserID(userID int) ([]TokenRecord, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("token repository 未初始化")
	}
	rows, err := r.db.Query(`
		SELECT token, user_id, expires_at, created_at
		FROM user_tokens
		WHERE user_id = ?
		ORDER BY created_at DESC, expires_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []TokenRecord{}
	for rows.Next() {
		var rec TokenRecord
		var token string
		var expiresAt string
		var createdAtMs int64
		if err := rows.Scan(&token, &rec.UserID, &expiresAt, &createdAtMs); err != nil {
			return nil, err
		}
		rec.ExpiresAt, err = time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return nil, err
		}
		if createdAtMs > 0 {
			rec.CreatedAt = time.UnixMilli(createdAtMs)
		}
		rec.Token = token
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *tokenRepository) Delete(token string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("token repository 未初始化")
	}
	_, err := r.db.Exec(`DELETE FROM user_tokens WHERE token = ?`, token)
	return err
}

func (r *tokenRepository) DeleteByUserID(userID int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("token repository 未初始化")
	}
	_, err := r.db.Exec(`DELETE FROM user_tokens WHERE user_id = ?`, userID)
	return err
}

func (r *tokenRepository) DeleteByUserIDExcept(userID int, exceptToken string) (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("token repository 未初始化")
	}
	res, err := r.db.Exec(`
		DELETE FROM user_tokens WHERE user_id = ? AND token != ?
	`, userID, exceptToken)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

func (r *tokenRepository) DeleteExpired() (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("token repository 未初始化")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Exec(`DELETE FROM user_tokens WHERE expires_at < ?`, now)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}
