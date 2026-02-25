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
