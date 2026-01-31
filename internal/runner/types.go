package runner

// Config holds configuration for initializing a Runner.
type Config struct {
	// Common configuration
	Namespace string // Namespace/project for workloads

	// Kubernetes-specific
	Kubeconfig      string // Path to kubeconfig file (empty for in-cluster)
	VNCSidecarImage string // VNC sidecar container image

	// Docker-specific (for future use)
	DockerHost string // Docker daemon socket/host

	// Nomad-specific (for future use)
	NomadAddr  string // Nomad server address
	NomadToken string // Nomad ACL token
}

// WorkloadConfig contains configuration for creating a workload.
type WorkloadConfig struct {
	// ID is a unique identifier for the workload (e.g., session ID)
	ID string

	// Name is a human-readable name for the workload
	Name string

	// AppID is the application identifier this workload belongs to
	AppID string

	// AppName is the human-readable application name
	AppName string

	// ContainerImage is the Docker image to run
	ContainerImage string

	// Command overrides the container entrypoint
	Command []string

	// Args are arguments to the command
	Args []string

	// EnvVars are environment variables to set
	EnvVars map[string]string

	// Labels are metadata labels for the workload
	Labels map[string]string

	// Resource limits and requests
	CPULimit      string // e.g., "2" for 2 cores
	MemoryLimit   string // e.g., "2Gi"
	CPURequest    string // e.g., "500m"
	MemoryRequest string // e.g., "512Mi"

	// EnableVNC indicates whether to add VNC sidecar for GUI apps
	EnableVNC bool

	// Ports to expose (port number -> protocol)
	Ports map[int]string
}

// DefaultWorkloadConfig returns a WorkloadConfig with sensible defaults.
func DefaultWorkloadConfig(id, appID, appName, image string) *WorkloadConfig {
	return &WorkloadConfig{
		ID:             id,
		AppID:          appID,
		AppName:        appName,
		ContainerImage: image,
		CPULimit:       "2",
		MemoryLimit:    "2Gi",
		CPURequest:     "500m",
		MemoryRequest:  "512Mi",
		EnableVNC:      true,
		Labels:         make(map[string]string),
		EnvVars:        make(map[string]string),
	}
}

// WorkloadStatus represents the current state of a workload.
type WorkloadStatus string

const (
	// StatusPending indicates the workload is being created.
	StatusPending WorkloadStatus = "pending"

	// StatusRunning indicates the workload is running and ready.
	StatusRunning WorkloadStatus = "running"

	// StatusFailed indicates the workload failed to start or crashed.
	StatusFailed WorkloadStatus = "failed"

	// StatusTerminated indicates the workload has been terminated.
	StatusTerminated WorkloadStatus = "terminated"

	// StatusUnknown indicates the status could not be determined.
	StatusUnknown WorkloadStatus = "unknown"
)

// IsTerminal returns true if the status represents a terminal state.
func (s WorkloadStatus) IsTerminal() bool {
	return s == StatusFailed || s == StatusTerminated
}

// IsReady returns true if the workload is ready to accept connections.
func (s WorkloadStatus) IsReady() bool {
	return s == StatusRunning
}

// Workload represents a running workload instance.
type Workload struct {
	// ID is the unique identifier for this workload
	ID string

	// Name is the runner-specific name (e.g., pod name, container name)
	Name string

	// Status is the current workload status
	Status WorkloadStatus

	// IP is the internal IP address of the workload (if available)
	IP string

	// VNCPort is the port for VNC WebSocket connections (if VNC is enabled)
	VNCPort int

	// Message contains additional status information (e.g., error messages)
	Message string

	// Labels are metadata labels on the workload
	Labels map[string]string
}
