include .env
export

.PHONY: run build test migrate-up migrate-down docker-up docker-down monitoring-up monitoring-down load

run:
	go run ./cmd/server

build:
	go build -o server ./cmd/server

test:
	go test ./...

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

docker-up:
	docker compose up -d db redis

docker-down:
	docker compose down

monitoring-up:
	docker compose up -d prometheus grafana

monitoring-down:
	docker compose stop prometheus grafana

load:
	./monitoring/loadtest.sh
