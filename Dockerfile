FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG COMMIT
ARG BUILD_TIME

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -ldflags="-X 'main.version=$VERSION' -X 'main.commit=$COMMIT' -X 'main.buildTime=$BUILD_TIME'" -o goapps-server github.com/topi314/goapps

FROM alpine

COPY --from=build /build/goapps-server /bin/goapps

EXPOSE 80

ENTRYPOINT ["/bin/goapps"]

CMD ["-config", "/var/lib/goapps/goapps.yml"]
