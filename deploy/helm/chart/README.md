# Launchpad Helm Chart

A Helm chart for deploying Launchpad - a centralized application launcher for large organizations.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- (Optional) An ingress controller for external access
- (Optional) A storage class for persistent storage

## Installation

### Add the Chart Repository

```bash
# If published to a Helm repository
helm repo add launchpad https://rjsadow.github.io/launchpad/charts
helm repo update
```

### Install from Local Source

```bash
cd deploy/helm
helm install launchpad ./chart -n launchpad --create-namespace
```

### Quick Start

```bash
# Basic installation
helm install launchpad ./chart -n launchpad --create-namespace

# With persistence enabled
helm install launchpad ./chart -n launchpad --create-namespace \
  --set persistence.enabled=true

# With custom branding
helm install launchpad ./chart -n launchpad --create-namespace \
  --set launchpad.branding.tenantName="My Company" \
  --set launchpad.branding.primaryColor="#0066cc"

# With ingress enabled
helm install launchpad ./chart -n launchpad --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=launchpad.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

## Uninstallation

```bash
helm uninstall launchpad -n launchpad
```

## Configuration

### Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| **Image** | | |
| `image.repository` | Container image repository | `ghcr.io/rjsadow/launchpad` |
| `image.pullPolicy` | Image pull policy | `Always` |
| `image.tag` | Image tag (defaults to appVersion) | `""` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| **Deployment** | | |
| `replicaCount` | Number of replicas | `1` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override full name | `""` |
| **Service Account** | | |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` |
| **Pod** | | |
| `podAnnotations` | Pod annotations | `{}` |
| `podSecurityContext.runAsNonRoot` | Run as non-root | `true` |
| `podSecurityContext.runAsUser` | User ID | `1000` |
| `podSecurityContext.runAsGroup` | Group ID | `1000` |
| `podSecurityContext.fsGroup` | FS Group ID | `1000` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.readOnlyRootFilesystem` | Read-only root filesystem | `true` |
| **Service** | | |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `80` |
| `service.targetPort` | Target container port | `8080` |
| **Ingress** | | |
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.annotations` | Ingress annotations | `{}` |
| `ingress.hosts` | Ingress hosts | See values.yaml |
| `ingress.tls` | TLS configuration | `[]` |
| **Resources** | | |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| **Health Probes** | | |
| `livenessProbe.httpGet.path` | Liveness probe path | `/api/config` |
| `livenessProbe.initialDelaySeconds` | Initial delay | `10` |
| `livenessProbe.periodSeconds` | Period | `30` |
| `livenessProbe.timeoutSeconds` | Timeout | `5` |
| `readinessProbe.httpGet.path` | Readiness probe path | `/api/config` |
| `readinessProbe.initialDelaySeconds` | Initial delay | `5` |
| `readinessProbe.periodSeconds` | Period | `10` |
| `readinessProbe.timeoutSeconds` | Timeout | `3` |
| **Launchpad Configuration** | | |
| `launchpad.server.port` | Server port | `8080` |
| `launchpad.server.database` | Database path | `/data/launchpad.db` |
| `launchpad.branding.logoUrl` | Logo URL | `""` |
| `launchpad.branding.primaryColor` | Primary brand color | `#398D9B` |
| `launchpad.branding.secondaryColor` | Secondary brand color | `#4AB7C3` |
| `launchpad.branding.tenantName` | Tenant name | `Launchpad` |
| `launchpad.kubernetes.namespace` | Session pod namespace | Release namespace |
| `launchpad.kubernetes.vncSidecarImage` | VNC sidecar image | `ghcr.io/rjsadow/launchpad-vnc-sidecar:latest` |
| `launchpad.session.timeout` | Session timeout (minutes) | `120` |
| `launchpad.session.cleanupInterval` | Cleanup interval (minutes) | `5` |
| `launchpad.session.podReadyTimeout` | Pod ready timeout (seconds) | `120` |
| **Persistence** | | |
| `persistence.enabled` | Enable persistent storage | `false` |
| `persistence.storageClass` | Storage class | `""` |
| `persistence.accessMode` | Access mode | `ReadWriteOnce` |
| `persistence.size` | Storage size | `1Gi` |
| `persistence.existingClaim` | Existing PVC name | `""` |
| **RBAC** | | |
| `rbac.create` | Create RBAC resources | `true` |
| **Node Selection** | | |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |

### Custom Values File

Create a custom `values.yaml`:

```yaml
# my-values.yaml
launchpad:
  branding:
    tenantName: "Acme Corp"
    primaryColor: "#0066cc"
    logoUrl: "https://acme.com/logo.png"
  session:
    timeout: 60

ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/websocket-services: "launchpad"
  hosts:
    - host: launchpad.acme.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: launchpad-tls
      hosts:
        - launchpad.acme.com

persistence:
  enabled: true
  size: 5Gi
```

Install with custom values:

```bash
helm install launchpad ./chart -n launchpad -f my-values.yaml
```

## Health Checks

The chart configures both liveness and readiness probes:

- **Liveness Probe**: Checks `/api/config` every 30 seconds after a 10-second initial delay. Restarts the container if 3 consecutive checks fail.

- **Readiness Probe**: Checks `/api/config` every 10 seconds after a 5-second initial delay. Removes the pod from service if 3 consecutive checks fail.

## Container Sessions

Container sessions (launching applications as Kubernetes pods) require:

1. RBAC permissions for pod management (created by default)
2. The VNC sidecar image available
3. Network connectivity between Launchpad and session pods

Configure the session namespace if different from the release namespace:

```bash
helm install launchpad ./chart \
  --set launchpad.kubernetes.namespace=launchpad-sessions
```

## Upgrading

```bash
helm upgrade launchpad ./chart -n launchpad -f my-values.yaml
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n launchpad
kubectl describe pod -n launchpad -l app.kubernetes.io/name=launchpad
```

### View Logs

```bash
kubectl logs -n launchpad -l app.kubernetes.io/name=launchpad
```

### Check Configuration

```bash
kubectl get configmap -n launchpad -l app.kubernetes.io/name=launchpad -o yaml
```
