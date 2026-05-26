# go-budget — Claude Context

## What This Service Is

The budgeting service for the personal-enterprise project. Manages accounts, bills, and transactions. Backed by MongoDB Atlas. Validates JWTs issued by go-auth — does not issue tokens.

Built from `go-service-template`. The structure, patterns, and tooling are inherited from that template — this file documents what is specific to go-budget on top of that foundation.

---

## Architecture

```
cmd/
  server/main.go     ← wiring: config, DB, stores, server.New()
app/
  app.go             ← Application struct: AccountStore, BillStore, TransactionStore, CategoryStore
server/
  server.go          ← chi router, global middleware
  routes.go          ← route registration — all routes auth-protected
config/
  config.go          ← standard template config, no additions needed
db_connection/
  db.go              ← MongoDB Connect() — returns (*mongo.Client, *mongo.Database), creates indexes
health/
  handler.go
middleware/
  auth.go
  logging.go
  requestid.go
account/
  account_model.go
  account_store.go
  account_handler.go
  account_routes.go
bill/
  bill_model.go
  bill_store.go
  bill_handler.go
  bill_routes.go
transaction/
  transaction_model.go
  transaction_store.go
  transaction_handler.go
  transaction_routes.go
category/
  category_model.go
  category_store.go
  category_handler.go
  category_routes.go
```

---

## Domain Conventions

### Monetary Values

All monetary amounts stored as integers (cents). Never `float64` for money. Frontend handles display formatting.

### All Routes Are Protected

Every domain in this service requires authentication. `authMiddleware` is applied to all mounts in `server/routes.go` — no unprotected endpoints except `/health`.

---

## Database

MongoDB Atlas. Database name: `budget`. Collections: `accounts`, `bills`, `categories`, `transactions`.

### Store pattern

Stores define a private doc struct with `bson` tags for MongoDB serialization, and a domain struct (in `_model.go`) with plain Go types for everything outside the store. The store's methods convert between the two via `toDomain()`. This keeps bson tags out of the domain layer.

IDs are stored as strings (`uuid.UUID.String()`), parsed back to `uuid.UUID` on read.

### Indexes (created at startup in db_connection/db.go)

- `accounts.userId`
- `bills.userId`
- `categories.userId`
- `transactions.userId`, `transactions.billMonth`, `transactions.date`

### Cascade behavior (application-layer, no DB enforcement)

- **Delete account**: nulls `transactions.sourceId` and `transactions.destinationId`; deletes bills with `sourceId = accountId`; nulls `transactions.billId` for those bills
- **Delete bill**: nulls `transactions.billId` where `billId = billId`
- **Delete category**: no cascade

### Dates

Transaction `date` stored as a YYYY-MM-DD string. String comparison is correct for ISO dates, so MongoDB range filters work without conversion.

### Connection string

`DATABASE_URL` is a MongoDB URI (`mongodb+srv://user:pass@cluster.mongodb.net/`). No database name in the URI — database selected in code via `client.Database("budget")`.

---

## Patterns

### Domain Structure

Four-file pattern: `<domain>_model.go`, `_store.go`, `_handler.go`, `_routes.go`. No service layer needed — handler → store is sufficient for all domains here.

### Logging

`slog.SetDefault(logger)` in main. JSON to stdout. All packages use `slog` directly.

### Testing

Integration tests via testcontainers — `mongo:6` container, no mocks. `TestMain` handles container lifecycle. (Not yet written.)

### Conventions

- File naming: `account_handler.go`, `bill_store.go`, etc.
- Receiver names: `h` for handlers, `s` for stores
- Errors: log with `slog.Error` server-side, generic message to client
- Routes function: `Routes(store *Store) chi.Router`
- User ID extracted from context via `middleware.UserIDFromContext` — all queries scoped to the authenticated user

---

## Environment Variables

| Variable | Description |
|---|---|
| `PORT` | HTTP port (defaults to 8080) |
| `DATABASE_URL` | MongoDB URI (`mongodb+srv://user:pass@cluster.mongodb.net/`) |
| `JWT_PUBLIC_KEY` | RSA public key PEM for validating JWTs issued by go-auth |
| `ALLOWED_ORIGINS` | Comma-separated list of allowed CORS origins |

Copy `.env.example` to `.env.local` for local dev. Never commit `.env.local`.

---

## Current State

- `account/`, `bill/`, `transaction/`, `category/` domains complete
- MongoDB migration complete — no Postgres, no sqlc, no golang-migrate
- Deployed to dev (Postgres). Needs redeployment after Cloud SQL attachment removed and MongoDB Atlas wired.
- Cloud Run dev URL: `https://go-budget-dev-iwpkmztv2a-uc.a.run.app`

---

## Open Decisions

- A `template` domain (transaction templates / presets) may be needed — leave out until the use case is clarified.
