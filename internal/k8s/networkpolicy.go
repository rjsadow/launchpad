package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NetworkPolicyConfig contains configuration for creating a session NetworkPolicy
type NetworkPolicyConfig struct {
	SessionID    string
	AppID        string
	AllowedHosts []string
}

// BuildNetworkPolicy creates a NetworkPolicy for a session pod based on allowed hosts.
// If AllowedHosts is empty, the policy denies all external egress (only DNS is allowed).
// If AllowedHosts contains entries, egress is allowed to those hosts on ports 80 and 443.
func BuildNetworkPolicy(config *NetworkPolicyConfig) *networkingv1.NetworkPolicy {
	policyName := fmt.Sprintf("launchpad-session-%s", config.SessionID)

	// Base egress rules: always allow DNS
	egressRules := []networkingv1.NetworkPolicyEgressRule{
		{
			// Allow DNS lookups to kube-dns
			To: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"k8s-app": "kube-dns",
						},
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr("UDP"),
					Port:     portPtr(53),
				},
				{
					Protocol: protocolPtr("TCP"),
					Port:     portPtr(53),
				},
			},
		},
	}

	// If allowed hosts are specified, add egress rules for HTTP/HTTPS
	// Note: Kubernetes NetworkPolicy doesn't support hostname-based rules,
	// so we document this limitation. For hostname enforcement, use a
	// network policy engine like Cilium with DNS-aware policies.
	if len(config.AllowedHosts) > 0 {
		// Allow outbound HTTP/HTTPS traffic
		// The actual host restriction requires DNS-aware CNI (e.g., Cilium)
		// For standard NetworkPolicy, we allow ports but cannot restrict by hostname
		egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: protocolPtr("TCP"),
					Port:     portPtr(80),
				},
				{
					Protocol: protocolPtr("TCP"),
					Port:     portPtr(443),
				},
			},
		})
	}
	// If AllowedHosts is empty, no HTTP/HTTPS egress is allowed (only DNS)

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: GetNamespace(),
			Labels: map[string]string{
				SessionLabelKey:   config.SessionID,
				AppLabelKey:       config.AppID,
				ComponentLabelKey: "session-network-policy",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			// Apply to the specific session pod
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					SessionLabelKey: config.SessionID,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: egressRules,
		},
	}

	return policy
}

// CreateNetworkPolicy creates a NetworkPolicy in the cluster
func CreateNetworkPolicy(ctx context.Context, policy *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	return client.NetworkingV1().NetworkPolicies(GetNamespace()).Create(ctx, policy, metav1.CreateOptions{})
}

// DeleteNetworkPolicy deletes a NetworkPolicy by name
func DeleteNetworkPolicy(ctx context.Context, policyName string) error {
	client, err := GetClient()
	if err != nil {
		return err
	}

	return client.NetworkingV1().NetworkPolicies(GetNamespace()).Delete(ctx, policyName, metav1.DeleteOptions{})
}

// DeleteSessionNetworkPolicy deletes the NetworkPolicy for a specific session
func DeleteSessionNetworkPolicy(ctx context.Context, sessionID string) error {
	policyName := fmt.Sprintf("launchpad-session-%s", sessionID)
	return DeleteNetworkPolicy(ctx, policyName)
}

// Helper functions
func protocolPtr(p string) *corev1.Protocol {
	protocol := corev1.Protocol(p)
	return &protocol
}

func portPtr(port int) *intstr.IntOrString {
	p := intstr.FromInt(port)
	return &p
}
