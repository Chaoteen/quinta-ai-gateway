# Subscription APIs

This document describes the Iteration 10.1A subscription API contract. It documents Subscription Foundation and Alpha management surfaces only. It does not define quota engine, billing, payment, voucher, invoice, or revenue-share behavior.

All endpoints use the project API envelope (`ApiSuccess` / `ApiError`). Error responses are usually JSON objects with `success: false` and a message; callers should not rely only on HTTP status codes.

## DTOs

### Admin Plan DTO

Admin plan APIs return `SubscriptionPlanDTO`:

```json
{
  "plan": {
    "id": 1,
    "code": "alpha_pro",
    "name": "Alpha Pro",
    "description": "Alpha plan",
    "monthly_price": 19.99,
    "yearly_price": 199.99,
    "token_quota": 1000000,
    "request_quota": 10000,
    "model_quota": "",
    "status": "enabled",
    "created_at": 1710000000,
    "updated_at": 1710000000
  }
}
```

Admin plan DTO intentionally does not expose legacy purchase, payment, order, or callback fields such as `price_amount`, `currency`, `duration_unit`, `duration_value`, `custom_seconds`, `enabled`, `sort_order`, `max_purchase_per_user`, `upgrade_group`, `total_amount`, `quota_reset_period`, `quota_reset_custom_seconds`, `stripe_price_id`, `creem_product_id`, `epay`, `payment`, or order callback data.

### Public Plan DTO

Public purchase entry APIs return `PublicSubscriptionPlanDTO`. It embeds the Alpha fields and retains purchase-flow compatibility fields:

```json
{
  "plan": {
    "id": 1,
    "code": "alpha_pro",
    "name": "Alpha Pro",
    "description": "Alpha plan",
    "monthly_price": 19.99,
    "yearly_price": 199.99,
    "token_quota": 1000000,
    "request_quota": 10000,
    "model_quota": "",
    "status": "enabled",
    "created_at": 1710000000,
    "updated_at": 1710000000,
    "title": "Alpha Pro",
    "subtitle": "Alpha plan",
    "price_amount": 19.99,
    "currency": "USD",
    "duration_unit": "month",
    "duration_value": 1,
    "custom_seconds": 0,
    "enabled": true,
    "max_purchase_per_user": 0,
    "upgrade_group": "",
    "total_amount": 1000000,
    "quota_reset_period": "never",
    "quota_reset_custom_seconds": 0,
    "stripe_price_id": "",
    "creem_product_id": ""
  }
}
```

Admin Plan DTO and Public Plan DTO are different DTOs. Public purchase compatibility fields must not be copied into admin plan responses.

### Admin User Subscription DTO

Admin user subscription APIs return `UserSubscriptionRecordDTO`:

```json
{
  "subscription": {
    "id": 1,
    "user_id": 100,
    "plan_id": 1,
    "plan_code": "alpha_pro",
    "plan_name": "Alpha Pro",
    "lifecycle_status": "active",
    "start_time": 1710000000,
    "end_time": 1712592000,
    "token_quota_snapshot": 1000000,
    "request_quota_snapshot": 10000,
    "model_quota_snapshot": "",
    "next_reset_time": 0,
    "created_at": 1710000000,
    "updated_at": 1710000000
  }
}
```

For root callers only, ownership fields may also be present:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

The DTO intentionally excludes `source`, `amount_total`, `amount_used`, `upgrade_group`, `prev_user_group`, and legacy payment/order semantic fields.

### Self Subscription DTO

Self subscription APIs return `SelfSubscriptionRecordDTO`:

```json
{
  "subscription": {
    "plan_code": "alpha_pro",
    "plan_name": "Alpha Pro",
    "lifecycle_status": "active",
    "start_time": 1710000000,
    "end_time": 1712592000,
    "token_quota_snapshot": 1000000,
    "request_quota_snapshot": 10000,
    "model_quota_snapshot": "",
    "next_reset_time": 0,
    "token_quota": 1000000,
    "token_used": 0,
    "token_remaining": 1000000
  }
}
```

Self Subscription DTO does not return a full GORM `UserSubscription` model.

## Admin Subscription Plan APIs

### List Plans

