# Curl Guide

Base URL:

```bash
export BASE_URL="http://localhost:8090"
```

## 1) Health

```bash
curl -i "$BASE_URL/health"
```

## 2) Ready

```bash
curl -i "$BASE_URL/ready"
```

## 3) Meta

```bash
curl -i "$BASE_URL/v1/meta"
```

## 4) Login with Privy token

```bash
curl -i -c cookies.txt \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/v1/auth/privy/login" \
  -d '{"privy_access_token":"<PRIVY_ACCESS_TOKEN>"}'
```

Expected:
- HTTP 200
- `lg_access` and `lg_refresh` cookies set

## 5) Get current user

```bash
curl -i -b cookies.txt "$BASE_URL/v1/auth/me"
```

Expected:
- HTTP 200 with user object

## 6) Refresh session

```bash
curl -i -b cookies.txt -c cookies.txt -X POST "$BASE_URL/v1/auth/refresh"
```

Expected:
- HTTP 200
- rotated cookies

## 7) Logout

```bash
curl -i -b cookies.txt -c cookies.txt -X POST "$BASE_URL/v1/auth/logout"
```

Expected:
- HTTP 200
- auth cookies cleared

## 8) Admin endpoint (requires admin role)

```bash
curl -i -b cookies.txt "$BASE_URL/admin/system/health"
```

Expected:
- HTTP 200 for admin user
- HTTP 403 for non-admin user

## 9) Upload loan CSV (requires lender/admin role)

```bash
curl -i -b cookies.txt \
  -X POST "$BASE_URL/v1/loans/upload" \
  -F "lender_id=<LENDER_UUID>" \
  -F "file=@./sample-loans.csv;type=text/csv"
```

Expected:
- HTTP 200 with `{ loan_ids, processed, errors: [] }` for valid CSV
- HTTP 400 with row-level `errors` for invalid CSV rows

## 10) List loans

```bash
curl -i -b cookies.txt "$BASE_URL/v1/loans?lender_id=<LENDER_UUID>&status=active&limit=20&offset=0"
```

## 11) Get loan by id

```bash
curl -i -b cookies.txt "$BASE_URL/v1/loans/<LOAN_ID>"
```

## 12) Record repayment

```bash
curl -i -b cookies.txt \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/v1/loans/<LOAN_ID>/repay" \
  -d '{"amount_minor":50000,"currency":"NGN"}'
```

## 13) Mark default

```bash
curl -i -b cookies.txt \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/v1/loans/<LOAN_ID>/default" \
  -d '{"reason":"missed scheduled payments"}'
```

## 14) Portfolio analytics

```bash
curl -i -b cookies.txt "$BASE_URL/v1/portfolio/analytics?lender_id=<LENDER_UUID>"
```

## 15) Portfolio health

```bash
curl -i -b cookies.txt "$BASE_URL/v1/portfolio/health?lender_id=<LENDER_UUID>"
```

## 16) Passport snapshot by borrower hash

```bash
curl -i -b cookies.txt "$BASE_URL/v1/passport/<BORROWER_HASH_HEX>"
```

## 17) Passport history

```bash
curl -i -b cookies.txt "$BASE_URL/v1/passport/<BORROWER_HASH_HEX>/history?limit=20&offset=0"
```

## 18) Passport NFT view

```bash
curl -i -b cookies.txt "$BASE_URL/v1/passport/<BORROWER_HASH_HEX>/nft"
```

## 19) List pools

```bash
curl -i -b cookies.txt "$BASE_URL/v1/pools?currency=NGN&status=open&limit=20&offset=0"
```

## 20) Get pool

```bash
curl -i -b cookies.txt "$BASE_URL/v1/pools/<POOL_ID>"
```

## 21) Get pool performance

```bash
curl -i -b cookies.txt "$BASE_URL/v1/pools/<POOL_ID>/performance?days=30"
```

## 22) Get lender profile

```bash
curl -i -b cookies.txt "$BASE_URL/v1/lenders/<LENDER_ID>/profile"
```

## 23) Admin onboard lender (admin role)

```bash
curl -i -b cookies.txt \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/admin/lenders" \
  -d '{"name":"New Lender","country_code":"NG","wallet_address":"0x8888888888888888888888888888888888888888"}'
```

## 24) Admin update lender status (admin role)

```bash
curl -i -b cookies.txt \
  -H "Content-Type: application/json" \
  -X PATCH "$BASE_URL/admin/lenders/<LENDER_ID>/status" \
  -d '{"kyc_status":"approved"}'
```
