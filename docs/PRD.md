# PRD: AI Meal Planner (Working Title: "MasakApa")

## 1. Latar Belakang & Tujuan

**Masalah:** Ibu rumah tangga / individu kebingungan menentukan menu masak harian, sekaligus kesulitan menyesuaikan menu dengan budget dan tujuan hidup (hemat, sehat, diet tertentu).

**Solusi:** Aplikasi yang memberi rekomendasi menu harian berbasis AI, dipersonalisasi dari goal & budget user, lengkap dengan estimasi harga bahan.

**Tujuan Produk:**
- Mengurangi decision fatigue harian ("masak apa hari ini?")
- Membantu user tetap on-budget dan/atau on-goal kesehatan
- Menyediakan shopping list otomatis dari menu terpilih

---

## 2. Target Pengguna

| Persona | Deskripsi | Kebutuhan Utama |
|---|---|---|
| Ibu rumah tangga | Masak harian untuk keluarga, waktu terbatas | Cepat, praktis, sesuai budget keluarga |
| Individu hemat | Anak kos/pekerja, budget ketat | Estimasi harga akurat, porsi kecil |
| Health-conscious | Fokus diet/kalori/nutrisi | Info gizi, filter bahan |

---

## 3. Platform & Tech Stack

**Platform:** Web app dengan PWA (installable, offline-capable untuk shopping list & resep tersimpan). Tidak perlu native app di awal — PWA cukup untuk "Add to Home Screen" di HP.

**Rekomendasi Stack** (disesuaikan skillset: JS, PHP, Go, Python):

| Layer | Pilihan | Alasan |
|---|---|---|
| Frontend | Next.js (React) + TypeScript | SSR/PWA support native, ekosistem besar, cocok untuk web+mobile-like UX |
| Backend API | Go (Fiber/Echo) | Performa bagus untuk banyak call ke AI API, concurrency mudah utk future scraping/price-fetch jobs |
| Database | PostgreSQL | Relasional cocok untuk user, resep, harga, riwayat |
| Cache | Redis | Cache hasil AI generation & harga (hindari re-call AI tiap request sama) |
| AI Provider | Claude API (Anthropic) atau OpenAI | Untuk generate resep + estimasi harga, dengan structured output (JSON) |
| Hosting | Railway/Fly.io (awal) → VPS/Cloud saat scale | Murah untuk MVP |

*Alternatif: kalau mau lebih cepat prototyping, backend bisa pakai Node.js/Express dulu, migrasi ke Go saat traffic naik. Tapi karena kamu sudah terbiasa Go, langsung Go juga tidak masalah — asal API design-nya clean dari awal (lihat section 7).*

---

## 4. Scope MVP (Fase 1)

**In-scope:**
1. Onboarding: set goal, budget, jumlah anggota keluarga, pantangan
2. Home screen: greeting kontekstual + CTA rekomendasi menu harian
3. AI generate 3 opsi menu (nama, deskripsi, estimasi total harga, list bahan)
4. Estimasi harga per bahan (via AI, bukan real scraping dulu)
5. Shopping list otomatis dari menu terpilih (checklist manual)
6. Riwayat menu yang pernah dipilih (histori sederhana)

**Out-of-scope (Fase 2+):**
- Real-time price scraping/API marketplace
- Meal planning mingguan/bulanan otomatis
- Integrasi ke e-commerce (auto order bahan)
- Nutrisi detail (kalori, makro) per menu
- Social feature (share menu, rating komunitas)
- Native mobile app

---

## 5. Strategi Harga Bahan (PENTING — desain untuk mudah diganti)

Karena kamu sudah bilang mau fleksibel (AI dulu, nanti bisa scraping/API), desain sistem harga sebagai **abstraction layer**, bukan hardcode ke AI call langsung.

```
PriceProvider (interface)
  ├── AIEstimateProvider    ← aktif di MVP
  ├── ScrapingProvider      ← Fase 2 (Tokopedia/Shopee)
  └── ThirdPartyAPIProvider ← Fase 2/3 (kalau ada API resmi, misal Panel Harga Pangan / Kemendag)
```

