# Gimli 🛡️

**Gimli** is a high-performance, resilient Go mail-dispatching microservice designed to consume email tasks from RabbitMQ, perform efficient template parsing and HTML/CSS minification, and reliably deliver them via SMTP. 

Engineered for **memory stability**, **strict resilience**, and **high configurability**, Gimli handles high-throughput pipelines gracefully without memory spikes.

---

## Key Features

- **Concurrent SMTP Connection Pooling:** Reuses persistent, pre-dialed SMTP connections to maximize throughput and eliminate TCP handshake overhead.
- **Robust Queue Management:** Integrated RabbitMQ consumer featuring exponential backoff retries and Dead-Letter Queue (DLQ) support for unrecoverable failures.
- **In-Flight Minification & Parsing:** Zero-allocation pattern matching for template variables (`{{key}}`) paired with production-safe HTML/CSS minification.
- **Constant Memory Footprint:** Bounded worker pools and prefetch limits ensure a flat, predictable memory profile even under massive loads.

---

## Prerequisites

- Go 1.26+
- RabbitMQ
- An SMTP server (or a local dev tool like Mailpit)

---

## Setup & Configuration

Copy `.env.example` to `.env` and fill in your configuration:

```env
RABBITMQ_URL=amqp://user:password@localhost:5672/
WORKER_COUNT=1000
RECONNECT_DELAY=5
PREFETCH_COUNT=3000
MAX_RETRIES

MAIN_QUEUE_NAME=gimli.mail
MAIN_QUEUE_EXCHANGE=gimli.exchange
MAIN_QUEUE_ROUTING_KEY=mail.send

RETRY_QUEUE_NAME=gimli.mail.retry
RETRY_QUEUE_DELAY_FACTOR=15
RETRY_QUEUE_EXCHANGE=gimli.retry

DLQ_QUEUE_NAME=gimli.mail.dlq
DLQ_QUEUE_EXCHANGE=gimli.dlx
DLQ_QUEUE_ROUTING_KEY=mail.dead

SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=testuser
SMTP_PASSWORD=testpass
SMTP_ALLOW_TLS=false
SMTP_POOL_SIZE=5
SMTP_TIMEOUT=3