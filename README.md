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

## Quick start

```bash
cd backend && go run .

cd passenger-miniapp && pnpm i && pnpm dev   # :5173
cd driver-pwa && pnpm i && pnpm dev          # :5174
```

Optional PostgreSQL:

```bash
docker compose up -d
export DATABASE_URL='host=localhost user=guzopay password=guzopay dbname=guzopay port=5432 sslmode=disable'
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
