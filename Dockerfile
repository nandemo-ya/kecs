FROM --platform=$BUILDPLATFORM golang:1.24.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS=linux
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -o controlplane ./cmd/controlplane

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:nonroot
COPY --from=builder /app/controlplane /controlplane

ENV PORT=8080
ENV ADMIN_PORT=8081

EXPOSE 8080 8081
ENTRYPOINT ["/controlplane", "server"]
