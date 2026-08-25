FROM golang:1.25-alpine AS builder

WORKDIR /app

# ВАЖНО: убираем GOTOOLCHAIN=local — теперь он не нужен
# Go 1.25 поддерживает все свежие пакеты

# Копируем go.mod и go.sum
COPY go.mod go.sum ./

# Скачиваем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -p 2 -ldflags="-w -s" -o main ./cmd/api

# Stage 2: Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./main"]