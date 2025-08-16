FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY main.go .
COPY go.mod .

RUN go build -o frontend main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/frontend .

EXPOSE 3000

CMD ["./frontend"]