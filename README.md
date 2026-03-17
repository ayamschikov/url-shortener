# URL Shortener

URL shortening service with Redis caching, rate limiting, click tracking, and link expiration.

## Stack

- Go 1.25
- PostgreSQL (pgx, no ORM)
- Redis (cache-aside, rate limiting)
- chi (HTTP router)
- goose (migrations)
- Docker + docker-compose

## API

| Method | Path       | Description                              |
|--------|------------|------------------------------------------|
| POST   | /shorten   | Create short URL (optional: custom alias, TTL) |
| GET    | /{code}    | Redirect to original URL                 |

## Features

- **Cache-aside** — popular URLs served from Redis
- **Rate limiting** — fixed window per IP
- **Click tracking** — async goroutine, non-blocking
- **Link expiration** — TTL support
- **Custom aliases** — user-defined short codes

## Quick start

```bash
cp .env.example .env
make docker-up
make migrate-up
make run
```

## Tests

```bash
make test
```

Unit tests (service, handler) use mocks. Integration tests (repository) use testcontainers with real PostgreSQL.

## Architecture

```
cmd/server/         entrypoint
internal/
  handler/          HTTP handlers
  middleware/       rate limiting
  service/          business logic, caching
  repository/       PostgreSQL queries
  cache/            Redis cache layer
  model/            domain structs
  httputil/         IP extraction helpers
migrations/         goose SQL migrations
```
