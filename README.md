# Uptime availability checker

The stack consists of:

- **Scheduler** — reads `config/services.yaml`, upserts per-site **check interval**, **HTTP timeout**, and **retry** settings into PostgreSQL, then enqueues a ping job to **AWS SQS** whenever each site’s interval has elapsed since the last enqueue.
- **Worker(s)** — long-poll **SQS** (`ReceiveMessage`), run an HTTP GET using that site’s **timeout** and **retries** (extra attempts after a failed request), record **up** / **down**, **latency**, and optional **error** in `probe_results`, then **delete** the message on success. Failed processing leaves the message invisible until the visibility timeout expires, then it is retried (at-least-once delivery).
- **API** — serves `GET /api/v1/status` (latest row per service) and a small HTML dashboard at `/`.
- **PostgreSQL** — stores services and ping history.
- **SQS** — decouples scheduling from execution; scale workers independently. In Docker Compose, **LocalStack** provides a compatible SQS API for local testing.

### Architecture (DDD / hexagonal)

The layout uses **dependency injection via interfaces** (ports) so the core stays independent of PostgreSQL, SQS, YAML, and HTTP details:

| Layer | Package | Role |
|-------|---------|------|
| **Domain** | `internal/domain` | Entities and value objects (`ServiceDefinition`, `ProbeResult`, `ProbeJob`). **Ports** live in `internal/domain/ports`: `ServiceRepository`, `JobQueue`, `AvailabilityProbe`, `ServiceLoader`, `DownNotifier`, `LatestStatusReader`. |
| **Application** | `internal/application` | Use cases: `SchedulerService`, `WorkerService`, `ListLatestServiceStatuses` — depend only on ports, not adapters. |
| **Adapters** | `internal/adapters/...` | **Driving:** YAML catalog loader. **Driven:** PostgreSQL repository, SQS queue client, HTTP availability probe, optional webhook alerter. |
| **Composition** | `cmd/*` | Wires concrete adapters into application services (composition root). |

## Run with Docker Compose

```bash
docker compose up --build
```

- Dashboard: [http://localhost:8080](http://localhost:8080)
- JSON: [http://localhost:8080/api/v1/status](http://localhost:8080/api/v1/status)

LocalStack SQS is on port **4566**. The `sqs-init` service creates the queue `uptime-ping-jobs` before workers and the scheduler start.

Scale workers (example: three replicas):

```bash
docker compose up --build --scale worker=3
```

### Environment variables

| Variable | Used by | Description |
|----------|---------|-------------|
| `DATABASE_URL` | all | PostgreSQL DSN |
| `SQS_QUEUE_URL` | scheduler, worker | Full queue URL from AWS or LocalStack |
| `AWS_REGION` / `AWS_DEFAULT_REGION` | scheduler, worker | AWS region (default `us-east-1`) |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | scheduler, worker | Credentials (use real IAM keys in AWS; `test`/`test` for LocalStack) |
| `AWS_ENDPOINT_URL` or `SQS_ENDPOINT` | scheduler, worker | Optional custom SQS API endpoint (e.g. `http://localstack:4566`) |
| `SQS_VISIBILITY_TIMEOUT` | worker | Seconds (default `60`). Must exceed worst-case work time: roughly **(1 + retries) × timeout** per attempt plus small backoffs—raise this if you increase `retries` or `timeout` in YAML |
| `SQS_WAIT_TIME_SECONDS` | worker | Long-poll wait (default `20`; max `20`) |
| `ALERT_WEBHOOK_URL` | worker | Optional. If set, the worker **POST**s JSON to this URL on every **down** probe (after the result is stored). |
| `CONFIG_PATH` | scheduler | Path to YAML config (default `/config/services.yaml` in Compose) |
| `SCHEDULER_TICK` | scheduler | How often the scheduler wakes to reload config and enqueue due jobs (default `5s`) |
| `HTTP_ADDR` | api | Listen address (default `:8080`) |

### Configuration file

Edit `config/services.yaml`. Each service needs a unique `name` and an HTTP(S) `endpoint`.

Optional per site:

| Field | Meaning | Default |
|-------|---------|---------|
| `interval` | Minimum time between enqueueing checks for this site (Go duration: `30s`, `1m`, …) | `30s` |
| `timeout` | Per-request HTTP timeout for each attempt | `15s` |
| `retries` | Number of **extra** HTTP attempts after the first failure (integer `0`–`20`) | `0` |

The scheduler stores these in PostgreSQL and only enqueues when `interval` has passed since the last enqueue for that service.

### Production (AWS)

Create an SQS queue in your account, set `SQS_QUEUE_URL` to the queue URL from the console or CLI, **omit** `AWS_ENDPOINT_URL`, and use IAM roles or environment credentials. Tune `SQS_VISIBILITY_TIMEOUT` to cover the longest possible single job: multiple retries each using `timeout`, plus short delays between retries.

## Local build (without Docker)

```bash
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker
go build -o bin/scheduler ./cmd/scheduler
```

You need PostgreSQL, an SQS API (AWS or LocalStack), and the same environment variables as above.

![alt text](<assets/images/architecture.png>)