# Gimli

Email worker that consumes mail tasks from RabbitMQ, applies template parsing and HTML/CSS minification, and sends via SMTP — with exponential backoff retries and dead-letter queue support.

## Prerequisites

- Go 1.26+
- RabbitMQ
- SMTP server

## Setup

Copy `.env.example` to `.env` and fill in your configuration:

```env
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

MAIN_QUEUE_NAME=gimli-tasks
MAIN_QUEUE_EXCHANGE=gimli-main
MAIN_QUEUE_ROUTING_KEY=email.send
MAIN_QUEUE_MAX_RETRIES=5

RETRY_QUEUE_NAME=gimli-retry
RETRY_QUEUE_EXCHANGE=gimli-retry
RETRY_QUEUE_DELAY_FACTOR=1.0

DLQ_NAME=gimli-dlq
DLQ_EXCHANGE=gimli-dlq
DLQ_ROUTING_KEY=email.dead

WORKER_COUNT=4
PREFETCH_COUNT=8
RECONNECT_DELAY=5

SMTP_HOST=localhost
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_ALLOW_TLS=true
SMTP_POOL_SIZE=4
```

## Build & Run

```sh
make build
make run
```

Or manually:

```sh
go build -o bin/gimli cmd/main/main.go
./bin/gimli
```

## Load Testing

```sh
make lt
```

## Architecture

```
cmd/main/main.go    → entry point
internal/
  config/           → env loading & config types
  mail/             → SMTP client pool, template parsing, minification
  queue/            → RabbitMQ consumer with retry/DLQ logic
```

Messages are JSON-encoded `Mail` structs consumed from the main queue. On failure they are re-published with exponential backoff; after max retries they land in the dead-letter queue.
