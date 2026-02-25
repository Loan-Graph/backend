# LoanGraph Backend

Go Gin API with Postgres, SQL migrations, and cookie-first authentication.

## Stack
- Go 1.23+
- Gin
- pgx
- PostgreSQL 16
- golang-migrate
- JWT (backend sessions)
- Privy access token verification (identity bootstrap)

## Product Positioning (Auth)
- Users onboard with email via Privy.
- Email verification happens in Privy.
- Privy wallet is created behind the scenes (no wallet setup friction).
- Backend issues LoanGraph JWT cookies for API authorization/auditability.

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

## Auth Endpoints
- `POST /v1/auth/privy/login`
- `POST /v1/auth/refresh`
- `POST /v1/auth/logout`
- `GET /v1/auth/me`

## Admin Endpoint (Role-Protected)
- `GET /admin/system/health` (requires `role=admin` in backend auth token)

## Loan Upload Endpoint
- `POST /v1/loans/upload` (requires `role=lender|admin`, multipart CSV with `lender_id` + `file`)
- `GET /v1/loans`
- `GET /v1/loans/:loanId`
- `POST /v1/loans/:loanId/repay`
- `POST /v1/loans/:loanId/default`
- `GET /v1/portfolio/analytics`
- `GET /v1/portfolio/health`
- `GET /v1/passport/:borrowerHash`
- `GET /v1/passport/:borrowerHash/history`
- `GET /v1/passport/:borrowerHash/nft`
- `GET /v1/pools`
- `GET /v1/pools/:poolId`
- `GET /v1/pools/:poolId/performance`
- `GET /v1/lenders/:lenderId/profile`
- `POST /admin/lenders`
- `PATCH /admin/lenders/:lenderId/status`
- `GET /v1/ws` (websocket upgrade)

## Auth Role Bootstrap
- Set `AUTH_BOOTSTRAP_ADMIN_SUBJECT=<privy subject>` in `.env` to promote that Privy subject to admin at login time.

## Make Commands
From `backend/`:

```bash
make run
make run-worker
make run-indexer
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
- Transport is cookie-first for web (`HttpOnly` auth cookies).
- Bearer-token transport is reserved for the mobile phase.
- Outbox worker processes queued chain jobs from `outbox_jobs` (`make run-worker`).
- Indexer processes `chain_events` and applies DB projections (`make run-indexer`).
- WebSocket hub streams pool repayment and lender portfolio events from DB-polled notifier.
- Global request size cap is configurable via `MAX_REQUEST_BODY_BYTES` (defaults to 60 MiB).
- Chain writer mode is configurable via `CHAIN_WRITER_MODE=stub|real`.
- `real` mode currently uses node-managed signing over JSON-RPC (`eth_sendTransaction`) with `CHAIN_WRITER_FROM_ADDRESS` and `CREDITCOIN_HTTP_RPC`.
- Indexer chain ingestion is opt-in via `INDEXER_INGEST_ENABLED=true` and uses `eth_blockNumber`/`eth_getLogs` from `CREDITCOIN_HTTP_RPC` to populate `chain_events` before projections.
