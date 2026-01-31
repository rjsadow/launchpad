package db

import (
	"encoding/json"
	"testing"
)

func TestNewAppSpec(t *testing.T) {
	spec := NewAppSpec("nginx:latest")

	if spec.Image != "nginx:latest" {
		t.Errorf("expected image 'nginx:latest', got '%s'", spec.Image)
	}

	if spec.Resources == nil {
		t.Error("expected default resources to be set")
	}

	if spec.NetworkRules == nil {
		t.Error("expected default network rules to be set")
	}
}

func TestAppSpecChaining(t *testing.T) {
	spec := NewAppSpec("myapp:v1").
		WithCommand("/bin/sh", "-c").
		WithArgs("echo hello").
		WithEnv("MY_VAR", "my_value").
		WithVolume("data", "/app/data", "emptyDir")

	if len(spec.Command) != 2 || spec.Command[0] != "/bin/sh" {
		t.Errorf("expected command ['/bin/sh', '-c'], got %v", spec.Command)
	}

	if len(spec.Args) != 1 || spec.Args[0] != "echo hello" {
		t.Errorf("expected args ['echo hello'], got %v", spec.Args)
	}

	if spec.Env["MY_VAR"] != "my_value" {
		t.Errorf("expected env MY_VAR='my_value', got '%s'", spec.Env["MY_VAR"])
	}

	if len(spec.Volumes) != 1 || spec.Volumes[0].Name != "data" {
		t.Errorf("expected volume 'data', got %v", spec.Volumes)
	}
}

func TestAppSpecResourceDefaults(t *testing.T) {
	spec := &AppSpec{Image: "test:latest"}

	if spec.GetCPURequest() != "500m" {
		t.Errorf("expected default CPU request '500m', got '%s'", spec.GetCPURequest())
	}

	if spec.GetCPULimit() != "2" {
		t.Errorf("expected default CPU limit '2', got '%s'", spec.GetCPULimit())
	}

	if spec.GetMemoryRequest() != "512Mi" {
		t.Errorf("expected default memory request '512Mi', got '%s'", spec.GetMemoryRequest())
	}

	if spec.GetMemoryLimit() != "2Gi" {
		t.Errorf("expected default memory limit '2Gi', got '%s'", spec.GetMemoryLimit())
	}
}

func TestAppSpecResourceOverrides(t *testing.T) {
	spec := &AppSpec{
		Image: "test:latest",
		Resources: &ResourceSpec{
			CPURequest:    "1",
			CPULimit:      "4",
			MemoryRequest: "1Gi",
			MemoryLimit:   "8Gi",
		},
	}

	if spec.GetCPURequest() != "1" {
		t.Errorf("expected CPU request '1', got '%s'", spec.GetCPURequest())
	}

	if spec.GetCPULimit() != "4" {
		t.Errorf("expected CPU limit '4', got '%s'", spec.GetCPULimit())
	}

	if spec.GetMemoryRequest() != "1Gi" {
		t.Errorf("expected memory request '1Gi', got '%s'", spec.GetMemoryRequest())
	}

	if spec.GetMemoryLimit() != "8Gi" {
		t.Errorf("expected memory limit '8Gi', got '%s'", spec.GetMemoryLimit())
	}
}

func TestAppSpecJSONSerialization(t *testing.T) {
	spec := NewAppSpec("myapp:v1").
		WithCommand("/app/start").
		WithEnv("DEBUG", "true").
		WithVolume("config", "/etc/myapp", "configMap")

	spec.Volumes[0].Source = "myapp-config"
	spec.Volumes[0].ReadOnly = true

	spec.NetworkRules = &NetworkRuleSpec{
		AllowInternet: true,
		ExposedPorts:  []int{8080, 8443},
		AllowedHosts:  []string{"api.example.com"},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("failed to marshal AppSpec: %v", err)
	}

	var restored AppSpec
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal AppSpec: %v", err)
	}

	if restored.Image != spec.Image {
		t.Errorf("image mismatch: expected '%s', got '%s'", spec.Image, restored.Image)
	}

	if len(restored.Command) != 1 || restored.Command[0] != "/app/start" {
		t.Errorf("command mismatch: expected ['/app/start'], got %v", restored.Command)
	}

	if restored.Env["DEBUG"] != "true" {
		t.Errorf("env mismatch: expected DEBUG='true', got '%s'", restored.Env["DEBUG"])
	}

	if len(restored.Volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(restored.Volumes))
	}
	vol := restored.Volumes[0]
	if vol.Name != "config" || vol.MountPath != "/etc/myapp" || vol.Type != "configMap" {
		t.Errorf("volume mismatch: got %+v", vol)
	}
	if vol.Source != "myapp-config" || !vol.ReadOnly {
		t.Errorf("volume source/readonly mismatch: got %+v", vol)
	}

	if restored.NetworkRules == nil {
		t.Fatal("expected network rules to be restored")
	}
	if !restored.NetworkRules.AllowInternet {
		t.Error("expected AllowInternet to be true")
	}
	if len(restored.NetworkRules.ExposedPorts) != 2 {
		t.Errorf("expected 2 exposed ports, got %d", len(restored.NetworkRules.ExposedPorts))
	}
	if len(restored.NetworkRules.AllowedHosts) != 1 || restored.NetworkRules.AllowedHosts[0] != "api.example.com" {
		t.Errorf("allowed hosts mismatch: got %v", restored.NetworkRules.AllowedHosts)
	}
}

func TestVolumeTypes(t *testing.T) {
	testCases := []struct {
		volumeType string
		source     string
	}{
		{"emptyDir", ""},
		{"configMap", "my-config"},
		{"secret", "my-secret"},
		{"pvc", "my-pvc"},
	}

	for _, tc := range testCases {
		vol := VolumeSpec{
			Name:      "test-vol",
			MountPath: "/data",
			Type:      tc.volumeType,
			Source:    tc.source,
		}

		data, err := json.Marshal(vol)
		if err != nil {
			t.Errorf("failed to marshal volume type %s: %v", tc.volumeType, err)
			continue
		}

		var restored VolumeSpec
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Errorf("failed to unmarshal volume type %s: %v", tc.volumeType, err)
			continue
		}

		if restored.Type != tc.volumeType {
			t.Errorf("volume type mismatch: expected '%s', got '%s'", tc.volumeType, restored.Type)
		}
	}
}

func TestDefaultResourceSpec(t *testing.T) {
	r := DefaultResourceSpec()

	if r.CPURequest != "500m" {
		t.Errorf("expected CPU request '500m', got '%s'", r.CPURequest)
	}
	if r.CPULimit != "2" {
		t.Errorf("expected CPU limit '2', got '%s'", r.CPULimit)
	}
	if r.MemoryRequest != "512Mi" {
		t.Errorf("expected memory request '512Mi', got '%s'", r.MemoryRequest)
	}
	if r.MemoryLimit != "2Gi" {
		t.Errorf("expected memory limit '2Gi', got '%s'", r.MemoryLimit)
	}
}

func TestDefaultNetworkRules(t *testing.T) {
	n := DefaultNetworkRules()

	if n.AllowInternet {
		t.Error("expected AllowInternet to be false by default")
	}
	if len(n.ExposedPorts) != 0 {
		t.Errorf("expected empty exposed ports, got %v", n.ExposedPorts)
	}
	if len(n.AllowedHosts) != 0 {
		t.Errorf("expected empty allowed hosts, got %v", n.AllowedHosts)
	}
}
