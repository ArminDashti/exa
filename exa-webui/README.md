# Exa WebUI

Personal expense tracker (Vue 3 + shadcn + PWA).

## Run locally

```powershell
npm install
npm run dev
```

Open `http://127.0.0.1:5197`

Ensure `exa-api` is running on port **8197** (proxy handles `/api` in dev).

Default login: `armin` / `dopadopa123`

## Pages

- Dashboard — totals and recent expenses
- Add Expense — date, store, amount, optional note
- Stores — CRUD
- Stats — date-range summary and per-store breakdown
- About Me — `/about-me`
