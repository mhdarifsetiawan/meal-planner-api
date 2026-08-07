# API Contract — meal-planner-api

Base URL local: `http://localhost:8080/api`
Auth: Bearer token (Supabase JWT) di header `Authorization`, kecuali endpoint yang ditandai *public*.

## Auth & Onboarding

```
POST /onboarding
  body: { goal, budget_amount, budget_period, household_size, restrictions[], city_id }
  → simpan/update user_preferences
```

## Home & Menu

```
GET  /home/suggestion
  → { greeting: string, budget_remaining: int, cta: "generate_menu" }

POST /menu/generate
  → rate-limited sesuai subscription_plans.features.max_generate_per_day
  → response:
  {
    "options": [
      {
        "recipe_id": int,
        "recipe_name": string,
        "description": string,
        "estimated_total_price": int,
        "ingredients": [
          { "name": string, "qty": string, "estimated_price": int, "price_source": "ai_estimate"|"crowdsource" }
        ]
      }
    ]
  }

POST /menu/select
  body: { recipe_id, selected_date }
  → generate shopping_list, return shopping_list_id

GET /history
  → riwayat meal_selections user (butuh plan dengan history_access = true)
```

## Shopping List

```
GET   /shopping-list/:id
PATCH /shopping-list/:id/item
  body: { item_index, checked }
```

## Subscription & Payment

```
GET  /subscription/plans                (public)
POST /subscription/subscribe
  body: { plan_id, coupon_code? }
  → create payment_transaction via active PaymentProvider (Dummy di MVP)

POST /webhook/wuzzpay                   (public, verifikasi signature)
  → update payment_transactions.status, aktifkan user_subscriptions kalau success
```

## Community Price Watch

```
GET  /price-watch/campaigns/active
  → list campaign + item aktif untuk user isi harga

POST /price-watch/submissions
  body: { watch_item_id, submitted_price }
  → city_id diambil dari users.city_id
  → status awal "pending", validasi konsensus jalan async/background job

GET  /price-watch/submissions/me
  → riwayat submission user + status + credit earned
```

## Admin Only (butuh role = admin)

```
POST  /admin/price-watch/campaigns
PATCH /admin/price-watch/campaigns/:id
POST  /admin/price-watch/items
PATCH /admin/price-watch/items/:id
GET   /admin/price-watch/submissions

GET   /admin/subscription-plans
PATCH /admin/subscription-plans/:id     (ubah harga/features tanpa redeploy)

POST  /admin/coupons
PATCH /admin/coupons/:id

GET   /admin/ai-provider-config
PATCH /admin/ai-provider-config/:id     (switch provider aktif)
```

## Format Response Standar

Sukses:
```json
{ "data": { ... }, "error": null }
```

Error:
```json
{ "data": null, "error": { "message": "..." } }
```