**Cara kerja:**
- Backend punya interface `PriceProvider` dengan method `GetPrice(ingredientName, unit, region) -> (price, source, confidence)`
- MVP: implementasi `AIEstimateProvider` — minta AI estimasi harga berdasarkan wilayah user (kalau ada) + training data AI, kasih label `"estimasi"` di UI (jangan seolah-olah harga pasti)
- Fase 2: tinggal buat implementasi baru, swap di config, tanpa ubah logic lain
- Simpan histori harga di DB (`ingredient_price_log`) supaya nanti bisa hybrid: AI estimate + koreksi dari data riil yang terkumpul

**Catatan penting:** Karena wilayah Indonesia (kamu di Surabaya misal) harga bisa beda jauh per kota, sebaiknya dari awal simpan field `region` di user profile, walau AI provider belum bisa akurat per-kota — biar struktur DB sudah siap untuk Fase 2.

---

## 6. Data Model (Draft ERD)

```
users
  id, name, email, region, created_at

user_preferences
  id, user_id, goal (enum: hemat/sehat/diet/bebas),
  budget_amount, budget_period (harian/mingguan),
  household_size, restrictions (jsonb: alergi/pantangan)

recipes
  id, name, description, goal_tags (jsonb),
  ai_generated (bool), created_at

recipe_ingredients
  id, recipe_id, ingredient_name, quantity, unit

ingredient_price_log
  id, ingredient_name, region, price, source (ai/scrape/api),
  confidence_score, recorded_at

meal_selections
  id, user_id, recipe_id, selected_date, total_estimated_price

shopping_lists
  id, user_id, meal_selection_id, items (jsonb: {name, qty, checked}),
  created_at
```

---

## 7. API Design (High-Level)

```
POST   /api/onboarding                  → simpan user_preference
GET    /api/home/suggestion             → greeting + CTA context
POST   /api/menu/generate               → trigger AI, return 3 opsi menu + estimasi harga
POST   /api/menu/select                 → simpan pilihan user → generate shopping list
GET    /api/shopping-list/:id           → ambil shopping list
PATCH  /api/shopping-list/:id/item      → toggle checklist
GET    /api/history                     → riwayat menu terpilih
```

**Kontrak `POST /api/menu/generate` (contoh response):**
```json
{
  "options": [
    {
      "recipe_name": "Tumis Kangkung Tempe",
      "description": "Menu sehat rendah kalori, cocok untuk budget hemat",
      "estimated_total_price": 18000,
      "ingredients": [
        {"name": "Kangkung", "qty": "1 ikat", "estimated_price": 3000},
        {"name": "Tempe", "qty": "1 papan", "estimated_price": 6000},
        {"name": "Cabai merah", "qty": "5 buah", "estimated_price": 2000}
      ],
      "price_source": "ai_estimate"
    }
  ]
}
```

---

## 8. Desain Prompt AI (Panduan, bukan final)

Struktur prompt ke AI harus **selalu request structured JSON output**, jangan free text (biar gampang di-parse backend). Contoh system prompt:

> "Kamu adalah asisten rekomendasi menu masakan Indonesia. Berdasarkan goal user (hemat/sehat/diet), budget, dan jumlah anggota keluarga, berikan 3 opsi menu dalam format JSON. Sertakan estimasi harga tiap bahan dalam Rupiah berdasarkan harga pasar umum Indonesia saat ini. Jangan sertakan penjelasan di luar JSON."

Perlu di-lock schema JSON-nya (pakai JSON schema validation di backend) supaya AI tidak "ngarang" struktur beda-beda tiap call.

---

## 9. Non-Functional Requirements

- **Latency:** AI generate menu maks 5 detik (pakai loading state di FE)
- **Cache:** Cache hasil generate per kombinasi (goal + budget range + region) selama beberapa jam untuk hemat cost AI call
- **Cost control:** Rate limit generate per user (misal maks 10x/hari) untuk kontrol biaya API AI
- **Offline:** Shopping list & resep tersimpan bisa diakses offline (PWA service worker)

