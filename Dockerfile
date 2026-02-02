# build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk --no-cache add git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot

# final image
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bot /app/bot
COPY internal/db/migrations.sql /app/internal/db/migrations.sql
COPY .env /app/.env

CMD ["/app/bot"]

