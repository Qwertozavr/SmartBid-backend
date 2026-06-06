# Ad Timer Design

## Goal

Add a backend-owned 24-hour auction timer for published ads. When the timer
expires, the backend finishes the ad and publishes an idempotent event for
consumers such as the Telegram bot.

## Scope

- Start a fixed 24-hour timer when an ad transitions to `published`.
- Reject bids after the timer expires.
- Automatically transition an expired published ad:
  - to `bought` when `pretendent_id` is present;
  - to `expired` when `pretendent_id` is absent.
- Add an authenticated-by-chat removal operation that transitions an ad to
  `removed`.
- Publish a completion event for `removed`, `bought`, and `expired`.

The Telegram bot is not responsible for timer execution or business-state
transitions. It consumes completion events and updates Telegram messages.

## State Machine

Allowed transitions:

```text
created   -> published | removed
published -> bought | expired | removed
removed, bought, expired -> no transitions
```

The domain layer owns transition validation. Attempts to transition from a
terminal state or along an unsupported path return a domain validation error.

## Time Ownership

The service layer owns the auction lifetime:

```go
const adLifetime = 24 * time.Hour
```

`AdService` receives a `now func() time.Time` dependency. Production wiring
uses `time.Now`; tests use a fixed function.

When publishing an ad, the service calculates:

```text
published_at = now()
expires_at   = published_at + adLifetime
```

The repository only persists the supplied timestamps and does not calculate
the lifetime.

## Data Model

Add the `expired` value to `ad_status`.

Add nullable timestamp columns to `ads`:

- `published_at timestamptz`
- `expires_at timestamptz`

Both fields remain null until the ad is published. Publishing sets them once;
subsequent bids do not change them.

The ad domain model and HTTP responses expose `published_at` and `expires_at`
as nullable timestamps.

## Publishing

Publishing is allowed only from `created`. The service validates `chat_id`,
checks the state-machine transition, calculates the timestamps, and performs
the status and timestamp update.

The existing `ad.created` outbox event remains associated with publication and
is not sent to the new completion topic.

## Bidding

A bid is accepted only when the ad is `published` and `expires_at` is later
than the service's current time.

The repository performs the price and pretender update conditionally against
the current `published` status and `expires_at`. This prevents a bid from being
accepted concurrently with timer completion.

An attempted bid against a non-published or expired ad returns a domain
validation error.

## Timer Worker

A background worker runs once per minute. A delay of up to one minute after
the exact expiration time is acceptable.

On each iteration, it finds expired `published` ads and completes them in
batches. Rows are claimed with PostgreSQL row locking suitable for multiple
backend instances, so only one worker processes each ad.

For each claimed ad:

- use `bought` when `pretendent_id` is present;
- use `expired` when `pretendent_id` is absent;
- update the status and create the completion outbox event in one transaction.

Processing is retryable. Ads that have already left `published` are skipped.

## Removal API

Add:

```http
POST /api/v1/ads/{id}/remove
Content-Type: application/json

{"chat_id":123}
```

The service verifies that the supplied `chat_id` matches the ad's `chat_id`.
Removal is allowed from `created` and `published`. Removal from a terminal
state returns a validation error.

The transition to `removed` and creation of its completion outbox event occur
in one transaction.

## Completion Event

Add a dedicated Kafka topic and event type:

- default topic: `ad-finished`
- event type: `ad.finished`
- DLQ: the existing configured Kafka DLQ mechanism

Payload:

```json
{
  "event_id": "uuid",
  "ad_id": "uuid",
  "status": "removed|bought|expired",
  "pretendent_id": null,
  "final_price": 105
}
```

`pretendent_id` is nullable. The event is written through the existing Outbox
Pattern in the same database transaction as the terminal state change.

The outbox uniqueness constraint on aggregate, aggregate ID, and event type
ensures one `ad.finished` event per ad. The event ID allows consumers to process
the message idempotently. Failed Kafka deliveries use the existing retry and
Dead Letter Queue flow.

## Configuration

Add `KAFKA_AD_FINISHED_TOPIC`, defaulting to `ad-finished`.

The 24-hour lifetime stays fixed in the service layer and is not configurable
in this iteration. The worker interval stays fixed at one minute.

## Error Handling And Security

- Unknown ads return `404`.
- Invalid state transitions, mismatched `chat_id`, and bids after expiration
  return `400`.
- Request bodies continue to reject unknown JSON fields and use the existing
  request-size limit.
- Repository errors and transaction failures return `500` without exposing
  storage details.
- State changes and corresponding outbox records are atomic.
- Conditional updates and row locking prevent race conditions between bids,
  removal, and automatic completion.

## Testing

Domain tests cover all allowed and rejected state transitions.

Service tests use a fixed `now` function and cover:

- publishing sets timestamps exactly 24 hours apart;
- bids before expiration succeed;
- bids at or after expiration fail;
- timer completion selects `bought` or `expired`;
- removal validates `chat_id` and source status;
- terminal transitions create the expected outbox payload.

Repository query tests cover conditional bid updates, publication timestamps,
expired-ad claiming, row locking, terminal status updates, and returned ad
fields.

HTTP router tests cover the `/remove` success and error responses and the new
timestamp response fields.

Application-level verification runs the complete Go test suite.
