# WebSocket Notifications — Mobile Frontend Contract

Real-time push channel the backend uses to tell a logged-in user's mobile app
about **price updates** and **status changes** (KYC, deposit, withdraw, swap,
transfer). The mobile app shows transaction events as a **snackbar**.

> **Delivery model: in-app only.** Messages are delivered *only* while the app
> is connected to the socket. If a transaction finishes while the app is
> backgrounded or killed, that message is **not** stored and **not** replayed —
> it is lost. On reconnect / app open, the app must reload state from the REST
> endpoints (see [Missed events](#missed-events)). There is no FCM/APNs push.

---

## 1. Connection

| | |
|---|---|
| URL | `ws://<host>:8080/ws/prices` (use `wss://` in production) |
| Auth | JWT access token as a query param: `?token=<JWT>` |
| Protocol | Standard WebSocket (text frames, JSON payloads) |

```
ws://localhost:8080/ws/prices?token=eyJhbGciOi...
```

Notes:
- The path is historically named `/ws/prices`, but it is a **single multiplexed
  stream**: it carries both the price feed *and* all user notifications.
  Distinguish them by the `type` field (below). One socket per app session is
  enough — do **not** open a second socket.
- Auth is the same JWT used for REST (`Authorization: Bearer <token>`). Because
  browsers/mobile WS clients can't easily set headers on the upgrade, the token
  goes in the `token` query param. A missing/invalid token → HTTP 401, no upgrade.
- The connection is bound to the user id inside the JWT. The server fans a
  message out to **all** of that user's open connections (multi-device safe).

### Keep-alive
The server sends a WebSocket **ping every ~54s** and drops the connection if it
gets no **pong** within 60s. Virtually all WS client libraries answer pings
automatically — no app code needed. Just don't disable automatic pong.

### Reconnect
Treat disconnects as normal (network changes, app sleep). On close, reconnect
with backoff (e.g. 1s, 2s, 5s, capped), then refresh state via REST so the UI
reflects anything missed while offline.

---

## 2. Message envelope

**Every** message — price ticks and notifications alike — uses this envelope:

```jsonc
{
  "type": "EVENT_TYPE",   // string discriminator, switch on this
  "data": { /* ... */ },  // event-specific payload (object, or map for prices)
  "timestamp": 1718600000000 // server time, Unix epoch milliseconds
}
```

Mobile handling requirement:
1. Parse JSON, read `type`.
2. If `type == "PRICE_UPDATE"` → update price UI, **do not** snackbar.
3. Any other `type` → render a snackbar (and optionally refresh the relevant
   screen). Map each `type` to a user-facing message per the catalog below.
4. Unknown `type` → ignore gracefully (forward-compatible; backend may add more).

---

## 3. Event catalog

### Price feed

| `type` | When | `data` |
|---|---|---|
| `PRICE_UPDATE` | every ~3s | `{ "MYRC": 1.0, "USDT": 4.7, "USDC": 4.7, "BTC": 300000.0, "ETH": 16000.0, "SUI": 4.5 }` (token → price in MYR) |

### KYC

| `type` | When | `data` | Suggested snackbar |
|---|---|---|---|
| `KYC_APPROVED` | admin approves KYC | `{ "kyc_id": "<uuid>" }` | "Your identity has been verified ✅" |
| `KYC_REJECTED` | admin rejects KYC | `{ "kyc_id": "<uuid>", "remarks": "blurry photo" }` | "KYC rejected: {remarks}" |

### Deposit (MYR → MYRC, async via payment webhook)

| `type` | When | `data` | Suggested snackbar |
|---|---|---|---|
| `DEPOSIT_SUCCESS` | MYRC credited after payment | `{ "deposit_id": "<uuid>", "amount": 100.0, "tx_hash": "0x.." }` | "Deposit of {amount} MYRC completed" |
| `DEPOSIT_FAILED` | on-chain step failed | `{ "deposit_id": "<uuid>", "amount": 100.0, "reason": "blockchain error" }` | "Deposit failed: {reason}" |

### Withdraw (MYRC → bank, async)

| `type` | When | `data` | Suggested snackbar |
|---|---|---|---|
| `WITHDRAW_SUCCESS` | bank payout done | `{ "withdraw_id": "<uuid>", "amount": 50.0, "tx_hash": "0x.." }` | "Withdrawal of {amount} MYR sent to your bank" |
| `WITHDRAW_FAILED` | chain or payout failed | `{ "withdraw_id": "<uuid>", "amount": 50.0, "reason": "bank payout error" }` | "Withdrawal failed: {reason}" |

### Swap (token → token, async via Kafka + Sui)

| `type` | When | `data` | Suggested snackbar |
|---|---|---|---|
| `SWAP_COMPLETED` | swap settled on chain | `{ "swap_id": "<uuid>", "from_token": "MYRC", "to_token": "SUI", "to_amount": 12.3, "tx_hash": "0x.." }` | "Swapped to {to_amount} {to_token}" |
| `SWAP_FAILED` | swap payout failed | `{ "swap_id": "<uuid>", "from_token": "MYRC", "to_token": "SUI", "reason": "blockchain error" }` | "Swap failed: {reason}" |

### Transfer (on-chain send, async via Kafka)

| `type` | When | `data` | Suggested snackbar |
|---|---|---|---|
| `TRANSFER_SUCCESS` | transfer confirmed | `{ "transfer_id": "<uuid>", "amount": 5.0, "tx_hash": "0x.." }` | "Sent {amount}" |
| `TRANSFER_FAILED` | transfer failed | `{ "transfer_id": "<uuid>", "amount": 5.0, "reason": "blockchain error" }` | "Transfer failed: {reason}" |

> **Crypto deposit/withdraw (BTC/ETH/SUI/USDT/USDC)** complete **synchronously**
> inside the HTTP request — the REST response already carries success/failure.
> Drive that snackbar from the HTTP response, **not** the socket. No WS event is
> emitted for those endpoints.

---

## 4. Missed events

Because delivery is in-app only, after (re)connecting, reload state from REST so
the UI is correct even if events fired while disconnected:

| Domain | REST endpoint |
|---|---|
| KYC status | `GET /api/v1/private/kyc/status` |
| Deposits | `GET /api/v1/private/deposit/...` (list/view) |
| Withdrawals | `GET /api/v1/private/withdraw/...` |
| Swaps | `GET /api/v1/private/swap/...` |
| Transfers | `GET /api/v1/private/transfer/...` |

(See `internal/routes/` for exact paths.)

---

## 5. Example session

Client connects:
```
ws://localhost:8080/ws/prices?token=<JWT>
```

Server pushes a price tick:
```json
{ "type": "PRICE_UPDATE", "data": { "MYRC": 1.0, "SUI": 4.5, "BTC": 300000.0 }, "timestamp": 1718600000000 }
```

User's pending deposit settles:
```json
{ "type": "DEPOSIT_SUCCESS", "data": { "deposit_id": "8f3a...", "amount": 100.0, "tx_hash": "0xabc..." }, "timestamp": 1718600012345 }
```
→ app shows snackbar: **"Deposit of 100 MYRC completed"**.
