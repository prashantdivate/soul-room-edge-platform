FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.work ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/control-api ./cmd/control-api && \
    CGO_ENABLED=0 go build -o /out/device-gateway ./cmd/device-gateway && \
    CGO_ENABLED=0 go build -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -o /out/dev-seed ./cmd/dev-seed && \
    CGO_ENABLED=0 go build -o /out/platform ./cmd/platform

FROM build AS test
RUN go test ./...

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/control-api /control-api
COPY --from=build /out/device-gateway /device-gateway
COPY --from=build /out/worker /worker
COPY --from=build /out/dev-seed /dev-seed
COPY --from=build /out/platform /platform
USER nonroot:nonroot
CMD ["/platform"]
