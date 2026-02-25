# Curl Guide

Base URL:

```bash
export BASE_URL="http://localhost:8090"
```

## 1) Health

```bash
curl -i "$BASE_URL/health"
```

Expected:
- HTTP 200
- `{"service":"loangraph-backend","status":"ok"}`

## 2) Ready

```bash
curl -i "$BASE_URL/ready"
```

Expected when DB is reachable:
- HTTP 200
- `{"database":"ok","status":"ready"}`

Expected when DB is down:
- HTTP 503
- `{"database":"error","status":"not_ready"}`

## 3) Meta

```bash
curl -i "$BASE_URL/v1/meta"
```

Expected:
- HTTP 200
- JSON with `name`, `version`, `env`

## 4) Not Found Example

```bash
curl -i "$BASE_URL/does-not-exist"
```

Expected:
- HTTP 404
- `{"error":"not_found"}`
