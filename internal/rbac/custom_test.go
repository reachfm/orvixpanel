package rbac

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.CustomRole{}))
	return db
}

func TestValidateName(t *testing.T) {
	good := []string{"billing-readonly", "devops_admin", "noc-operator-2", "x"}
	for _, n := range good {
		require.NoError(t, ValidateName(n), "expected %q valid", n)
	}
	bad := []string{
		"", " ", "-leading-dash", "_leading_under",
		"with space", "with\nnewline", "with/slash", "x" + string(make([]byte, 65)),
	}
	for _, n := range bad {
		require.Error(t, ValidateName(n), "expected %q invalid", n)
	}
	// Built-in names reserved.
	for _, n := range []string{"root_admin", "account_owner", "support"} {
		require.ErrorIs(t, ValidateName(n), ErrBuiltinRoleClash)
	}
}

func TestCreateAndLookup(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	ctx := context.Background()

	grants := []PermissionGrant{
		{Resource: "domain", Actions: []string{"read", "create"}},
		{Resource: "hosting", Actions: []string{"*"}},
	}
	row, err := s.Create(ctx, "t1", "site-operator", "manage site domains", grants)
	require.NoError(t, err)
	require.Equal(t, "site-operator", row.Name)
	require.NotEmpty(t, row.Permissions)

	got, err := s.Get(ctx, "t1", "site-operator")
	require.NoError(t, err)
	require.Equal(t, "site-operator", got.Name)

	parsed, err := ParsePermissions(got)
	require.NoError(t, err)
	require.Len(t, parsed, 2)
	require.Equal(t, "domain", parsed[0].Resource)
}

func TestCreateRejectsEmptyPermissions(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	_, err := s.Create(context.Background(), "t1", "blank", "no perms", nil)
	require.Error(t, err)
}

func TestCreateRejectsBuiltinName(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	_, err := s.Create(context.Background(), "t1", "root_admin", "hijack", []PermissionGrant{{Resource: "*", Actions: []string{"*"}}})
	require.ErrorIs(t, err, ErrBuiltinRoleClash)
}

func TestUpdateAndDelete(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	ctx := context.Background()

	_, err := s.Create(ctx, "t1", "r1", "d", []PermissionGrant{{Resource: "x", Actions: []string{"read"}}})
	require.NoError(t, err)

	newPerms := []PermissionGrant{{Resource: "y", Actions: []string{"*"}}}
	require.NoError(t, s.Update(ctx, "t1", "r1", "updated", newPerms))

	got, err := s.Get(ctx, "t1", "r1")
	require.NoError(t, err)
	parsed, err := ParsePermissions(got)
	require.NoError(t, err)
	require.Equal(t, "y", parsed[0].Resource)

	require.NoError(t, s.Delete(ctx, "t1", "r1"))
	_, err = s.Get(ctx, "t1", "r1")
	require.ErrorIs(t, err, ErrRoleNotFound)
}

func TestHasPermissionForBuiltinAndCustom(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	ctx := context.Background()

	// Built-in: root_admin = "*" "*"
	require.True(t, HasPermissionFor(ctx, s, "t1", "root_admin", "anything", "delete"))
	// Built-in: account_owner cannot do billing
	require.False(t, HasPermissionFor(ctx, s, "t1", "account_owner", "billing", "write"))
	// Built-in: account_owner CAN do domain.create
	require.True(t, HasPermissionFor(ctx, s, "t1", "account_owner", "domain", "create"))

	// Custom: site-operator
	_, err := s.Create(ctx, "t1", "site-operator", "", []PermissionGrant{
		{Resource: "domain", Actions: []string{"read", "create"}},
	})
	require.NoError(t, err)
	require.True(t, HasPermissionFor(ctx, s, "t1", "site-operator", "domain", "create"))
	require.False(t, HasPermissionFor(ctx, s, "t1", "site-operator", "domain", "delete"))
	require.False(t, HasPermissionFor(ctx, s, "t1", "site-operator", "hosting", "read"))

	// Unknown role
	require.False(t, HasPermissionFor(ctx, s, "t1", "no-such-role", "x", "y"))
}

func TestAssignRole(t *testing.T) {
	db := newTestDB(t)
	s := New(db)
	ctx := context.Background()
	require.NoError(t, db.AutoMigrate(&models.User{}))

	u := &models.User{Email: "u@x", PasswordHash: "h", Role: "account_owner", TenantID: "t1"}
	require.NoError(t, db.Create(u).Error)

	// Assign built-in
	require.NoError(t, AssignRole(ctx, db, u.ID, "billing"))
	var got models.User
	require.NoError(t, db.First(&got, "id = ?", u.ID).Error)
	require.Equal(t, "billing", got.Role)

	// Assign custom
	_, err := s.Create(ctx, "t1", "noc", "", []PermissionGrant{{Resource: "x", Actions: []string{"*"}}})
	require.NoError(t, err)
	require.NoError(t, AssignRole(ctx, db, u.ID, "noc"))
	require.NoError(t, db.First(&got, "id = ?", u.ID).Error)
	require.Equal(t, "noc", got.Role)

	// Assign unknown
	require.ErrorIs(t, AssignRole(ctx, db, u.ID, "no-such-role"), ErrRoleNotFound)
}

func TestRoleCache(t *testing.T) {
	c := NewRoleCache()
	c.Set("t1", "r1", []PermissionGrant{{Resource: "x", Actions: []string{"*"}}})
	g, ok := c.Get("t1", "r1")
	require.True(t, ok)
	require.Len(t, g, 1)
	c.Invalidate("t1", "r1")
	_, ok = c.Get("t1", "r1")
	require.False(t, ok)
}
