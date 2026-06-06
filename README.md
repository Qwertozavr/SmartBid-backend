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

KAFKA_BROKERS=kafka:9092
KAFKA_HOST_PORT=9092
KAFKA_AD_CREATED_TOPIC=ad-created
KAFKA_AD_FINISHED_TOPIC=ad-finished
KAFKA_DLQ_TOPIC=ad-created-dlq
```

`DB_PORT` - порт PostgreSQL внутри Docker-сети, по нему backend подключается к контейнеру `db`.
`DB_HOST_PORT` - порт PostgreSQL на твоей машине, например для подключения через IDE или `psql`.
`KAFKA_BROKERS` - адрес Kafka внутри Docker-сети, по нему backend публикует события.
`KAFKA_HOST_PORT` - порт Kafka на твоей машине.
`KAFKA_AD_CREATED_TOPIC` - топик для событий о создании объявлений.
`KAFKA_AD_FINISHED_TOPIC` - топик для событий о завершении объявлений.
`KAFKA_DLQ_TOPIC` - топик для событий, которые не удалось доставить после повторных попыток.

```bash
docker compose up --build
```

При старте контейнера приложение автоматически выполняет миграции через `goose`, а затем запускает HTTP-сервер.
Kafka разворачивается в Docker Compose, а init-контейнер создаёт топики `ad-created`, `ad-finished` и `ad-created-dlq`.

События о создании объявлений публикуются через Outbox Pattern: создание объявления и запись события выполняются в одной транзакции PostgreSQL, после чего фоновый dispatcher доставляет событие в Kafka. Payload события содержит идемпотентный `event_id` и `ad_id`.

При публикации backend запускает фиксированный 24-часовой таймер. Раз в минуту worker завершает просроченные объявления: с претендентом в статус `bought`, без претендента в `expired`. Переход в конечный статус и событие `ad.finished` записываются атомарно через Outbox Pattern. Telegram-бот выступает идемпотентным Consumer события завершения.

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
- `POST /api/v1/ad/{id}/increase` - поднятие цены объявления на 5%.
- `POST /api/v1/ads/{id}/remove` - удаление объявления владельцем чата.

Создание объявления поддерживает `application/json` с полями `title`, `description` и `photo`. Поле `photo` передаётся как `[]byte`, поэтому в JSON кодируется стандартно для Go - base64-строкой.
Поднятие цены принимает `application/json` с полем `pretendent_id` и возвращает идентификатор объявления с новой ценой.
