.PHONY: run db-up db-down redis-up redis-down services-up services-down migrate-up migrate-down status dev-setup test redis-cli

DATABASE_URL=postgres://urlshortener:password@localhost:5432/urlshortener?sslmode=disable
REDIS_URL=redis://localhost:6379

run:
	go run cmd/main.go

# Database commands
db-up:
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5

db-down:
	docker-compose stop postgres

# Redis commands
redis-up:
	docker-compose up -d redis
	@echo "Waiting for Redis to be ready..."
	@sleep 3

redis-down:
	docker-compose stop redis

redis-cli:
	docker exec -it urlshortener-cache redis-cli

# All services
services-up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

services-down:
	docker-compose down

# Migrations
migrate-up:
	migrate -database "$(DATABASE_URL)" -path migrations up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path migrations down 1

# Development
dev-setup: services-up
	@echo "Setting up development environment..."
	@sleep 3
	$(MAKE) migrate-up
	@echo "Development environment ready!"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"
	@echo "Redis Commander: http://localhost:8081"

# Testing
test:
	go test -v ./...

test-coverage:
	go test -v -cover ./...

# Status and monitoring
status:
	docker-compose ps

logs:
	docker-compose logs -f

logs-app:
	docker-compose logs -f app

# Clean up
clean:
	docker-compose down -v
	rm -rf tmp/

# Build
build:
	go build -o bin/urlshortener cmd/main.go

# Run with hot reload (requires air)
dev:
	air