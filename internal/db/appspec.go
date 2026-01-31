package db

// ResourceSpec defines CPU and memory requirements for a container application
type ResourceSpec struct {
	CPURequest    string `json:"cpu_request,omitempty"`    // CPU request (e.g., "500m")
	CPULimit      string `json:"cpu_limit,omitempty"`      // CPU limit (e.g., "2")
	MemoryRequest string `json:"memory_request,omitempty"` // Memory request (e.g., "512Mi")
	MemoryLimit   string `json:"memory_limit,omitempty"`   // Memory limit (e.g., "2Gi")
}

// VolumeSpec defines a volume mount for a container application
type VolumeSpec struct {
	Name      string `json:"name"`                // Volume name (must be unique within app)
	MountPath string `json:"mount_path"`          // Path inside container where volume is mounted
	Type      string `json:"type"`                // Volume type: "emptyDir", "configMap", "secret", "pvc"
	Source    string `json:"source,omitempty"`    // Source reference (configMap/secret name or PVC name)
	ReadOnly  bool   `json:"read_only,omitempty"` // Mount as read-only
}

// NetworkRuleSpec defines network access rules for a container application
type NetworkRuleSpec struct {
	AllowInternet bool     `json:"allow_internet,omitempty"` // Allow outbound internet access
	ExposedPorts  []int    `json:"exposed_ports,omitempty"`  // Ports exposed by the container
	AllowedHosts  []string `json:"allowed_hosts,omitempty"`  // Allowed egress destination hosts
}

// AppSpec defines the complete specification for a containerized application.
// This model captures all configuration needed to launch a container in Kubernetes.
type AppSpec struct {
	// Image is the container image to run (required for container apps)
	Image string `json:"image"`

	// Command overrides the container entrypoint
	Command []string `json:"command,omitempty"`

	// Args provides arguments to the command
	Args []string `json:"args,omitempty"`

	// Env defines environment variables for the container
	Env map[string]string `json:"env,omitempty"`

	// Resources defines CPU and memory requirements
	Resources *ResourceSpec `json:"resources,omitempty"`

	// Volumes defines volume mounts for the container
	Volumes []VolumeSpec `json:"volumes,omitempty"`

	// NetworkRules defines network access policies
	NetworkRules *NetworkRuleSpec `json:"network_rules,omitempty"`
}

// DefaultResourceSpec returns sensible default resource requirements
func DefaultResourceSpec() *ResourceSpec {
	return &ResourceSpec{
		CPURequest:    "500m",
		CPULimit:      "2",
		MemoryRequest: "512Mi",
		MemoryLimit:   "2Gi",
	}
}

// DefaultNetworkRules returns default network rules (restricted)
func DefaultNetworkRules() *NetworkRuleSpec {
	return &NetworkRuleSpec{
		AllowInternet: false,
		ExposedPorts:  []int{},
		AllowedHosts:  []string{},
	}
}

// NewAppSpec creates a new AppSpec with the given image and default settings
func NewAppSpec(image string) *AppSpec {
	return &AppSpec{
		Image:        image,
		Resources:    DefaultResourceSpec(),
		NetworkRules: DefaultNetworkRules(),
	}
}

// WithCommand sets the command and returns the AppSpec for chaining
func (s *AppSpec) WithCommand(command ...string) *AppSpec {
	s.Command = command
	return s
}

// WithArgs sets the args and returns the AppSpec for chaining
func (s *AppSpec) WithArgs(args ...string) *AppSpec {
	s.Args = args
	return s
}

// WithEnv adds an environment variable and returns the AppSpec for chaining
func (s *AppSpec) WithEnv(key, value string) *AppSpec {
	if s.Env == nil {
		s.Env = make(map[string]string)
	}
	s.Env[key] = value
	return s
}

// WithVolume adds a volume mount and returns the AppSpec for chaining
func (s *AppSpec) WithVolume(name, mountPath, volumeType string) *AppSpec {
	s.Volumes = append(s.Volumes, VolumeSpec{
		Name:      name,
		MountPath: mountPath,
		Type:      volumeType,
	})
	return s
}

// GetCPURequest returns the CPU request or the default value
func (s *AppSpec) GetCPURequest() string {
	if s.Resources != nil && s.Resources.CPURequest != "" {
		return s.Resources.CPURequest
	}
	return "500m"
}

// GetCPULimit returns the CPU limit or the default value
func (s *AppSpec) GetCPULimit() string {
	if s.Resources != nil && s.Resources.CPULimit != "" {
		return s.Resources.CPULimit
	}
	return "2"
}

// GetMemoryRequest returns the memory request or the default value
func (s *AppSpec) GetMemoryRequest() string {
	if s.Resources != nil && s.Resources.MemoryRequest != "" {
		return s.Resources.MemoryRequest
	}
	return "512Mi"
}

// GetMemoryLimit returns the memory limit or the default value
func (s *AppSpec) GetMemoryLimit() string {
	if s.Resources != nil && s.Resources.MemoryLimit != "" {
		return s.Resources.MemoryLimit
	}
	return "2Gi"
}