- Method: `GET`
- Path: `/api/subscription/admin/plans`
- Permission: root, tenant_admin, organization_admin, finance, auditor.
- Request params:
  - `page`: optional positive integer, defaults to `1`.
  - `limit`: optional positive integer, defaults to `50`, capped at `200`.
  - `q`: optional search text; matches `code`, `name`, or legacy `title`.
  - `status`: optional, normalized to `enabled` or `disabled`.
- Request body: none.
- Response DTO: array of `SubscriptionPlanDTO`.
- Error cases:
  - Database query failure.
  - Permission failure.
- Notes:
  - The response uses Admin Plan DTO only.
  - It does not expose payment provider IDs or legacy purchase fields.
  - Current implementation allows read access for non-root admin roles listed above; plan mutation remains root-only.

### Create Plan

- Method: `POST`
- Path: `/api/subscription/admin/plans`
- Permission: root only.
- Request params: none.
- Request body:
  - `plan`: Admin Plan DTO input fields: `code`, `name`, `description`, `monthly_price`, `yearly_price`, `token_quota`, `request_quota`, `model_quota`, `status`.
- Response DTO: `SubscriptionPlanDTO`.
- Error cases:
  - Invalid JSON body.
  - Empty `code` or `name`.
  - Negative or out-of-range prices.
  - Negative quota values.
  - Duplicate `code`.
  - Database create failure.
  - Permission failure.
- Notes:
  - Only Alpha management fields are accepted as the admin contract.
  - The model layer still normalizes compatibility fields internally for public purchase compatibility.

### Update Plan

- Method: `PUT`
- Path: `/api/subscription/admin/plans/:id`
- Permission: root only.
- Request params:
  - `id`: positive integer plan id.
- Request body:
  - `plan`: Admin Plan DTO input fields: `code`, `name`, `description`, `monthly_price`, `yearly_price`, `token_quota`, `request_quota`, `model_quota`, `status`.
- Response DTO: `SubscriptionPlanDTO`.
- Error cases:
  - Invalid `id`.
  - Invalid JSON body.
  - Plan not found.
  - Empty `code` or `name`.
  - Negative or out-of-range prices.
  - Negative quota values.
  - Duplicate `code`.
  - Database update failure.
  - Permission failure.
- Notes:
  - Does not accept or return legacy payment/provider fields through the admin DTO.

### Enable Or Disable Plan

- Method: `PATCH`
- Path: `/api/subscription/admin/plans/:id`
- Permission: root only.
- Request params:
  - `id`: positive integer plan id.
- Request body:
  - `enabled`: optional boolean.
  - `status`: optional string, normalized to `enabled` or `disabled`.
- Response DTO: `null`.
- Error cases:
  - Invalid `id`.
  - Invalid JSON body.
  - Missing both `enabled` and `status`.
  - Database update failure.
  - Permission failure.
- Notes:
  - The endpoint updates both legacy `enabled` and Alpha `status` internally.
  - The admin response still does not expose the legacy `enabled` field.

## Admin User Subscription APIs

### List All User Subscriptions

- Method: `GET`
- Path: `/api/subscription/admin/user-subscriptions`
- Permission: root, tenant_admin, organization_admin, finance, auditor.
- Request params:
  - `page`: optional positive integer, defaults to `1`.
  - `limit`: optional positive integer, defaults to `50`, capped at `200`.
  - `status`: optional lifecycle status filter.
  - `user_id`: optional user id filter.
- Request body: none.
- Response DTO: array of `UserSubscriptionRecordDTO`.
- Error cases:
  - Target user is outside the caller's access scope.
  - Database query failure.
  - Permission failure.
- Notes:
  - Query is ownership-scoped for non-root callers.
  - Root responses may include ownership fields. Non-root responses omit them.
  - The DTO does not expose full GORM `UserSubscription` rows.

### List One User's Subscriptions

- Method: `GET`
- Path: `/api/subscription/admin/users/:id/subscriptions`
- Permission: root, tenant_admin, organization_admin, finance, auditor.
- Request params:
  - `id`: positive integer user id.
- Request body: none.
- Response DTO: array of `UserSubscriptionRecordDTO`.
- Error cases:
  - Invalid `id`.
  - User not found or outside caller's access scope.
  - Database query failure.
  - Permission failure.
