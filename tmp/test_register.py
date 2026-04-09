import urllib.request
import json

data = {
    "username": "testuser",
    "phone_number": "1234567890",
    "email": "test@test.com",
    "password": "Password1!",
    "pin": "1234",
    "account_name": "My Account",
    "custody_type": "SERVER"
}

req = urllib.request.Request(
    'http://localhost:8080/api/v1/auth/register',
    data=json.dumps(data).encode('utf-8'),
    headers={'Content-Type': 'application/json'},
    method='POST'
)

try:
    with urllib.request.urlopen(req) as f:
        print(f.status)
        print(f.read().decode('utf-8'))
except urllib.error.HTTPError as e:
    print(e.status)
    print(e.read().decode('utf-8'))
except Exception as e:
    print(e)
