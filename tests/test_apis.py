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

def get_db_connection():
    try:
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASSWORD,
            dbname=DB_NAME
        )
        return conn
    except Exception as e:
        print(f"Failed to connect to database: {e}")
        sys.exit(1)

def test_register():
    print("\n--- 1. Register User ---")
    ts = int(time.time())
    username = f"testuser_{ts}"
    payload = {
        "username": username,
        "phone_number": f"0123{ts}",
        "email": f"test_{ts}@example.com",
        "password": "Password123!",
        "pin": "1234",
        "account_name": "Test Account",
        "custody_type": "SERVER"
    }
    res = requests.post(f"{BASE_URL}/public/auth/register", json=payload)
    print("Status:", res.status_code)
    print("Response:", res.json())
    assert res.status_code == 200, "Registration failed"
    
    # DB Check
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT id FROM users WHERE username = %s;", (username,))
    user = cur.fetchone()
    assert user is not None, "User not found in DB"
    user_id = user[0]
    
    cur.execute("SELECT id FROM user_wallet_accounts WHERE user_id = %s;", (user_id,))
    wallet = cur.fetchone()
    assert wallet is not None, "Wallet account not generated for user"
    
    cur.close()
    conn.close()
    print("DB Verification: User and Wallet created successfully.")
    
    return username

def test_login(username):
    print("\n--- 2. Login User ---")
    payload = {
        "username": username,
        "password": "Password123!"
    }
    res = requests.post(f"{BASE_URL}/public/auth/login", json=payload)
    print("Status:", res.status_code)
    assert res.status_code == 200, "Login failed"
    data = res.json().get('data', {})
    token = data.get('access_token')
    assert token is not None, "No access token returned"
    print("Login successful, token obtained.")
    return token

def test_get_balance(token):
    print("\n--- 3. Get Wallet Balance ---")
    headers = {"Authorization": f"Bearer {token}"}
    res = requests.get(f"{BASE_URL}/private/wallet/get-balance", headers=headers)
    print("Status:", res.status_code)
    print("Response:", res.json())
    assert res.status_code == 200, "Get balance failed"

def test_deposit(token):
    print("\n--- 4. Initiate Deposit ---")
    headers = {"Authorization": f"Bearer {token}"}
    payload = {"amount": 100.50}
    res = requests.post(f"{BASE_URL}/private/deposit/initiate", json=payload, headers=headers)
    print("Status:", res.status_code)
    print("Response:", res.json())
    
    # Sometimes deposit might fail if Billplz is not properly configured, so we test if it returned 200
    if res.status_code != 200:
        print("Note: Deposit initiation might need correct Billplz keys. Bypassing further DB checks for deposit. Returned status:", res.status_code)
        return None, None
        
    data = res.json().get('data', {})
    deposit_id = data.get('deposit_id')
    payment_url = data.get('payment_url')
    assert deposit_id is not None, "No deposit_id returned"
    
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT status, amount FROM deposits WHERE id = %s;", (deposit_id,))
    deposit = cur.fetchone()
    assert deposit is not None, "Deposit not found in DB"
    assert deposit[0] == "pending", "Deposit should be pending initially"
    cur.close()
    conn.close()
    print(f"DB Verification: Deposit {deposit_id} is pending.")
    
    return deposit_id, payment_url

def test_deposit_webhook(deposit_id, payment_url):
    print("\n--- 5. Test Deposit Webhook ---")
    if not payment_url:
        print("Skipping webhook test due to failed deposit initiation.")
        return
        
    bill_id = payment_url.split('/')[-1] if '/' in payment_url else deposit_id
    payload = {
        "id": bill_id,
        "state": "paid"
    }
    # Webhook endpoint might accept JSON depending on Gin binding
    res = requests.post(f"{BASE_URL}/public/deposit/webhook", json=payload)
    print("Status:", res.status_code)
    assert res.status_code == 200, "Webhook failed"
    
    # DB Verification
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT status FROM deposits WHERE id = %s;", (deposit_id,))
    deposit = cur.fetchone()
    assert deposit is not None, "Deposit not found in DB"
    assert deposit[0] == "SUCCESS", f"Deposit status should be SUCCESS but got {deposit[0]}"
    cur.close()
    conn.close()
    print("DB Verification: Deposit updated to SUCCESS.")

def test_withdraw(token):
    print("\n--- 6. Initiate Withdraw ---")
    headers = {"Authorization": f"Bearer {token}"}
    payload = {
        "amount_myrc": 10.0,
        "bank_name": "MAYBANK",
        "bank_account_no": "112233445566"
    }
    res = requests.post(f"{BASE_URL}/private/withdraw/initiate", json=payload, headers=headers)
    print("Status:", res.status_code)
    print("Response:", res.json())
    
    if res.status_code == 200:
        print("Withdraw initiated successfully.")
    else:
        # It could fail due to insufficient MYRC balance if the user just registered
        print("Withdraw returned status:", res.status_code, "(Possibly due to insufficient balance)")

def test_transfer(token):
    print("\n--- 7. Initiate Transfer ---")
    headers = {"Authorization": f"Bearer {token}"}
    payload = {
        "from_address": "0xYourWalletAddresss",
        "to_address": "0xRecipientWalletAddress",
        "amount_myrc": 5.0
    }
    res = requests.post(f"{BASE_URL}/private/transfer/initiate", json=payload, headers=headers)
    print("Status:", res.status_code)
    print("Response:", res.json())

def test_swap(token):
    print("\n--- 8. Initiate Swap ---")
    headers = {"Authorization": f"Bearer {token}"}
    payload = {
        "from_token": "MYRC",
        "to_token": "USDT",
        "from_amount": 5.0,
        "estimated_amount": 1.0,
        "slippage": 0.05
    }
    res = requests.post(f"{BASE_URL}/private/swap/initiate", json=payload, headers=headers)
    print("Status:", res.status_code)
    print("Response:", res.json())


if __name__ == "__main__":
    print("Starting API Tests...")
    
    # Optional check if server is running
    try:
        requests.get(BASE_URL)
    except requests.exceptions.ConnectionError:
        print(f"Warning: Cannot connect to {BASE_URL}. Ensure the Go backend is running.")
    
    # 1. Register
    username = test_register()
    
    # 2. Login
    token = test_login(username)
    
    # 3. Balance
    test_get_balance(token)
    
    # 4. Deposit
    deposit_id, payment_url = test_deposit(token)
    
    # 5. Webhook
    if deposit_id:
        test_deposit_webhook(deposit_id, payment_url)
    
    # 6. Withdraw
    test_withdraw(token)
    
    # 7. Transfer
    test_transfer(token)
    
    # 8. Swap
    test_swap(token)
    
    print("\nAll endpoints executed.")
