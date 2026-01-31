package runner

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// RunnerTypeKubernetes is the type identifier for the Kubernetes runner.
	RunnerTypeKubernetes = "kubernetes"

	// Default VNC sidecar image
	defaultVNCSidecarImage = "ghcr.io/rjsadow/launchpad-vnc-sidecar:latest"

	// Label keys
	labelSessionID  = "launchpad.io/session-id"
	labelAppID      = "launchpad.io/app-id"
	labelComponent  = "app.kubernetes.io/component"
	labelManagedBy  = "app.kubernetes.io/managed-by"
	componentValue  = "session"
	managedByValue  = "launchpad"

	// Volume name for X11 socket
	x11SocketVolume = "x11-socket"
)

func init() {
	Register(RunnerTypeKubernetes, NewKubernetesRunner)
}

// KubernetesRunner implements the Runner interface for Kubernetes.
type KubernetesRunner struct {
	client          *kubernetes.Clientset
	namespace       string
	vncSidecarImage string

	mu sync.RWMutex
}

// NewKubernetesRunner creates a new Kubernetes runner.
func NewKubernetesRunner(cfg *Config) (Runner, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		config, err = buildConfigFromKubeconfig(cfg.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	namespace := cfg.Namespace
	if namespace == "" {
		namespace = resolveNamespace()
	}

	vncImage := cfg.VNCSidecarImage
	if vncImage == "" {
		vncImage = defaultVNCSidecarImage
	}

	return &KubernetesRunner{
		client:          client,
		namespace:       namespace,
		vncSidecarImage: vncImage,
	}, nil
}

// buildConfigFromKubeconfig builds a REST config from kubeconfig file.
func buildConfigFromKubeconfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig at %s: %w", kubeconfigPath, err)
	}

	return config, nil
}

// resolveNamespace determines the namespace to use.
func resolveNamespace() string {
	// Check environment variable
	if ns := os.Getenv("LAUNCHPAD_NAMESPACE"); ns != "" {
		return ns
	}

	// Try to read in-cluster namespace
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return string(data)
	}

	return "default"
}

// Type returns the runner type identifier.
func (r *KubernetesRunner) Type() string {
	return RunnerTypeKubernetes
}

// Close releases resources (no-op for Kubernetes).
func (r *KubernetesRunner) Close() error {
	return nil
}

// CreateWorkload creates a new pod for the workload.
func (r *KubernetesRunner) CreateWorkload(ctx context.Context, cfg *WorkloadConfig) (*Workload, error) {
	pod := r.buildPodSpec(cfg)

	createdPod, err := r.client.CoreV1().Pods(r.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}

	return r.workloadFromPod(createdPod), nil
}

// DeleteWorkload deletes the pod for the workload.
func (r *KubernetesRunner) DeleteWorkload(ctx context.Context, workloadID string) error {
	podName := r.podName(workloadID)
	err := r.client.CoreV1().Pods(r.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		// Check if already deleted
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete pod: %w", err)
	}
	return nil
}

// GetWorkload returns the current state of a workload.
func (r *KubernetesRunner) GetWorkload(ctx context.Context, workloadID string) (*Workload, error) {
	podName := r.podName(workloadID)
	pod, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrWorkloadNotFound
		}
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	return r.workloadFromPod(pod), nil
}

// WaitForReady waits for the workload to be ready.
func (r *KubernetesRunner) WaitForReady(ctx context.Context, workloadID string, timeout time.Duration) error {
	podName := r.podName(workloadID)

	err := wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		pod, err := r.client.CoreV1().Pods(r.namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if pod is ready
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		// Check if pod has failed
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return false, fmt.Errorf("%w: pod is in terminal state: %s", ErrWorkloadFailed, pod.Status.Phase)
		}

		return false, nil
	})

	if err != nil {
		if ctx.Err() != nil {
			return ErrTimeout
		}
		return err
	}
	return nil
}

// ListWorkloads lists all workloads managed by this runner.
func (r *KubernetesRunner) ListWorkloads(ctx context.Context, labels map[string]string) ([]*Workload, error) {
	// Build label selector
	var parts []string
	parts = append(parts, labelSessionID)
	parts = append(parts, fmt.Sprintf("%s=%s", labelManagedBy, managedByValue))

	// Add additional label filters
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	selector := strings.Join(parts, ",")

	pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	workloads := make([]*Workload, len(pods.Items))
	for i := range pods.Items {
		workloads[i] = r.workloadFromPod(&pods.Items[i])
	}

	return workloads, nil
}

// podName returns the pod name for a workload ID.
func (r *KubernetesRunner) podName(workloadID string) string {
	return fmt.Sprintf("launchpad-session-%s", workloadID)
}

