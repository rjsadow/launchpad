package rbac

import (
	"context"
	"testing"
)

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		permission Permission
		expected   bool
	}{
		// Admin permissions
		{"admin can create apps", RoleAdmin, PermissionAppCreate, true},
		{"admin can delete apps", RoleAdmin, PermissionAppDelete, true},
		{"admin can read audit", RoleAdmin, PermissionAuditRead, true},
		{"admin can view all sessions", RoleAdmin, PermissionSessionAll, true},

		// App-author permissions
		{"app-author can create apps", RoleAppAuthor, PermissionAppCreate, true},
		{"app-author can update apps", RoleAppAuthor, PermissionAppUpdate, true},
		{"app-author cannot delete apps", RoleAppAuthor, PermissionAppDelete, false},
		{"app-author cannot read audit", RoleAppAuthor, PermissionAuditRead, false},
		{"app-author cannot view all sessions", RoleAppAuthor, PermissionSessionAll, false},

		// User permissions
		{"user can read apps", RoleUser, PermissionAppRead, true},
		{"user cannot create apps", RoleUser, PermissionAppCreate, false},
		{"user cannot update apps", RoleUser, PermissionAppUpdate, false},
		{"user cannot delete apps", RoleUser, PermissionAppDelete, false},
		{"user can create sessions", RoleUser, PermissionSessionCreate, true},
		{"user cannot view all sessions", RoleUser, PermissionSessionAll, false},
		{"user cannot read audit", RoleUser, PermissionAuditRead, false},
		{"user cannot read analytics", RoleUser, PermissionAnalyticsRead, false},
		{"user can write analytics", RoleUser, PermissionAnalyticsWrite, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPermission(tt.role, tt.permission)
			if result != tt.expected {
				t.Errorf("HasPermission(%s, %s) = %v, expected %v", tt.role, tt.permission, result, tt.expected)
			}
		})
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"admin", true},
		{"app-author", true},
		{"user", true},
		{"Admin", false},
		{"ADMIN", false},
		{"superuser", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := IsValidRole(tt.role)
			if result != tt.expected {
				t.Errorf("IsValidRole(%s) = %v, expected %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestValidRoles(t *testing.T) {
	roles := ValidRoles()
	if len(roles) != 3 {
		t.Errorf("Expected 3 roles, got %d", len(roles))
	}

	expected := map[Role]bool{
		RoleAdmin:     true,
		RoleAppAuthor: true,
		RoleUser:      true,
	}

	for _, role := range roles {
		if !expected[role] {
			t.Errorf("Unexpected role: %s", role)
		}
	}
}

func TestGetPermissions(t *testing.T) {
	adminPerms := GetPermissions(RoleAdmin)
	if len(adminPerms) == 0 {
		t.Error("Admin should have permissions")
	}

	userPerms := GetPermissions(RoleUser)
	if len(userPerms) == 0 {
		t.Error("User should have permissions")
	}

	// Admin should have more permissions than user
	if len(adminPerms) <= len(userPerms) {
		t.Errorf("Admin should have more permissions than user: admin=%d, user=%d", len(adminPerms), len(userPerms))
	}

	// Invalid role should return nil
	invalidPerms := GetPermissions(Role("invalid"))
	if invalidPerms != nil {
		t.Error("Invalid role should return nil permissions")
	}
}

func TestUserHasPermission(t *testing.T) {
	adminUser := &User{ID: "admin1", Role: RoleAdmin}
	regularUser := &User{ID: "user1", Role: RoleUser}

	if !adminUser.HasPermission(PermissionAppDelete) {
		t.Error("Admin should have app:delete permission")
	}

	if regularUser.HasPermission(PermissionAppDelete) {
		t.Error("User should not have app:delete permission")
	}

	if !regularUser.HasPermission(PermissionAppRead) {
		t.Error("User should have app:read permission")
	}
}

func TestUserIsAdmin(t *testing.T) {
	adminUser := &User{ID: "admin1", Role: RoleAdmin}
	authorUser := &User{ID: "author1", Role: RoleAppAuthor}
	regularUser := &User{ID: "user1", Role: RoleUser}

	if !adminUser.IsAdmin() {
		t.Error("Admin user should be admin")
	}

	if authorUser.IsAdmin() {
		t.Error("App-author should not be admin")
	}

	if regularUser.IsAdmin() {
		t.Error("Regular user should not be admin")
	}
}

func TestContextWithUser(t *testing.T) {
	user := &User{ID: "test-user", Role: RoleUser}
	ctx := context.Background()

	// User not in context
	_, ok := UserFromContext(ctx)
	if ok {
		t.Error("Should not find user in empty context")
	}

	// Add user to context
	ctxWithUser := ContextWithUser(ctx, user)
	foundUser, ok := UserFromContext(ctxWithUser)
	if !ok {
		t.Error("Should find user in context")
	}
	if foundUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, foundUser.ID)
	}
	if foundUser.Role != user.Role {
		t.Errorf("Expected role %s, got %s", user.Role, foundUser.Role)
	}
}

func TestRequirePermission(t *testing.T) {
	adminUser := &User{ID: "admin1", Role: RoleAdmin}
	regularUser := &User{ID: "user1", Role: RoleUser}

	// No user in context
	ctx := context.Background()
	err := RequirePermission(ctx, PermissionAppRead)
	if err == nil {
		t.Error("Should error when no user in context")
	}

	// Admin with permission
	adminCtx := ContextWithUser(ctx, adminUser)
	err = RequirePermission(adminCtx, PermissionAppDelete)
	if err != nil {
		t.Errorf("Admin should have app:delete permission: %v", err)
	}

	// User without permission
	userCtx := ContextWithUser(ctx, regularUser)
	err = RequirePermission(userCtx, PermissionAppDelete)
	if err == nil {
		t.Error("User should not have app:delete permission")
	}

	// User with permission
	err = RequirePermission(userCtx, PermissionAppRead)
	if err != nil {
		t.Errorf("User should have app:read permission: %v", err)
	}
}
