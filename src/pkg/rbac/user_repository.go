package rbac

import (
	"database/sql"
	"fmt"
)

type userRepository struct {
	db *sql.DB
}

func newUserRepository(db *sql.DB) UserRepository {
	if db == nil {
		return nil
	}
	return &userRepository{db: db}
}

func (r *userRepository) Count() (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("user repository 未初始化")
	}
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (r *userRepository) FindByUsername(username string) (UserRecord, error) {
	if r == nil || r.db == nil {
		return UserRecord{}, fmt.Errorf("user repository 未初始化")
	}
	var rec UserRecord
	err := r.db.QueryRow(`
		SELECT id, username, password_hash, salt, role_id
		FROM users
		WHERE username = ?
	`, username).Scan(&rec.ID, &rec.Username, &rec.PasswordHash, &rec.Salt, &rec.RoleID)
	if err == sql.ErrNoRows {
		return UserRecord{}, sql.ErrNoRows
	}
	return rec, err
}

func (r *userRepository) Create(username, passwordHash, salt string, roleID int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("user repository 未初始化")
	}
	_, err := r.db.Exec(`
		INSERT INTO users (username, password_hash, salt, role_id)
		VALUES (?, ?, ?, ?)
	`, username, passwordHash, salt, roleID)
	return err
}

func (r *userRepository) UpdateRole(userID, roleID int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("user repository 未初始化")
	}
	_, err := r.db.Exec(`UPDATE users SET role_id = ? WHERE id = ?`, roleID, userID)
	return err
}

func (r *userRepository) UpdatePassword(userID int, passwordHash, salt string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("user repository 未初始化")
	}
	_, err := r.db.Exec(`UPDATE users SET password_hash = ?, salt = ? WHERE id = ?`, passwordHash, salt, userID)
	return err
}

func (r *userRepository) Delete(userID int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("user repository 未初始化")
	}
	res, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *userRepository) List() ([]UserResponse, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("user repository 未初始化")
	}
	rows, err := r.db.Query(`
		SELECT u.id, u.username, u.role_id, r.name, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []UserResponse{}
	for rows.Next() {
		var u UserResponse
		if err := rows.Scan(&u.ID, &u.Username, &u.RoleID, &u.RoleName, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *userRepository) AdminCredentials() (passwordHash, salt string, ok bool, err error) {
	if r == nil || r.db == nil {
		return "", "", false, fmt.Errorf("user repository 未初始化")
	}
	err = r.db.QueryRow(`SELECT password_hash, salt FROM users WHERE username = ?`, "admin").Scan(&passwordHash, &salt)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return passwordHash, salt, true, nil
}
