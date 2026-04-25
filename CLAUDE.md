# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

Tradeoff: These guidelines bias toward caution over speed. For trivial tasks, use judgment.

1. Think Before Coding
   Don't assume. Don't hide confusion. Surface tradeoffs.

Before implementing:

State your assumptions explicitly. If uncertain, ask.
If multiple interpretations exist, present them - don't pick silently.
If a simpler approach exists, say so. Push back when warranted.
If something is unclear, stop. Name what's confusing. Ask. 2. Simplicity First
Minimum code that solves the problem. Nothing speculative.

No features beyond what was asked.
No abstractions for single-use code.
No "flexibility" or "configurability" that wasn't requested.
No error handling for impossible scenarios.
If you write 200 lines and it could be 50, rewrite it.
Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

3. Surgical Changes
   Touch only what you must. Clean up only your own mess.

When editing existing code:

Don't "improve" adjacent code, comments, or formatting.
Don't refactor things that aren't broken.
Match existing style, even if you'd do it differently.
If you notice unrelated dead code, mention it - don't delete it.
When your changes create orphans:

Remove imports/variables/functions that YOUR changes made unused.
Don't remove pre-existing dead code unless asked.
The test: Every changed line should trace directly to the user's request.

4. Goal-Driven Execution
   Define success criteria. Loop until verified.

Transform tasks into verifiable goals:

"Add validation" → "Write tests for invalid inputs, then make them pass"
"Fix the bug" → "Write a test that reproduces it, then make it pass"
"Refactor X" → "Ensure tests pass before and after"
For multi-step tasks, state a brief plan:

1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
   Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

These guidelines are working if: fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

**Run locally with hot reload:**

```bash
air
```

**Build manually:**

```bash
go build -o ./tmp/main.exe ./cmd/api/main.go   # API server
go build -o ./tmp/worker.exe ./cmd/worker/main.go  # Kafka worker
```

**Run with Docker (full stack including PostgreSQL + Kafka):**

```bash
docker-compose up
```

**Run integration tests (Python):**

```bash
python tests/test_apis.py
```

**Simulate Kafka events for testing:**

```bash
go run tests/simulate_swap_event.go
go run tests/sui.go
```

There are no native Go `*_test.go` files — all tests are Python integration tests in `tests/`.

## Architecture

Two entry points with separate responsibilities:

- `cmd/api/main.go` — Gin REST API server on port 8080; handles HTTP, dependency injection, database migration
- `cmd/worker/main.go` — Kafka consumer worker for async blockchain transaction processing

### Request Flow

```
HTTP Request
  → Handler (internal/handlers/)
    → Biz (internal/bizs/)          ← orchestrates business logic
      → Service (internal/services/) ← stateless domain logic
      → Repository (internal/repositories/) ← DB access via GORM
      → FSM (internal/fsm/)         ← validates state transitions
      → Kafka Producer (internal/kafka/) ← emits async events

Kafka Event
  → Worker (cmd/worker/)
    → Biz (internal/bizs/)          ← same biz layer reused
      → Blockchain Client (internal/clients/chains/)
      → FSM state transition
      → Repository update
```

### Biz Layer (`internal/bizs/`)

The biz layer is the critical orchestration layer between handlers and services. It:

- Uses `pkg/database.RunInTx()` to wrap multi-step DB operations in a transaction (passed via context)
- Calls FSM to validate state before committing transitions
- Emits Kafka events after successful DB writes (via BizContext)
- Used by both HTTP handlers AND the Kafka worker — same biz, different callers

### State Machine (`internal/fsm/`)

Every async operation (deposit, swap, transfer, withdraw) has its own FSM defined in `internal/fsm/`. The engine (`fsm/engine.go`) validates transitions — attempting an illegal state transition returns an error before any DB write.

### Transaction Helper (`pkg/database/tx.go`)

`RunInTx(ctx, fn)` starts a DB transaction and injects it into context. `GetDB(ctx)` returns the transaction if one is active, otherwise returns the global DB. All repositories use `GetDB(ctx)` so they automatically participate in the caller's transaction.

### Kafka Topics

| Topic                   | Producer         | Consumer |
| ----------------------- | ---------------- | -------- |
| `swap_events_topic`     | swap handler     | worker   |
| `deposit_events_topic`  | deposit handler  | worker   |
| `transfer_events_topic` | transfer handler | worker   |
| `withdraw_events_topic` | withdraw handler | worker   |

### External Integrations

- **Sui** (`internal/clients/chains/sui.go`) — primary blockchain; MYRC token minting/burning/transferring on Sui testnet
- **Billplz** (`internal/clients/billplz.go`) — MYR payment gateway; webhook at `/api/v1/public/deposit/webhook`
- **WebSocket** (`internal/ws/`) — live price feed to clients at `ws://host/ws/prices`

## Key Environment Variables

All vars are validated at startup in `pkg/configs/config.go` — missing required vars are fatal.

```
# Database (also in docker-compose)
DB_HOST, DB_USER, DB_PASSWORD, DB_NAME, DB_PORT

# JWT
JWT_SECRET, JWT_ACCESS_TIME, JWT_REFRESH_TIME

# Kafka
KAFKA_BROKERS   # localhost:9092 locally; kafka:29092 inside Docker

# Sui blockchain
SUI_CHAIN_URL, SUI_TREASURY_ADDRESS, SUI_TREASURY_PRIVATE_KEY
MYRC_SUI_PACKAGE_ID, MYRC_SUI_TREASURY_CAP_ID
SUI_ADMIN_ADDRESS, SUI_ADMIN_PRIVATE_KEY

# Billplz payment gateway
BILLPLZ_API_URL, BILLPLZ_API_KEY, BILLPLZ_COLLECTION_ID, BILLPLZ_CALLBACK_URL
```

Copy `.env.example` to `.env` and fill in blockchain keys and Billplz credentials before running.

## API Route Structure

Routes are defined in `internal/routes/` and registered in `cmd/api/main.go`.

- `POST /api/v1/public/auth/register|login`
- `GET|POST /api/v1/private/wallet/*` — requires JWT middleware
- `POST /api/v1/private/deposit/initiate` + `GET` routes — async via Billplz webhook
- `POST /api/v1/private/swap/initiate` + `GET` routes — async via Kafka + Sui
- `POST /api/v1/private/transfer/initiate` + `GET` routes
- `POST /api/v1/private/withdraw/initiate` + `GET` routes
- `GET /ws/prices` — WebSocket
