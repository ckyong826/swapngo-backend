# Frontend Build Brief — Real-Time Snackbar Notifications (React Native)

> Paste this whole file to the mobile/frontend Claude. It is the implementation
> brief. The exact wire contract it depends on lives in
> [`websocket-notifications.md`](./websocket-notifications.md) — both are
> consistent; this file is the "how to build it in RN" layer.

---

## Goal

Show the user a **snackbar** the moment a backend status changes:
KYC approved/rejected, and deposit / withdraw / swap / transfer success/failed.
Also keep a live **price feed** updating. Both arrive over **one** WebSocket.

This is **in-app only**: notifications arrive only while the app is open and the
socket is connected. Anything that happens while the app is closed/backgrounded
is **not** replayed — so on (re)connect you must refresh affected screens from
REST. There is no FCM/APNs push in scope.

---

## The backend, in one paragraph

A Go backend exposes a single authenticated WebSocket. It streams a price tick
every ~3s and pushes per-user event notifications when async operations finish
(payments settle, on-chain swaps/transfers/withdrawals complete, admin
approves/rejects KYC). Every message uses the same JSON envelope with a `type`
discriminator. Crypto deposit/withdraw for BTC/ETH/SUI/USDT/USDC are
**synchronous REST** — their result is in the HTTP response, no socket event.

---

## Connection contract (must match exactly)

| | |
|---|---|
| URL | `wss://<host>/ws/prices` (dev: `ws://<host>:8080/ws/prices`) |
| Auth | JWT in query param: `?token=<accessToken>` (same token as REST `Authorization: Bearer`) |
| Frames | text, JSON |
| One socket | the path is named `/ws/prices` but multiplexes price + notifications. Open **exactly one** socket per session. |
| Keep-alive | server pings every ~54s. RN's native WebSocket answers pongs automatically — no JS code needed. Don't add manual ping. |
| Auth failure | bad/missing token → upgrade fails (HTTP 401). Treat as "need fresh token / re-login". |

### Envelope (every message)

```jsonc
{
  "type": "EVENT_TYPE",       // switch on this
  "data": { /* event fields */ },
  "timestamp": 1718600000000  // unix epoch millis
}
```

### Event catalog

| `type` | `data` fields | UI action |
|---|---|---|
| `PRICE_UPDATE` | `{ MYRC, USDT, USDC, BTC, ETH, SUI }` (token→price in MYR) | update price store; **no snackbar** |
| `KYC_APPROVED` | `{ kyc_id }` | snackbar "Identity verified ✅"; refresh KYC screen |
| `KYC_REJECTED` | `{ kyc_id, remarks }` | snackbar "KYC rejected: {remarks}" |
| `DEPOSIT_SUCCESS` | `{ deposit_id, amount, tx_hash }` | snackbar "Deposit of {amount} MYRC completed"; refresh wallet |
| `DEPOSIT_FAILED` | `{ deposit_id, amount, reason }` | snackbar "Deposit failed: {reason}" |
| `WITHDRAW_SUCCESS` | `{ withdraw_id, amount, tx_hash }` | snackbar "Withdrawal of {amount} MYR sent"; refresh wallet |
| `WITHDRAW_FAILED` | `{ withdraw_id, amount, reason }` | snackbar "Withdrawal failed: {reason}" |
| `SWAP_COMPLETED` | `{ swap_id, from_token, to_token, to_amount, tx_hash }` | snackbar "Swapped to {to_amount} {to_token}"; refresh balances |
| `SWAP_FAILED` | `{ swap_id, from_token, to_token, reason }` | snackbar "Swap failed: {reason}" |
| `TRANSFER_SUCCESS` | `{ transfer_id, amount, tx_hash }` | snackbar "Sent {amount}"; refresh balances |
| `TRANSFER_FAILED` | `{ transfer_id, amount, reason }` | snackbar "Transfer failed: {reason}" |

Unknown `type` → ignore (forward-compatible; backend may add events).

---

## What to build (React Native)

1. A **singleton socket service** with auto-reconnect + backoff, bound to the
   auth token, started after login and stopped on logout.
2. An **event router** that maps `type` → snackbar text + optional screen
   refresh, and feeds `PRICE_UPDATE` into the price store.
3. A **snackbar** call (use `react-native-snackbar`, or your existing toast).
4. A **reconnect-refresh** hook: on (re)connect, re-fetch wallet / tx history so
   missed events don't leave stale UI.

### Suggested deps
- WS: native `WebSocket` (built-in) is fine with a manual reconnect wrapper, or
  `reconnecting-websocket`.
- Snackbar: `react-native-snackbar`.
- State: whatever the app already uses (Redux/Zustand/Context). Don't introduce a
  new one.

### Socket service (reference implementation)

