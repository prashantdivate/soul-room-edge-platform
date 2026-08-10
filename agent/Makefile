.PHONY: fmt test race build vet clean

fmt:
	gofmt -w ./cmd ./internal ./pkg

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build ./cmd/edge-agent ./cmd/edge-agentctl ./cmd/device-simulator

clean:
	go clean
