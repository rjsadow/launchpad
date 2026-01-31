# Autoscaling and Backpressure

This document describes Launchpad's autoscaling policies, session queueing behavior,
and backpressure mechanisms for handling high load.

## Overview

Launchpad implements a multi-layered approach to handle load:

1. **Session Queueing** - Limits concurrent pod creations to prevent API overload
2. **Backpressure** - Returns HTTP 429 when the system is at capacity
3. **Horizontal Pod Autoscaler (HPA)** - Scales Launchpad instances based on demand

## Session Queueing

### How It Works

When a session creation request arrives:

1. Check if queue is full → Return `429 Too Many Requests` immediately
2. Join the queue (increment counter)
3. Acquire semaphore slot (may block if at max concurrent)
4. Create the Kubernetes pod
5. Release semaphore slot and leave queue

```text
Request → Queue Full? ─Yes→ 429 Too Many Requests
              │
              No
              ↓
         Join Queue
              ↓
         Wait for Semaphore ←─┐
              │               │
              ↓               │
         Create Pod           │
              │               │
              ↓               │
         Release Semaphore ───┘
              │
              ↓
         Leave Queue
              │
              ↓
         Return Session
```

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `LAUNCHPAD_MAX_CONCURRENT_SESSIONS` | 10 | Maximum simultaneous pod creations |
| `LAUNCHPAD_MAX_QUEUED_SESSIONS` | 50 | Maximum requests waiting in queue |

### Tuning Guidelines

- **Max Concurrent Sessions**: Start with 10. Increase if your Kubernetes API server
  can handle more concurrent requests and you have sufficient node capacity.

- **Max Queued Sessions**: Set based on expected burst traffic. A larger queue
  absorbs traffic spikes but increases latency. Set to 0 to disable queueing
  (immediate rejection).

## Backpressure

When the queue is full, Launchpad returns HTTP 429 (Too Many Requests):

```json
{
  "error": "too many session requests: server at capacity"
}
```

### Client Handling

Clients should implement exponential backoff when receiving 429:

```javascript
async function createSessionWithRetry(appId, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    const response = await fetch('/api/sessions', {
      method: 'POST',
      body: JSON.stringify({ app_id: appId }),
    });

    if (response.status === 429) {
      const delay = Math.pow(2, i) * 1000; // 1s, 2s, 4s
      await new Promise(resolve => setTimeout(resolve, delay));
      continue;
    }

    return response.json();
  }
  throw new Error('Server at capacity, please try again later');
}
```

## Horizontal Pod Autoscaler (HPA)

For production deployments, configure HPA to scale Launchpad based on load.

### Basic HPA Configuration

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: launchpad
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: launchpad
  minReplicas: 2
  maxReplicas: 10
  metrics:
    # Scale based on CPU utilization
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70

    # Scale based on memory utilization
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### Advanced HPA with Custom Metrics

For more precise scaling, use custom metrics based on queue depth:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: launchpad
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: launchpad
  minReplicas: 2
  maxReplicas: 20
  metrics:
    # Scale when queue utilization exceeds 50%
    - type: Pods
      pods:
        metric:
          name: launchpad_queue_utilization
        target:
          type: AverageValue
          averageValue: "50"

  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300  # Wait 5 min before scaling down
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0    # Scale up immediately
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
        - type: Pods
          value: 4
          periodSeconds: 15
      selectPolicy: Max
```

### Exposing Metrics for HPA

To use custom metrics, expose them via a `/metrics` endpoint (Prometheus format):

```text
# HELP launchpad_sessions_queued Current number of session requests in queue
# TYPE launchpad_sessions_queued gauge
launchpad_sessions_queued 5

# HELP launchpad_sessions_concurrent Current number of concurrent pod creations
# TYPE launchpad_sessions_concurrent gauge
launchpad_sessions_concurrent 8

# HELP launchpad_queue_utilization Percentage of queue capacity used
# TYPE launchpad_queue_utilization gauge
launchpad_queue_utilization 10
```

## Session Pod Resource Limits

Each session pod has the following default resource limits:

| Container | CPU Request | CPU Limit | Memory Request | Memory Limit |
|-----------|------------|-----------|----------------|--------------|
| VNC Sidecar | 500m | 500m | 512Mi | 512Mi |
| Application | 500m | 2 | 512Mi | 2Gi |

### Kubernetes Resource Quotas

For multi-tenant deployments, use ResourceQuotas to limit total session resources:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: launchpad-sessions
  namespace: launchpad
spec:
  hard:
    # Limit total pods
    pods: "100"

    # Limit total CPU
    requests.cpu: "50"
    limits.cpu: "100"

    # Limit total memory
    requests.memory: "50Gi"
    limits.memory: "100Gi"
```

## Load Testing

Before production deployment, test your configuration:

```bash
# Install hey (HTTP load generator)
go install github.com/rakyll/hey@latest

# Test session creation endpoint
hey -n 1000 -c 50 -m POST \
  -H "Content-Type: application/json" \
  -d '{"app_id":"test-app","user_id":"load-test"}' \
  http://localhost:8080/api/sessions
```

Monitor:
- 429 response rate
- Average latency
- Pod creation success rate
- Kubernetes API server load

## Recommended Production Settings

| Deployment Size | Max Concurrent | Max Queued | HPA Min | HPA Max |
|-----------------|----------------|------------|---------|---------|
| Small (<100 users) | 5 | 20 | 1 | 3 |
| Medium (100-500) | 10 | 50 | 2 | 5 |
| Large (500-2000) | 20 | 100 | 3 | 10 |
| Enterprise (2000+) | 50 | 200 | 5 | 20 |

Adjust based on:
- Kubernetes cluster capacity
- Session pod resource requirements
- Acceptable queue latency
- Budget constraints
