# CLAUDE.md — meal-planner-api

Konteks proyek untuk AI coding assistant. Baca file ini SEBELUM membuat/mengubah kode apapun di repo ini.

## Tentang Proyek

Backend API untuk aplikasi rekomendasi menu masak berbasis AI ("MasakApa"). User dapat rekomendasi menu harian sesuai goal (hemat/sehat/diet) dan budget, lengkap estimasi harga bahan. Ada fitur crowdsourcing harga sembako dari komunitas, sistem subscription (Free/Premium), dan payment.

Full spec produk ada di `docs/PRD.md` (copy dari dokumen utama). Kalau ada keputusan desain yang tidak jelas dari file ini, cek PRD dulu sebelum improvisasi.

## Tech Stack (WAJIB, jangan ganti tanpa konfirmasi)

- Bahasa: **Go** (versi 1.22+)
- Web framework: **Fiber** (atau Echo — pilih salah satu di awal proyek, konsisten)
- Database: **PostgreSQL** (Supabase di production, Docker Postgres di local)
- DB access: **pgx** driver, query builder/ORM boleh **sqlc** (generate type-safe Go dari SQL) — hindari GORM kalau bisa, lebih suka raw SQL + sqlc untuk kontrol query
- Migration: **golang-migrate**, file SQL plain di `migrations/`
- Auth: validasi JWT dari **Supabase Auth** (Google OAuth ditangani Supabase, backend cuma verifikasi token)
- Cache: **Redis** (untuk cache hasil AI generate & rate limiting)

## Struktur Folder (WAJIB diikuti)

```
cmd/
  api/main.go            → entry point
internal/
  ai/                     → AIProvider interface + implementasi (openai.go, groq.go, gemini.go)
  price/                  → PriceProvider interface + implementasi (ai_estimate.go, crowdsource.go)
  payment/                → PaymentProvider interface + implementasi (dummy.go, wuzzpay.go)
  auth/                   → middleware verifikasi Supabase JWT
  subscription/           → logic subscription & rate limit per tier
  pricewatch/             → logic crowdsource campaign, submission, validasi konsensus
  handler/                → HTTP handlers (per resource, mis. menu_handler.go, auth_handler.go)
  repository/             → DB query layer (per entity)
  model/                  → struct/domain types
migrations/               → SQL migration files (golang-migrate format)
docs/
  SCHEMA.sql              → reference schema lengkap (source of truth untuk migration)
  API_CONTRACT.md         → kontrak endpoint
docker-compose.yml        → Postgres + Redis lokal
```

## Arsitektur Wajib: Provider Pattern (Interface + Swappable Implementation)

Tiga domain ini WAJIB pakai interface abstraction — jangan hardcode langsung ke satu vendor:

### 1. AIProvider (`internal/ai`)
```go
type AIProvider interface {
    GenerateMenu(ctx context.Context, params MenuGenerateParams) (*MenuOptions, error)
}
```
Implementasi aktif: `OpenAIProvider`. Provider aktif ditentukan dari tabel `ai_provider_config`, bukan env var hardcode — baca config ini saat request masuk (atau cache di Redis beberapa menit).

### 2. PriceProvider (`internal/price`)
```go
type PriceProvider interface {
    GetPrice(ctx context.Context, ingredientName string, cityID int) (*PriceResult, error)
}
```
MVP: `AIEstimateProvider` aktif. `CrowdsourceProvider` mengambil data dari `ingredient_price_log` yang `source = "crowdsource"` — kalau ada data crowdsource valid untuk kombinasi ingredient+kota, prioritaskan ini di atas AI estimate.

### 3. PaymentProvider (`internal/payment`)
```go
type PaymentProvider interface {
    CreateTransaction(ctx context.Context, sub Subscription, amount int, coupon *Coupon) (*TransactionRef, error)
    HandleWebhook(ctx context.Context, payload []byte) (*TransactionStatus, error)
}
```
MVP: `DummyPaymentProvider` aktif — langsung return status success tanpa call eksternal. JANGAN implementasi `WuzzpayProvider` dulu sampai diminta eksplisit (kredensial Wuzzpay belum aktif).

**Aturan penting:** semua provider baru harus implement interface yang sama, return struct yang sama, supaya business logic (handler, subscription logic, dll) TIDAK PERNAH tahu vendor mana yang lagi aktif.

## Konvensi Kode

- Semua response API dalam JSON, format konsisten: `{"data": ..., "error": null}` atau `{"data": null, "error": {"message": "..."}}`
- Error handling: jangan `panic`, selalu return error ke caller, log di handler layer
- Semua endpoint yang butuh auth: pasang middleware `auth.RequireAuth()` di route, ambil `user_id` dari context, JANGAN percaya `user_id` dari request body/query
- Endpoint admin: middleware tambahan `auth.RequireAdmin()`
- Rate limiting generate menu: cek `subscription.features.max_generate_per_day` dari plan aktif user SEBELUM call AIProvider (hindari cost AI kalau limit sudah habis)
- Semua nominal uang dalam **integer (rupiah, tanpa desimal)**, jangan pakai float untuk uang
- Naming table & column: `snake_case`, sudah didefinisikan lengkap di `docs/SCHEMA.sql` — JANGAN improvisasi nama kolom baru tanpa cek dulu skema yang sudah ada

## Environment Variables

Lihat `.env.example` untuk daftar lengkap. Yang penting:
- `DATABASE_URL` — beda value local (Docker) vs production (Supabase), keduanya format Postgres standar
- `SUPABASE_JWT_SECRET` — untuk verifikasi token
- `AI_PROVIDER_API_KEY_*` — per provider (OpenAI dulu)
- `REDIS_URL`

## Yang TIDAK Boleh Dilakukan AI Assistant Tanpa Konfirmasi

- Jangan ganti tech stack inti (Go, Postgres, Fiber/Echo) ke pilihan lain
- Jangan implementasi `WuzzpayProvider` asli sebelum diminta eksplisit — pakai Dummy terus
- Jangan hardcode API key di kode, selalu dari env var
- Jangan bikin tabel baru di luar `docs/SCHEMA.sql` tanpa update schema itu dulu
- Jangan skip validasi input di handler layer (semua request body harus divalidasi sebelum masuk business logic)
