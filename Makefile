.PHONY: run db-up db-down migrate-up migrate-down status dev-setup

DATABASE_URL=postgres://urlshortener:password@localhost:5432/urlshortener?sslmode=disable

run:
	go run cmd/main.go

db-up:
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5

db-down:
	docker-compose down

migrate-up:
	migrate -database "$(DATABASE_URL)" -path migrations up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path migrations down 1

status:
	docker-compose ps

dev-setup: db-up
	@echo "Setting up development environment..."
	@sleep 3
	$(MAKE) migrate-up