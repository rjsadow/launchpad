.PHONY: all build clean dev frontend backend migrate-up migrate-down migrate-status

all: build

# Build the frontend
frontend:
	cd web && npm install && npm run build

# Build the Go binary (requires frontend to be built first)
backend: frontend
	go build -o launchpad .

# Build everything
build: backend

# Run development server (frontend only)
dev:
	cd web && npm run dev

# Clean build artifacts
clean:
	rm -rf web/dist web/node_modules launchpad

# Run the production server
run: build
	./launchpad

# Database migrations
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status
