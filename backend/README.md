# Backend data import

`cmd/migrate-mongo` copies the legacy MongoDB collections into the PostgreSQL
schema used by this backend. MongoDB is only opened as a read source: the
importer issues collection reads and never inserts, updates, deletes, or
changes its schema.

Set `MONGO_URI` and `DATABASE_URL` in `backend/.env`. The database name is read
from the MongoDB URI; set `MONGO_DATABASE` if the URI does not include it.
`backend/.env.example` contains placeholders.

From the `backend` directory, run a dry run first:

```powershell
go run ./cmd/migrate-mongo
```

The default dry run reads and validates source documents but does not connect
to or modify PostgreSQL. To write the imported data after reviewing the dry-run
report, run:

```powershell
go run ./cmd/migrate-mongo --apply
```

The destination schema must already exist. The apply run is one PostgreSQL
transaction and does not truncate or delete destination data. Stable,
collection-scoped UUIDs map MongoDB ObjectIds so reruns are safe; conflicts on
the same primary or relationship key are left unchanged, while conflicting
unique email, username, or video file IDs fail the transaction rather than
silently merging unrelated records. Stop writes to the legacy application
while applying the import so cross-collection references reflect one stable
source state.

Rows whose required references point to missing or skipped source records are
omitted instead of creating invalid PostgreSQL relationships; the command
reports those counts under `skipped_orphans.*`. Apply reports also show new
rows written under `inserted.*` and final PostgreSQL table totals. Review the
skipped counts in both the dry-run and apply reports.

The importer covers users, videos, comments and replies, notifications,
playlists and their ordered video membership, subscriptions, likes and views,
watch history, and watch-later entries. Legacy password hashes are copied
unchanged.

## Manual Render deployment

No Render manifest is required. Create a Go web service using `backend` as the
root directory, `go build -o aurahub .` as the build command, and
`./aurahub` as the start command. Configure `DATABASE_URL`, `VALKEY_URL`,
`JWT_SECRET`, and `FRONTEND_ORIGINS` in Render's environment settings.
`UPLOAD_FOLDER_ID` and `FREEIMAGE_API_KEY` are needed for the corresponding
upload integrations.

Provision PostgreSQL and Valkey-compatible services first. Before the first
API start, initialize the PostgreSQL schema from the `backend` directory with
`go run ./cmd/initdb`. The initializer can be rerun: it leaves existing tables
and enum types in place and creates only missing ones; it does not alter an
existing schema. Restart or redeploy the web service after configuring
environment variables. Keep `.env` files local; `backend/.gitignore` excludes
them while allowing `.env.example` to remain tracked.
