FROM golang:1.22 AS source
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

FROM source AS test
RUN go test ./...

FROM source AS raspberry-pi-build
RUN mkdir -p /out && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /out/edge-agent ./cmd/edge-agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /out/edge-agentctl ./cmd/edge-agentctl

FROM scratch AS raspberry-pi
COPY --from=raspberry-pi-build /out/edge-agent /edge-agent
COPY --from=raspberry-pi-build /out/edge-agentctl /edge-agentctl
COPY configs/raspberry-pi.yaml /config.yaml
COPY packaging/systemd/edge-agent.service /edge-agent.service
