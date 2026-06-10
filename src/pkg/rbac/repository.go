package rbac

import (
	"database/sql"
	"fmt"
	"time"
)

// UserRecord is the persisted credential row for authentication.
type UserRecord struct {
	ID           int
	Username     string
	PasswordHash string
	Salt         string
	RoleID       int
	CreatedAt    time.Time
}

// TokenRecord is a resolved session token with user and role context.
type TokenRecord struct {
	Token     string
	UserID    int
	ExpiresAt time.Time
	CreatedAt time.Time
	Username  string
	RoleID    int
	RoleName  string
}

// UserRepository manages RBAC users.
type UserRepository interface {
	Count() (int, error)
	FindByUsername(username string) (UserRecord, error)
	Create(username, passwordHash, salt string, roleID int) error
	UpdateRole(userID, roleID int) error
	UpdatePassword(userID int, passwordHash, salt string) error
	Delete(userID int) error
	List() ([]UserResponse, error)
	AdminCredentials() (passwordHash, salt string, ok bool, err error)
}

// RoleRepository manages RBAC roles and permissions.
type RoleRepository interface {
	List() ([]RoleResponse, error)
	Permissions(roleID int) ([]string, error)
	EnsureRole(id int, name, description string) error
	EnsurePermission(code, name, ptype string) error
	EnsureRolePermission(roleID int, permissionCode string) error
	SeedDefaults() error
}

// TokenRepository manages login sessions.
type TokenRepository interface {
	Create(token string, userID int, expiresAt time.Time) error
	Lookup(token string) (*TokenRecord, error)
	ListByUserID(userID int) ([]TokenRecord, error)
	Delete(token string) error
	DeleteByUserID(userID int) error
	DeleteByUserIDExcept(userID int, exceptToken string) (int, error)
	DeleteExpired() (int, error)
}

var (
	sqlDB     *sql.DB
	userRepo  UserRepository
	roleRepo  RoleRepository
	tokenRepo TokenRepository
)

func initRepositories(db *sql.DB) {
	sqlDB = db
	userRepo = newUserRepository(db)
	roleRepo = newRoleRepository(db)
	tokenRepo = newTokenRepository(db)
}

func requireUserRepo() (UserRepository, error) {
	if userRepo == nil {
		return nil, fmt.Errorf("user repository 未初始化")
	}
	return userRepo, nil
}

func requireRoleRepo() (RoleRepository, error) {
	if roleRepo == nil {
		return nil, fmt.Errorf("role repository 未初始化")
	}
	return roleRepo, nil
}

func requireTokenRepo() (TokenRepository, error) {
	if tokenRepo == nil {
		return nil, fmt.Errorf("token repository 未初始化")
	}
	return tokenRepo, nil
}

// SetRepositoriesForTest replaces repository implementations (for unit tests).
func SetRepositoriesForTest(users UserRepository, roles RoleRepository, tokens TokenRepository) {
	userRepo = users
	roleRepo = roles
	tokenRepo = tokens
}

// ResetRepositoriesForTest clears repository wiring after tests.
func ResetRepositoriesForTest() {
	userRepo = nil
	roleRepo = nil
	tokenRepo = nil
	sqlDB = nil
}