---

## 10. Roadmap

| Fase | Fitur | Estimasi |
|---|---|---|
| Fase 1 (MVP) | Onboarding, generate menu AI, estimasi harga AI, shopping list | 4-6 minggu |
| Fase 2 | Real price data (scraping/API), riwayat & analitik budget mingguan | +3-4 minggu |
| Fase 3 | Meal plan mingguan otomatis, info nutrisi, native app | +4-6 minggu |

---

## 11. Keputusan Final (update dari diskusi)

1. **AI Provider:** Multi-provider (OpenAI awal, bisa ganti Groq/Gemini/dll) → wajib pakai abstraction layer, lihat section 12.
2. **Region:** Granularity per kota, dengan tabel terpisah kota & provinsi (relasi hierarki).
3. **Monetisasi:** Subscription dengan harga fleksibel dari DB, support kupon/voucher, ada paket Rp0 dengan limit tertentu.
4. **Auth:** Wajib login via Google dari awal.
5. **Database:** Postgres-based, fleksibel antara Supabase (production) dan Docker Postgres lokal (development) — sama-sama Postgres jadi tidak perlu ganti driver, cukup ganti connection string.
6. **Deployment:** Frontend → Vercel (Next.js), Backend → Fly.io (Go API), DB → Supabase.

---

## 12. Arsitektur AI Provider (Multi-Model)

Sama seperti `PriceProvider`, buat interface `AIProvider` supaya gampang switch OpenAI ↔ Groq ↔ Gemini ↔ Claude tanpa ubah business logic:

```
AIProvider (interface)
  GenerateMenu(ctx, params) -> (MenuOptions, error)

Implementasi:
  ├── OpenAIProvider    ← default aktif
  ├── GroqProvider
  ├── GeminiProvider
  └── ClaudeProvider    (opsional, kualitas bahasa Indonesia biasanya bagus)
```

**Desain penting:**
- Provider aktif disimpan di tabel `ai_provider_config` (bukan hardcode/env saja), supaya bisa di-switch dari admin panel tanpa redeploy: `id, provider_name, model_name, is_active, api_key_ref, priority`
- Semua provider wajib return **format JSON yang sama** (menu options schema) — normalisasi response di masing-masing implementasi provider, jangan bocorkan perbedaan struktur API tiap vendor ke business logic
- API key jangan disimpan plaintext di DB — simpan reference ke secret manager (Fly.io secrets), kolom `api_key_ref` cuma nama env var
- Kalau mau resiliency: bisa tambah fallback logic (provider A gagal → coba provider B), tapi ini optional untuk Fase 1

---

## 13. Auth: Google Login

Karena stack pakai Supabase, paling efisien pakai **Supabase Auth** (built-in Google OAuth provider), bukan bikin OAuth flow manual:

- Frontend (Next.js) pakai Supabase JS client untuk trigger Google login
- Supabase Auth generate JWT session
- Backend (Go di Fly.io) validasi JWT dari Supabase (verify signature pakai Supabase JWT secret/JWKS) di middleware, tidak perlu simpan session sendiri
- Tabel `users` di section 6 tetap ada, tapi `id`-nya sinkron dengan `auth.users.id` dari Supabase (pakai id yang sama, bukan bikin tabel auth sendiri)

Ini menghindari kamu bikin ulang OAuth + session management dari nol.

---

## 14. Monetisasi & Subscription (Schema Tambahan)

```
subscription_plans
  id, name, price, billing_period (bulanan/tahunan),
  features (jsonb: {"max_generate_per_day": 3, "price_estimate": true, ...}),
  is_active, created_at

coupons
  id, code, discount_type (percent/fixed),
  discount_value, max_uses, used_count,
  expires_at, is_active

user_subscriptions
  id, user_id, plan_id, coupon_id (nullable),
  status (active/expired/canceled),
  started_at, ends_at

payment_transactions
  id, user_subscription_id, amount, status (pending/success/failed),
  payment_gateway, gateway_ref, created_at
```

