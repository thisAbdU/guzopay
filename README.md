# GuzoPay

Contactless, hardware-free transit fare payment MVP for the M-Pesa Ethiopia Hackathon 2026.

## Stack

- **Backend:** Go + Gin + GORM (SQLite by default, PostgreSQL via `DATABASE_URL`) + WebSockets
- **Passenger:** React + Tailwind (M-Pesa Mini-App style)
- **Driver:** React + Tailwind + Lucide PWA dashboard

Real Daraja integration target: [mpesa-go-sdk](https://github.com/Safaricom-Ethiopia-PLC/mpesa-go-sdk) — C2B callback stub is at `backend/mpesa/c2b_webhook.go`.

### M-Pesa Mini App (passenger)

Aligned with [Safaricom Mini App docs](https://developer.safaricom.et/mini-app-documentation) (HTML5 WebView + JSBridge):

1. Load `hylid-bridge` in `passenger-miniapp/index.html`
2. Detect `MiniProgram` UserAgent → `my.getAuthCode` → `POST /api/v1/mpesa/auth`
3. Pay: `POST /api/v1/pay/initiate` → `my.tradePay({ tradeNO })` → `POST /api/v1/pay/confirm`
4. Browser preview falls back to simulated `POST /api/v1/pay` (JSAPIs unavailable outside SuperApp)

Register the deployed passenger URL as the mini program **Entrance URL** on the platform.

## Quick start

```bash
# API (port 8080) — seeds taxi 4A2Q / Dawit Alemu
cd backend && go run .

# Passenger mini-app (port 5173)
cd passenger-miniapp && pnpm i && pnpm dev

# Driver PWA (port 5174)
cd driver-pwa && pnpm i && pnpm dev
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
| POST | `/api/v1/pay` | Browser-simulated fare payment |
| POST | `/api/v1/pay/initiate` | Create PENDING order + `tradeNO` for `my.tradePay` |
| POST | `/api/v1/pay/confirm` | Complete after `tradePay` result `9000` |
| POST | `/api/v1/mpesa/auth` | Exchange mini-app `auth_code` (stub → applyToken) |
| POST | `/api/v1/mpesa/notify-payment` | Wallet `notifyPayment` webhook |
| POST | `/api/v1/mpesa/c2b-callback` | Daraja C2B confirmation |
| GET | `/api/v1/driver/:code/dashboard` | Wallet, today totals, last 20 txs |
| POST | `/api/v1/driver/:code/proximity` | `{ "enabled": true }` |
| PUT | `/api/v1/driver/:code/fare-tiers` | `{ "fare_tiers": [{ "name", "amount" }] }` |
| GET | `/ws/driver/:code` | Real-time payment events |
