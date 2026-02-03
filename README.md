# Dorm Verification Bot

Телеграм‑бот для автоматической модерации заявок в чат общежития.  
Бот сам пишет пользователю после заявки в группу, собирает ФИО и документы, пересылает анкету в чат админов и добавляет пользователя, если админы одобряют.

## Возможности

- Авто‑обработка join requests в группу общежития.
- Диалог с пользователем в ЛС:
  - запрос ФИО;
  - сбор фото/кружков с документами;
  - кнопка «Готово, отправить админам».
- Пересылка ВСЕХ сообщений с медиа + ФИО и ссылкой на профиль в админский чат (без хранения файлов у бота).
- Админские кнопки «✅ Одобрить» / «❌ Отклонить».
- Белый/чёрный списки пользователей (PostgreSQL):
  - whitelist → автоодобрение заявок;
  - blacklist → автодеклайн заявок;
  - команда `/status` для просмотра статуса.
- Команды управления для админов.
- Docker + docker‑compose, healthcheck для БД, systemd‑юнит для автозапуска.

---

## Стек

- Go 1.21
- Telegram Bot API (`github.com/go-telegram-bot-api/telegram-bot-api/v5`) [web:25]
- PostgreSQL 16
- Docker, docker compose [web:158]
- systemd (для запуска на VDS)

---

## Архитектура

```text
dorm-bot/
├── cmd/
│   └── bot/
│       └── main.go          # входная точка приложения
├── internal/
│   ├── config/
│   │   └── config.go        # загрузка конфигурации из env
│   ├── db/
│   │   ├── db.go            # подключение к PostgreSQL
│   │   └── migrations.sql   # схема БД (users/whitelist/blacklist)
│   ├── models/
│   │   └── models.go        # Store: работа с БД, статусы/списки
│   └── bot/
│       ├── state.go         # состояния диалога и сессии
│       └── handlers.go      # логика бота: join requests, диалоги, команды
├── Dockerfile               # multi-stage сборка Go‑бинарника
├── docker-compose.yml       # bot + postgres + healthchecks
├── go.mod
└── .env.example             # пример конфигурации
```

---

## База данных

Миграции лежат в `internal/db/migrations.sql` и применяются автоматически при старте.

Создаются таблицы:

- `users`:
    - `telegram_id BIGINT UNIQUE`
    - `full_name TEXT`
    - `username TEXT`
    - `status TEXT` (`pending`, `approved`, `rejected`, `blocked`, `unverified`)
- `whitelist` (`telegram_id BIGINT UNIQUE`)
- `blacklist` (`telegram_id BIGINT UNIQUE`, `reason TEXT`)

---

## Конфигурация

Настройки читаются из `.env` и переменных окружения.

Пример: `.env.example`

```env
BOT_TOKEN=your_telegram_bot_token
ADMIN_CHAT_ID=123456789        # id админа или чата админов
GROUP_ID=-1001234567890        # id группы общежития (supergroup)
DATABASE_URL=postgres://dorm:password@postgres:5432/dorm_bot?sslmode=disable
```

Обязательные переменные:

- `BOT_TOKEN` – токен бота от BotFather.
- `ADMIN_CHAT_ID` – tg_id админа (если один) или чата админов.
- `GROUP_ID` – id группы общаги (в формате `-100...`).
- `DATABASE_URL` – строка подключения к Postgres.

---

## Сборка и запуск локально

Требуется Docker и docker compose.

1. Скопировать .env:
```bash
cp .env.example .env
# отредактировать значения
```
2. Собрать и поднять контейнеры:

```bash
docker compose build --no-cache
docker compose up -d
```
3. Проверить:

```bash
docker compose ps
docker compose logs -f bot
```

---

## Деплой на сервер (VDS)

1. Подключиться по SSH:

```bash
ssh root@YOUR_SERVER_IP
```

2. Установить Docker + Compose (если нет):

```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# проверить
docker --version
docker compose version
```

3. Клонировать репозиторий:

```bash
mkdir -p /opt/dorm-bot
cd /opt/dorm-bot

git clone <URL_репозитория> .
cp .env.example .env
nano .env   # указать BOT_TOKEN, ADMIN_CHAT_ID, GROUP_ID
```

4. Запустить:

```bash
docker compose build --no-cache
docker compose up -d
docker compose logs -f bot
```

5. (Опционально) Автостарт через systemd:

Создать `/etc/systemd/system/dorm-bot.service`:

```ini
[Unit]
Description=Dorm Telegram bot (docker compose)
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
WorkingDirectory=/opt/dorm-bot
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
RemainAfterExit=yes
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Активировать:

```bash
systemctl daemon-reload
systemctl enable dorm-bot
systemctl start dorm-bot
systemctl status dorm-bot
```

Postgres имеет healthcheck (`pg_isready`), бот стартует только после того, как БД готова.

---

## Логика работы бота

### Поток для пользователя

1. Пользователь подаёт заявку на вступление в группу.
2. Бот получает `chat_join_request` и:
    - если пользователь в whitelist → автоматически approve;
    - если в blacklist → автоматически decline;
    - иначе пишет пользователю в ЛС.
3. Диалог:
    - просит ФИО;
    - просит отправить подтверждающие документы (фото/кружки).
4. Пользователь шлёт все медиа, затем жмёт кнопку  
   **«✅ Готово, отправить админам»**.
5. Бот:
    - пересылает в чат админов шапку анкеты (ФИО, ссылка на профиль, ID);
    - пересылает все собранные сообщения с медиа;
    - добавляет под анкетой кнопки «✅ Одобрить» / «❌ Отклонить».

### Поток для админа

В админском чате под анкетой:

- **«✅ Одобрить»**:
    - approve join request;
    - добавить пользователя в whitelist;
    - статус `approved`;
    - отправить приветственное сообщение пользователю.

- **«❌ Отклонить»**:
    - либо сразу отклонить join request;
    - либо (если включено) запросить причину в ЛС админа и отправить её пользователю.

---

## Команды бота

Команды обрабатываются в ЛС с ботом.

### Для всех пользователей

- `/start` – краткое описание того, что бот делает, и как начать (подать заявку в группу).
- `/help` – пояснение процесса проверки.

### Для админов

(Проверка через `isAdmin` по `ADMIN_CHAT_ID`.)

- `/start` – описание админских возможностей и список команд.
- `/help` – детальнее про логику, списки и кнопки.
- `/status <telegram_id>` – показать, где находится пользователь:
    - `whitelisted`
    - `blacklisted`
    - `unverified` (ни в одном списке).

Работа со списками:

- `/whitelist_add <telegram_id>` – добавить в белый список (будет автоодобрение).
- `/whitelist_remove <telegram_id>` – удалить из белого списка.
- `/blacklist_add <telegram_id> [причина]` – добавить в чёрный список (заявки будут сразу отклоняться).
- `/blacklist_remove <telegram_id>` – удалить из чёрного списка.

---

## Разработка

Запуск без Docker (на локальной машине с установленным Postgres):

```bash
export BOT_TOKEN=...
export ADMIN_CHAT_ID=...
export GROUP_ID=...
export DATABASE_URL=postgres://dorm:password@localhost:5432/dorm_bot?sslmode=disable

go run ./cmd/bot
```

Полезные команды:

```bash
go fmt ./...
go vet ./...
go test ./...
```

---

## TODO / Идеи для улучшений

- Более гибкая система прав админов (список admin_ids вместо одного `ADMIN_CHAT_ID`).
- История анкет и действий админов.
- Web‑панель или бота‑админа для просмотра статистики.
- Локализация сообщений и конфиг текста.