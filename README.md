# MasakApa — Backend API (`meal-planner-api`)

Backend API untuk aplikasi rekomendasi menu masak berbasis AI **"MasakApa"**. User dapat menerima rekomendasi menu harian sesuai goal (hemat/sehat/diet) dan budget, lengkap dengan estimasi harga bahan, fitur crowdsourcing harga sembako komunitas, sistem subscription, dan integrasi payment gateway.

---

## 🚀 Tech Stack

- **Language:** Go (1.22+)
- **Web Framework:** Fiber v2
- **Database:** PostgreSQL (pgx driver v5)
- **Database Migration:** `golang-migrate` (plain SQL)
- **Cache & Rate Limiting:** Redis
- **Auth:** Supabase Auth (verifikasi JWT Google OAuth)

---

## 📁 Folder Structure

```text
cmd/
  api/
    main.go            -> Entrypoint aplikasi Fiber
internal/
  ai/                  -> Interface & provider AI (OpenAI, Groq, Gemini)
  price/               -> Provider estimasi harga (AI estimate, crowdsource)
  payment/             -> Interface & provider payment (Dummy, Wuzzpay)
  auth/                -> Middleware verifikasi Supabase JWT
  subscription/        -> Logic subscription & rate limiting per tier
  pricewatch/          -> Logic crowdsource campaign, submission & validasi konsensus
  handler/             -> HTTP handlers per resource
  repository/          -> DB query layer per entity
  model/               -> Struct & domain types
migrations/            -> SQL migration files (golang-migrate format)
docs/
  SCHEMA.sql           -> Reference database schema (source of truth)
  API_CONTRACT.md      -> Kontrak endpoint REST API
docker-compose.yml     -> Postgres (port 5434) & Redis (port 6380) lokal
```

---

## 🛠️ Environment Variables

Salin `.env.example` ke `.env` sebelum menjalankan server:

```bash
cp .env.example .env
```

Isi `.env`:

```ini
DATABASE_URL=postgres://postgres:postgres@localhost:5434/masakapa?sslmode=disable
REDIS_URL=redis://localhost:6380
SUPABASE_JWT_SECRET=your_supabase_jwt_secret
AI_PROVIDER_API_KEY_OPENAI=your_openai_api_key
PORT=8080
ENV=development
```

---

## 🚦 Cara Menjalankan (Local Development)

### 1. Jalankan Container (Postgres & Redis)

```bash
docker compose up -d
```

### 2. Jalankan Database Migration

```bash
export $(grep -v '^#' .env | xargs)
migrate -path migrations/ -database "$DATABASE_URL" up
```

### 3. Jalankan Server API

```bash
go run ./cmd/api/main.go
```

Server akan berjalan pada `http://localhost:8080`.

### 4. Health Check

```bash
curl http://localhost:8080/health
```

Output:
```json
{"data":{"service":"masakapa-api","status":"ok"},"error":null}
```

---

## 📄 Format Response API

Semua endpoint mengembalikan format JSON yang konsisten:

- **Success Response:**
  ```json
  {
    "data": { ... },
    "error": null
  }
  ```
- **Error Response:**
  ```json
  {
    "data": null,
    "error": {
      "message": "Error description here"
    }
  }
  ```
