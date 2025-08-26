.PHONY: run db-up db-down

run:
	go run cmd/main.go

db-up:
	docker-compose up -d postgres

db-down:
	docker-compose down

status:
	docker-compose ps