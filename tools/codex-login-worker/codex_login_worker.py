"""Single-use subprocess. stdin/stdout are a private credential channel."""
import contextlib
import io
import json
import logging
import os
from pathlib import Path
import sys
import time
from urllib.parse import urljoin, urlparse

ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(ROOT / 'third_party/codex_protocol'))
os.umask(0o077)
logging.disable(logging.CRITICAL)

from login_core import LoginError, select_workspace, validate_tokens, safe_diagnostics, required_proxy
diagnostics = {'stage': 'startup'}


def authorize(data):
    proxy = required_proxy(data.get('proxy'))
    from curl_cffi import CurlOpt
    import pyotp
    import dotenv
    dotenv.load_dotenv = lambda *args, **kwargs: False
    from config import codex as cfg
    from core import codex_oauth as flow
    from core.session import BrowserSession

    cfg.CODEX_AUTH_URL_SOURCE = 'local'
    cfg.CODEX_OAUTH_DRIVER = 'protocol'
    flow._NET_MAX_ATTEMPTS = 1
    session = BrowserSession(proxy=proxy, detect_exit_geo=False)
    session.session.trust_env = False
    session.session.curl_options[CurlOpt.NOPROXY] = ''
    original_request = session.session.request
    allowed_posts = {
        ('sentinel.openai.com', '/backend-api/sentinel/req'),
        ('auth.openai.com', '/api/accounts/authorize/continue'),
        ('auth.openai.com', '/api/accounts/password/verify'),
        ('auth.openai.com', '/api/accounts/mfa/verify'),
        ('auth.openai.com', '/api/accounts/workspace/select'),
        ('auth.openai.com', '/oauth/token'),
    }

    def request(method, url, **kwargs):
        follow = kwargs.pop('allow_redirects', True)
        for _ in range(15):
            parsed = urlparse(url)
            if parsed.scheme != 'https' or parsed.hostname not in {'auth.openai.com', 'chatgpt.com', 'sentinel.openai.com'}:
                raise LoginError('unexpected_redirect')
            if method.upper() != 'GET' and (parsed.hostname, parsed.path) not in allowed_posts:
                raise LoginError('additional_verification_required')
            kwargs['timeout'] = 20
            response = original_request(method, url, allow_redirects=False, **kwargs)
            if response.status_code >= 400:
                diagnostics.update(http_status=response.status_code, endpoint=parsed.path)
                code = 'login_rejected'
                if response.status_code == 429:
                    code = 'rate_limited'
                elif response.status_code == 403:
                    code = ('mfa_rejected' if diagnostics.get('stage') == 'totp'
                            and parsed.path.endswith('/mfa/verify') else 'additional_verification_required')
                elif response.status_code >= 500:
                    code = 'upstream_unavailable'
                elif parsed.path.endswith('/password/verify'):
                    code = 'invalid_password'
                elif parsed.path.endswith('/mfa/verify'):
                    code = 'invalid_totp'
                raise LoginError(code)
            if not follow or response.status_code not in (301, 302, 303, 307, 308):
                return response
            location = response.headers.get('location')
            if not location:
                return response
            url = urljoin(url, location)
            if response.status_code in (301, 302, 303) and method.upper() != 'GET':
                method = 'GET'
                kwargs.pop('data', None)
                kwargs.pop('json', None)
        raise LoginError('redirect_limit')

    session.session.request = request
    try:
        diagnostics['stage'] = 'oauth_bootstrap'
        verifier, challenge = flow._generate_pkce()
        state = flow._generate_state()
        flow._bootstrap_authorize(session, state, challenge)
        diagnostics['stage'] = 'email'
        email_payload = flow._submit_email(session, data['email'])
        if flow._flow_page_type(email_payload) not in ('login_password', ''):
            raise LoginError('additional_verification_required')
        diagnostics['stage'] = 'password'
        password_payload = flow._submit_password(session, data['password'])
        factor = flow._extract_totp_factor_id(password_payload, flow._auth_session_payload(session))
        if not factor:
            raise LoginError('totp_factor_missing')
        remaining = 30 - time.time() % 30
        if remaining < 5:
            time.sleep(remaining + .1)
        diagnostics['stage'] = 'totp'
        mfa = flow._submit_mfa_totp(session, pyotp.TOTP(data['totp_secret']).now(), factor)
        auth_payload = mfa.get('oai-client-auth-session')
        if isinstance(auth_payload, str):
            auth_payload = flow._decode_jwt_segment(auth_payload.split('.')[0])
        if not isinstance(auth_payload, dict) or not auth_payload.get('workspaces'):
            auth_payload = flow._auth_session_payload(session)
        if not auth_payload.get('workspaces'):
            continuation = mfa.get('continue_url')
            if not continuation:
                raise LoginError('workspace_unavailable')
            session.get(urljoin('https://auth.openai.com', continuation))
            auth_payload = flow._auth_session_payload(session)
        diagnostics['stage'] = 'workspace'
        workspace = select_workspace(auth_payload.get('workspaces') or [], data.get('account_id'))
        flow._get_workspace_id = lambda _: workspace['id']
        callback = flow._select_workspace_and_get_callback(session, state)
        code = flow._extract_code(callback, state)
        diagnostics['stage'] = 'token_exchange'
        tokens = flow.exchange_codex_token(session, code, verifier)
        diagnostics['stage'] = 'identity_validation'
        identity = validate_tokens(tokens, data['email'], workspace['id'])
        return {'success': True, 'tokens': tokens, 'account_id': workspace['id'],
                'workspace_kind': workspace.get('kind'), 'workspace_name': workspace.get('name', ''),
                'plan_type': identity.get('https://api.openai.com/auth', {}).get('chatgpt_plan_type', '')}
    finally:
        session.session.close()


if __name__ == '__main__':
    try:
        payload = json.loads(sys.stdin.buffer.read(16384))
        # Reference code logs URLs and snippets. Neither stdout nor stderr escapes this boundary.
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            result = authorize(payload)
    except LoginError as exc:
        result = {'success': False, 'code': exc.code, 'diagnostics': safe_diagnostics(diagnostics)}
    except Exception as exc:
        diagnostics['exception_type'] = type(exc).__name__
        result = {'success': False, 'code': 'login_failed', 'diagnostics': safe_diagnostics(diagnostics)}
    sys.stdout.write(json.dumps(result))
