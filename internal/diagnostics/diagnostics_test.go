package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector("1.0.0")
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
	if c.version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", c.version)
	}
}

func TestCollect(t *testing.T) {
	c := NewCollector("1.0.0")

	// Set up a mock DB stats function
	c.SetDBStatsFunc(func() (DatabaseStats, error) {
		return DatabaseStats{
			AppCount:     10,
			SessionCount: 5,
			AuditCount:   100,
		}, nil
	})

	c.SetConfig(map[string]any{
		"port":   8080,
		"tenant": "TestTenant",
	})

	report, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if report.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", report.Version)
	}

	if report.System.NumCPU == 0 {
		t.Error("expected NumCPU > 0")
	}

	if report.Database.AppCount != 10 {
		t.Errorf("expected 10 apps, got %d", report.Database.AppCount)
	}

	if report.Config["port"] != 8080 {
		t.Errorf("expected port 8080, got %v", report.Config["port"])
	}
}

func TestHandleDiagnostics(t *testing.T) {
	c := NewCollector("1.0.0")
	h := NewHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics", nil)
	w := httptest.NewRecorder()

	h.HandleDiagnostics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var report DiagnosticsReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if report.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", report.Version)
	}
}

func TestHandleDiagnosticsMethodNotAllowed(t *testing.T) {
	c := NewCollector("1.0.0")
	h := NewHandler(c)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/diagnostics", nil)
	w := httptest.NewRecorder()

	h.HandleDiagnostics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleSupportBundle(t *testing.T) {
	c := NewCollector("1.0.0")
	h := NewHandler(c)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/support-bundle", nil)
	w := httptest.NewRecorder()

	h.HandleSupportBundle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/zip" {
		t.Errorf("expected content-type application/zip, got %s", contentType)
	}

	// Check content disposition
	contentDisp := w.Header().Get("Content-Disposition")
	if !strings.HasPrefix(contentDisp, "attachment; filename=launchpad-support-bundle-") {
		t.Errorf("unexpected content-disposition: %s", contentDisp)
	}

	// Verify it's a valid ZIP file
	reader, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("failed to read ZIP: %v", err)
	}

	// Check for expected files
	foundDiagnostics := false
	foundReadme := false
	for _, f := range reader.File {
		if f.Name == "diagnostics.json" {
			foundDiagnostics = true
		}
		if f.Name == "README.txt" {
			foundReadme = true
		}
	}

	if !foundDiagnostics {
		t.Error("diagnostics.json not found in ZIP")
	}
	if !foundReadme {
		t.Error("README.txt not found in ZIP")
	}
}

func TestHandleSupportBundleMethodNotAllowed(t *testing.T) {
	c := NewCollector("1.0.0")
	h := NewHandler(c)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/support-bundle", nil)
	w := httptest.NewRecorder()

	h.HandleSupportBundle(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestCollectSafeEnvironment(t *testing.T) {
	c := NewCollector("1.0.0")

	// The collector should only return safe environment variables
	env := c.collectSafeEnvironment()

	// These should not be in the result (unless explicitly set)
	sensitiveKeys := []string{
		"AWS_SECRET_ACCESS_KEY",
		"DATABASE_PASSWORD",
		"API_KEY",
	}

	for _, key := range sensitiveKeys {
		if _, exists := env[key]; exists {
			t.Errorf("sensitive key %s should not be in environment", key)
		}
	}
}
