# Hard Multi-Tenancy

This document defines the multi-tenancy architecture for Launchpad, covering
tenant isolation, routing, role-based access control, and quota management.

## Overview

Hard multi-tenancy ensures complete isolation between tenants at the data,
network, and resource level. Each tenant operates as if they have a dedicated
Launchpad instance while sharing the underlying infrastructure.

### Isolation Guarantees

| Layer       | Isolation Method               | Guarantee                        |
| ----------- | ------------------------------ | -------------------------------- |
| Data        | Tenant ID column + row filters | No cross-tenant data access      |
| Network     | Namespace + NetworkPolicy      | No cross-tenant network traffic  |
| Resources   | ResourceQuota per namespace    | No resource starvation           |
| Compute     | Dedicated node pools (optional)| Physical isolation for compliance|

## Architecture

### Tenant Model

```go
// Tenant represents an organization or isolated environment
type Tenant struct {
    ID          string    `json:"id"`          // Unique tenant identifier (UUID)
    Name        string    `json:"name"`        // Human-readable name
    Slug        string    `json:"slug"`        // URL-safe identifier
    Status      string    `json:"status"`      // active, suspended, deleted
    Plan        string    `json:"plan"`        // free, standard, enterprise
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Database Schema

All tenant-scoped tables include a `tenant_id` column with foreign key
constraint and index:

```sql
-- Tenants table
CREATE TABLE tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    plan TEXT NOT NULL DEFAULT 'free',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);

-- Tenant-scoped applications
ALTER TABLE applications ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE applications ADD CONSTRAINT fk_applications_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);
CREATE INDEX idx_applications_tenant ON applications(tenant_id);

-- Tenant-scoped sessions
ALTER TABLE sessions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE sessions ADD CONSTRAINT fk_sessions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);
CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);

-- Tenant-scoped audit logs
ALTER TABLE audit_log ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE audit_log ADD CONSTRAINT fk_audit_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);
CREATE INDEX idx_audit_tenant ON audit_log(tenant_id);

-- Tenant-scoped analytics
ALTER TABLE analytics ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE analytics ADD CONSTRAINT fk_analytics_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);
CREATE INDEX idx_analytics_tenant ON analytics(tenant_id);
```

### Migration Strategy

For existing single-tenant deployments:

```sql
-- 1. Create default tenant
INSERT INTO tenants (id, name, slug, status, plan)
VALUES ('default', 'Default Tenant', 'default', 'active', 'enterprise');

-- 2. Update existing records to belong to default tenant
UPDATE applications SET tenant_id = 'default' WHERE tenant_id IS NULL;
UPDATE sessions SET tenant_id = 'default' WHERE tenant_id IS NULL;
UPDATE audit_log SET tenant_id = 'default' WHERE tenant_id IS NULL;
UPDATE analytics SET tenant_id = 'default' WHERE tenant_id IS NULL;
```

## Tenant Routing

### Request Flow

```text
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐     ┌─────────┐
│   Request   │────▶│   Ingress    │────▶│ Tenant Resolver │────▶│ Handler │
│             │     │  (hostname)  │     │   Middleware    │     │         │
└─────────────┘     └──────────────┘     └─────────────────┘     └─────────┘
                           │                      │
                           │                      ▼
                           │              ┌───────────────┐
                           │              │ Tenant Context│
                           │              │  (in request) │
                           │              └───────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ tenant.slug  │
                    │   from URL   │
                    └──────────────┘
