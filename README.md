# Dorm Bot - Telegram Verification Bot

Бот для проверки жителей общажи в Telegram.

## Требования

- Go 1.21+
- PostgreSQL 16
- Docker & Docker Compose

## Установка

### Локально

```bash
# Клонируем репо
git clone <repo_url>
cd dorm-bot

# Копируем .env
cp .env.example .env

# Редактируем .env с твоими значениями
nano .env

# Загружаем зависимости
go mod download

# Запускаем бота
go run main.go