- Notes:
  - Results are ordered by `end_time desc, id desc`.
  - Root responses may include ownership fields. Non-root responses omit them.

### Assign Subscription To User

- Method: `POST`
- Path: `/api/subscription/admin/users/:id/subscriptions`
- Permission: root and tenant_admin.
- Request params:
  - `id`: positive integer user id.
- Request body:
  - `plan_id`: positive integer plan id.
- Response DTO: `null` or `{ "message": "..." }`.
- Error cases:
  - Invalid `id` or `plan_id`.
  - Caller is not root or tenant_admin.
  - Target user is outside caller's write scope.
  - Plan not found.
  - Plan purchase limit reached.
  - Subscription duration or reset configuration invalid.
  - Database create failure.
- Notes:
  - Non-root writes are tenant scoped.
  - Assignment creates a user subscription snapshot from the selected plan.

### Cancel User Subscription

- Method: `PATCH`
- Path: `/api/subscription/admin/user-subscriptions/:id/cancel`
- Permission: root and tenant_admin.
- Request params:
  - `id`: positive integer user subscription id.
- Request body: none.
- Response DTO: `null` or `{ "message": "..." }`.
- Error cases:
  - Invalid `id`.
  - Caller is not root or tenant_admin.
  - Subscription is outside caller's write scope.
  - Subscription not found.
  - Database update failure.
- Notes:
  - Sets lifecycle to `cancelled` through the model lifecycle path.

### Suspend User Subscription

- Method: `PATCH`
- Path: `/api/subscription/admin/user-subscriptions/:id/suspend`
- Permission: root and tenant_admin.
- Request params:
  - `id`: positive integer user subscription id.
- Request body: none.
- Response DTO: `null` or `{ "message": "..." }`.
- Error cases:
  - Invalid `id`.
  - Caller is not root or tenant_admin.
  - Subscription is outside caller's write scope.
  - Subscription not found.
  - Database update failure.
- Notes:
  - Sets lifecycle to `suspended` through the model lifecycle path.

### Renew User Subscription

- Method: `PATCH`
- Path: `/api/subscription/admin/user-subscriptions/:id/renew`
- Permission: root and tenant_admin.
- Request params:
  - `id`: positive integer user subscription id.
- Request body: none.
- Response DTO: `UserSubscriptionRecordDTO`.
- Error cases:
  - Invalid `id`.
  - Caller is not root or tenant_admin.
  - Subscription is outside caller's write scope.
  - Subscription or plan not found.
  - Subscription duration or reset configuration invalid.
  - Database create/update failure.
- Notes:
  - Renewal creates a new subscription from the original subscription's plan.
  - Ownership is copied from the original subscription.

## Self Subscription APIs

### Get Self Subscription State

- Method: `GET`
- Path: `/api/subscription/self`
- Permission: authenticated user.
- Request params: none.
- Request body: none.
- Response DTO:
  - `billing_preference`: normalized user preference.
  - `subscriptions`: active subscriptions, array of `SelfSubscriptionRecordDTO`.
  - `all_subscriptions`: all subscriptions including expired records, array of `SelfSubscriptionRecordDTO`.
- Error cases:
  - Authentication failure.
  - User settings lookup may fall back to default preference if unavailable.
  - Subscription lookup errors return empty arrays in the current implementation.
- Notes:
  - This endpoint does not return `model.SubscriptionSummary` directly.
  - This endpoint does not return a full GORM `UserSubscription` model.
  - User-visible remaining quota uses `token_quota`, `token_used`, and `token_remaining`.

## Public Subscription Plans API

### List Public Plans

- Method: `GET`
- Path: `/api/subscription/plans`
- Permission: authenticated user.
- Request params: none.
- Request body: none.
- Response DTO: array of `PublicSubscriptionPlanDTO`.
- Error cases:
  - Authentication failure.
  - Database query failure.
- Notes:
  - Returns only plans where legacy `enabled = true`.
  - Keeps purchase-flow fields such as `price_amount`, `currency`, `duration_unit`, `duration_value`, `stripe_price_id`, `creem_product_id`, and `enabled`.
  - These public purchase fields are intentionally limited to the public purchase DTO and are not part of the admin Alpha plan DTO.