```

### Tenant Resolution Methods

Launchpad supports multiple tenant resolution strategies:

| Method       | Example                          | Use Case                   |
| ------------ | -------------------------------- | -------------------------- |
| Subdomain    | `acme.launchpad.example.com`     | SaaS deployments           |
| Path prefix  | `launchpad.example.com/t/acme`   | Single-domain deployments  |
| Header       | `X-Tenant-ID: acme`              | API/service-to-service     |
| JWT claim    | `tenant_id` in token             | SSO-integrated deployments |

### Middleware Implementation

```go
// TenantMiddleware extracts and validates tenant context
func TenantMiddleware(resolver TenantResolver) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID, err := resolver.Resolve(r)
            if err != nil {
                http.Error(w, "tenant not found", http.StatusNotFound)
                return
            }

            tenant, err := db.GetTenant(tenantID)
            if err != nil || tenant == nil {
                http.Error(w, "tenant not found", http.StatusNotFound)
                return
            }

            if tenant.Status != "active" {
                http.Error(w, "tenant suspended", http.StatusForbidden)
                return
            }

            // Add tenant to request context
            ctx := context.WithValue(r.Context(), TenantContextKey, tenant)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// TenantFromContext retrieves the tenant from request context
func TenantFromContext(ctx context.Context) *Tenant {
    tenant, _ := ctx.Value(TenantContextKey).(*Tenant)
    return tenant
}
```

### Ingress Configuration

For subdomain-based routing:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: launchpad-tenants
  namespace: launchpad
  annotations:
    nginx.ingress.kubernetes.io/use-regex: "true"
spec:
  rules:
    - host: "*.launchpad.example.com"
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: launchpad
                port:
                  number: 8080
```

For path-based routing:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: launchpad-tenants
  namespace: launchpad
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$2
spec:
  rules:
    - host: launchpad.example.com
      http:
        paths:
          - path: /t/([^/]+)(/|$)(.*)
            pathType: ImplementationSpecific
            backend:
              service:
                name: launchpad
                port:
                  number: 8080
```

## Tenant-Scoped RBAC

### Role Model

```sql
-- Roles defined per tenant
CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    permissions TEXT NOT NULL,  -- JSON array of permission strings
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);

-- User-role assignments
CREATE TABLE user_roles (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (role_id) REFERENCES roles(id),
    UNIQUE(tenant_id, user_id, role_id)
);

