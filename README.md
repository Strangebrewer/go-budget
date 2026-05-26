# go-budget

Budgeting service for the personal-enterprise project, nomsayn? Manages accounts, bills, and transactions. All monetary values are stored as integers (cents) — formatting is handled on the frontend.

> Active development — endpoints and schema documented once stable.

---

## Stack

- **Language**: Go
- **Router**: [chi](https://github.com/go-chi/chi)
- **Database**: Postgres via [pgx](https://github.com/jackc/pgx)
- **Query generation**: [sqlc](https://sqlc.dev)
- **Migrations**: [golang-migrate](https://github.com/golang-migrate/migrate)
- **Auth**: Stateless RSA JWT validation — tokens issued by go-auth, verified independently here
- **Logging**: `slog` with JSON output

---

## Running Locally

Copy `.env.example` to `.env.local` and fill in values.

```bash
# Start the server
go run ./cmd/server

# Run migrations
go run ./cmd/migrate up
go run ./cmd/migrate down   # rolls back one step

# Run tests
go test ./...
```

---

## Environment Variables

| Variable | Description |
|---|---|
| `PORT` | HTTP port (defaults to 8080) |
| `DATABASE_URL` | Postgres connection string (`postgres://user:pass@host/budget`) |
| `JWT_PUBLIC_KEY` | RSA public key PEM for validating JWTs issued by go-auth |
| `ALLOWED_ORIGINS` | Comma-separated list of allowed CORS origins |
