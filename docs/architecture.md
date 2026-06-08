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
- `internal/price` - provider-neutral price estimation port.
- `internal/price/openrouter` - OpenRouter price estimation adapter.
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

## Price estimation

Оценка цены отделена от конкретного внешнего провайдера интерфейсом
`price.Estimator`. OpenRouter-клиент реализует этот интерфейс как адаптер, а
`internal/app` создаёт адаптер из конфигурации и внедряет его в `AdService`.
Для запуска приложения обязателен `OPENROUTER_API_KEY`; это относится и к
запуску через Docker Compose.

При создании объявления сервис после валидации передаёт оценщику заголовок,
описание и необязательное фото. OpenRouter возвращает внутренний
`recommended_price` как целое количество рублей. Сервис отвечает за перевод
результата в копейки умножением на `100` и сохраняет его в существующее поле
`Price`, используемое базой данных и API; отдельное поле `recommended_price`
не добавляется.

Обычные ошибки сети или провайдера, включая собственный timeout адаптера,
некорректный или неположительный результат и переполнение при переводе в
копейки приводят к резервной цене `100` копеек. Отмена или превышение deadline
контекста вызывающей стороны/HTTP-запроса возвращаются вызывающей стороне и
прерывают создание объявления до вызова repository, поэтому запись в базу
данных не выполняется. Write timeout HTTP-сервера должен оставлять достаточно
времени для оценки цены и последующего создания объявления.

## Ad timer

Слой сервиса задаёт фиксированную длительность объявления `24h` и вычисляет
`published_at` и `expires_at`. Фоновый worker раз в минуту блокирует
просроченные опубликованные объявления через PostgreSQL `FOR UPDATE SKIP
LOCKED` и переводит их в `bought` либо `expired`. Ставки используют условное
обновление и не принимаются после `expires_at`.

Подробное описание сценариев, события завершения и гарантий конкурентности:
[Жизненный цикл объявлений](ad-lifecycle.md).
