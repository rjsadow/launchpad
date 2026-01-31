# Launchpad

A centralized application launcher for large organizations.

<!-- Screenshot placeholder: Add screenshot of the main dashboard here -->
![Launchpad Dashboard](docs/assets/screenshot-dashboard.png)

## The Problem

Large organizations have dozens or hundreds of internal applications scattered across different URLs, wikis, and bookmarks. Employees waste time searching for the right tool, and new hires struggle to discover what's available.

**Launchpad solves this** by providing a single, searchable portal where users can find and launch any application instantly.

## Core Concepts

### Apps

An **App** is any application registered in Launchpad. Apps can be:

- **URL-based**: Opens an external URL in a new tab (e.g., GitHub, Jira, internal tools)
- **Container-based**: Launches a containerized application in Kubernetes with VNC access (e.g., Firefox, LibreOffice, GIMP)

### Sessions

A **Session** is created when a user launches a container-based app. Each session represents a running Kubernetes pod that the user can interact with via WebSocket/VNC streaming. Sessions are isolated per user and automatically cleaned up when terminated.

### Users

A **User** is anyone accessing Launchpad. Users can browse available apps, search by name or category, and launch applications. Future versions will support SSO authentication and personalized favorites.

## Quickstart

### Option 1: Docker (Recommended)

```bash
docker run -p 8080:8080 ghcr.io/rjsadow/launchpad:latest
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/rjsadow/launchpad.git
cd launchpad

# Build and run
make build
./launchpad --port 8080
```

### Option 3: Development Mode

```bash
# Run frontend dev server with hot reload
cd web && npm install && npm run dev
```

### Seed with Sample Apps

```bash
./launchpad --seed examples/apps-with-containers.json
```

## Features

- **Application Grid**: Visual display of all available applications
- **Search**: Quickly find apps by name or description
- **Categories**: Logical grouping of related applications
- **Container Apps**: Launch containerized applications with VNC streaming
- **Dark Mode**: Toggle between light and dark themes
- **Keyboard Shortcuts**: Navigate and launch apps without a mouse
- **Analytics**: Track application usage and popularity
- **Audit Logging**: Record all administrative actions

<!-- GIF placeholder: Add demo GIF showing search and launch flow -->
![Demo](docs/assets/demo.gif)

## Configuration

Apps are configured via JSON. Create an `apps.json` file:

```json
{
  "applications": [
    {
      "id": "github",
      "name": "GitHub",
      "description": "Code hosting platform",
      "url": "https://github.com",
      "icon": "https://github.githubassets.com/favicons/favicon.svg",
      "category": "Development",
      "launch_type": "url"
    },
    {
      "id": "firefox",
      "name": "Firefox Browser",
      "description": "Firefox in a secure container",
      "url": "",
      "icon": "https://mozilla.org/firefox.svg",
      "category": "Browsers",
      "launch_type": "container",
      "container_image": "jlesage/firefox:latest"
    }
  ]
}
```

Seed on startup:

```bash
./launchpad --seed apps.json
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/apps` | GET | List all applications |
| `/api/apps` | POST | Create a new application |
| `/api/apps/{id}` | GET/PUT/DELETE | Manage a specific app |
| `/api/sessions` | GET/POST | List or create sessions |
| `/api/sessions/{id}` | GET/DELETE | Get or terminate a session |
| `/api/analytics/stats` | GET | Get usage statistics |
| `/api/audit` | GET | Get audit logs |

## Deployment

See [docs/deployment.md](docs/deployment.md) for production deployment guides including:

- Docker Compose
- Kubernetes with Helm
- High availability configuration

For container-based apps, see [docs/KUBERNETES.md](docs/KUBERNETES.md).

## Architecture

```
┌─────────────────────────────────────────┐
│           User's Browser                │
│  ┌───────────────────────────────────┐  │
│  │         Launchpad UI              │  │
│  │  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ │  │
│  │  │App 1│ │App 2│ │App 3│ │App N│ │  │
│  │  └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ │  │
│  └─────┼───────┼───────┼───────┼─────┘  │
└────────┼───────┼───────┼───────┼────────┘
         │       │       │       │
         ▼       ▼       ▼       ▼
    ┌────────────────────────────────┐
    │      Go Backend Server         │
    │  (Embedded React Frontend)     │
    └────────────────────────────────┘
              │
              ▼
    ┌────────────────────────────────┐
    │   Kubernetes (for containers)  │
    │   ┌─────────────────────────┐  │
    │   │    Container Sessions   │  │
    │   └─────────────────────────┘  │
    └────────────────────────────────┘
```

## Tech Stack

- **Frontend**: React, TypeScript, Tailwind CSS, Vite
- **Backend**: Go with embedded static files
- **Database**: SQLite
- **Container Orchestration**: Kubernetes (optional, for container apps)

## Contributing

Contributions are welcome! Please see the [ROADMAP.md](ROADMAP.md) for planned features.

## License

MIT
