# Brand Cleanup Audit

## Scope

Search terms:

- `New API`
- `NewAPI`
- `new-api`
- `One API`
- `OneAPI`
- `one-api`
- `docs.newapi.pro`
- `github.com/songquanpeng/one-api`

Searched paths:

- `web/default`
- `docs`
- `README*`
- landing page, footer, header, about, help, docs, FAQ, support, community, partner, SEO-facing frontend and documentation surfaces

## User-Visible Residuals Found And Fixed

The following user-visible residuals were found and replaced or removed:

| Area | Before | Action |
| --- | --- | --- |
| Main README | Legacy project-name references in product positioning | Reworded as Quinta AI Gateway product positioning |
| Multilingual README files | Legacy project badges, docs links, Docker examples, old project links | Replaced with concise Quinta AI Gateway project descriptions |
| `docs/installation/BT.md` | Legacy product name, old Docker image examples, `docs.newapi.pro` links | Rewritten as Quinta AI Gateway deployment guidance |
| Footer demo links | `docs.newapi.pro` links | Replaced with local `/about` and `/pricing` links |
| Footer related links | `One API` label and `github.com/songquanpeng/one-api` link | Replaced with Quinta AI Gateway repository link |
| About empty state | `One API` attribution link | Replaced with Quinta AI Gateway platform description |
| i18n locale files | Unused `One API` footer and standalone keys | Removed and replaced with Quinta AI Gateway footer keys |
| Update checker | Old upstream release endpoint and user agent | Changed to Quinta AI Gateway release endpoint and dashboard user agent |
| Logo SVG | Legacy DOM id | Renamed to Quinta AI Gateway DOM id |

Final user-visible check:

- `README*`: no user-visible legacy brand/link residuals remain.
- `docs/installation`: no legacy brand/link residuals remain.
- `web/default/index.html`, frontend layout/header/footer/about/home/i18n locale surfaces: no `docs.newapi.pro`, `github.com/songquanpeng/one-api`, standalone `One API`, `New API`, `NewAPI`, or `new-api` product-brand residuals remain.
- Footer no longer links to `docs.newapi.pro`.
- Footer no longer displays or links to `One API`.
- Landing page default title and description remain `Quinta AI Gateway` / `Enterprise AI Gateway & MaaS Platform`.

## Developer-Visible Residuals

These are not user-facing product branding and were not changed:

| File | Residual | Reason |
| --- | --- | --- |
| `web/default/package.json` | package name `newapi-web` | Package metadata/name is outside user-visible brand cleanup scope |
| `web/default/bun.lock` | package lock entry `newapi-web` | Lockfile mirrors package name |
| `web/default/src/features/keys/api.ts` | comment text `Create a new API key` | Generic phrase, not product branding |
| `web/default/src/i18n/locales/*.json` | phrases like `Add a new API key...`, `Enter one API key...` | Generic English grammar for API-key workflows, not `One API` brand usage |

## Technical Path Residuals

These are compatibility/API identifiers and were not changed:

| File | Residual | Reason |
| --- | --- | --- |
| `web/default/src/lib/api.ts` | `New-Api-User` request header | Existing backend/header compatibility contract |
| `docs/openapi/api.json` | `NewApiUser`, `New-Api-User` schema/security entries | Generated OpenAPI technical contract |
| `docs/openapi/relay.json` | `NewApiUser`, `New-Api-User` schema/security entries | Generated OpenAPI technical contract |
| `web/default/src/features/chat/lib/send-to-fluent.ts` | `fluent-new-api-container`, `id: 'new-api'` | Fluent/chat integration protocol identifier |
| `web/default/src/features/chat/lib/chat-links.ts` | `id: 'new-api'`, `platform: 'new-api'` | Chat integration protocol identifier |
| `web/default/src/features/keys/components/data-table-row-actions.tsx` | `_type: 'newapi_channel_conn'` | Client integration payload type |
| `web/default/src/features/system-settings/models/constants.ts` | `/llm-metadata/api/newapi/ratio_config-v1-base.json` | Static metadata path |

## External Link Audit

Removed from user-visible frontend/docs:

- `docs.newapi.pro`
- `github.com/songquanpeng/one-api`

Remaining external links that include legacy terms are limited to none in user-visible frontend/docs after cleanup. OpenAPI generated contracts and protocol identifiers do not create navigation links.

## i18n

Updated `en`, `zh`, `fr`, `ja`, `ru`, and `vi` locale files:

- Added `footer.gateway.projectAttributionSuffix`.
- Added `footer.columns.related.links.gateway`.
- Added `| Enterprise AI Gateway & MaaS Platform`.
- Removed unused `One API` and legacy footer attribution keys.

`npm run i18n:sync` completed with `missingCount: 0` for all locales.

## Verification Commands

```bash
rg -n -i "New API|NewAPI|new-api|One API|OneAPI|one-api|docs\\.newapi\\.pro|github\\.com/songquanpeng/one-api" web/default docs README* --hidden --glob '!web/default/node_modules/**' --glob '!web/default/dist/**'
rg -n "docs\\.newapi\\.pro|github\\.com/songquanpeng/one-api|One API|New API|NewAPI|new-api" README* docs/installation web/default/src/components web/default/src/features/home web/default/src/features/about web/default/src/components/layout web/default/index.html web/default/src/i18n/locales --hidden --glob '!**/_reports/**'
npm run i18n:sync
```

The broad search still reports technical-path residuals listed above. The user-visible focused search returns no legacy brand/link matches.

## Items Intentionally Not Modified

- Go module path.
- Package/import paths.
- Backend request header compatibility.
- Generated OpenAPI security scheme names.
- Protocol ids used by existing client/chat integrations.
