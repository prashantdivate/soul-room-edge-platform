.PHONY: fmt test race build vet clean bundle bundle-arm64 bundle-armv7 bundle-amd64

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

bundle: bundle-amd64 bundle-arm64 bundle-armv7

bundle-amd64:
	docker build --target embedded-linux-amd64 --output type=local,dest=dist/embedded-linux-amd64 .

bundle-arm64:
	docker build --target embedded-linux-arm64 --output type=local,dest=dist/embedded-linux-arm64 .

bundle-armv7:
	docker build --target embedded-linux-armv7 --output type=local,dest=dist/embedded-linux-armv7 .

clean:
	go clean
