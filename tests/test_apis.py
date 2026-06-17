import requests
import psycopg2
import time
import sys

BASE_URL = "http://localhost:8080/api/v1"
DB_HOST = "localhost"
DB_PORT = 5433
DB_USER = "root"
DB_PASSWORD = "secretpassword"
DB_NAME = "swapngo"

POLL_INTERVAL = 3   # seconds between DB polls
POLL_TIMEOUT  = 30  # max seconds to wait for async Kafka processing


def get_db_connection():
    try:
        return psycopg2.connect(
            host=DB_HOST, port=DB_PORT,
            user=DB_USER, password=DB_PASSWORD,
            dbname=DB_NAME
        )
    except Exception as e:
        print(f"Failed to connect to database: {e}")
        sys.exit(1)


def poll_db_status(table, record_id, target_statuses, timeout=POLL_TIMEOUT):
    """Poll a table until the row's status is in target_statuses or timeout."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute(f"SELECT status FROM {table} WHERE id = %s;", (record_id,))
        row = cur.fetchone()
        cur.close()
        conn.close()
        if row and row[0] in target_statuses:
            return row[0]
        print(f"  [poll] {table} {record_id} -> {row[0] if row else 'not found'}, waiting...")
        time.sleep(POLL_INTERVAL)
    return None


# ─────────────────────────────────────────────
# Auth helpers
# ─────────────────────────────────────────────

def register_user(suffix=""):
    ts = int(time.time())
    username = f"testuser_{ts}{suffix}"
    payload = {
        "username": username,
        "phone_number": f"0123{ts}{suffix[-1:] if suffix else ''}",
        "email": f"test_{ts}{suffix}@example.com",
        "password": "Password123!",
        "pin": "1234",
        "account_name": "Test Account",
        "custody_type": "SERVER"
    }
    res = requests.post(f"{BASE_URL}/public/auth/register", json=payload)
    assert res.status_code == 200, f"Registration failed: {res.text}"

    # Fetch user_id from DB
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id FROM users WHERE username = %s;", (username,))
    row = cur.fetchone()
    assert row is not None, "User not found in DB after registration"
    user_id = str(row[0])

    cur.execute("SELECT id FROM accounts WHERE user_id = %s;", (row[0],))
    acc = cur.fetchone()
    assert acc is not None, f"Account not created for user {username}"
    cur.close()
    conn.close()

    print(f"Registered: {username}  user_id={user_id}")
    return username, user_id


def login(username):
    res = requests.post(f"{BASE_URL}/public/auth/login", json={
        "username": username,
        "password": "Password123!"
    })
    assert res.status_code == 200, f"Login failed: {res.text}"
    token = res.json().get("data", {}).get("access_token")
    assert token, "No access_token in login response"
    return token


# ─────────────────────────────────────────────
# Test cases
# ─────────────────────────────────────────────

def test_register():
    print("\n=== 1. Register Users ===")
    username, user_id = register_user()

    # DB check: wallets should be created
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id FROM accounts WHERE user_id = %s;", (user_id,))
    acc = cur.fetchone()
    assert acc is not None, "Account not found in DB"
    cur.execute("SELECT count(*) FROM wallets WHERE account_id = %s;", (acc[0],))
    count = cur.fetchone()[0]
    assert count > 0, "No wallets generated for the account"
    cur.close()
    conn.close()
    print(f"DB OK: account + {count} wallet(s) created.")
    return username, user_id


def test_login(username):
    print("\n=== 2. Login ===")
    token = login(username)
    print("Login OK — token obtained.")
    return token


def test_get_balance(token):
    print("\n=== 3. Get Wallet Balance ===")
    res = requests.get(
        f"{BASE_URL}/private/wallet/get-balance",
        headers={"Authorization": f"Bearer {token}"}
    )
    print("Status:", res.status_code, "| Response:", res.json())
    assert res.status_code == 200, f"Get balance failed: {res.text}"


def test_deposit(token):
    print("\n=== 4. Initiate Deposit ===")
    res = requests.post(
        f"{BASE_URL}/private/deposit/initiate",
        json={"amount": 100.0},
        headers={"Authorization": f"Bearer {token}"}
    )
    print("Status:", res.status_code, "| Response:", res.json())

    if res.status_code != 200:
        print("NOTE: Deposit failed (likely missing Billplz keys). Skipping deposit tests.")
        return None, None

    data = res.json().get("data", {})
    deposit_id = data.get("deposit_id")
    payment_url = data.get("payment_url")
    assert deposit_id, "No deposit_id in response"

    # DB check: status should be PENDING
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT status, amount_myrc FROM deposits WHERE id = %s;", (deposit_id,))
    row = cur.fetchone()
    cur.close()
    conn.close()
    assert row is not None, "Deposit not found in DB"
    assert row[0] == "PENDING", f"Expected PENDING, got {row[0]}"
    print(f"DB OK: deposit {deposit_id} is PENDING (amount_myrc={row[1]})")

    return deposit_id, payment_url


def test_deposit_webhook(deposit_id, payment_url):
    print("\n=== 5. Deposit Webhook (simulate Billplz paid callback) ===")
    if not deposit_id:
        print("Skipping — no deposit_id.")
        return

    # Extract the Billplz bill ID from the payment URL
    bill_id = payment_url.rstrip("/").split("/")[-1] if payment_url else deposit_id

    res = requests.post(
        f"{BASE_URL}/public/deposit/webhook",
        # Billplz sends form-encoded; our handler accepts both via ShouldBind
        data={"id": bill_id, "state": "paid"}
    )
    print("Status:", res.status_code, "| Response:", res.text)
    assert res.status_code == 200, f"Webhook failed: {res.text}"

    # After webhook the synchronous part moves status to PROCESSING_WEB3
    # The Kafka worker then finishes it asynchronously (SUCCESS)
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT status FROM deposits WHERE id = %s;", (deposit_id,))
    row = cur.fetchone()
    cur.close()
    conn.close()
    assert row is not None, "Deposit not found in DB"
    assert row[0] in ("PROCESSING_WEB3", "SUCCESS"), \
        f"Expected PROCESSING_WEB3 or SUCCESS after webhook, got {row[0]}"
    print(f"DB OK: deposit status after webhook = {row[0]}")

    if row[0] == "PROCESSING_WEB3":
        print("  Polling for Kafka async processing to complete...")
        final = poll_db_status("deposits", deposit_id, {"SUCCESS", "FAILED"})
        print(f"  Final deposit status: {final}")


def test_withdraw(token):
    print("\n=== 6. Initiate Withdraw ===")
    res = requests.post(
        f"{BASE_URL}/private/withdraw/initiate",
        json={
            "amount_myrc": 10.0,
            "bank_name": "MAYBANK",
            "bank_account_no": "112233445566"
        },
        headers={"Authorization": f"Bearer {token}"}
    )
    print("Status:", res.status_code, "| Response:", res.json())

    if res.status_code == 200:
        data = res.json().get("data", {})
        withdraw_id = data.get("id") if isinstance(data, dict) else None
        print("Withdraw initiated OK.")
        if withdraw_id:
            print("  Polling for Kafka async processing...")
            final = poll_db_status("withdrawals", withdraw_id, {"SUCCESS", "FAILED"})
            print(f"  Final withdraw status: {final}")
    else:
        print("Withdraw failed (possibly insufficient MYRC balance — expected for a fresh account).")


def test_transfer(token, receiver_user_id):
    print("\n=== 7. Initiate Transfer ===")
    res = requests.post(
        f"{BASE_URL}/private/transfer/initiate",
        json={
            "receiver_user_id": receiver_user_id,
            "amount": 5.0
        },
        headers={"Authorization": f"Bearer {token}"}
    )
    print("Status:", res.status_code, "| Response:", res.json())

    if res.status_code == 200:
        data = res.json().get("data", {})
        transfer_id = data.get("id") if isinstance(data, dict) else None
        print("Transfer initiated OK.")
        if transfer_id:
            print("  Polling for Kafka async processing...")
            final = poll_db_status("transfers", transfer_id, {"SUCCESS", "FAILED"})
            print(f"  Final transfer status: {final}")
    else:
        print("Transfer failed (possibly insufficient balance — expected for a fresh account).")


def test_swap(token):
    print("\n=== 8. Initiate Swap ===")
    res = requests.post(
        f"{BASE_URL}/private/swap/initiate",
        json={
            "from_token": "SUI",
            "to_token": "MYRC",
            "from_amount": 0.01,
            "estimated_amount": 10.0,
            "slippage": 0.05
        },
        headers={"Authorization": f"Bearer {token}"}
    )
    print("Status:", res.status_code, "| Response:", res.json())

    if res.status_code == 200:
        data = res.json().get("data", {})
        swap_id = data.get("swap_id")
        print(f"Swap initiated OK — swap_id={swap_id}")
        if swap_id:
            print("  Polling for Kafka async processing...")
            final = poll_db_status("swaps", swap_id, {"SUCCESS", "FAILED"})
            print(f"  Final swap status: {final}")
    else:
        print("Swap failed — check blockchain connectivity and token balances.")


# ─────────────────────────────────────────────
# Ledger helpers (CEX model: token_balances is the source of truth)
# ─────────────────────────────────────────────

def get_account_id(user_id):
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id FROM accounts WHERE user_id = %s;", (user_id,))
    row = cur.fetchone()
    cur.close()
    conn.close()
    assert row is not None, f"No account for user {user_id}"
    return str(row[0])


def get_ledger_balance(account_id, token):
    """Return the ledger balance for (account, token), 0 if no row."""
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute(
        "SELECT balance FROM token_balances WHERE account_id = %s AND token = %s;",
        (account_id, token),
    )
    row = cur.fetchone()
    cur.close()
    conn.close()
    return float(row[0]) if row else 0.0


def test_crypto_simulate_deposit(token, account_id):
    print("\n=== 9. Simulated Crypto Deposit (BTC) ===")
    before = get_ledger_balance(account_id, "BTC")
    res = requests.post(
        f"{BASE_URL}/private/crypto/deposit/simulate",
        json={"token": "BTC", "amount": 0.5},
        headers={"Authorization": f"Bearer {token}"},
    )
    print("Status:", res.status_code, "| Response:", res.json())
    assert res.status_code == 200, f"Simulated deposit failed: {res.text}"
    after = get_ledger_balance(account_id, "BTC")
    assert abs(after - (before + 0.5)) < 1e-9, f"BTC ledger not credited: {before} -> {after}"
    print(f"Ledger OK: BTC {before} -> {after}")


def test_ledger_swap(token, account_id):
    """Pure-ledger swap: BTC -> ETH must move ledger rows, reach SUCCESS, no chain tx."""
    print("\n=== 10. Ledger Swap (BTC -> ETH) ===")
    btc_before = get_ledger_balance(account_id, "BTC")
    eth_before = get_ledger_balance(account_id, "ETH")
    res = requests.post(
        f"{BASE_URL}/private/swap/initiate",
        json={
            "from_token": "BTC",
            "to_token": "ETH",
            "from_amount": 0.1,
            "estimated_amount": 0,
            "slippage": 0.05,
        },
        headers={"Authorization": f"Bearer {token}"},
    )
    print("Status:", res.status_code, "| Response:", res.json())
    assert res.status_code == 200, f"Swap initiate failed: {res.text}"
    swap_id = res.json().get("data", {}).get("swap_id")
    if swap_id:
        final = poll_db_status("swaps", swap_id, {"SUCCESS", "FAILED"})
        print(f"  Final swap status: {final}")
        assert final == "SUCCESS", f"Expected SUCCESS, got {final}"
    btc_after = get_ledger_balance(account_id, "BTC")
    eth_after = get_ledger_balance(account_id, "ETH")
    assert abs(btc_after - (btc_before - 0.1)) < 1e-9, f"BTC not debited: {btc_before} -> {btc_after}"
    assert eth_after > eth_before, f"ETH not credited: {eth_before} -> {eth_after}"
    print(f"Ledger OK: BTC {btc_before}->{btc_after}, ETH {eth_before}->{eth_after}")


def test_crypto_simulate_withdraw(token, account_id):
    print("\n=== 11. Simulated Crypto Withdraw (ETH) ===")
    before = get_ledger_balance(account_id, "ETH")
    if before <= 0:
        print("Skipping — no ETH balance to withdraw.")
        return
    amount = min(before, 0.01)
    res = requests.post(
        f"{BASE_URL}/private/crypto/withdraw",
        json={"token": "ETH", "amount": amount},
        headers={"Authorization": f"Bearer {token}"},
    )
    print("Status:", res.status_code, "| Response:", res.json())
    assert res.status_code == 200, f"Withdraw failed: {res.text}"
    after = get_ledger_balance(account_id, "ETH")
    assert abs(after - (before - amount)) < 1e-9, f"ETH not debited: {before} -> {after}"
    print(f"Ledger OK: ETH {before} -> {after} (withdrew {amount})")


# ─────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────

if __name__ == "__main__":
    print("Starting SwapNGo API Tests...")

    try:
        requests.get(BASE_URL, timeout=3)
    except requests.exceptions.ConnectionError:
        print(f"WARNING: Cannot connect to {BASE_URL}. Make sure the Go API is running.")

    # Register two users: sender + receiver (needed for transfer test)
    print("\n--- Registering sender ---")
    username, _user_id = test_register()

    print("\n--- Registering receiver ---")
    time.sleep(1)  # ensure different timestamp
    receiver_username, receiver_user_id = register_user("_recv")

    token = test_login(username)

    test_get_balance(token)

    deposit_id, payment_url = test_deposit(token)
    if deposit_id:
        test_deposit_webhook(deposit_id, payment_url)

    test_withdraw(token)

    test_transfer(token, receiver_user_id)

    test_swap(token)

    # Crypto rails + pure-ledger swap (CEX model)
    account_id = get_account_id(_user_id)
    test_crypto_simulate_deposit(token, account_id)
    test_ledger_swap(token, account_id)
    test_crypto_simulate_withdraw(token, account_id)

    print("\n✓ All endpoint tests executed.")
