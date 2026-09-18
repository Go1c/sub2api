"""Private sub2api login worker. Never exposes an unauthenticated login endpoint."""
import asyncio
import hmac
import json
import os
import shutil
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from curl_cffi.requests import AsyncSession
from login_core import LoginMaterial, LoginError, login

TOKEN = os.environ.get('WORKER_TOKEN', '')
SLOTS = threading.BoundedSemaphore(2)


async def run(data):
    material = LoginMaterial(data['email'], data['password'], data['totp_secret'])
    result = await login(material, account_id=data.get('account_id'), proxy=data.get('proxy'))
    proxy = data.get('proxy')
    async with AsyncSession(proxies={'all': proxy} if proxy else None, timeout=20) as client:
        response = await client.get('https://chatgpt.com/backend-api/wham/usage', headers={
            'Authorization': 'Bearer ' + result['tokens']['access_token'],
            'ChatGPT-Account-Id': result['account_id'], 'Accept': 'application/json',
        }, allow_redirects=False)
    if response.status_code != 200 or not isinstance(response.json().get('rate_limit'), dict):
        raise LoginError('credential_probe_failed')
    return result


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def send_json(self, status, value):
        body = json.dumps(value).encode()
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(body)))
        self.send_header('Cache-Control', 'no-store')
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        self.send_json(200 if self.path == '/health' else 404, {'ready': self.path == '/health'})

    def do_POST(self):
        if not hmac.compare_digest(self.headers.get('Authorization', ''), 'Bearer ' + TOKEN):
            self.send_json(401, {'success': False})
            return
        if self.path != '/login':
            self.send_json(404, {'success': False})
            return
        try:
            length = int(self.headers.get('Content-Length', '0'))
        except ValueError:
            length = 0
        if not 0 < length <= 16384:
            self.send_json(413, {'success': False})
            return
        if not SLOTS.acquire(blocking=False):
            self.send_json(503, {'success': False, 'code': 'worker_busy'})
            return
        try:
            self.connection.settimeout(10)
            data = json.loads(self.rfile.read(length))
            for key in ('email', 'password', 'totp_secret'):
                if not isinstance(data.get(key), str) or not data[key]:
                    raise ValueError('invalid input')
            result = asyncio.run(run(data))
        except LoginError as exc:
            result = {'success': False, 'code': exc.code}
        except Exception:
            result = {'success': False, 'code': 'login_failed'}
        finally:
            SLOTS.release()
        try:
            self.send_json(200, result)
        except (BrokenPipeError, ConnectionResetError, TimeoutError):
            pass


if __name__ == '__main__':
    if len(TOKEN) < 32 or not shutil.which('node'):
        raise SystemExit('WORKER_TOKEN (32+ characters) and Node.js are required')
    ThreadingHTTPServer(('0.0.0.0', 8081), Handler).serve_forever()
