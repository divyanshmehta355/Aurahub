# Aurahub Backend

High-concurrency Go (Fiber v2) REST API and asynchronous event processing backend for Aurahub.

## Database Initialization

Before the first API start, initialize the PostgreSQL schema from the `backend` directory:

```powershell
go run ./cmd/initdb
```

The initializer is idempotent: it leaves existing tables and enum types in place and creates only missing ones; it does not alter an existing schema.

## Search Indexing & Backfill

To backfill and synchronize video documents into OpenSearch:

```powershell
go run ./cmd/index-opensearch
```

If OpenSearch is not configured, the backend automatically operates in standby mode and queries PostgreSQL directly.

## Running Tests

All test suites are isolated in the `tests/` directory:

```powershell
go test -v ./...
```

## Manual Render Deployment

No Render manifest is required. Create a Go web service using `backend` as the root directory, `go build -o aurahub .` as the build command, and `./aurahub` as the start command.

Configure `DATABASE_URL`, `VALKEY_URL`, `JWT_SECRET`, and `FRONTEND_ORIGINS` in Render's environment settings. `UPLOAD_FOLDER_ID` and `FREEIMAGE_API_KEY` are needed for the corresponding upload integrations.

Provision PostgreSQL and Valkey-compatible services first, run `go run ./cmd/initdb`, and deploy the service. Keep `.env` files local; `backend/.gitignore` excludes them while allowing `.env.example` to remain tracked.
