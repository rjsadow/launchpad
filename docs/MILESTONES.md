# GitHub Milestones

This document defines the GitHub milestones for the Launchpad project and maps
ROADMAP phases to release milestones.

## Milestone Definitions

### MVP (Minimum Viable Product)

**Description:** Core functionality for a working application launcher.
Users can view, search, and launch applications from a responsive web interface.

**Target:** Initial public release

**Corresponds to:** ROADMAP Phase 1

**Criteria for completion:**

- Static landing page with branding
- Application grid/list view displaying configured apps
- Click-to-launch functionality (opens apps in new tab)
- Search/filter applications by name
- JSON-based application configuration
- Responsive design (desktop + mobile)
- Basic deployment documentation

### Beta

**Description:** Enhanced user experience with personalization features and
authentication support. Ready for pilot deployments with select users.

**Target:** Limited availability release

**Corresponds to:** ROADMAP Phase 2 + Phase 3

**Criteria for completion:**

- Categories/folders for application grouping
- User preferences (favorites, recent apps)
- Dark mode toggle
- Keyboard navigation and shortcuts
- Application health status indicators
- SSO integration (SAML/OIDC)
- Role-based application visibility
- User-specific favorites stored server-side
- Admin panel for managing applications

### GA (General Availability)

**Description:** Production-ready release with enterprise features including
analytics, multi-tenancy, and high availability support.

**Target:** Production release

**Corresponds to:** ROADMAP Phase 4

**Criteria for completion:**

- Application usage analytics
- Custom branding per tenant/department
- API for programmatic app management
- Audit logging
- High availability deployment guide
- Container app support (Kubernetes pods)
- Comprehensive documentation

## Current Implementation Status

Based on the codebase analysis, here is the current status:

### Completed Features

| Feature | Phase | Status |
|---------|-------|--------|
| React + TypeScript frontend | MVP | Done |
| Go backend with embedded frontend | MVP | Done |
| Application grid/list view | MVP | Done |
| Click-to-launch functionality | MVP | Done |
| Search/filter applications | MVP | Done |
| JSON-based configuration | MVP | Done |
| Responsive design | MVP | Done |
| Categories for app grouping | Beta | Done |
| User preferences (favorites, recent) | Beta | Done |
| Dark mode toggle | Beta | Done |
| Application usage analytics | GA | Done |
| Custom branding per tenant | GA | Done |
| API for app management | GA | Done |
| Audit logging | GA | Done |
| Kubernetes Pod Orchestration | GA | Done |
| Session management (VNC) | GA | Done |
| Centralized configuration | GA | Done |

### Remaining Features

| Feature | Phase | Milestone |
|---------|-------|-----------|
| Keyboard navigation and shortcuts | Phase 2 | Beta |
| Application health status indicators | Phase 2 | Beta |
| SSO integration (SAML/OIDC) | Phase 3 | Beta |
| Role-based application visibility | Phase 3 | Beta |
| User-specific favorites (server-side) | Phase 3 | Beta |
| Admin panel for managing applications | Phase 3 | Beta |
| High availability deployment guide | Phase 4 | GA |

## GitHub Milestone Setup Instructions

Since `gh` CLI authentication is unavailable, create milestones manually:

### 1. Create Milestones

Navigate to: `https://github.com/rjsadow/launchpad/milestones/new`

Create three milestones:

**MVP**

- Title: `MVP`
- Description: `Minimum Viable Product - Core launcher functionality with
  search, filtering, and responsive design. See docs/MILESTONES.md for details.`
- Due date: (set based on project timeline)

**Beta**

- Title: `Beta`
- Description: `Enhanced UX and authentication - Keyboard shortcuts, health
  indicators, SSO, RBAC, and admin panel. See docs/MILESTONES.md for details.`
- Due date: (set based on project timeline)

**GA**

- Title: `GA`
- Description: `General Availability - Production-ready with enterprise
  features, HA documentation, and full API. See docs/MILESTONES.md for details.`
- Due date: (set based on project timeline)

### 2. Issue Assignment Recommendations

When `gh` CLI is available, use these commands to assign issues:

```bash
# Assign MVP issues (if any remain)
gh issue edit <issue-number> --milestone MVP

# Assign Beta issues
gh issue edit <issue-number> --milestone Beta

# Assign GA issues
gh issue edit <issue-number> --milestone GA
```

### 3. Creating New Issues for Remaining Features

For each remaining feature, create a GitHub issue:

**Beta Milestone Issues:**

```bash
# Keyboard navigation
gh issue create --title "Add keyboard navigation and shortcuts" \
  --body "Implement keyboard shortcuts for navigating between apps, searching, and launching. Part of Phase 2 Enhanced UX." \
  --label "enhancement" --milestone "Beta"

# Health status indicators
gh issue create --title "Add application health status indicators" \
  --body "Show visual indicators for application availability/health. Part of Phase 2 Enhanced UX." \
  --label "enhancement" --milestone "Beta"

# SSO integration
gh issue create --title "Implement SSO integration (SAML/OIDC)" \
  --body "Add single sign-on support via SAML 2.0 and OpenID Connect. Part of Phase 3 Authentication." \
  --label "enhancement" --milestone "Beta"

# Role-based visibility
gh issue create --title "Add role-based application visibility" \
  --body "Control which applications users can see based on their roles/groups. Part of Phase 3 Authentication." \
  --label "enhancement" --milestone "Beta"

# Server-side favorites
gh issue create --title "Store user favorites server-side" \
  --body "Persist user favorites in the database (currently client-side only). Part of Phase 3 Personalization." \
  --label "enhancement" --milestone "Beta"

# Admin panel
gh issue create --title "Create admin panel for managing applications" \
  --body "Web UI for administrators to add, edit, and remove applications. Part of Phase 3 Authentication." \
  --label "enhancement" --milestone "Beta"
```

**GA Milestone Issues:**

```bash
# HA deployment guide
gh issue create --title "Add high availability deployment guide" \
  --body "Documentation for deploying Launchpad in HA configuration. Part of Phase 4 Enterprise." \
  --label "documentation" --milestone "GA"
```

## Mapping Existing Issues

Review open issues at `https://github.com/rjsadow/launchpad/issues` and assign
to milestones based on:

- **MVP:** Core functionality bugs or missing MVP features
- **Beta:** UX improvements, auth features, admin features
- **GA:** Enterprise features, performance, scalability, documentation

## Version Tagging Convention

When milestones are complete, tag releases:

- MVP complete: `v1.0.0`
- Beta complete: `v1.1.0` or `v2.0.0-beta.1`
- GA complete: `v2.0.0`
