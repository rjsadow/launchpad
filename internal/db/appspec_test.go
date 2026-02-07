package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})
	return db
}

func TestCreateAndGetAppSpec(t *testing.T) {
	db := setupTestDB(t)

	spec := AppSpec{
		ID:          "spec-1",
		Name:        "My App",
		Image:       "nginx:latest",
		TemplateRef: "tpl-web",
		Command:     []string{"/bin/sh", "-c"},
		Args:        []string{"nginx -g 'daemon off;'"},
		Env: []EnvVar{
			{Name: "PORT", Value: "8080"},
			{Name: "ENV", Value: "production"},
		},
		Resources: &ResourceLimits{
			CPURequest:    "100m",
			CPULimit:      "500m",
			MemoryRequest: "128Mi",
			MemoryLimit:   "512Mi",
		},
		Volumes: []VolumeMount{
			{Name: "data", MountPath: "/var/data", Size: "1Gi", ReadOnly: false},
		},
		NetworkRules: []NetworkRule{
			{Direction: "ingress", Port: 80, Protocol: "TCP"},
			{Direction: "egress", Port: 443, Protocol: "TCP", CIDRBlock: "0.0.0.0/0"},
		},
		Port: 8080,
	}

	if err := db.CreateAppSpec(spec); err != nil {
		t.Fatalf("CreateAppSpec failed: %v", err)
	}

	got, err := db.GetAppSpec("spec-1")
	if err != nil {
		t.Fatalf("GetAppSpec failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetAppSpec returned nil")
	}

	if got.Name != "My App" {
		t.Errorf("Name = %q, want %q", got.Name, "My App")
	}
	if got.Image != "nginx:latest" {
		t.Errorf("Image = %q, want %q", got.Image, "nginx:latest")
	}
	if got.TemplateRef != "tpl-web" {
		t.Errorf("TemplateRef = %q, want %q", got.TemplateRef, "tpl-web")
	}
	if got.Port != 8080 {
		t.Errorf("Port = %d, want %d", got.Port, 8080)
	}

	// Command
	if len(got.Command) != 2 || got.Command[0] != "/bin/sh" {
		t.Errorf("Command = %v, want [/bin/sh -c]", got.Command)
	}

	// Args
	if len(got.Args) != 1 {
		t.Errorf("Args length = %d, want 1", len(got.Args))
	}

	// Env
	if len(got.Env) != 2 {
		t.Errorf("Env length = %d, want 2", len(got.Env))
	}
	if got.Env[0].Name != "PORT" || got.Env[0].Value != "8080" {
		t.Errorf("Env[0] = %+v, want {PORT 8080}", got.Env[0])
	}

	// Resources
	if got.Resources == nil {
		t.Fatal("Resources is nil")
	}
	if got.Resources.CPURequest != "100m" {
		t.Errorf("CPURequest = %q, want %q", got.Resources.CPURequest, "100m")
	}
	if got.Resources.MemoryLimit != "512Mi" {
		t.Errorf("MemoryLimit = %q, want %q", got.Resources.MemoryLimit, "512Mi")
	}

	// Volumes
	if len(got.Volumes) != 1 {
		t.Fatalf("Volumes length = %d, want 1", len(got.Volumes))
	}
	if got.Volumes[0].Name != "data" || got.Volumes[0].MountPath != "/var/data" {
		t.Errorf("Volumes[0] = %+v", got.Volumes[0])
	}
	if got.Volumes[0].Size != "1Gi" {
		t.Errorf("Volumes[0].Size = %q, want %q", got.Volumes[0].Size, "1Gi")
	}

	// NetworkRules
	if len(got.NetworkRules) != 2 {
		t.Fatalf("NetworkRules length = %d, want 2", len(got.NetworkRules))
	}
	if got.NetworkRules[0].Direction != "ingress" || got.NetworkRules[0].Port != 80 {
		t.Errorf("NetworkRules[0] = %+v", got.NetworkRules[0])
	}
	if got.NetworkRules[1].CIDRBlock != "0.0.0.0/0" {
		t.Errorf("NetworkRules[1].CIDRBlock = %q, want %q", got.NetworkRules[1].CIDRBlock, "0.0.0.0/0")
	}
}

func TestGetAppSpec_NotFound(t *testing.T) {
	db := setupTestDB(t)

	got, err := db.GetAppSpec("nonexistent")
	if err != nil {
		t.Fatalf("GetAppSpec failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent spec, got %+v", got)
	}
}