**Catatan:**
- Plan gratis (Rp0) tetap masuk row di `subscription_plans` dengan `price = 0` dan `features` yang dibatasi (misal `max_generate_per_day: 3`) — jadi flow-nya konsisten, tidak perlu logic khusus "user tanpa subscription"
- Rate limiting (generate menu) baca dari `features` plan aktif user, bukan hardcode di code
- Untuk payment gateway lokal Indonesia (dukung QRIS, VA, e-wallet), rekomendasi: **Midtrans** atau **Xendit** — keduanya punya SDK Go & webhook untuk update status transaksi

---

## 15. Database Strategy: Supabase vs Local Docker Postgres

Karena keduanya sama-sama Postgres, **tidak perlu abstraction layer database** (ORM-agnostic dsb) — cukup:

- Pakai driver Postgres standar di Go (`pgx` atau lewat GORM/sqlc)
- Koneksi diatur lewat `DATABASE_URL` env var — beda value antara local (`postgres://localhost:5432/...` dari Docker) dan production (connection string Supabase)
- Migration pakai tool yang vendor-neutral: **golang-migrate** atau **Atlas** — tulis migration dalam plain SQL, jalan sama persis di local & Supabase
- Supabase punya fitur tambahan (Row Level Security, Realtime, Storage) — kalau nanti mau pakai fitur itu di luar Postgres standar, baru perlu dipikirkan lagi supaya tidak vendor-locked

---

## 16. Deployment Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Vercel     │ ───> │   Fly.io     │ ───> │  Supabase   │
│ Next.js (FE) │      │  Go API (BE) │      │  Postgres + │
│              │      │              │      │  Auth       │
└─────────────┘      └──────────────┘      └─────────────┘
```

- **Frontend (Vercel):** Next.js, env var untuk `NEXT_PUBLIC_API_URL` (arahkan ke Fly.io), `NEXT_PUBLIC_SUPABASE_URL/ANON_KEY` untuk auth
- **Backend (Fly.io):** Go API, terima request dari Vercel domain (perlu setup CORS eksplisit ke domain Vercel), koneksi ke Supabase Postgres pakai `DATABASE_URL`, secrets (AI API key, Supabase service key) disimpan via `fly secrets set`
- **Local dev:** `docker-compose.yml` untuk Postgres lokal + Go API jalan langsung, frontend `next dev` pointing ke `localhost` API

## 17. Keputusan Final (lanjutan)

1. **Payment Gateway: Wuzzpay** (docs.wuzzpay.com) — implementasi via abstraction, lihat section 20
2. **Admin panel:** Perlu, sederhana, deploy di Vercel (lihat struktur repo section 18)
3. **Subscription tier:** **2 tier — Free & Premium** (bukan 3). Basic dihapus untuk sekarang karena belum ada value differentiation yang jelas di MVP. Skema `subscription_plans` tetap fleksibel (data-driven), jadi tier baru bisa ditambah kapan saja dari admin panel tanpa ubah code.

**Limit awal per tier:**

| Fitur | Free | Premium |
|---|---|---|
| Generate menu | 3x/hari | Unlimited |
| Estimasi harga bahan | ✅ | ✅ |
| Shopping list | ✅ (tanpa histori) | ✅ + histori tersimpan |
| Riwayat menu | ❌ | ✅ |
| Filter goal custom per hari | ❌ (pakai default onboarding) | ✅ |

*(Disimpan di kolom `features` jsonb pada `subscription_plans`, gampang diubah tanpa redeploy.)*

---

## 18. Struktur Repository

**Rekomendasi: 2 repo, bukan 3** — frontend digabung monorepo, backend terpisah karena beda bahasa.

```
meal-planner-web/              (monorepo — Turborepo/pnpm workspaces)
├── apps/
│   ├── user/                  → Next.js, end-user app, deploy Vercel project #1
│   └── admin/                 → Next.js, admin panel, deploy Vercel project #2
├── packages/
│   ├── shared-types/          → TypeScript types (Recipe, Subscription, dll)
│   ├── supabase-client/       → config Supabase client shared
│   └── ui/                    → shared UI components (opsional)
└── turbo.json

