package rbac

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"
)

type memoryUserRepository struct {
	mu     sync.Mutex
	nextID int
	users  map[int]UserRecord
	byName map[string]int
}

func newMemoryUserRepository(seed ...UserRecord) *memoryUserRepository {
	repo := &memoryUserRepository{
		nextID: 1,
		users:  map[int]UserRecord{},
		byName: map[string]int{},
	}
	for _, user := range seed {
		repo.users[user.ID] = user
		repo.byName[user.Username] = user.ID
		if user.ID >= repo.nextID {
			repo.nextID = user.ID + 1
		}
	}
	return repo
}

func (r *memoryUserRepository) Count() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users), nil
}

func (r *memoryUserRepository) FindByUsername(username string) (UserRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byName[username]
	if !ok {
		return UserRecord{}, sql.ErrNoRows
	}
	return r.users[id], nil
}

func (r *memoryUserRepository) findByID(userID int) (UserRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userID]
	if !ok {
		return UserRecord{}, sql.ErrNoRows
	}
	return user, nil
}

func (r *memoryUserRepository) Create(username, passwordHash, salt string, roleID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byName[username]; exists {
		return fmt.Errorf("username already exists")
	}
	id := r.nextID
	r.nextID++
	rec := UserRecord{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Salt:         salt,
		RoleID:       roleID,
		CreatedAt:    time.Now(),
	}
	r.users[id] = rec
	r.byName[username] = id
	return nil
}

func (r *memoryUserRepository) UpdateRole(userID, roleID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userID]
	if !ok {
		return sql.ErrNoRows
	}
	user.RoleID = roleID
	r.users[userID] = user
	return nil
}

func (r *memoryUserRepository) UpdatePassword(userID int, passwordHash, salt string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userID]
	if !ok {
		return sql.ErrNoRows
	}
	user.PasswordHash = passwordHash
	user.Salt = salt
	r.users[userID] = user
	return nil
}

func (r *memoryUserRepository) Delete(userID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userID]
	if !ok {
		return sql.ErrNoRows
	}
	delete(r.users, userID)
	delete(r.byName, user.Username)
	return nil
}

func (r *memoryUserRepository) List() ([]UserResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]UserResponse, 0, len(r.users))
	for _, user := range r.users {
		out = append(out, UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			RoleID:    user.RoleID,
			CreatedAt: user.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (r *memoryUserRepository) AdminCredentials() (passwordHash, salt string, ok bool, err error) {
	user, err := r.FindByUsername("admin")
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return user.PasswordHash, user.Salt, true, nil
}

type memoryRoleRepository struct {
	mu          sync.Mutex
	roles       map[int]RoleResponse
	permissions map[int]map[string]struct{}
}

func newMemoryRoleRepository() *memoryRoleRepository {
	return &memoryRoleRepository{
		roles: map[int]RoleResponse{
			1: {ID: 1, Name: "admin", Description: "admin"},
		},
		permissions: map[int]map[string]struct{}{
			1: {"menu:overview": {}, "menu:config": {}},
		},
	}
}

func (r *memoryRoleRepository) List() ([]RoleResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RoleResponse, 0, len(r.roles))
	for _, role := range r.roles {
		out = append(out, role)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *memoryRoleRepository) Permissions(roleID int) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := r.permissions[roleID]
	out := make([]string, 0, len(set))
	for code := range set {
		out = append(out, code)
	}
	sort.Strings(out)
	return out, nil
}

func (r *memoryRoleRepository) EnsureRole(id int, name, description string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[id] = RoleResponse{ID: id, Name: name, Description: description}
	return nil
}

func (r *memoryRoleRepository) EnsurePermission(code, name, ptype string) error {
	return nil
}

func (r *memoryRoleRepository) EnsureRolePermission(roleID int, permissionCode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.permissions[roleID] == nil {
		r.permissions[roleID] = map[string]struct{}{}
	}
	r.permissions[roleID][permissionCode] = struct{}{}
	return nil
}

func (r *memoryRoleRepository) SeedDefaults() error {
	return nil
}

func (r *memoryRoleRepository) roleName(roleID int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if role, ok := r.roles[roleID]; ok {
		return role.Name
	}
	return ""
}

type memoryTokenRecord struct {
	UserID    int
	ExpiresAt time.Time
	CreatedAt time.Time
}

type memoryTokenRepository struct {
	mu     sync.Mutex
	tokens map[string]memoryTokenRecord
	users  *memoryUserRepository
	roles  *memoryRoleRepository
}

func newMemoryTokenRepository(users *memoryUserRepository, roles *memoryRoleRepository) *memoryTokenRepository {
	return &memoryTokenRepository{
		tokens: map[string]memoryTokenRecord{},
		users:  users,
		roles:  roles,
	}
}

func (r *memoryTokenRepository) Create(token string, userID int, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token] = memoryTokenRecord{
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	return nil
}

func (r *memoryTokenRepository) Lookup(token string) (*TokenRecord, error) {
	r.mu.Lock()
	base, ok := r.tokens[token]
	r.mu.Unlock()
	if !ok {
		return nil, sql.ErrNoRows
	}
	user, err := r.users.findByID(base.UserID)
	if err != nil {
		return nil, err
	}
	return &TokenRecord{
		Token:     token,
		UserID:    base.UserID,
		ExpiresAt: base.ExpiresAt,
		CreatedAt: base.CreatedAt,
		Username:  user.Username,
		RoleID:    user.RoleID,
		RoleName:  r.roles.roleName(user.RoleID),
	}, nil
}

func (r *memoryTokenRepository) ListByUserID(userID int) ([]TokenRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]TokenRecord, 0)
	for token, rec := range r.tokens {
		if rec.UserID != userID {
			continue
		}
		user, err := r.users.findByID(rec.UserID)
		if err != nil {
			continue
		}
		out = append(out, TokenRecord{
			Token:     token,
			UserID:    rec.UserID,
			ExpiresAt: rec.ExpiresAt,
			CreatedAt: rec.CreatedAt,
			Username:  user.Username,
			RoleID:    user.RoleID,
			RoleName:  r.roles.roleName(user.RoleID),
		})
	}
	return out, nil
}

func (r *memoryTokenRepository) Delete(token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tokens, token)
	return nil
}

func (r *memoryTokenRepository) DeleteByUserID(userID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for token, rec := range r.tokens {
		if rec.UserID == userID {
			delete(r.tokens, token)
		}
	}
	return nil
}

func (r *memoryTokenRepository) DeleteByUserIDExcept(userID int, exceptToken string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	removed := 0
	for token, rec := range r.tokens {
		if rec.UserID == userID && token != exceptToken {
			delete(r.tokens, token)
			removed++
		}
	}
	return removed, nil
}

func (r *memoryTokenRepository) DeleteExpired() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	removed := 0
	for token, rec := range r.tokens {
		if now.After(rec.ExpiresAt) {
			delete(r.tokens, token)
			removed++
		}
	}
	return removed, nil
}

func newMemoryRBACStores(adminPassword string) (UserRepository, RoleRepository, TokenRepository) {
	salt := "memory-salt"
	hash := HashPassword(adminPassword, salt)
	users := newMemoryUserRepository(UserRecord{
		ID:           1,
		Username:     "admin",
		PasswordHash: hash,
		Salt:         salt,
		RoleID:       1,
		CreatedAt:    time.Now(),
	})
	roles := newMemoryRoleRepository()
	tokens := newMemoryTokenRepository(users, roles)
	return users, roles, tokens
}
