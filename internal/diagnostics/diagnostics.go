// Package diagnostics provides system diagnostic collection and support bundle
// generation for enterprise supportability.
package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

// SystemInfo contains information about the system and runtime.
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	Hostname     string `json:"hostname"`
	StartTime    string `json:"start_time,omitempty"`
}

// MemoryStats contains memory usage statistics.
type MemoryStats struct {
	Alloc        uint64 `json:"alloc_bytes"`
	TotalAlloc   uint64 `json:"total_alloc_bytes"`
	Sys          uint64 `json:"sys_bytes"`
	NumGC        uint32 `json:"num_gc"`
	HeapAlloc    uint64 `json:"heap_alloc_bytes"`
	HeapInuse    uint64 `json:"heap_inuse_bytes"`
	HeapReleased uint64 `json:"heap_released_bytes"`
}

// DatabaseStats contains database statistics.
type DatabaseStats struct {
	AppCount     int   `json:"app_count"`
	SessionCount int   `json:"session_count"`
	AuditCount   int   `json:"audit_count"`
	DBSizeBytes  int64 `json:"db_size_bytes,omitempty"`
}

// ComponentStatus represents the status of a system component.
type ComponentStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// DiagnosticsReport is the complete diagnostics bundle.
type DiagnosticsReport struct {
	Timestamp   time.Time         `json:"timestamp"`
	Version     string            `json:"version"`
	System      SystemInfo        `json:"system"`
	Memory      MemoryStats       `json:"memory"`
	Database    DatabaseStats     `json:"database"`
	Components  []ComponentStatus `json:"components"`
	Config      map[string]any    `json:"config,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// Collector gathers diagnostic information.
type Collector struct {
	version   string
	startTime time.Time
	dbStats   func() (DatabaseStats, error)
	config    map[string]any
}

// NewCollector creates a new diagnostics collector.
func NewCollector(version string) *Collector {
	return &Collector{
		version:   version,
		startTime: time.Now(),
		config:    make(map[string]any),
	}
}

// SetDBStatsFunc sets the function to collect database statistics.
func (c *Collector) SetDBStatsFunc(f func() (DatabaseStats, error)) {
	c.dbStats = f
}

// SetConfig sets the configuration snapshot (non-sensitive only).
func (c *Collector) SetConfig(config map[string]any) {
	c.config = config
}

// Collect gathers all diagnostic information.
func (c *Collector) Collect(ctx context.Context) (*DiagnosticsReport, error) {
	hostname, _ := os.Hostname()

	// Collect memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	report := &DiagnosticsReport{
		Timestamp: time.Now().UTC(),
		Version:   c.version,
		System: SystemInfo{
			GoVersion:    runtime.Version(),
			GOOS:         runtime.GOOS,
			GOARCH:       runtime.GOARCH,
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			Hostname:     hostname,
			StartTime:    c.startTime.Format(time.RFC3339),
		},
		Memory: MemoryStats{
			Alloc:        m.Alloc,
			TotalAlloc:   m.TotalAlloc,
			Sys:          m.Sys,
			NumGC:        m.NumGC,
			HeapAlloc:    m.HeapAlloc,
			HeapInuse:    m.HeapInuse,
			HeapReleased: m.HeapReleased,
		},
		Components: []ComponentStatus{},
		Config:     c.config,
	}

	// Collect database stats if available
	if c.dbStats != nil {
		dbStats, err := c.dbStats()
		if err != nil {
			report.Components = append(report.Components, ComponentStatus{
				Name:    "database",
				Status:  "error",
				Message: err.Error(),
			})
		} else {
			report.Database = dbStats
			report.Components = append(report.Components, ComponentStatus{
				Name:   "database",
				Status: "healthy",
			})
		}
	}

	// Collect environment (filtered for non-sensitive vars)
	report.Environment = c.collectSafeEnvironment()

	return report, nil
}

// collectSafeEnvironment returns environment variables safe for diagnostics.
func (c *Collector) collectSafeEnvironment() map[string]string {
	safeVars := map[string]string{}
	safeKeys := []string{
		"LAUNCHPAD_PORT",
		"LAUNCHPAD_NAMESPACE",
		"LAUNCHPAD_SESSION_TIMEOUT",
		"LAUNCHPAD_SESSION_CLEANUP_INTERVAL",
		"LAUNCHPAD_POD_READY_TIMEOUT",
		"LAUNCHPAD_TENANT_NAME",
		"LAUNCHPAD_PRIMARY_COLOR",
		"LAUNCHPAD_SECONDARY_COLOR",
	}

	for _, key := range safeKeys {
		if val := os.Getenv(key); val != "" {
			safeVars[key] = val
		}
	}

	return safeVars
}

// Handler provides HTTP handlers for diagnostics endpoints.
type Handler struct {
	collector *Collector
}

// NewHandler creates a new diagnostics handler.
func NewHandler(collector *Collector) *Handler {
	return &Handler{collector: collector}
}

// HandleDiagnostics handles the /api/admin/diagnostics endpoint.
func (h *Handler) HandleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	report, err := h.collector.Collect(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect diagnostics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// HandleSupportBundle handles the /api/admin/support-bundle endpoint.
// Returns a downloadable ZIP file containing diagnostics and logs.
func (h *Handler) HandleSupportBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Collect diagnostics
	report, err := h.collector.Collect(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to collect diagnostics: %v", err), http.StatusInternalServerError)
		return
	}

	// Create ZIP archive in memory
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add diagnostics.json
	diagJSON, _ := json.MarshalIndent(report, "", "  ")
	if err := addFileToZip(zipWriter, "diagnostics.json", diagJSON); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create bundle: %v", err), http.StatusInternalServerError)
		return
	}

	// Add system info summary
	summary := fmt.Sprintf(`Launchpad Support Bundle
Generated: %s
Version: %s
Hostname: %s
Go Version: %s
OS/Arch: %s/%s
`,
		report.Timestamp.Format(time.RFC3339),
		report.Version,
		report.System.Hostname,
		report.System.GoVersion,
		report.System.GOOS,
		report.System.GOARCH,
	)
	if err := addFileToZip(zipWriter, "README.txt", []byte(summary)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create bundle: %v", err), http.StatusInternalServerError)
		return
	}

	// Close the ZIP writer
	if err := zipWriter.Close(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to finalize bundle: %v", err), http.StatusInternalServerError)
		return
	}

	// Send the ZIP file
	filename := fmt.Sprintf("launchpad-support-bundle-%s.zip", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	io.Copy(w, buf)
}

func addFileToZip(zw *zip.Writer, filename string, data []byte) error {
	f, err := zw.Create(filename)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}
