# Secure Development Notes

The project baseline follows the intent of GOST R 56939-2016, "Information protection. Secure software development. General requirements", at an initial implementation level.

## Implemented Baseline

- configuration is externalized through `.env`;
- SQL queries use pgx parameters instead of string concatenation;
- domain validation is performed before persistence;
- HTTP request body size is limited;
- unknown JSON fields are rejected;
- internal repository errors are not returned directly to API clients;
- panic recovery middleware is enabled;
- basic security headers are added to HTTP responses;
- database migrations are versioned and applied through `goose`;
- integration events are persisted through the Outbox Pattern before publication;
- Kafka messages include stable `event_id` values for idempotent consumers;
- failed event delivery is retried and then routed to a Dead Letter Queue topic;
- `.env` is excluded from git.

## Required Next Steps

- add authentication and authorization;
- add structured audit/security events;
- add unit and integration tests for negative scenarios;
- add `go vet`, `govulncheck`, and `gosec` to CI;
- document threat model and security requirements;
- document vulnerability handling and dependency update process;
- introduce secrets management for production deployments.
