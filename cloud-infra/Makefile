.PHONY: fmt test race vet build dev-up dev-down migrate seed backup restore

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build ./cmd/control-api ./cmd/device-gateway ./cmd/worker ./cmd/adminctl ./cmd/dev-seed ./cmd/platform

dev-up:
	docker compose up --build

dev-down:
	docker compose down

migrate:
	go run ./cmd/adminctl migrate

seed:
	go run ./cmd/dev-seed

backup:
	docker compose --profile maintenance run --rm backup

restore:
	docker compose --profile maintenance run --rm restore
