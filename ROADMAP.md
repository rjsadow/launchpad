# Launchpad Roadmap

A centralized application launcher for large organizations.

## Vision

Provide users a single, reliable portal to launch dozens or hundreds of
custom applications. Simple, effective, reliable, secure.

## MVP (Phase 1)

### Core Features

- [x] Static landing page with branding
- [x] Application grid/list view displaying configured apps
- [x] Click-to-launch functionality (opens apps in new tab)
- [x] Search/filter applications by name
- [x] JSON-based application configuration
- [x] Responsive design (desktop + mobile)

### Tech Stack (Recommended)

- **Frontend:** React + TypeScript
- **Styling:** Tailwind CSS (using brand colors: #398D9B, #4AB7C3)
- **Build:** Vite
- **Deployment:** Static hosting (GitHub Pages, Netlify, or self-hosted)

### Data Model (MVP)

```json
{
  "applications": [
    {
      "id": "app-1",
      "name": "Application Name",
      "description": "Brief description",
      "url": "https://app.example.com",
      "icon": "path/to/icon.png",
      "category": "Development"
    }
  ]
}
```

---

## Phase 2: Enhanced UX

- [x] Categories/folders for application grouping
- [x] User preferences (favorites, recent apps)
- [x] Dark mode toggle
- [ ] Keyboard navigation and shortcuts
- [ ] Application health status indicators

---

## Phase 3: Authentication & Personalization

- [ ] SSO integration (SAML/OIDC)
- [ ] Role-based application visibility
- [ ] User-specific favorites stored server-side
- [ ] Admin panel for managing applications

---

## Phase 4: Enterprise Features

- [x] Application usage analytics
- [x] Custom branding per tenant/department
- [x] API for programmatic app management
- [x] Audit logging
- [ ] High availability deployment guide

### Additional Enterprise Features (Implemented)

- [x] Kubernetes Pod Orchestration for container apps
- [x] Session management with VNC streaming
- [x] Centralized configuration management
- [x] WebSocket support for real-time updates

---

## Milestones

See [docs/MILESTONES.md](docs/MILESTONES.md) for GitHub milestone definitions
and issue mapping.

| Milestone | Phases | Status |
|-----------|--------|--------|
| MVP | Phase 1 | Complete |
| Beta | Phase 2 + 3 | In Progress |
| GA | Phase 4 | In Progress |

---

## Out of Scope (for now)

- Application hosting (Launchpad only links to apps)
- User management (delegated to identity provider)
- Application monitoring beyond simple health checks
