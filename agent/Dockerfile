FROM golang:1.22 AS source
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

FROM source AS test
RUN go test ./...

FROM source AS embedded-linux-arm64-build
RUN mkdir -p /out && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /out/edge-agent ./cmd/edge-agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /out/edge-agentctl ./cmd/edge-agentctl

FROM scratch AS embedded-linux-arm64
COPY --from=embedded-linux-arm64-build /out/edge-agent /edge-agent
COPY --from=embedded-linux-arm64-build /out/edge-agentctl /edge-agentctl
COPY configs/embedded-linux.yaml /config.yaml
COPY packaging/systemd/edge-agent.service /edge-agent.service

FROM source AS embedded-linux-armv7-build
RUN mkdir -p /out && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags="-s -w" -o /out/edge-agent ./cmd/edge-agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags="-s -w" -o /out/edge-agentctl ./cmd/edge-agentctl

FROM scratch AS embedded-linux-armv7
COPY --from=embedded-linux-armv7-build /out/edge-agent /edge-agent
COPY --from=embedded-linux-armv7-build /out/edge-agentctl /edge-agentctl
COPY configs/embedded-linux.yaml /config.yaml
COPY packaging/systemd/edge-agent.service /edge-agent.service