CREATE INDEX idx_user_roles_tenant ON user_roles(tenant_id);
CREATE INDEX idx_user_roles_user ON user_roles(user_id);
```

### Permission Model

Permissions follow the format `resource:action`:

| Permission             | Description                        |
| ---------------------- | ---------------------------------- |
| `apps:read`            | View applications                  |
| `apps:write`           | Create/update applications         |
| `apps:delete`          | Delete applications                |
| `apps:launch`          | Launch applications                |
| `sessions:read`        | View own sessions                  |
| `sessions:read_all`    | View all tenant sessions           |
| `sessions:terminate`   | Terminate sessions                 |
| `users:read`           | View tenant users                  |
| `users:write`          | Manage tenant users                |
| `roles:read`           | View tenant roles                  |
| `roles:write`          | Manage tenant roles                |
| `audit:read`           | View audit logs                    |
| `analytics:read`       | View analytics                     |
| `tenant:admin`         | Full tenant administration         |

### Default Roles

Each tenant is created with these default roles:

```json
{
  "roles": [
    {
      "name": "admin",
      "description": "Full tenant administrator",
      "permissions": ["tenant:admin"]
    },
    {
      "name": "app-manager",
      "description": "Manage applications",
      "permissions": ["apps:read", "apps:write", "apps:delete", "analytics:read"]
    },
    {
      "name": "user",
      "description": "Standard user",
      "permissions": ["apps:read", "apps:launch", "sessions:read"]
    },
    {
      "name": "viewer",
      "description": "Read-only access",
      "permissions": ["apps:read"]
    }
  ]
}
```

### RBAC Middleware

```go
// RequirePermission middleware checks user has required permission
func RequirePermission(permission string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenant := TenantFromContext(r.Context())
            user := UserFromContext(r.Context())

            if tenant == nil || user == nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            hasPermission, err := checkPermission(tenant.ID, user.ID, permission)
            if err != nil {
                http.Error(w, "internal error", http.StatusInternalServerError)
                return
            }

            if !hasPermission {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func checkPermission(tenantID, userID, permission string) (bool, error) {
    // Check if user has direct permission or tenant:admin
    permissions, err := db.GetUserPermissions(tenantID, userID)
    if err != nil {
        return false, err
    }

    for _, p := range permissions {
        if p == permission || p == "tenant:admin" {
            return true, nil
        }
    }

    return false, nil
}
```

### API Endpoint Authorization

| Endpoint               | Method | Required Permission      |
| ---------------------- | ------ | ------------------------ |
| `/api/apps`            | GET    | `apps:read`              |
| `/api/apps`            | POST   | `apps:write`             |
| `/api/apps/{id}`       | GET    | `apps:read`              |
| `/api/apps/{id}`       | PUT    | `apps:write`             |
| `/api/apps/{id}`       | DELETE | `apps:delete`            |
| `/api/apps/{id}/launch`| POST   | `apps:launch`            |
| `/api/sessions`        | GET    | `sessions:read`          |
| `/api/sessions/{id}`   | DELETE | `sessions:terminate`     |
| `/api/users`           | GET    | `users:read`             |
| `/api/users`           | POST   | `users:write`            |
| `/api/roles`           | GET    | `roles:read`             |
| `/api/roles`           | POST   | `roles:write`            |
| `/api/audit`           | GET    | `audit:read`             |
| `/api/analytics`       | GET    | `analytics:read`         |

## Tenant-Level Quotas

### Quota Model

```sql
CREATE TABLE tenant_quotas (
    tenant_id TEXT PRIMARY KEY,
    max_applications INTEGER NOT NULL DEFAULT 50,
    max_active_sessions INTEGER NOT NULL DEFAULT 10,
    max_users INTEGER NOT NULL DEFAULT 100,
    max_storage_gb INTEGER NOT NULL DEFAULT 10,
    cpu_limit_cores REAL NOT NULL DEFAULT 4.0,
    memory_limit_gb REAL NOT NULL DEFAULT 8.0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE tenant_usage (
    tenant_id TEXT PRIMARY KEY,
    application_count INTEGER NOT NULL DEFAULT 0,
    active_session_count INTEGER NOT NULL DEFAULT 0,
    user_count INTEGER NOT NULL DEFAULT 0,
    storage_used_gb REAL NOT NULL DEFAULT 0,
    cpu_used_cores REAL NOT NULL DEFAULT 0,
    memory_used_gb REAL NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);
```

### Quota Limits by Plan

| Resource            | Free    | Standard | Enterprise |
| ------------------- | ------- | -------- | ---------- |
| Applications        | 10      | 50       | Unlimited  |
| Active Sessions     | 2       | 10       | 100        |
| Users               | 5       | 100      | Unlimited  |
| Storage (GB)        | 1       | 10       | 100        |
| CPU (cores)         | 1       | 4        | 16         |
| Memory (GB)         | 2       | 8        | 32         |

### Quota Enforcement

```go
// QuotaEnforcer checks and enforces tenant quotas
type QuotaEnforcer struct {
    db *db.DB
}

// CheckQuota verifies tenant has capacity for the requested resource
func (q *QuotaEnforcer) CheckQuota(tenantID, resource string, requested int) error {
    quota, err := q.db.GetTenantQuota(tenantID)
    if err != nil {
        return fmt.Errorf("failed to get quota: %w", err)
    }

    usage, err := q.db.GetTenantUsage(tenantID)
    if err != nil {
        return fmt.Errorf("failed to get usage: %w", err)
    }

    switch resource {
    case "applications":
        if usage.ApplicationCount+requested > quota.MaxApplications {
            return ErrQuotaExceeded{Resource: resource, Limit: quota.MaxApplications}
        }
    case "sessions":
        if usage.ActiveSessionCount+requested > quota.MaxActiveSessions {
            return ErrQuotaExceeded{Resource: resource, Limit: quota.MaxActiveSessions}
        }
    case "users":
        if usage.UserCount+requested > quota.MaxUsers {
            return ErrQuotaExceeded{Resource: resource, Limit: quota.MaxUsers}
        }
    }

    return nil
}

// ErrQuotaExceeded indicates a quota limit has been reached
type ErrQuotaExceeded struct {
    Resource string
    Limit    int
}

func (e ErrQuotaExceeded) Error() string {
    return fmt.Sprintf("quota exceeded for %s: limit is %d", e.Resource, e.Limit)
}
```

### Kubernetes Resource Quotas

Each tenant gets a dedicated namespace with ResourceQuota:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: launchpad-tenant-${TENANT_ID}
  labels:
    launchpad.io/tenant: "${TENANT_ID}"
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-quota
  namespace: launchpad-tenant-${TENANT_ID}
spec:
  hard:
    # Compute quotas (vary by plan)
    requests.cpu: "${CPU_REQUEST_LIMIT}"
    requests.memory: "${MEMORY_REQUEST_LIMIT}"
    limits.cpu: "${CPU_LIMIT}"
    limits.memory: "${MEMORY_LIMIT}"
    # Object count quotas
    pods: "${MAX_SESSIONS}"
    persistentvolumeclaims: "${MAX_PVCS}"
    # Storage quotas
    requests.storage: "${STORAGE_LIMIT}"
---
apiVersion: v1
kind: LimitRange
metadata:
  name: tenant-limits
  namespace: launchpad-tenant-${TENANT_ID}
spec:
  limits:
    - type: Pod
      max:
        cpu: "2"
        memory: "4Gi"
      min:
        cpu: "50m"
        memory: "64Mi"
    - type: Container
      default:
        cpu: "500m"
        memory: "512Mi"
      defaultRequest:
        cpu: "100m"
        memory: "128Mi"
```

### Network Isolation

NetworkPolicy ensures tenant session pods cannot communicate across namespaces:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tenant-isolation
  namespace: launchpad-tenant-${TENANT_ID}
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Allow traffic from Launchpad control plane
    - from:
        - namespaceSelector:
            matchLabels:
              app: launchpad
        - podSelector:
            matchLabels:
              app: launchpad
    # Allow traffic within tenant namespace
    - from:
        - podSelector: {}
  egress:
    # Allow DNS resolution
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
    # Allow external internet access (configurable)
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
            except:
              - 10.0.0.0/8
              - 172.16.0.0/12
              - 192.168.0.0/16
```

## Usage Tracking

### Usage Update Triggers

Usage counters are updated via database triggers or application-level hooks:

```sql
-- Trigger to update application count
CREATE TRIGGER update_app_count_insert AFTER INSERT ON applications
BEGIN
    INSERT INTO tenant_usage (tenant_id, application_count, updated_at)
    VALUES (NEW.tenant_id, 1, CURRENT_TIMESTAMP)
    ON CONFLICT(tenant_id) DO UPDATE SET
        application_count = application_count + 1,
        updated_at = CURRENT_TIMESTAMP;
END;

CREATE TRIGGER update_app_count_delete AFTER DELETE ON applications
BEGIN
    UPDATE tenant_usage
    SET application_count = application_count - 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE tenant_id = OLD.tenant_id;
END;
```

### Usage API

```go
// GetTenantUsage returns current resource usage for a tenant
func (db *DB) GetTenantUsage(tenantID string) (*TenantUsage, error) {
    var usage TenantUsage
    err := db.conn.QueryRow(`
        SELECT
            tenant_id,
            application_count,
            active_session_count,
            user_count,
            storage_used_gb,
            cpu_used_cores,
            memory_used_gb,
            updated_at
        FROM tenant_usage
        WHERE tenant_id = ?
    `, tenantID).Scan(
        &usage.TenantID,
        &usage.ApplicationCount,
        &usage.ActiveSessionCount,
        &usage.UserCount,
        &usage.StorageUsedGB,
        &usage.CPUUsedCores,
        &usage.MemoryUsedGB,
        &usage.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return &TenantUsage{TenantID: tenantID}, nil
    }
    return &usage, err
}
```

### Usage Dashboard Endpoint

| Endpoint                    | Method | Response                        |
| --------------------------- | ------ | ------------------------------- |
| `/api/tenant/usage`         | GET    | Current usage vs. quota limits  |
| `/api/tenant/usage/history` | GET    | Historical usage over time      |

Example response:

```json
{
  "tenant_id": "acme-corp",
  "plan": "standard",
  "usage": {
    "applications": { "current": 23, "limit": 50, "percentage": 46 },
    "sessions": { "current": 5, "limit": 10, "percentage": 50 },
    "users": { "current": 47, "limit": 100, "percentage": 47 },
    "storage_gb": { "current": 3.2, "limit": 10, "percentage": 32 },
    "cpu_cores": { "current": 1.5, "limit": 4, "percentage": 38 },
    "memory_gb": { "current": 3.8, "limit": 8, "percentage": 48 }
  }
}
```

## Tenant Lifecycle

### Tenant Provisioning

1. Create tenant record in database
2. Create Kubernetes namespace
3. Apply ResourceQuota and LimitRange
4. Apply NetworkPolicy
5. Create default roles
6. Create admin user with `tenant:admin` role

```bash
# Provisioning script
kubectl create namespace launchpad-tenant-${TENANT_ID}
kubectl label namespace launchpad-tenant-${TENANT_ID} launchpad.io/tenant=${TENANT_ID}
kubectl apply -f tenant-resources.yaml -n launchpad-tenant-${TENANT_ID}
```

### Tenant Suspension

When a tenant is suspended:

1. Update tenant status to `suspended`
2. Terminate all active sessions
3. Block new session creation
4. Maintain read-only access to data
5. Scale down tenant namespace resources

### Tenant Deletion

When a tenant is deleted:

1. Update tenant status to `deleted`
2. Terminate all sessions
3. Delete Kubernetes namespace and all resources
4. Archive or delete tenant data per retention policy
5. Remove from routing

## Configuration

### Environment Variables

```bash
# Multi-tenancy mode
LAUNCHPAD_MULTI_TENANT=true
LAUNCHPAD_TENANT_RESOLVER=subdomain  # subdomain, path, header, jwt

# Default tenant (for backward compatibility)
LAUNCHPAD_DEFAULT_TENANT=default

# Quota defaults
LAUNCHPAD_DEFAULT_MAX_APPS=50
LAUNCHPAD_DEFAULT_MAX_SESSIONS=10
LAUNCHPAD_DEFAULT_MAX_USERS=100
```

### Tenant Configuration File

```json
{
  "multi_tenancy": {
    "enabled": true,
    "resolver": "subdomain",
    "domain_suffix": ".launchpad.example.com",
    "default_tenant": "default"
  },
  "quotas": {
    "plans": {
      "free": {
        "max_applications": 10,
        "max_sessions": 2,
        "max_users": 5,
        "storage_gb": 1,
        "cpu_cores": 1,
        "memory_gb": 2
      },
      "standard": {
        "max_applications": 50,
        "max_sessions": 10,
        "max_users": 100,
        "storage_gb": 10,
        "cpu_cores": 4,
        "memory_gb": 8
      },
      "enterprise": {
        "max_applications": -1,
        "max_sessions": 100,
        "max_users": -1,
        "storage_gb": 100,
        "cpu_cores": 16,
        "memory_gb": 32
      }
    }
  }
}
```

## Related Documentation

- [Data Persistence Strategy](DATA_PERSISTENCE.md) - Database schema details
- [Kubernetes Deployment](KUBERNETES.md) - Namespace and resource configuration
- [Disaster Recovery](DISASTER_RECOVERY.md) - Tenant backup and restore
