# API Contract — meal-planner-api

Base URL local: `http://localhost:8080/api/v1`  
Production URL: `https://masakapa-api.fly.dev/api/v1`  
Auth: Bearer token (Supabase JWT) di header `Authorization`, kecuali endpoint yang ditandai *(public)*.

---

## Auth, User & Onboarding

```http
GET  /cities (public)
  → List kota terdaftar untuk onboarding & crowdsource price watch

GET  /me
  → Detail profile user login (id, email, name, role, city_id)

GET  /preferences
  → User budget & preference profile

POST /onboarding
  body: { goal, budget_amount, budget_period, household_size, restrictions[], city_id }
  → Simpan/update user_preferences
```

---

## Home & Menu Recommendation

```http
GET  /home/suggestion
  → { greeting: string, budget_remaining: int, cta: "generate_menu" }

POST /menu/generate
  → Rate-limited sesuai subscription_plans.features.max_generate_per_day
  → Response:
  {
    "data": {
      "options": [
        {
          "recipe_name": string,
          "description": string,
          "estimated_total_price": int,
          "goal_tags": string[],
          "ingredients": [
            { "name": string, "quantity": string, "unit": string, "estimated_price": int, "price_source": string }
          ]
        }
      ]
    },
    "error": null
  }

GET  /menu/latest
  → Ambil opsi rekomendasi menu AI terbaru milik user

POST /menu/select
  body: { recipe_name, description, estimated_total_price, ingredients[] }
  → Generate shopping_list & meal_selection, return { shopping_list_id }

GET  /history
  → Riwayat meal_selections user (butuh plan dengan history_access = true)

DELETE /history/:id
  → Hapus item riwayat pilihan menu milik user (id = meal_selection_id)
```

---

## Shopping List

```http
GET   /shopping-list/:id
  → Ambil detail shopping list dan items

PATCH /shopping-list/:id/item
  body: { item_index, checked }
  → Update status checklist item belanja
```

---

## Subscription & Payment

```http
GET  /subscription/plans (public)
  → List paket langganan (Free vs Premium)

POST /subscription/subscribe
  body: { plan_id, coupon_code? }
  → Create payment_transaction via active PaymentProvider (Dummy di MVP)

POST /webhook/wuzzpay (public, verifikasi signature)
  → Update payment_transactions.status, aktifkan user_subscriptions jika success
```

---

## Community Price Watch (Crowdsource) & Credits

```http
GET  /price-watch/campaigns/active
  → List campaign + item aktif untuk user isi harga

POST /price-watch/submissions
  body: { watch_item_id, submitted_price }
  → city_id diambil dari users.city_id
  → status awal "pending", validasi konsensus jalan async/background job

GET  /price-watch/submissions/me
  → Riwayat submission user + status + credit earned

GET  /credits/me
  → Saldo credit user (reward crowdsourcing)
```

---

## Admin Only (butuh Bearer token + role = admin)

```http
GET   /admin/master-ingredients?category=...&search=...
POST  /admin/master-ingredients
  body: { category, name, default_unit, aliases[] }
PUT   /admin/master-ingredients/:id
  body: { category, name, default_unit }
POST  /admin/master-ingredients/:id/aliases
  body: { alias_name }
DELETE /admin/master-ingredients/aliases/:alias_id

POST  /admin/price-watch/campaigns
PATCH /admin/price-watch/campaigns/:id
POST  /admin/price-watch/items
PATCH /admin/price-watch/items/:id
GET   /admin/price-watch/submissions

GET   /admin/subscription-plans
PATCH /admin/subscription-plans/:id

POST  /admin/coupons
PATCH /admin/coupons/:id

GET   /admin/ai/configs
POST  /admin/ai/configs/select
  body: { "provider_name": "groq" }
  → Switch active AI provider (openai | groq | gemini)
```

---

## Format Response Standar

Sukses:
```json
{
  "data": { ... },
  "error": null
}
```

Error:
```json
{
  "data": null,
  "error": {
    "message": "Pesan deskripsi error",
    "code": "ERROR_CODE"
  }
}
```
