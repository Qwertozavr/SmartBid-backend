# SmartBid-backend
Система предоставляет пользователям возможность размещать объявления о продаже товаров различных категорий, управлять параметрами публикации и отслеживать состояние активных объявлений. После создания объявления информация передаётся в backend-сервис, где проходит обработку, сохраняется в базе данных и подготавливается к публикации.

## Структура

- `cmd/smartbid` - точка входа приложения.
- `internal/app` - сборка зависимостей приложения.
- `internal/config` - конфигурация из переменных окружения.
- `internal/domain` - доменные модели и ошибки.
- `internal/http` - роуты и HTTP-хендлеры.
- `internal/http/dto` - DTO для HTTP-запросов и ответов.
- `internal/http/middleware` - HTTP middleware.
- `internal/service` - бизнес-логика.
- `internal/repository` - работа с хранилищами данных.
- `pkg/database` - подключение к PostgreSQL.
- `migrations` - SQL-миграции goose.

Подробнее:

- [Архитектура](docs/architecture.md)
- [Безопасная разработка](docs/security.md)

## Запуск

Основные параметры запуска лежат в `.env`:

```env
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8080

DB_HOST=db
DB_PORT=5432
DB_HOST_PORT=5432
DB_NAME=smartbid
DB_USER=smartbid
DB_PASSWORD=smartbid
DB_SSLMODE=disable
```

`DB_PORT` - порт PostgreSQL внутри Docker-сети, по нему backend подключается к контейнеру `db`.
`DB_HOST_PORT` - порт PostgreSQL на твоей машине, например для подключения через IDE или `psql`.

```bash
docker compose up --build
```

При старте контейнера приложение автоматически выполняет миграции через `goose`, а затем запускает HTTP-сервер.

Проверка работы приложения:

```bash
curl http://localhost:8080/ping
```

Ожидаемый ответ:

```json
{"status":"ok"}
```

## API

- `GET /ping` - health-check приложения.
- `POST /api/v1/ads` - создание объявления.
- `GET /api/v1/ads/{id}` - получение объявления по идентификатору.

Создание объявления поддерживает `application/json` с полями `title`, `description` и `photo`. Поле `photo` передаётся как `[]byte`, поэтому в JSON кодируется стандартно для Go - base64-строкой.