// workloadFromPod converts a Kubernetes pod to a Workload.
func (r *KubernetesRunner) workloadFromPod(pod *corev1.Pod) *Workload {
	w := &Workload{
		ID:      pod.Labels[labelSessionID],
		Name:    pod.Name,
		IP:      pod.Status.PodIP,
		VNCPort: 6080, // WebSocket port for noVNC
		Labels:  pod.Labels,
	}

	// Determine status
	switch pod.Status.Phase {
	case corev1.PodPending:
		w.Status = StatusPending
	case corev1.PodRunning:
		// Check if actually ready
		ready := false
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
		if ready {
			w.Status = StatusRunning
		} else {
			w.Status = StatusPending
		}
	case corev1.PodSucceeded:
		w.Status = StatusTerminated
	case corev1.PodFailed:
		w.Status = StatusFailed
		// Extract failure message
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Terminated != nil && cs.State.Terminated.Message != "" {
				w.Message = cs.State.Terminated.Message
				break
			}
		}
	default:
		w.Status = StatusUnknown
	}

	return w
}

// buildPodSpec creates a Kubernetes Pod specification for a workload.
func (r *KubernetesRunner) buildPodSpec(cfg *WorkloadConfig) *corev1.Pod {
	podName := r.podName(cfg.ID)

	// Build environment variables for app container
	appEnv := []corev1.EnvVar{
		{Name: "DISPLAY", Value: ":99"},
	}
	for key, value := range cfg.EnvVars {
		appEnv = append(appEnv, corev1.EnvVar{Name: key, Value: value})
	}

	// Parse resource limits
	cpuLimit := resource.MustParse(cfg.CPULimit)
	memoryLimit := resource.MustParse(cfg.MemoryLimit)
	cpuRequest := resource.MustParse(cfg.CPURequest)
	memoryRequest := resource.MustParse(cfg.MemoryRequest)

	// Build labels
	podLabels := map[string]string{
		labelSessionID: cfg.ID,
		labelAppID:     cfg.AppID,
		labelComponent: componentValue,
		labelManagedBy: managedByValue,
	}
	maps.Copy(podLabels, cfg.Labels)

	containers := []corev1.Container{
		// Application container
		{
			Name:    "app",
			Image:   cfg.ContainerImage,
			Command: cfg.Command,
			Args:    cfg.Args,
			Env:     appEnv,
			VolumeMounts: []corev1.VolumeMount{
				{Name: x11SocketVolume, MountPath: "/tmp/.X11-unix"},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    cpuLimit,
					corev1.ResourceMemory: memoryLimit,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    cpuRequest,
					corev1.ResourceMemory: memoryRequest,
				},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: boolPtr(false),
				ReadOnlyRootFilesystem:   boolPtr(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			},
		},
	}

	// Add VNC sidecar if enabled
	if cfg.EnableVNC {
		vncContainer := corev1.Container{
			Name:  "vnc-sidecar",
			Image: r.vncSidecarImage,
			Ports: []corev1.ContainerPort{
				{Name: "vnc", ContainerPort: 5900, Protocol: corev1.ProtocolTCP},
				{Name: "websocket", ContainerPort: 6080, Protocol: corev1.ProtocolTCP},
			},
			Env: []corev1.EnvVar{
				{Name: "DISPLAY", Value: ":99"},
				{Name: "VNC_PASSWORD", Value: ""}, // No password for internal use
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: x11SocketVolume, MountPath: "/tmp/.X11-unix"},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: boolPtr(false),
				ReadOnlyRootFilesystem:   boolPtr(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(6080),
					},
				},
				InitialDelaySeconds: 2,
				PeriodSeconds:       5,
				TimeoutSeconds:      2,
				SuccessThreshold:    1,
				FailureThreshold:    6,
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(6080),
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       30,
				TimeoutSeconds:      5,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
		}
		// Prepend VNC container so it starts first
		containers = append([]corev1.Container{vncContainer}, containers...)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: r.namespace,
			Labels:    podLabels,
			Annotations: map[string]string{
				"launchpad.io/app-name":   cfg.AppName,
				"launchpad.io/created-at": time.Now().UTC().Format(time.RFC3339),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: boolPtr(true),
				RunAsUser:    int64Ptr(1000),
				RunAsGroup:   int64Ptr(1000),
				FSGroup:      int64Ptr(1000),
			},
			Volumes: []corev1.Volume{
				{
					Name: x11SocketVolume,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			Containers: containers,
		},
	}

	return pod
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}

func isNotFound(err error) bool {
	return apierrors.IsNotFound(err)
}
