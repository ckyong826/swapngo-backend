import urllib.request
import json
import time

BASE_URL = "http://localhost:8080/api/v1"

def request(method, path, payload=None, token=None):
    url = BASE_URL + path
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = f'Bearer {token}'
    
    data = json.dumps(payload).encode('utf-8') if payload else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    
    try:
        with urllib.request.urlopen(req) as res:
            res_data = res.read().decode('utf-8')
            return json.loads(res_data), res.status
    except Exception as e:
        if hasattr(e, 'read'):
            res_data = e.read().decode('utf-8')
            return json.loads(res_data) if res_data else str(e), e.code
        return str(e), 500

print("Testing Server is UP...")
try:
    urllib.request.urlopen("http://localhost:8080")
except Exception as e:
    pass

print("\n--- 1. Register User ---")
reg_payload = {
    "username": f"testuser_{int(time.time())}",
    "phone_number": f"0123{int(time.time())}",
    "email": f"test_{int(time.time())}@example.com",
    "password": "Password123!",
    "pin": "1234",
    "account_name": "Test Account",
    "custody_type": "SERVER"
}
res, code = request('POST', '/public/auth/register', reg_payload)
print(code, res)

print("\n--- 2. Login User ---")
login_payload = {
    "username": reg_payload['username'],
    "password": "Password123!"
}
res, code = request('POST', '/public/auth/login', login_payload)
print(code, res)
if isinstance(res, dict):
    token = res.get('data', {}).get('access_token')
else:
    token = None
if not token:
    print("Failed to get token!")
    exit(1)

print("\n--- 3. Initiate Deposit ---")
deposit_payload = {
    "amount": 100.50
}
res, code = request('POST', '/private/deposit/initiate', deposit_payload, token)
print(code, res)
if isinstance(res, dict):
    deposit_id = res.get('data', {}).get('deposit_id', 'testing_id_if_failed')
    payment_url = res.get('data', {}).get('payment_url', 'no_url')
else:
    deposit_id = 'testing_id_if_failed'
    payment_url = 'no_url'

print("\n--- 4. Handle Webhook ---")
# Billplz format uses application/x-www-form-urlencoded but our Go backend uses binding (which can parse json if sent as json)
# Let's send it as JSON here, or if backend expects form, we can modify the request.
# The depositHandler: ctx.ShouldBind(&req) handles both JSON and Form correctly.
webhook_payload = {
    "id": payment_url.split('/')[-1] if '/' in payment_url else deposit_id, # simulated Bill ID
    "state": "paid"
}
res, code = request('POST', '/public/deposit/webhook', webhook_payload)
print(code, res)
