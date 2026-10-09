# Aurahub 🎬

Aurahub is a modern, high-performance video streaming platform built with Go (Fiber), SvelteKit, PostgreSQL, Valkey, Apache Kafka, and OpenSearch.

---

## 🚀 One-Command Zero-Config Docker Setup

Run the entire platform locally — including **PostgreSQL**, **Valkey**, **Kafka (KRaft)**, **OpenSearch**, the **Go API**, and the **SvelteKit Web App** — with a single command:

```bash
docker compose up --build
```

*(To run in the background: `docker compose up -d --build`)*

### ✨ What happens automatically (No manual configuration needed!)
1. **PostgreSQL**: Starts on port `5432`. Missing tables, custom ENUMs (`video_visibility`, `streamtape_status`, etc.), full-text search tsvector columns, and performance GIN indexes are **automatically created** upon boot. If tables already exist, it verifies and proceeds immediately.
2. **Valkey**: High-performance Redis-compatible caching engine starts on port `6379`.
3. **Apache Kafka (KRaft mode)**: Runs without ZooKeeper on ports `9092` and `29092`. Topics `aurahub.video.lifecycle` and `aurahub.video.upload` are **automatically initialized** with 3 partitions.
4. **OpenSearch**: Starts on port `9200`. The `aurahub_videos` index with custom **edge-ngram tokenizers** (min 2, max 20) and BM25 field weightings is **automatically verified/created** by the backend upon startup.
5. **Backend**: Compiled into an ultra-lean Alpine container on port `8080`, listening for events and serving the API.
6. **Frontend**: SvelteKit web client running with hot reloading on port `5173`, automatically proxying `/api` requests to the backend.

Open your browser to:
👉 **[http://localhost:5173](http://localhost:5173)**

---

## 🔌 Service Ports & Endpoints

| Service | Port | Description |
|---|---|---|
| **Frontend Web App** | `5173` | SvelteKit UI ([http://localhost:5173](http://localhost:5173)) |
| **Backend API** | `8080` | Go Fiber REST API ([http://localhost:8080](http://localhost:8080)) |
| **API Health Check** | `8080` | [http://localhost:8080/api/health](http://localhost:8080/api/health) |
| **PostgreSQL** | `5432` | User: `aurahub`, Pass: `aurahub_secret`, DB: `aurahub` |
| **Valkey (Redis)** | `6379` | Key-value cache & session store |
| **Kafka Broker** | `9092` / `29092` | Event streaming engine |
| **OpenSearch** | `9200` | Full-text BM25 search & autocomplete engine |

---

## 🛠️ Stopping the Environment

To stop all containers:
```bash
docker compose down
```

To stop all containers and clear local volume data (for a clean reset):
```bash
docker compose down -v
```

---

## 💻 Running Without Docker (Local Development)

If you prefer running services directly on your host machine:

### 1. Start the Backend
```bash
cd backend
go run main.go
```
*(The backend will automatically verify and initialize PostgreSQL tables and OpenSearch indexes if connected).*

### 2. Backfill Existing Videos into OpenSearch
```bash
cd backend
go run ./cmd/index-opensearch
```

### 3. Start the Frontend
```bash
cd frontend
npm install
npm run dev
```
