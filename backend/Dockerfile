FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w -extldflags '-static'" -o /power-twin-backend ./cmd/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /power-twin-backend /usr/local/bin/power-twin-backend
COPY --from=builder /app/config.json /etc/power-twin/config.json

EXPOSE 8080 61850

ENTRYPOINT ["power-twin-backend"]
CMD ["-config", "/etc/power-twin/config.json"]
