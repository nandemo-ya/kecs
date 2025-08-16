FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY main.go .
COPY go.mod .

RUN go build -o backend main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/backend .

EXPOSE 8080

CMD ["./backend"]