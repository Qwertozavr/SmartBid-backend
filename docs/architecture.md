# Architecture

SmartBid backend is organized as a layered service that follows Clean Architecture at the project bootstrap level.

## Paradigm

The project uses distributed programming principles:

- the backend service runs as a separate container;
- PostgreSQL runs as a separate container;
- services communicate over the Docker network;
- runtime settings are provided through environment variables;
- database migrations are applied by `goose` before the HTTP server starts.

## Layers

- `cmd/smartbid` - application entry point.
- `internal/app` - dependency composition.
- `internal/domain` - business entities, statuses, and domain errors.
- `internal/service` - application business logic.
- `internal/repository` - repository ports.
- `internal/repository/postgres` - PostgreSQL repository implementation.
- `internal/http/router` - route declarations.
- `internal/http/handler` - HTTP adapters.
- `internal/http/dto` - HTTP request and response models.
- `internal/http/middleware` - HTTP middleware.
- `pkg/database` - database connection infrastructure.

## Patterns

- Clean Architecture: domain and service layers do not depend on HTTP or PostgreSQL implementation details.
- Repository: storage access is described by `repository.AdRepository` and implemented by `postgres.AdRepository`.
- Service: `service.AdService` contains application operations and validation flow.
- State Machine: допустимы переходы `created -> published|removed` и `published -> bought|expired|removed`; конечные состояния неизменяемы.
- Outbox Pattern: переход в конечный статус и событие `ad.finished` сохраняются в одной транзакции.
- Producer-Consumer: backend публикует события, Telegram-бот идемпотентно обрабатывает их как Consumer.
- Adapter: HTTP, PostgreSQL и Kafka изолированы адаптерами от бизнес-логики.
- Dead Letter Queue: события после исчерпания попыток доставки публикуются в DLQ.

## Ad timer

Слой сервиса задаёт фиксированную длительность объявления `24h` и вычисляет
`published_at` и `expires_at`. Фоновый worker раз в минуту блокирует
просроченные опубликованные объявления через PostgreSQL `FOR UPDATE SKIP
LOCKED` и переводит их в `bought` либо `expired`. Ставки используют условное
обновление и не принимаются после `expires_at`.