meal-planner-api/              (repo terpisah — Go)
├── cmd/
├── internal/
│   ├── ai/                    → AIProvider implementations
│   ├── price/                 → PriceProvider implementations
│   ├── payment/                → Wuzzpay integration
│   ├── subscription/
│   └── auth/                  → Supabase JWT middleware
├── migrations/                → golang-migrate SQL files
└── docker-compose.yml         → Postgres lokal untuk dev
```

**Kenapa 2 repo, bukan 3:** admin panel & user app share types dan Supabase config — kalau dipisah repo, tiap ada perubahan schema/type harus update manual di 2 tempat (rawan out-of-sync). Tetap deploy sebagai 2 project Vercel independen dari 1 repo (set "Root Directory" per project). Kalau nanti admin panel butuh dikelola tim/akses terpisah, baru worth di-extract jadi repo sendiri.

---

## 19. Payment Provider: Dummy dulu, Wuzzpay Menyusul

Sama seperti `AIProvider` dan `PriceProvider` (section 5 & 12), payment juga dibungkus interface supaya bisa ganti implementasi tanpa ubah business logic lain:

```
PaymentProvider (interface)
  CreateTransaction(ctx, subscription, amount, coupon) -> (TransactionRef, PaymentURL, error)
  HandleWebhook(ctx, payload) -> (TransactionStatus, error)

Implementasi:
  ├── DummyPaymentProvider   ← AKTIF di MVP — selalu return status "success" instan, tidak call API eksternal
  └── WuzzpayProvider        ← Fase berikutnya, setelah KYC Wuzzpay selesai (lihat catatan Wuzzpay di bawah)
```

**`DummyPaymentProvider` behavior:**
- `CreateTransaction` langsung insert row `payment_transactions` dengan `status = "success"`, tidak ada redirect ke payment page eksternal
- Tidak ada webhook beneran — langsung trigger `user_subscriptions.status = "active"` setelah create
- Ini memungkinkan seluruh flow subscription (upgrade, limit per tier, dsb) sudah bisa di-test end-to-end walau Wuzzpay belum aktif

**Catatan Wuzzpay (untuk referensi nanti saat swap provider):**
- Wuzzpay = payment gateway aggregator, merchant onboarding via KYC (KTP/NPWP/Legalitas Usaha), status bertahap `UNDER_REVIEW` → `APPROVED_PENDING_PAYMENT` → `ACTIVE`
- Ada biaya aktivasi keanggotaan one-time **Rp99.000** sebelum API bisa dipakai
- **Action item paralel (tidak blocking dev):** mulai proses registrasi + KYC Wuzzpay dari sekarang, karena butuh waktu review admin — supaya pas fitur subscription selesai di-develop dengan dummy provider, tinggal swap ke `WuzzpayProvider` begitu akun `ACTIVE`
- Detail format webhook & signature verification perlu dicek lagi di API Reference mereka saat mulai implementasi `WuzzpayProvider`

---

## 21. Fitur: Community Price Watch (Crowdsourced Price Input)

### 21.1 Konsep

Admin/kamu **kurasi item sembako** yang aktif butuh data harga terkini (bukan open submission bebas apapun). Item ini dipublish jadi halaman khusus di user app, user tinggal pilih item → isi harga sesuai kota mereka → submit. Kalau tervalidasi (lihat 21.3), user dapat reward credit.

Ini melengkapi `PriceProvider` (section 5) sebagai implementasi baru: `CrowdsourceProvider`, jalan berdampingan dengan `AIEstimateProvider` — data crowdsource yang sudah tervalidasi lebih diprioritaskan daripada estimasi AI kalau tersedia.

### 21.2 Schema Tambahan

```
price_watch_campaigns
  id, title, description,
  is_active, created_by (admin user_id),
  created_at, updated_at

