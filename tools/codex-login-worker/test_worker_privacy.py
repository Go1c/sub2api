import json
from pathlib import Path
import sys
from types import SimpleNamespace
from unittest.mock import patch


def test_node_cookie_is_not_in_process_arguments():
    sys.path.insert(0, str(Path(__file__).parent / 'third_party/codex_protocol'))
    from core.sentinel_runner import generate_sentinel_token
    token = json.dumps({'p': 'p', 'c': 'c', 'id': 'device', 'flow': 'mfa_verify'})
    with patch('core.sentinel_runner.subprocess.run', return_value=SimpleNamespace(returncode=0, stdout=token, stderr='')) as run:
        generate_sentinel_token({}, 'mfa_verify', 'device', cookie='login_session=private-session-cookie')
    assert 'private-session-cookie' not in ' '.join(run.call_args.args[0])
    assert run.call_args.kwargs['input'] == 'login_session=private-session-cookie'


def test_http_worker_preserves_only_allowlisted_failure_diagnostics():
    from login_core import LoginError
    from server import failure_response
    error = LoginError('credential_probe_failed', {
        'stage': 'credential_probe', 'http_status': 403, 'exception_type': 'Timeout',
        'endpoint': '/oauth/token?code=SECRET', 'body': 'SECRET', 'password': 'SECRET',
    })
    result = failure_response(error)
    assert result['diagnostics'] == {'stage': 'credential_probe', 'http_status': 403, 'exception_type': 'Timeout'}
    assert 'SECRET' not in json.dumps(result)
    assert failure_response(RuntimeError('SECRET')) == {'success': False, 'code': 'login_failed', 'diagnostics': {}}


def test_usage_probe_returns_exact_status_without_response_body():
    import asyncio
    from unittest.mock import AsyncMock
    import server
    from login_core import LoginError
    import pytest
    response = SimpleNamespace(status_code=401, text='PRIVATE-TOKEN')
    client = AsyncMock()
    client.get.return_value = response
    context = AsyncMock()
    context.__aenter__.return_value = client
    with patch.object(server, 'login', new=AsyncMock(return_value={
        'tokens': {'access_token': 'PRIVATE-TOKEN'}, 'account_id': 'workspace',
    })), patch.object(server, 'AsyncSession', return_value=context), \
         patch.object(server, 'select_proxy', new=AsyncMock(return_value={'id': 2, 'url': 'http://proxy:80'})):
        with pytest.raises(LoginError) as failure:
            asyncio.run(server.run({'email': 'demo@example.com', 'password': 'private', 'totp_secret': 'test'}))
    assert failure.value.diagnostics == {'stage': 'credential_probe', 'http_status': 401, 'proxy_id': 2}


def test_http_failure_reply_carries_safe_stage_end_to_end():
    import threading
    import urllib.request
    from http.server import ThreadingHTTPServer
    from unittest.mock import AsyncMock
    import server
    from login_core import LoginError
    http = ThreadingHTTPServer(('127.0.0.1', 0), server.Handler)
    thread = threading.Thread(target=http.serve_forever, daemon=True)
    with patch.object(server, 'TOKEN', 'test-token'), patch.object(server, 'run', new=AsyncMock(side_effect=LoginError(
        'login_failed', {'stage': 'workspace', 'exception_type': 'RuntimeError', 'body': 'PRIVATE-TOKEN'}
    ))):
        thread.start()
        try:
            request = urllib.request.Request(
                f'http://127.0.0.1:{http.server_port}/login',
                data=json.dumps({'email': 'demo@example.com', 'password': 'PRIVATE-TOKEN', 'totp_secret': 'PRIVATE-TOKEN'}).encode(),
                headers={'Authorization': 'Bearer test-token', 'Content-Type': 'application/json'},
            )
            opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
            with opener.open(request, timeout=5) as response:
                body = response.read().decode()
            assert 'PRIVATE-TOKEN' not in body
            assert json.loads(body)['diagnostics'] == {'stage': 'workspace', 'exception_type': 'RuntimeError'}
        finally:
            http.shutdown()
            http.server_close()
            thread.join()
