package rbac

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type UserSession struct {
	UserID      int      `json:"userId"`
	Username    string   `json:"username"`
	RoleName    string   `json:"roleName"`
	Permissions []string `json:"permissions"`
}

type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	RoleID    int       `json:"roleId"`
	RoleName  string    `json:"roleName"`
	CreatedAt time.Time `json:"createdAt"`
}

type RoleResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AuthenticateUser checks credentials and returns a secure token on success.
func AuthenticateUser(username, password string) (string, error) {
	var userID int
	var passwordHash, salt string
	var roleID int

	err := db.QueryRow(`
		SELECT id, password_hash, salt, role_id 
		FROM users 
		WHERE username = ?
	`, username).Scan(&userID, &passwordHash, &salt, &roleID)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("用户名或密码错误")
		}
		return "", err
	}

	calculatedHash := HashPassword(password, salt)
	if calculatedHash != passwordHash {
		return "", fmt.Errorf("用户名或密码错误")
	}

	// Generate UUID-like secure token
	token, err := generateSecureToken()
	if err != nil {
		return "", err
	}

	// Persist token (valid for 24 hours)
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO user_tokens (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, token, userID, expiresAt)

	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateToken returns UserSession if token is valid and not expired.
func ValidateToken(token string) (*UserSession, error) {
	var userID int
	var expiresAt time.Time
	var username, roleName string
	var roleID int

	err := db.QueryRow(`
		SELECT t.user_id, t.expires_at, u.username, u.role_id, r.name
		FROM user_tokens t
		JOIN users u ON t.user_id = u.id
		JOIN roles r ON u.role_id = r.id
		WHERE t.token = ?
	`, token).Scan(&userID, &expiresAt, &username, &roleID, &roleName)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, err
	}

	if time.Now().After(expiresAt) {
		// Clean up expired token
		_, _ = db.Exec(`DELETE FROM user_tokens WHERE token = ?`, token)
		return nil, fmt.Errorf("token expired")
	}

	permissions, err := GetRolePermissions(roleID)
	if err != nil {
		return nil, err
	}

	return &UserSession{
		UserID:      userID,
		Username:    username,
		RoleName:    roleName,
		Permissions: permissions,
	}, nil
}

// InvalidateToken revokes the session.
func InvalidateToken(token string) error {
	_, err := db.Exec(`DELETE FROM user_tokens WHERE token = ?`, token)
	return err
}

// GetRolePermissions returns all permission codes linked to a role.
func GetRolePermissions(roleID int) ([]string, error) {
	rows, err := db.Query(`
		SELECT permission_code 
		FROM role_permissions 
		WHERE role_id = ?
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		permissions = append(permissions, code)
	}
	return permissions, nil
}

// CreateUser registers a new user with a hashed password.
func CreateUser(username, password string, roleID int) error {
	salt := generateSalt()
	passwordHash := HashPassword(password, salt)

	_, err := db.Exec(`
		INSERT INTO users (username, password_hash, salt, role_id)
		VALUES (?, ?, ?, ?)
	`, username, passwordHash, salt, roleID)
	return err
}

// UpdateUserRole updates role of a user.
func UpdateUserRole(userID int, roleID int) error {
	_, err := db.Exec(`UPDATE users SET role_id = ? WHERE id = ?`, roleID, userID)
	return err
}

// DeleteUser deletes a user and cleans up their tokens.
func DeleteUser(userID int) error {
	if userID == 1 {
		return fmt.Errorf("cannot delete default admin user")
	}
	_, _ = db.Exec(`DELETE FROM user_tokens WHERE user_id = ?`, userID)
	_, err := db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	return err
}

// ListUsers returns all users in system.
func ListUsers() ([]UserResponse, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.role_id, r.name, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserResponse
	for rows.Next() {
		var u UserResponse
		if err := rows.Scan(&u.ID, &u.Username, &u.RoleID, &u.RoleName, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// ListRoles returns all roles in system.
func ListRoles() ([]RoleResponse, error) {
	rows, err := db.Query(`SELECT id, name, description FROM roles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []RoleResponse
	for rows.Next() {
		var r RoleResponse
		if err := rows.Scan(&r.ID, &r.Name, &r.Description); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 24)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