```ts
// notificationSocket.ts
type Envelope = { type: string; data: any; timestamp: number };

type Handlers = {
  onPrice: (prices: Record<string, number>) => void;
  onNotification: (type: string, data: any) => void;
  onStatus?: (s: "connected" | "disconnected") => void;
};

const WS_BASE =
  __DEV__ ? "ws://10.0.2.2:8080" : "wss://api.your-host.com"; // 10.0.2.2 = host from Android emulator

export class NotificationSocket {
  private ws?: WebSocket;
  private closedByUser = false;
  private retry = 0;
  private getToken: () => Promise<string | null>;
  private handlers: Handlers;

  constructor(getToken: () => Promise<string | null>, handlers: Handlers) {
    this.getToken = getToken;
    this.handlers = handlers;
  }

  async connect() {
    this.closedByUser = false;
    const token = await this.getToken();
    if (!token) return; // not logged in

    const url = `${WS_BASE}/ws/prices?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(url);
    this.ws = ws;

    ws.onopen = () => {
      this.retry = 0;
      this.handlers.onStatus?.("connected");
      // IMPORTANT: refresh state on every (re)connect — missed events aren't replayed
      // e.g. store.dispatch(refreshWalletAndHistory())
    };

    ws.onmessage = (e) => {
      let msg: Envelope;
      try { msg = JSON.parse(e.data as string); } catch { return; }
      if (!msg?.type) return;
      if (msg.type === "PRICE_UPDATE") this.handlers.onPrice(msg.data);
      else this.handlers.onNotification(msg.type, msg.data);
    };

    ws.onclose = () => {
      this.handlers.onStatus?.("disconnected");
      if (!this.closedByUser) this.scheduleReconnect();
    };

    ws.onerror = () => { try { ws.close(); } catch {} }; // triggers onclose → reconnect
  }

  private scheduleReconnect() {
    const delay = Math.min(1000 * 2 ** this.retry, 15000); // 1s,2s,4s… cap 15s
    this.retry++;
    setTimeout(() => this.connect(), delay);
  }

  disconnect() {
    this.closedByUser = true;
    try { this.ws?.close(); } catch {}
  }
}
```

### Event router (reference)

```ts
import Snackbar from "react-native-snackbar";

export function handleNotification(type: string, data: any) {
  const msg = (() => {
    switch (type) {
      case "KYC_APPROVED":     return "Identity verified ✅";
      case "KYC_REJECTED":     return `KYC rejected: ${data.remarks ?? ""}`;
      case "DEPOSIT_SUCCESS":  return `Deposit of ${data.amount} MYRC completed`;
      case "DEPOSIT_FAILED":   return `Deposit failed: ${data.reason ?? ""}`;
      case "WITHDRAW_SUCCESS": return `Withdrawal of ${data.amount} MYR sent`;
      case "WITHDRAW_FAILED":  return `Withdrawal failed: ${data.reason ?? ""}`;
      case "SWAP_COMPLETED":   return `Swapped to ${data.to_amount} ${data.to_token}`;
      case "SWAP_FAILED":      return `Swap failed: ${data.reason ?? ""}`;
      case "TRANSFER_SUCCESS": return `Sent ${data.amount}`;
      case "TRANSFER_FAILED":  return `Transfer failed: ${data.reason ?? ""}`;
      default:                 return null; // unknown → ignore
    }
  })();
  if (!msg) return;

  const isFailure = type.endsWith("_FAILED") || type === "KYC_REJECTED";
  Snackbar.show({
    text: msg,
    duration: Snackbar.LENGTH_LONG,
    backgroundColor: isFailure ? "#B00020" : "#2E7D32",
  });

  // Optional: trigger a targeted refresh so the screen is consistent
  // if (type.startsWith("DEPOSIT") || type.startsWith("WITHDRAW")) refreshWallet();
}
```

### Wiring (reference)

```ts
// after successful login:
const socket = new NotificationSocket(
  () => getAccessTokenFromSecureStore(),
  {
    onPrice: (prices) => priceStore.set(prices),
    onNotification: (type, data) => handleNotification(type, data),
    onStatus: (s) => { if (s === "connected") refreshWalletAndHistory(); },
  }
);
socket.connect();

// on logout:
socket.disconnect();

// optional: on AppState 'active', if socket closed, socket.connect();
```

---

## Requirements / acceptance criteria

- [ ] Exactly **one** socket open per logged-in session (verify no duplicate
      connects on re-render / navigation).
- [ ] `PRICE_UPDATE` updates prices and does **not** raise a snackbar.
- [ ] Each of the 10 notification types raises a snackbar with correct text;
      failures styled distinctly (red) from successes (green).
- [ ] On reconnect, wallet + transaction history are refreshed from REST.
- [ ] Token is read fresh on every (re)connect (handles token refresh); on 401 /
      repeated failure after logout, stop reconnecting.
- [ ] Socket disconnects cleanly on logout (no leaked reconnect loop).
- [ ] Unknown `type` is ignored without crashing.
- [ ] App backgrounding then foregrounding re-establishes the socket.

## Out of scope / gotchas

- **Crypto deposit/withdraw (BTC/ETH/SUI/USDT/USDC)**: synchronous REST. Show
  the snackbar from the HTTP response of those calls — **no** socket event fires.
- No offline/queued delivery. Missed-while-closed events are recovered only via
  the REST refresh on reconnect.
- Android emulator reaches host backend at `10.0.2.2`, iOS simulator at
  `localhost`. Use real LAN IP / domain on a physical device.
- Don't implement manual ping/pong — the native socket layer handles pongs.

## REST endpoints for the reconnect refresh

| Domain | Endpoint (under `/api/v1/private`) |
|---|---|
| KYC status | `GET /kyc/status` |
| Deposits | `GET /deposit/...` (list/view) |
| Withdrawals | `GET /withdraw/...` |
| Swaps | `GET /swap/...` |
| Transfers | `GET /transfer/...` |

(Confirm exact paths against the backend `internal/routes/` before wiring.)
