SHELL := /bin/bash

.PHONY: help run run-worker test tidy fmt vet migrate-up migrate-down compose-up compose-down

help:
	@echo "make run           - run API locally"
	@echo "make run-worker    - run outbox worker locally"
	@echo "make test          - run go tests"
	@echo "make tidy          - go mod tidy"
	@echo "make fmt           - format go files"
	@echo "make vet           - go vet"
	@echo "make migrate-up    - run SQL migrations up using docker compose migrate service"
	@echo "make migrate-down  - run SQL migrations down using docker compose migrate service"
	@echo "make compose-up    - start backend + postgres"
	@echo "make compose-down  - stop backend + postgres"

run:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -w ./cmd ./internal ./test

vet:
	go vet ./...

migrate-up:
	docker compose run --rm migrate up

migrate-down:
	docker compose run --rm migrate down 1

compose-up:
	docker compose up --build -d postgres api

compose-down:
	docker compose down
