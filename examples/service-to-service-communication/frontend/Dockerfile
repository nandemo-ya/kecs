FROM golang:1.25.2-bookworm AS builder

WORKDIR /app
COPY main.go .
COPY go.mod .

RUN CGO_ENABLED=1 go build -o frontend main.go

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates wget && rm -rf /var/lib/apt/lists/*

WORKDIR /root/
COPY --from=builder /app/frontend .

EXPOSE 3000

CMD ["./frontend"]