price_watch_items
  id, campaign_id, ingredient_name, unit (kg/ikat/liter/dll),
  icon_url (opsional), display_order, is_active

price_submissions
  id, watch_item_id, user_id, city_id,
  submitted_price, status (pending/validated/rejected),
  validated_at, created_at

user_credits
  id, user_id, balance, updated_at

credit_transactions
  id, user_id, amount, type (earn_submission/spend_generate/dll),
  reference_id (submission_id atau lainnya), created_at
```

### 21.3 Validasi & Reward (Consensus-Based)

Supaya tidak bisa di-spam demi credit, validasi otomatis pakai konsensus, bukan review manual admin (tidak scalable):

1. Submission baru → status `pending`
2. Sistem cek submission lain untuk **item + kota + rentang waktu sama** (misal 7 hari terakhir)
3. Kalau sudah terkumpul **minimal N submission** (misal 3) dari **user berbeda**, dan nilai-nilainya saling berdekatan (misal dalam rentang ±20% dari median) → **semua submission itu jadi `validated`** sekaligus
4. Rata-rata/median nilai yang tervalidasi → masuk ke `ingredient_price_log` dengan `source = "crowdsource"`, `city_id` terkait
5. Tiap user yang submission-nya tervalidasi → dapat credit (insert row di `credit_transactions`, update `user_credits.balance`)
6. Submission yang jadi outlier (di luar rentang konsensus) → tetap `pending` (nunggu submission baru lain) atau `rejected` kalau sudah melewati batas waktu tanpa konsensus tercapai

**Reward baru cair setelah validasi tercapai**, bukan instan saat submit — mencegah user asal isi angka.

### 21.4 Halaman Admin (Manajemen Campaign)

Di admin panel (repo `apps/admin`, section 18), perlu:
- CRUD `price_watch_campaigns` — buat/edit/nonaktifkan campaign
- CRUD `price_watch_items` dalam campaign — pilih ingredient, unit, urutan tampil, aktif/nonaktif per item
- Dashboard monitoring submission masuk per item/kota (opsional, untuk lihat progress crowdsourcing)

### 21.5 Halaman User (Input Harga)

Halaman baru di user app: list item dari campaign aktif (card dengan nama bahan + unit), user pilih satu → input field harga (auto-fill kota dari profile) → submit. Setelah submit, tampilkan status `pending` — beri tahu user reward akan masuk setelah tervalidasi (transparansi, hindari komplain "kok credit-nya belum masuk").

### 21.6 Pemanfaatan Credit

`user_credits.balance` bisa dipakai untuk (fleksibel, ditentukan nanti):
- Tambahan limit generate menu harian (di luar limit tier subscription)
- Nanti bisa diperluas jadi benefit lain (misal diskon upgrade Premium) — karena schema generic (`credit_transactions.type`), gampang nambah use-case baru tanpa ubah struktur data

### 21.7 API Endpoints Tambahan

```
GET   /api/price-watch/campaigns/active      → list campaign + item aktif untuk user app
POST  /api/price-watch/submissions           → submit harga
GET   /api/price-watch/submissions/me        → riwayat submission user + status

# Admin only
POST   /api/admin/price-watch/campaigns
PATCH  /api/admin/price-watch/campaigns/:id
POST   /api/admin/price-watch/items
PATCH  /api/admin/price-watch/items/:id
GET    /api/admin/price-watch/submissions    → monitoring semua submission
```

## 22. Open Questions (sisa)

1. Threshold konsensus validasi: N minimal submission = 3? Rentang toleransi ±20%? (angka awal, bisa di-tuning setelah lihat data real)
2. Batas waktu submission `pending` sebelum otomatis `rejected` (misal 14 hari tanpa konsensus tercapai)?
3. Nilai credit per submission tervalidasi — flat rate atau bisa beda per campaign (misal campaign "urgent" reward lebih besar)?
4. Detail webhook signature Wuzzpay (dicek saat implementasi provider aslinya)