func TestListAppSpecs(t *testing.T) {
	db := setupTestDB(t)

	// Empty list
	specs, err := db.ListAppSpecs()
	if err != nil {
		t.Fatalf("ListAppSpecs failed: %v", err)
	}
	if specs != nil {
		t.Errorf("expected nil for empty list, got %d items", len(specs))
	}

	// Add two specs
	db.CreateAppSpec(AppSpec{ID: "b-spec", Name: "Beta App", Image: "beta:1"})
	db.CreateAppSpec(AppSpec{ID: "a-spec", Name: "Alpha App", Image: "alpha:1"})

	specs, err = db.ListAppSpecs()
	if err != nil {
		t.Fatalf("ListAppSpecs failed: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}

	// Should be ordered by name
	if specs[0].Name != "Alpha App" {
		t.Errorf("first spec = %q, want %q (ordered by name)", specs[0].Name, "Alpha App")
	}
}

func TestUpdateAppSpec(t *testing.T) {
	db := setupTestDB(t)

	spec := AppSpec{ID: "spec-1", Name: "Original", Image: "img:1", Port: 80}
	db.CreateAppSpec(spec)

	spec.Name = "Updated"
	spec.Image = "img:2"
	spec.Port = 9090
	spec.Env = []EnvVar{{Name: "NEW_VAR", Value: "hello"}}

	if err := db.UpdateAppSpec(spec); err != nil {
		t.Fatalf("UpdateAppSpec failed: %v", err)
	}

	got, _ := db.GetAppSpec("spec-1")
	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}
	if got.Image != "img:2" {
		t.Errorf("Image = %q, want %q", got.Image, "img:2")
	}
	if got.Port != 9090 {
		t.Errorf("Port = %d, want %d", got.Port, 9090)
	}
	if len(got.Env) != 1 || got.Env[0].Name != "NEW_VAR" {
		t.Errorf("Env = %+v, want [{NEW_VAR hello}]", got.Env)
	}
}

func TestUpdateAppSpec_NotFound(t *testing.T) {
	db := setupTestDB(t)

	err := db.UpdateAppSpec(AppSpec{ID: "nonexistent", Name: "x", Image: "x"})
	if err == nil {
		t.Error("expected error for nonexistent spec, got nil")
	}
}

func TestDeleteAppSpec(t *testing.T) {
	db := setupTestDB(t)

	db.CreateAppSpec(AppSpec{ID: "spec-1", Name: "To Delete", Image: "img:1"})

	if err := db.DeleteAppSpec("spec-1"); err != nil {
		t.Fatalf("DeleteAppSpec failed: %v", err)
	}

	got, _ := db.GetAppSpec("spec-1")
	if got != nil {
		t.Errorf("spec still exists after delete: %+v", got)
	}
}

func TestDeleteAppSpec_NotFound(t *testing.T) {
	db := setupTestDB(t)

	err := db.DeleteAppSpec("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent spec, got nil")
	}
}

func TestCreateAppSpec_Duplicate(t *testing.T) {
	db := setupTestDB(t)

	spec := AppSpec{ID: "spec-1", Name: "App", Image: "img:1"}
	db.CreateAppSpec(spec)

	err := db.CreateAppSpec(spec)
	if err == nil {
		t.Error("expected error for duplicate ID, got nil")
	}
}

func TestAppSpec_MinimalFields(t *testing.T) {
	db := setupTestDB(t)

	spec := AppSpec{ID: "minimal", Name: "Minimal"}
	if err := db.CreateAppSpec(spec); err != nil {
		t.Fatalf("CreateAppSpec with minimal fields failed: %v", err)
	}

	got, err := db.GetAppSpec("minimal")
	if err != nil {
		t.Fatalf("GetAppSpec failed: %v", err)
	}
	if got.Name != "Minimal" {
		t.Errorf("Name = %q, want %q", got.Name, "Minimal")
	}
	if got.Resources != nil {
		t.Errorf("Resources should be nil for minimal spec, got %+v", got.Resources)
	}
	if got.Env != nil {
		t.Errorf("Env should be nil for minimal spec, got %+v", got.Env)
	}
	if got.Volumes != nil {
		t.Errorf("Volumes should be nil for minimal spec, got %+v", got.Volumes)
	}
	if got.NetworkRules != nil {
		t.Errorf("NetworkRules should be nil for minimal spec, got %+v", got.NetworkRules)
	}
}
