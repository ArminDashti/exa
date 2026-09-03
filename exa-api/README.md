# Exa API

Expense tracker backend (Gin + PostgreSQL).

## Run locally

```powershell
copy .env.example .env
docker compose -f docker-compose.local.yml up -d
```

API: `http://127.0.0.1:8197/health`

Default login (seeded): `armin` / `dopadopa123`

## Endpoints

Base: `/api/v1`

- `POST /auth/login`
- `GET/POST/PATCH/DELETE /stores`
- `GET/POST/PATCH/DELETE /expenses` (filters: `from`, `to`, `store_id`, `limit`)
- `GET /stats/summary` (filters: `from`, `to`)
- `GET /stats/by-store` (filters: `from`, `to`)
