# LoanGraph Backend (Phase 1 Bootstrap)

Go Gin API bootstrap with Postgres and SQL migrations.

## Stack
- Go 1.23+
- Gin
- pgx
- PostgreSQL 16
- golang-migrate

## Quickstart
1. Copy env file:

```bash
cp backend/.env.example backend/.env
```

2. Start services:

```bash
docker compose up --build -d postgres api
```

3. Run migrations:

```bash
docker compose run --rm migrate up
```

4. Check endpoints:

```bash
curl http://localhost:8090/health
curl http://localhost:8090/ready
curl http://localhost:8090/v1/meta
```

## Make Commands
From `backend/`:

```bash
make run
make test
make tidy
make migrate-up
make migrate-down
make compose-up
make compose-down
```

## Docs
- OpenAPI: `backend/docs/openapi.yaml`
- Postman collection: `backend/docs/postman/LoanGraph-Backend.postman_collection.json`
- Postman env: `backend/docs/postman/LoanGraph-Local.postman_environment.json`
- Curl guide: `backend/docs/curl.md`

## Notes
- No AWS/S3 integrations are included in this phase.
- Auth implementation is intentionally deferred to next increment.
