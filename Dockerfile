FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY src/* .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o backup-scheduler .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata rclone curl

COPY --from=builder /app/backup-scheduler /usr/local/bin/backup-scheduler
RUN chmod +x /usr/local/bin/backup-scheduler

WORKDIR /app

ENTRYPOINT ["/usr/local/bin/backup-scheduler"]
CMD ["--help"]
