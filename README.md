# GuzoPay

Contactless, hardware-free transit fare payment MVP for the M-Pesa Ethiopia Hackathon 2026.

## Stack

- **Backend:** Go + Gin + GORM (SQLite by default, PostgreSQL via `DATABASE_URL`) + WebSockets
- **Passenger:** React + Tailwind (M-Pesa Mini-App style)
- **Driver:** React + Tailwind + Lucide PWA dashboard

### Daraja sandbox (passenger → GuzoPay → driver payout)

Wired against `https://apisandbox.safaricom.et` (token, C2B register/simulate, B2C payout).

1. Put keys in `backend/.env` (see `.env.example`). **Never commit `.env`.**
2. Set `MPESA_SHORTCODE`, initiator + security credential from the portal.
3. `ngrok http 8080` → `CALLBACK_BASE_URL=https://….ngrok-free.app`
4. `curl -X POST localhost:8080/api/v1/mpesa/register-urls`
5. Pay: `POST /api/v1/pay/daraja` → simulate C2B → confirmation webhook credits ledger.
6. Withdraw: driver PWA → B2C to driver phone.

### M-Pesa HTML5 Mini App (Ethiopia)

1. Create **Mini Program Based on HTML5** → Entrance URL = deployed passenger app.
2. Enable features **PaymentET** + **UserScopes**.
3. Inside SuperApp the passenger app uses:
   - `call('userScopes', …)` for name/phone
   - `call('payWithMpesa', { businessID, billReference, amount, currency: 'ETB' })`
4. Backend C2B confirmation still credits the driver ledger (`BillRefNumber` = vehicle code).
5. Browser preview falls back to Daraja simulate or local `POST /pay`.

## Seed drivers

API start runs `SeedDrivers()`. Re-run: `cd backend && go run ./cmd/seed`  
Fleet: `4A2Q`, `4421`, `A402`. Driver PWA loads them via `GET /api/v1/drivers`.

Passenger identity: Mini App `userScopes`, or typed `07…` number. C2B confirmation uses Safaricom First/Last name. Money leaves the commuter’s M-Pesa MSISDN.

**Torch / feature phones:** cannot run Mini App — Pay Bill to shortcode with Bill Ref = vehicle code (shown in passenger UI). STK push not wired yet (needs `MPESA_PASSKEY`).

## Quick start (Docker — recommended)

**Prerequisites:** [Docker Engine](https://docs.docker.com/engine/install/) or Docker Desktop, and Docker Compose v2 (`docker compose`). You do not need Go, Node.js, or PostgreSQL installed on the host.

```bash
git clone https://github.com/thisAbdU/guzopay.git
cd guzopay
cp .env.example .env   # optional; defaults match docker-compose.yml
docker compose up --build
```

| Service | URL |
|---------|-----|
| Backend API | http://localhost:8080 |
| Passenger mini-app | http://localhost:5173 |
| Driver PWA | http://localhost:5174 |
| PostgreSQL (host) | `localhost:5432` (user/db/password: `guzopay`) |

Follow logs: `docker compose logs -f`
Stop the stack: `docker compose down` (named volume `pgdata` is kept; database data survives a normal `down`).

### Docker development workflow

- **Rebuild one service** after Dockerfile or dependency changes:
  `docker compose up --build backend` (or `driver-pwa`, `passenger-miniapp`).
- **Logs for one service:** `docker compose logs -f backend`
- **Restart after env changes:** `docker compose up -d --force-recreate backend`
- **Hot reload:** Frontends mount source from the host; edits under `driver-pwa/` and `passenger-miniapp/` reload via Vite. The Go backend image must be **rebuilt** for backend code changes: `docker compose up --build backend`.
- **New npm/pnpm packages:** Rebuild the frontend image so `pnpm install` runs again:
  `docker compose build --no-cache driver-pwa` (or `passenger-miniapp`).

Browser requests use `VITE_API_URL` / `VITE_WS_URL` (default `http://localhost:8080` and `ws://localhost:8080`) because JavaScript runs on your machine, not inside the frontend container. The backend connects to PostgreSQL using the Compose service hostname `postgres` via `DATABASE_URL`.

### Database (Docker stack)

- **Connection:** The backend reads `DATABASE_URL` (libpq DSN). In Compose, the default is
  `host=postgres user=guzopay password=guzopay dbname=guzopay port=5432 sslmode=disable`.
- **Schema & seed:** On startup, GORM `AutoMigrate` runs for `Driver` and `Payment`, then seeds demo taxi **4A2Q** / Dawit Alemu if missing (`backend/database/db.go`). There is no separate migration CLI.
- **Shell access:**
  `docker compose exec postgres psql -U guzopay -d guzopay`

### Troubleshooting

- **Port already in use:** Stop local processes on 5432, 8080, 5173, or 5174, or change host port mappings in `docker-compose.yml`.
- **Backend exits immediately:** Wait for PostgreSQL healthcheck; check `docker compose logs backend`. Ensure `DATABASE_URL` uses `host=postgres`, not `localhost`, inside Compose.
- **Frontends cannot reach API:** Confirm the API at http://localhost:8080/api/v1/taxi/4A2Q and set `VITE_API_URL` / `VITE_WS_URL` in `.env` if you use non-default ports.
- **Stale frontend dependencies:** Rebuild with `docker compose build --no-cache <service>`.

## Quick start (local tools)

```bash
cd backend && go run .

cd passenger-miniapp && pnpm i && pnpm dev   # :5173
cd driver-pwa && pnpm i && pnpm dev          # :5174
```

Optional PostgreSQL only (without full stack):

```bash
docker compose up -d postgres
export DATABASE_URL='host=localhost user=guzopay password=guzopay dbname=guzopay port=5432 sslmode=disable'
cd backend && go run .
```

## Demo flow

1. Open driver PWA → enable **Proximity** and **Audio**.
2. Open passenger mini-app → nearby taxi `#4A2Q` appears (or type `4A2Q`).
3. Tap **Pay 15.00 ETB via M-Pesa** → receipt on passenger; live row + ding on driver.

## API

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/taxi/:code` | Driver/route for vehicle code |
| GET | `/api/v1/taxi/nearby` | Taxis with proximity enabled |
| POST | `/api/v1/pay` | Local mock complete (no Daraja) |
| POST | `/api/v1/pay/daraja` | Sandbox simulate C2B + PENDING payment |
| GET | `/api/v1/pay/:id` | Poll payment status |
| POST | `/api/v1/pay/initiate` | Mini App order stub (`tradeNO`) |
| POST | `/api/v1/pay/confirm` | Complete after `tradePay` |
| POST | `/api/v1/mpesa/register-urls` | Register C2B validation/confirmation |
| POST | `/api/v1/mpesa/c2b-validation` | C2B validation listener |
| POST | `/api/v1/mpesa/c2b-callback` | C2B confirmation → credit ledger |
| POST | `/api/v1/mpesa/b2c-result` | B2C payout result |
| POST | `/api/v1/driver/:code/payout` | Withdraw ledger → driver M-Pesa |
| GET | `/api/v1/driver/:code/dashboard` | Wallet, today totals, last 20 txs |
| POST | `/api/v1/driver/:code/proximity` | `{ "enabled": true }` |
| PUT | `/api/v1/driver/:code/fare-tiers` | `{ "fare_tiers": [{ "name", "amount" }] }` |
| GET | `/ws/driver/:code` | Real-time payment events |
