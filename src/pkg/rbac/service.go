package rbac

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type UserSession struct {
	UserID      int      `json:"userId"`
	Username    string   `json:"username"`
	RoleName    string   `json:"roleName"`
	Permissions []string `json:"permissions"`
}

// ErrNoSessionToken indicates no RBAC session token was provided.
var ErrNoSessionToken = fmt.Errorf("no session token")

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
// Legacy SHA256 hashes are upgraded to Argon2id after a successful login.
func AuthenticateUser(username, password string) (string, error) {
	users, err := requireUserRepo()
	if err != nil {
		return "", err
	}
	tokens, err := requireTokenRepo()
	if err != nil {
		return "", err
	}

	rec, err := users.FindByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("用户名或密码错误")
		}
		return "", err
	}

	if !VerifyPassword(password, rec.PasswordHash, rec.Salt) {
		return "", fmt.Errorf("用户名或密码错误")
	}

	if !IsArgon2Hash(rec.PasswordHash) {
		hash, salt := HashPasswordArgon2id(password)
		if err := users.UpdatePassword(rec.ID, hash, salt); err != nil {
			return "", err
		}
	}

	token, err := generateSecureToken()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := tokens.Create(token, rec.ID, expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

// ValidateToken returns UserSession if token is valid and not expired.
func ValidateToken(token string) (*UserSession, error) {
	tokens, err := requireTokenRepo()
	if err != nil {
		return nil, err
	}
	roles, err := requireRoleRepo()
	if err != nil {
		return nil, err
	}

	rec, err := tokens.Lookup(token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, err
	}
	if time.Now().After(rec.ExpiresAt) {
		_ = tokens.Delete(token)
		return nil, fmt.Errorf("token expired")
	}

	permissions, err := roles.Permissions(rec.RoleID)
	if err != nil {
		return nil, err
	}

	return &UserSession{
		UserID:      rec.UserID,
		Username:    rec.Username,
		RoleName:    rec.RoleName,
		Permissions: permissions,
	}, nil
}

// InvalidateToken revokes the session.
func InvalidateToken(token string) error {
	tokens, err := requireTokenRepo()
	if err != nil {
		return err
	}
	return tokens.Delete(token)
}

// GetRolePermissions returns all permission codes linked to a role.
func GetRolePermissions(roleID int) ([]string, error) {
	roles, err := requireRoleRepo()
	if err != nil {
		return nil, err
	}
	return roles.Permissions(roleID)
}

// EnsureRolePermission grants a permission to a role if missing.
func EnsureRolePermission(roleID int, permissionCode string) error {
	roles, err := requireRoleRepo()
	if err != nil {
		return err
	}
	return roles.EnsureRolePermission(roleID, permissionCode)
}

// NeedsSetup reports whether the system has no users and requires initial admin setup.
func NeedsSetup() (bool, error) {
	users, err := requireUserRepo()
	if err != nil {
		return false, err
	}
	count, err := users.Count()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// SetupInitialAdmin creates the first admin user when the database has no users.
func SetupInitialAdmin(username, password string) (string, error) {
	needs, err := NeedsSetup()
	if err != nil {
		return "", err
	}
	if !needs {
		return "", fmt.Errorf("系统已完成初始化")
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return "", fmt.Errorf("用户名和密码不能为空")
	}
	if len(password) < 8 {
		return "", fmt.Errorf("密码长度至少 8 位")
	}
	hash, salt := HashPasswordArgon2id(password)
	users, err := requireUserRepo()
	if err != nil {
		return "", err
	}
	if err := users.Create(username, hash, salt, 1); err != nil {
		return "", err
	}
	return AuthenticateUser(username, password)
}

// CreateUser registers a new user with an Argon2id hashed password.
func CreateUser(username, password string, roleID int) error {
	users, err := requireUserRepo()
	if err != nil {
		return err
	}
	hash, salt := HashPasswordArgon2id(password)
	return users.Create(username, hash, salt, roleID)
}

// UpdateUserRole updates role of a user.
func UpdateUserRole(userID int, roleID int) error {
	users, err := requireUserRepo()
	if err != nil {
		return err
	}
	return users.UpdateRole(userID, roleID)
}

// DeleteUser deletes a user and cleans up their tokens.
func DeleteUser(userID int) error {
	if userID == 1 {
		return fmt.Errorf("cannot delete default admin user")
	}
	users, err := requireUserRepo()
	if err != nil {
		return err
	}
	tokens, err := requireTokenRepo()
	if err != nil {
		return err
	}
	_ = tokens.DeleteByUserID(userID)
	return users.Delete(userID)
}

// ListUsers returns all users in system.
func ListUsers() ([]UserResponse, error) {
	users, err := requireUserRepo()
	if err != nil {
		return nil, err
	}
	return users.List()
}

// ListRoles returns all roles in system.
func ListRoles() ([]RoleResponse, error) {
	roles, err := requireRoleRepo()
	if err != nil {
		return nil, err
	}
	return roles.List()
}

func generateSecureToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
