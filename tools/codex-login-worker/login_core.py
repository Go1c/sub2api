"""Credential parsing and isolated Codex password/TOTP authorization."""
import asyncio
import base64
from dataclasses import dataclass, field
import json
import os
from pathlib import Path
import re
import signal
import sys
import tempfile

import jwt

CLIENT_ID = 'app_EMoamEEZ73f0CkXaXp7hrann'
MAX_ACCOUNTS = 100


DIAGNOSTIC_STAGES = frozenset({
    'startup', 'oauth_bootstrap', 'email', 'password', 'totp', 'workspace',
    'token_exchange', 'identity_validation', 'credential_probe',
})
DIAGNOSTIC_EXCEPTIONS = frozenset({
    'Timeout', 'TimeoutError', 'ConnectionError', 'ProxyError', 'SSLError',
    'ConnectTimeout', 'ReadTimeout', 'RequestsError', 'RequestException',
    'RuntimeError', 'ValueError', 'JSONDecodeError', 'FileNotFoundError',
    'ModuleNotFoundError', 'ImportError', 'OSError',
})


def safe_diagnostics(value):
    if not isinstance(value, dict):
        return {}
    result = {}
    if isinstance(value.get('stage'), str) and value['stage'] in DIAGNOSTIC_STAGES:
        result['stage'] = value['stage']
    if type(value.get('http_status')) is int and 100 <= value['http_status'] <= 599:
        result['http_status'] = value['http_status']
    if isinstance(value.get('exception_type'), str) and value['exception_type'] in DIAGNOSTIC_EXCEPTIONS:
        result['exception_type'] = value['exception_type']
    return result


class LoginError(Exception):
    def __init__(self, code, diagnostics=None):
        self.code = code
        self.diagnostics = diagnostics or {}
        super().__init__(code)


@dataclass
class LoginMaterial:
    email: str
    password: str = field(repr=False)
    totp_secret: str = field(repr=False)
    account_id: str | None = None


def parse_import(content):
    if len(content.encode('utf-8')) > 1024 * 1024:
        raise LoginError('file_too_large')
    content = content.lstrip('\ufeff').strip('\r\n ')
    if content.startswith(('{', '[')):
        try:
            rows = json.loads(content)
        except ValueError:
            raise LoginError('invalid_json') from None
        if isinstance(rows, dict):
            rows = rows.get('accounts', rows.get('data', [rows]))
    else:
        rows = []
        for line in content.splitlines():
            if not line.strip():
                continue
            parts = re.split(r'----|\t|\|', line)
            rows.append(dict(zip(('email', 'password', 'totp_secret'), parts)) if len(parts) == 3 else None)
    if not isinstance(rows, list) or not 1 <= len(rows) <= MAX_ACCOUNTS:
        raise LoginError('invalid_account_count')
    items, errors, seen = [], [], set()
    for index, row in enumerate(rows, 1):
        try:
            if not isinstance(row, dict):
                raise LoginError('invalid_record')
            email = row.get('email', '')
            password = row.get('password', '')
            secret = next((row[k] for k in ('totp_secret', '2fa_secret', '2fa_sk', '2fa', 'otp_secret', 'secret') if k in row), '')
            if not all(isinstance(v, str) for v in (email, password, secret)):
                raise LoginError('invalid_record')
            email = email.strip().lower()
            if len(email) > 254 or not re.fullmatch(r'[^\s@]+@[^\s@]+\.[^\s@]+', email):
                raise LoginError('invalid_email')
            if not password or len(password) > 1024:
                raise LoginError('invalid_password')
            secret = re.sub(r'[\s-]', '', secret).upper().rstrip('=')
            try:
                decoded = base64.b32decode(secret + '=' * (-len(secret) % 8))
            except (ValueError, base64.binascii.Error):
                raise LoginError('invalid_totp_secret') from None
            if not 10 <= len(decoded) <= 128:
                raise LoginError('invalid_totp_secret')
            if email in seen:
                raise LoginError('duplicate_email')
            seen.add(email)
            account_id = row.get('account_id')
            if account_id is not None and (not isinstance(account_id, str) or not re.fullmatch(r'[A-Za-z0-9_-]{1,100}', account_id)):
                raise LoginError('invalid_account_id')
            items.append(LoginMaterial(email, password, secret, account_id))
        except LoginError as exc:
            errors.append({'index': index, 'code': exc.code})
    return items, errors


def select_workspace(workspaces, expected_id=None):
    valid = [w for w in workspaces if isinstance(w, dict) and w.get('id')]
    if expected_id:
        found = [w for w in valid if w['id'] == expected_id]
        if len(found) != 1:
            raise LoginError('workspace_unavailable')
        return found[0]
    teams = [w for w in valid if str(w.get('kind', w.get('type', ''))).lower() in ('organization', 'team', 'business', 'enterprise')]
    personal = [w for w in valid if str(w.get('kind', w.get('type', ''))).lower() == 'personal']
    choices = teams or personal
    if len(choices) != 1:
        raise LoginError('workspace_selection_required')
    return choices[0]


def validate_tokens(tokens, email, account_id):
    if not all(isinstance(tokens.get(k), str) and tokens[k] for k in ('access_token', 'refresh_token', 'id_token')):
        raise LoginError('incomplete_tokens')
    # These values come directly from the TLS-protected token endpoint, not imported JWTs.
    try:
        identity = jwt.decode(tokens['id_token'], options={'verify_signature': False})
        access = jwt.decode(tokens['access_token'], options={'verify_signature': False})
        actual_email = identity.get('email') or identity.get('https://api.openai.com/profile', {}).get('email')
        if not actual_email or actual_email.casefold() != email.casefold():
            raise LoginError('identity_mismatch')
        for value in (identity, access):
            if value.get('https://api.openai.com/auth', {}).get('chatgpt_account_id') != account_id:
                raise LoginError('workspace_mismatch')
        return identity
    except (ValueError, TypeError, jwt.PyJWTError):
        raise LoginError('invalid_tokens') from None


async def login(material, *, account_id=None, proxy=None):
    """No credentials in argv, inherited app environment, logs or temporary files."""
    worker = Path(__file__).with_name('codex_login_worker.py')
    env = {key: os.environ[key] for key in ('PATH', 'SYSTEMROOT', 'SSL_CERT_FILE', 'SSL_CERT_DIR') if key in os.environ}
    with tempfile.TemporaryDirectory(prefix='codex-login-') as cwd:
        proc = await asyncio.create_subprocess_exec(
            sys.executable, str(worker), cwd=cwd, env=env,
            stdin=asyncio.subprocess.PIPE, stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.DEVNULL, start_new_session=True,
        )
        try:
            stdout, _ = await asyncio.wait_for(proc.communicate(json.dumps({
                'email': material.email, 'password': material.password, 'totp_secret': material.totp_secret,
                'account_id': account_id, 'proxy': proxy or '',
            }).encode()), timeout=150)
        except (asyncio.TimeoutError, asyncio.CancelledError) as exc:
            if proc.returncode is None:
                os.killpg(proc.pid, signal.SIGKILL)
            await proc.wait()
            if isinstance(exc, asyncio.CancelledError):
                raise
            raise LoginError('login_timeout') from None
        try:
            result = json.loads(stdout)
        except (ValueError, UnicodeError):
            raise LoginError('login_worker_failed') from None
        if not result.get('success'):
            raise LoginError(result.get('code', 'login_failed'), result.get('diagnostics'))
        validate_tokens(result['tokens'], material.email, result['account_id'])
        if account_id and result['account_id'] != account_id:
            raise LoginError('workspace_mismatch')
        return result
