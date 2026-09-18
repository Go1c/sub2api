"""Private sub2api login worker. Never exposes an unauthenticated login endpoint."""
import asyncio
import hmac
import json
import os
import random
import time
import shutil
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from curl_cffi import CurlOpt
from curl_cffi.requests import AsyncSession
from login_core import LoginMaterial, LoginError, login, safe_diagnostics, required_proxy

TOKEN = os.environ.get('WORKER_TOKEN', '')
SLOTS = threading.BoundedSemaphore(2)


def proxy_session(proxy, timeout):
    # Explicit NOPROXY prevents NO_PROXY=* from bypassing the configured proxy.
    return AsyncSession(proxies={'all': required_proxy(proxy)}, trust_env=False,
                        curl_options={CurlOpt.NOPROXY: ''}, timeout=timeout, impersonate='chrome146')


async def probe_proxy(proxy):
    try:
        async with proxy_session(proxy, 5) as client:
            response = await client.get('https://auth.openai.com/log-in', allow_redirects=False)
        return response.status_code == 200
    except Exception:
        return False


async def select_proxy(candidates):
    if not isinstance(candidates, list) or not candidates or len(candidates) > 256:
        raise LoginError('proxy_required', {'stage': 'proxy_selection'})
    valid = []
    for item in candidates:
        if not isinstance(item, dict) or type(item.get('id')) is not int or item['id'] <= 0:
            continue
        try:
            valid.append({'id': item['id'], 'url': required_proxy(item.get('url'))})
        except LoginError:
            continue
    random.shuffle(valid)
    started = time.monotonic()
    for item in valid:
        remaining = 30 - (time.monotonic() - started)
        if remaining <= 0:
            break
        try:
            reachable = await asyncio.wait_for(probe_proxy(item['url']), timeout=min(6, remaining))
        except asyncio.TimeoutError:
            reachable = False
        # Probe carries neither account material nor login cookies. Do not rotate after login starts.
        if reachable:
            return item
    raise LoginError('proxy_unavailable', {'stage': 'proxy_selection'})


async def run(data):
    selected = await select_proxy(data.get('proxy_candidates'))
    proxy = selected['url']
    try:
        material = LoginMaterial(data['email'], data['password'], data['totp_secret'])
        result = await login(material, account_id=data.get('account_id'), proxy=proxy)
        diagnostics = {'stage': 'credential_probe', 'proxy_id': selected['id']}
        try:
            async with proxy_session(proxy, 20) as client:
                response = await client.get('https://chatgpt.com/backend-api/wham/usage', headers={
                    'Authorization': 'Bearer ' + result['tokens']['access_token'],
                    'ChatGPT-Account-Id': result['account_id'], 'Accept': 'application/json',
                }, allow_redirects=False)
            diagnostics['http_status'] = response.status_code
            if response.status_code != 200 or not isinstance(response.json().get('rate_limit'), dict):
                raise LoginError('credential_probe_failed', diagnostics)
        except LoginError:
            raise
        except Exception as exc:
            diagnostics['exception_type'] = type(exc).__name__
            raise LoginError('credential_probe_failed', safe_diagnostics(diagnostics)) from None
        result['proxy_id'] = selected['id']
        return result
    except LoginError as exc:
        details = safe_diagnostics(exc.diagnostics)
        details['proxy_id'] = selected['id']
        raise LoginError(exc.code, details) from None


def failure_response(exc):
    return {
        'success': False,
        'code': exc.code if isinstance(exc, LoginError) else 'login_failed',
        'diagnostics': safe_diagnostics(getattr(exc, 'diagnostics', {})),
    }


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
        if not 0 < length <= 131072:
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
        except Exception as exc:
            result = failure_response(exc)
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
